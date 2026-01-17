package gen

import (
	"api/internal/api/models"
	"api/internal/gen/ir"
	"bytes"
	"fmt"
	"go/format"
)

// FileBuilder generates a complete Go file for a job
type FileBuilder struct {
	job           *models.Job
	ctx           *GeneratorContext
	structs       []*ir.StructDecl
	funcs         []*ir.FuncDecl
	nodeIDs       map[int]bool // nodes to include in generation
	dbConnections []models.DBConnectionConfig
	steps         []Step
	nodeByID      map[int]*models.Node
}

// NewFileBuilder creates a new file builder
func NewFileBuilder(job *models.Job) *FileBuilder {
	nodeByID := make(map[int]*models.Node)
	for i := range job.Nodes {
		nodeByID[job.Nodes[i].ID] = &job.Nodes[i]
	}
	return &FileBuilder{
		job:           job,
		ctx:           NewGeneratorContext(),
		structs:       make([]*ir.StructDecl, 0),
		funcs:         make([]*ir.FuncDecl, 0),
		nodeIDs:       make(map[int]bool),
		dbConnections: make([]models.DBConnectionConfig, 0),
		nodeByID:      nodeByID,
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

// Build generates all code for the job
func (b *FileBuilder) Build() error {
	for i := range b.job.Nodes {
		node := &b.job.Nodes[i]

		// Skip nodes not in the pipeline (if filter is set)
		if len(b.nodeIDs) > 0 && !b.nodeIDs[node.ID] {
			continue
		}

		gen, ok := DefaultRegistry.Get(node.Type)
		if !ok {
			// Skip nodes without generators (like start nodes)
			continue
		}

		// Generate struct
		s, err := gen.GenerateStruct(node)
		if err != nil {
			return fmt.Errorf("failed to generate struct for node %d: %w", node.ID, err)
		}
		if s != nil {
			b.structs = append(b.structs, s)
		}

		// Generate function
		fn, err := gen.GenerateFunc(node, b.ctx)
		if err != nil {
			return fmt.Errorf("failed to generate func for node %d: %w", node.ID, err)
		}
		if fn != nil {
			b.funcs = append(b.funcs, fn)
		}
	}

	return nil
}

// EmitFile generates the complete Go source file
func (b *FileBuilder) EmitFile(pkgName string) ([]byte, error) {
	file := ir.NewFile(pkgName)

	// Generate main executor function first (may add imports)
	executorFunc := b.generateMainFunc()

	// Add imports
	for path, alias := range b.ctx.Imports {
		if alias != "" {
			file.ImportAlias(alias, path)
		} else {
			file.Import(path)
		}
	}

	// Add structs
	for _, s := range b.structs {
		file.AddStruct(s)
	}

	// Add functions
	for _, fn := range b.funcs {
		file.AddFunc(fn)
	}

	file.AddFunc(executorFunc)

	file.AddFunc(b.generateEntrypoint())

	// Emit to buffer
	var buf bytes.Buffer
	if err := ir.EmitFile(&buf, file.Build()); err != nil {
		return nil, fmt.Errorf("failed to emit file: %w", err)
	}

	// Format the code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Return unformatted on error (for debugging)
		return buf.Bytes(), fmt.Errorf("failed to format: %w (raw output available)", err)
	}

	return formatted, nil
}

// generateMainFunc generates the main execution function that orchestrates the pipeline
func (b *FileBuilder) generateMainFunc() *ir.FuncDecl {
	b.ctx.AddImport("context")
	b.ctx.AddImport("sync")
	b.ctx.AddImport("database/sql")
	b.ctx.AddImport("fmt")

	body := make([]ir.Stmt, 0)

	// 1. Open database connections
	for _, conn := range b.dbConnections {
		connID := conn.GetConnectionID()
		driverName := conn.GetDriverName()
		connString := conn.BuildConnectionString()

		// db_<connID>, err := sql.Open("<driver>", "<connString>")
		body = append(body,
			ir.DefineMulti(
				[]ir.Expr{ir.Id(fmt.Sprintf("db_%s", connID)), ir.Id("err")},
				[]ir.Expr{ir.Call("sql.Open", ir.Lit(driverName), ir.Lit(connString))},
			),
			ir.If(ir.Neq(ir.Id("err"), ir.Nil()),
				ir.Return(ir.Call("fmt.Errorf",
					ir.Lit(fmt.Sprintf("failed to connect to %s: %%w", connID)),
					ir.Id("err"),
				)),
			),
			ir.Defer(ir.Call(fmt.Sprintf("db_%s.Close", connID))),
		)
	}

	// 2. Create channels for data flow
	// Collect all output ports that need channels
	channelInfos := b.collectChannels()
	for _, ch := range channelInfos {
		// ch_<portID> := make(chan *Node<X>Row, bufferSize)
		body = append(body,
			ir.Define(
				ir.Id(fmt.Sprintf("ch_%d", ch.portID)),
				ir.Call("make", ir.Raw(fmt.Sprintf("chan *%s", ch.rowType)), ir.Lit(ch.bufferSize)),
			),
		)
	}

	// 3. Create wait group and error channel
	body = append(body,
		ir.Var("wg", "sync.WaitGroup"),
		ir.Define(ir.Id("errChan"), ir.Call("make", ir.Raw("chan error"), ir.Lit(len(b.funcs)))),
	)

	// 4. Launch goroutines for each node
	for _, step := range b.steps {
		for _, node := range step.nodes {
			if node.Type == models.NodeTypeStart {
				continue
			}

			stmts := b.generateNodeLaunch(&node, channelInfos)
			body = append(body, stmts...)
		}
	}

	// 5. Wait for completion in a goroutine, then close error channel
	body = append(body,
		ir.Go(ir.ClosureCall(nil, nil,
			ir.ExprStatement(ir.Call("wg.Wait")),
			ir.ExprStatement(ir.Call("close", ir.Id("errChan"))),
		)),
	)

	// 6. Collect errors
	body = append(body,
		ir.Var("firstErr", "error"),
		ir.RangeValue("err", ir.Id("errChan"),
			ir.If(ir.And(ir.Neq(ir.Id("err"), ir.Nil()), ir.Eq(ir.Id("firstErr"), ir.Nil())),
				ir.Assign(ir.Id("firstErr"), ir.Id("err")),
			),
		),
		ir.Return(ir.Id("firstErr")),
	)

	return ir.NewFunc("Execute").
		Param("ctx", "context.Context").
		Returns("error").
		Body(body...).
		Build()
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
				// port.Node is the destination node, port.NodeID is the owner (source) node
				toNodeID := port.Node.ID
				channels = append(channels, channelInfo{
					portID:     port.ID,
					fromNodeID: node.ID,
					toNodeID:   toNodeID,
					rowType:    b.ctx.StructName(&node),
					bufferSize: 1000, // default buffer size
				})
			}
		}
	}

	return channels
}

// generateNodeLaunch generates the goroutine launch for a node
func (b *FileBuilder) generateNodeLaunch(node *models.Node, channels []channelInfo) []ir.Stmt {
	stmts := make([]ir.Stmt, 0)

	switch node.Type {
	case models.NodeTypeDBInput:
		stmts = append(stmts, b.generateDBInputLaunch(node, channels)...)
	case models.NodeTypeDBOutput:
		stmts = append(stmts, b.generateDBOutputLaunch(node, channels)...)
	case models.NodeTypeMap:
		stmts = append(stmts, b.generateMapLaunch(node, channels)...)
	}

	return stmts
}

// generateDBInputLaunch generates goroutine launch for db_input node
func (b *FileBuilder) generateDBInputLaunch(node *models.Node, channels []channelInfo) []ir.Stmt {
	config, err := node.GetDBInputConfig()
	if err != nil {
		return nil
	}

	funcName := b.ctx.FuncName(node)
	connID := config.Connection.GetConnectionID()
	dbVar := fmt.Sprintf("db_%s", connID)

	// Find output channel for this node
	var outChanVar string
	for _, ch := range channels {
		if ch.fromNodeID == node.ID {
			outChanVar = fmt.Sprintf("ch_%d", ch.portID)
			break
		}
	}

	if outChanVar == "" {
		// No output channel, node might be a sink or error
		return nil
	}

	// wg.Add(1)
	// go func() {
	//     defer wg.Done()
	//     defer close(ch_X)
	//     if err := executeNodeX(ctx, db_Y, ch_X); err != nil {
	//         errChan <- err
	//     }
	// }()
	return []ir.Stmt{
		ir.ExprStatement(ir.Call("wg.Add", ir.Lit(1))),
		ir.Go(ir.ClosureCall(nil, nil,
			ir.Defer(ir.Call("wg.Done")),
			ir.Defer(ir.Call("close", ir.Id(outChanVar))),
			ir.IfInit(
				ir.Define(ir.Id("err"), ir.Call(funcName, ir.Id("ctx"), ir.Id(dbVar), ir.Id(outChanVar))),
				ir.Neq(ir.Id("err"), ir.Nil()),
				ir.Send(ir.Id("errChan"), ir.Id("err")),
			),
		)),
	}
}

// generateDBOutputLaunch generates goroutine launch for db_output node
func (b *FileBuilder) generateDBOutputLaunch(node *models.Node, channels []channelInfo) []ir.Stmt {
	config, err := node.GetDBOutputConfig()
	if err != nil {
		return nil
	}

	funcName := b.ctx.FuncName(node)
	connID := config.Connection.GetConnectionID()
	dbVar := fmt.Sprintf("db_%s", connID)

	// Find input channel for this node (from the source node)
	var inChanVar string
	var inputRowType string
	for _, port := range node.InputPort {
		if port.Type == models.PortTypeInput {
			// Find the channel that connects to this port
			for _, ch := range channels {
				if ch.toNodeID == node.ID {
					inChanVar = fmt.Sprintf("ch_%d", ch.portID)
					inputRowType = ch.rowType
					break
				}
			}
			break
		}
	}

	if inChanVar == "" {
		// No input channel found
		return nil
	}

	_ = inputRowType // Used by the generated function signature

	// wg.Add(1)
	// go func() {
	//     defer wg.Done()
	//     if err := executeNodeX(ctx, db_Y, ch_in); err != nil {
	//         errChan <- err
	//     }
	// }()
	return []ir.Stmt{
		ir.ExprStatement(ir.Call("wg.Add", ir.Lit(1))),
		ir.Go(ir.ClosureCall(nil, nil,
			ir.Defer(ir.Call("wg.Done")),
			ir.IfInit(
				ir.Define(ir.Id("err"), ir.Call(funcName, ir.Id("ctx"), ir.Id(dbVar), ir.Id(inChanVar))),
				ir.Neq(ir.Id("err"), ir.Nil()),
				ir.Send(ir.Id("errChan"), ir.Id("err")),
			),
		)),
	}
}

// generateMapLaunch generates goroutine launch for map node
func (b *FileBuilder) generateMapLaunch(node *models.Node, channels []channelInfo) []ir.Stmt {
	config, err := node.GetMapConfig()
	if err != nil {
		return nil
	}

	funcName := b.ctx.FuncName(node)

	// Find output channel for this node
	var outChanVar string
	for _, ch := range channels {
		if ch.fromNodeID == node.ID {
			outChanVar = fmt.Sprintf("ch_%d", ch.portID)
			break
		}
	}

	if outChanVar == "" {
		return nil
	}

	if len(config.Inputs) == 1 {
		// Single input map
		return b.generateSingleInputMapLaunch(node, &config, funcName, outChanVar, channels)
	}

	// Multiple inputs (join)
	return b.generateJoinMapLaunch(node, &config, funcName, outChanVar, channels)
}

// generateSingleInputMapLaunch generates launch for single-input map
func (b *FileBuilder) generateSingleInputMapLaunch(node *models.Node, config *models.MapConfig, funcName, outChanVar string, channels []channelInfo) []ir.Stmt {
	// Find input channel
	var inChanVar string
	for _, port := range node.InputPort {
		if port.Type == models.PortTypeInput {
			for _, ch := range channels {
				if ch.toNodeID == node.ID {
					inChanVar = fmt.Sprintf("ch_%d", ch.portID)
					break
				}
			}
			break
		}
	}

	if inChanVar == "" {
		return nil
	}

	// wg.Add(1)
	// go func() {
	//     defer wg.Done()
	//     defer close(outChan)
	//     if err := executeNodeX(ctx, in, outChan); err != nil {
	//         errChan <- err
	//     }
	// }()
	return []ir.Stmt{
		ir.ExprStatement(ir.Call("wg.Add", ir.Lit(1))),
		ir.Go(ir.ClosureCall(nil, nil,
			ir.Defer(ir.Call("wg.Done")),
			ir.Defer(ir.Call("close", ir.Id(outChanVar))),
			ir.IfInit(
				ir.Define(ir.Id("err"), ir.Call(funcName, ir.Id("ctx"), ir.Id(inChanVar), ir.Id(outChanVar))),
				ir.Neq(ir.Id("err"), ir.Nil()),
				ir.Send(ir.Id("errChan"), ir.Id("err")),
			),
		)),
	}
}

// generateJoinMapLaunch generates launch for multi-input map (join)
func (b *FileBuilder) generateJoinMapLaunch(node *models.Node, config *models.MapConfig, funcName, outChanVar string, channels []channelInfo) []ir.Stmt {
	if config.Join == nil {
		return nil
	}

	// Find left and right input channels based on port IDs in config
	var leftChanVar, rightChanVar string

	leftInput := config.GetInputByName(config.Join.LeftInput)
	rightInput := config.GetInputByName(config.Join.RightInput)

	if leftInput == nil || rightInput == nil {
		return nil
	}

	// Find channels by matching port IDs
	for _, ch := range channels {
		if ch.toNodeID == node.ID {
			// Check which input this channel corresponds to
			for _, port := range node.InputPort {
				if port.Type == models.PortTypeInput && ch.portID == port.ID {
					if int(port.ID) == leftInput.PortID {
						leftChanVar = fmt.Sprintf("ch_%d", ch.portID)
					} else if int(port.ID) == rightInput.PortID {
						rightChanVar = fmt.Sprintf("ch_%d", ch.portID)
					}
				}
			}
		}
	}

	// Fallback: if we couldn't match by port ID, use first two incoming channels
	if leftChanVar == "" || rightChanVar == "" {
		inChans := make([]string, 0)
		for _, ch := range channels {
			if ch.toNodeID == node.ID {
				inChans = append(inChans, fmt.Sprintf("ch_%d", ch.portID))
			}
		}
		if len(inChans) >= 2 {
			leftChanVar = inChans[0]
			rightChanVar = inChans[1]
		}
	}

	if leftChanVar == "" || rightChanVar == "" {
		return nil
	}

	// wg.Add(1)
	// go func() {
	//     defer wg.Done()
	//     defer close(outChan)
	//     if err := executeNodeX(ctx, leftIn, rightIn, outChan); err != nil {
	//         errChan <- err
	//     }
	// }()
	return []ir.Stmt{
		ir.ExprStatement(ir.Call("wg.Add", ir.Lit(1))),
		ir.Go(ir.ClosureCall(nil, nil,
			ir.Defer(ir.Call("wg.Done")),
			ir.Defer(ir.Call("close", ir.Id(outChanVar))),
			ir.IfInit(
				ir.Define(ir.Id("err"), ir.Call(funcName, ir.Id("ctx"), ir.Id(leftChanVar), ir.Id(rightChanVar), ir.Id(outChanVar))),
				ir.Neq(ir.Id("err"), ir.Nil()),
				ir.Send(ir.Id("errChan"), ir.Id("err")),
			),
		)),
	}
}

// generateEntrypoint generates the main function that calls Execute
func (b *FileBuilder) generateEntrypoint() *ir.FuncDecl {
	b.ctx.AddImport("context")
	b.ctx.AddImport("log")
	b.ctx.AddImport("os")

	return ir.NewFunc("main").
		Body(
			ir.Define(ir.Id("ctx"), ir.Call("context.Background")),
			ir.IfInit(
				ir.Define(ir.Id("err"), ir.Call("Execute", ir.Id("ctx"))),
				ir.Neq(ir.Id("err"), ir.Nil()),
				ir.ExprStatement(ir.Call("log.Fatalf", ir.Lit("execution failed: %v"), ir.Id("err"))),
			),
			ir.ExprStatement(ir.Call("log.Println", ir.Lit("Pipeline completed successfully"))),
		).
		Build()
}

// GetContext returns the generator context
func (b *FileBuilder) GetContext() *GeneratorContext {
	return b.ctx
}

// GetStructs returns all generated structs
func (b *FileBuilder) GetStructs() []*ir.StructDecl {
	return b.structs
}

// GetFuncs returns all generated functions
func (b *FileBuilder) GetFuncs() []*ir.FuncDecl {
	return b.funcs
}
