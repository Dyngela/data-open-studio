package gen

import (
	"api/internal/api/models"
	"fmt"
	"strings"
)

// FileBuilder generates a complete Go file for a job using templates
type FileBuilder struct {
	job           *models.Job
	ctx           *GeneratorContext
	engine        *TemplateEngine
	nodeIDs       map[int]bool // nodes to include in generation
	dbConnections []models.DBConnectionConfig
	steps         []Step
	nodeByID      map[int]*models.Node

	// Template data
	templateData *TemplateData

	// Progress reporting config
	natsURL  string
	tenantID string
	jobID    uint
}

// NewFileBuilder creates a new file builder
func NewFileBuilder(job *models.Job) *FileBuilder {
	nodeByID := make(map[int]*models.Node)
	for i := range job.Nodes {
		nodeByID[job.Nodes[i].ID] = &job.Nodes[i]
	}

	engine, err := NewTemplateEngine()
	if err != nil {
		// Fallback to basic engine, shouldn't happen in practice
		panic(fmt.Sprintf("failed to create template engine: %v", err))
	}

	return &FileBuilder{
		job:           job,
		ctx:           NewGeneratorContext(),
		engine:        engine,
		nodeIDs:       make(map[int]bool),
		dbConnections: make([]models.DBConnectionConfig, 0),
		nodeByID:      nodeByID,
		templateData: &TemplateData{
			Imports:       make([]ImportData, 0),
			Structs:       make([]StructData, 0),
			NodeFunctions: make([]NodeFunctionData, 0),
			DBConnections: make([]DBConnectionData, 0),
			Channels:      make([]ChannelData, 0),
			NodeLaunches:  make([]NodeLaunchData, 0),
		},
	}
}

// SetNodes sets the specific node IDs to generate code for
func (b *FileBuilder) SetNodes(nodeIDs []int) {
	b.nodeIDs = make(map[int]bool)
	for _, id := range nodeIDs {
		b.nodeIDs[id] = true
	}
}

// SetDBConnections sets the database connections needed
func (b *FileBuilder) SetDBConnections(conns []models.DBConnectionConfig) {
	b.dbConnections = conns
}

// SetSteps sets the execution steps
func (b *FileBuilder) SetSteps(steps []Step) {
	b.steps = steps
}

// SetProgressConfig sets the NATS URL, tenant ID and job ID for progress reporting
func (b *FileBuilder) SetProgressConfig(natsURL, tenantID string, jobID uint) {
	b.natsURL = natsURL
	b.tenantID = tenantID
	b.jobID = jobID
}

// Build generates all code for the job
func (b *FileBuilder) Build() error {
	// Pass 1: Generate all structs first so NodeStructNames is fully populated
	for i := range b.job.Nodes {
		node := &b.job.Nodes[i]

		if len(b.nodeIDs) > 0 && !b.nodeIDs[node.ID] {
			continue
		}

		gen, ok := DefaultRegistry.Get(node.Type)
		if !ok {
			continue
		}

		structData, err := gen.GenerateStructData(node)
		if err != nil {
			return fmt.Errorf("failed to generate struct for node %d: %w", node.ID, err)
		}
		if structData != nil {
			b.templateData.Structs = append(b.templateData.Structs, *structData)
			b.ctx.NodeStructNames[node.ID] = structData.Name
		}
	}

	// Pass 2: Generate all functions (now all struct names are available)
	for i := range b.job.Nodes {
		node := &b.job.Nodes[i]

		if len(b.nodeIDs) > 0 && !b.nodeIDs[node.ID] {
			continue
		}

		gen, ok := DefaultRegistry.Get(node.Type)
		if !ok {
			continue
		}

		funcData, err := gen.GenerateFuncData(node, b.ctx)
		if err != nil {
			return fmt.Errorf("failed to generate func for node %d: %w", node.ID, err)
		}
		if funcData != nil {
			b.templateData.NodeFunctions = append(b.templateData.NodeFunctions, *funcData)
		}
	}

	// Auto-detect imports required by struct field types (e.g. time.Time)
	b.resolveStructImports()

	// Collect channels
	channels := b.collectChannels()
	for _, ch := range channels {
		b.templateData.Channels = append(b.templateData.Channels, ChannelData{
			PortID:     ch.portID,
			FromNodeID: ch.fromNodeID,
			ToNodeID:   ch.toNodeID,
			RowType:    ch.rowType,
			BufferSize: ch.bufferSize,
		})
	}

	// Collect DB connections
	for _, conn := range b.dbConnections {
		b.templateData.DBConnections = append(b.templateData.DBConnections, DBConnectionData{
			ID:         conn.GetConnectionID(),
			Driver:     conn.GetDriverName(),
			ConnString: conn.BuildConnectionString(),
		})
	}

	// Collect node launches
	for _, step := range b.steps {
		for _, node := range step.nodes {
			if node.Type == models.NodeTypeStart {
				continue
			}
			launchData := b.generateNodeLaunchData(&node, channels)
			if launchData != nil {
				b.templateData.NodeLaunches = append(b.templateData.NodeLaunches, *launchData)
			}
		}
	}

	// Collect imports
	for path, alias := range b.ctx.Imports {
		b.templateData.Imports = append(b.templateData.Imports, ImportData{
			Path:  path,
			Alias: alias,
		})
	}

	// Set progress config
	if b.natsURL != "" && b.tenantID != "" && b.jobID > 0 {
		b.templateData.UseFlags = false
		b.templateData.NatsURL = b.natsURL
		b.templateData.TenantID = b.tenantID
		b.templateData.JobID = b.jobID
	} else {
		b.templateData.UseFlags = true
	}

	b.templateData.NodeCount = len(b.templateData.NodeFunctions)

	return nil
}

// EmitFile generates the complete Go source file using templates
func (b *FileBuilder) EmitFile() ([]byte, error) {
	result, err := b.engine.GenerateMainFile(b.templateData)
	if err != nil {
		// Print raw output for debugging
		if strings.Contains(err.Error(), "failed to format") {
			fmt.Println("========== RAW GENERATED CODE (BEFORE FORMAT) ==========")
			fmt.Println(string(result))
			fmt.Println("========== END RAW CODE ==========")
		}
	}
	return result, err
}

// channelInfo holds info about a channel between nodes
type channelInfo struct {
	portID     uint
	fromNodeID int
	toNodeID   int
	rowType    string
	bufferSize int
}

// collectChannels collects all channels needed for data flow
func (b *FileBuilder) collectChannels() []channelInfo {
	channels := make([]channelInfo, 0)
	seen := make(map[uint]bool)

	for _, node := range b.job.Nodes {
		if !b.nodeIDs[node.ID] {
			continue
		}

		for _, port := range node.OutputPort {
			if port.Type == models.PortTypeOutput && !seen[port.ID] {
				seen[port.ID] = true
				toNodeID := port.ConnectedNodeID
				channels = append(channels, channelInfo{
					portID:     port.ID,
					fromNodeID: node.ID,
					toNodeID:   int(toNodeID),
					rowType:    b.ctx.StructName(&node),
					bufferSize: 1000, // default buffer size
				})
			}
		}
	}

	return channels
}

// generateNodeLaunchData generates launch data for a node using the generator interface
func (b *FileBuilder) generateNodeLaunchData(node *models.Node, channels []channelInfo) *NodeLaunchData {
	funcName := b.ctx.FuncName(node)

	// Get the generator for this node type
	gen, ok := DefaultRegistry.Get(node.Type)
	if !ok {
		return nil
	}

	// Build DB connections map
	dbConns := make(map[string]string)
	for _, conn := range b.dbConnections {
		dbConns[conn.GetConnectionID()] = fmt.Sprintf("db_%s", conn.GetConnectionID())
	}

	// Let the generator determine its own args
	args := gen.GetLaunchArgs(node, channels, dbConns)

	if len(args) == 0 {
		return nil
	}

	// Determine output channel (if any)
	hasOutput := false
	outputChan := ""
	for _, ch := range channels {
		if ch.fromNodeID == node.ID {
			outputChan = fmt.Sprintf("ch_%d", ch.portID)
			hasOutput = true
			break
		}
	}

	return &NodeLaunchData{
		NodeID:           node.ID,
		NodeName:         node.Name,
		FuncName:         funcName,
		Args:             args,
		HasOutputChannel: hasOutput,
		OutputChannel:    outputChan,
	}
}

// resolveStructImports scans struct field types and adds missing imports
func (b *FileBuilder) resolveStructImports() {
	// Map of type substring → required import path
	typeImports := map[string]string{
		"time.": "time",
		"sql.":  "database/sql",
	}

	for _, s := range b.templateData.Structs {
		for _, f := range s.Fields {
			for prefix, importPath := range typeImports {
				if strings.Contains(f.Type, prefix) {
					b.ctx.AddImport(importPath)
				}
			}
		}
	}
}

// GetContext returns the generator context
func (b *FileBuilder) GetContext() *GeneratorContext {
	return b.ctx
}
