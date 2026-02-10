package gen

import (
	"api/internal/api/models"
	"fmt"
)

// NodeGenerator generates code data for a specific node type
type NodeGenerator interface {
	// NodeType returns the type of node this generator handles
	NodeType() models.NodeType

	// GenerateStructData generates the struct data for this node (if applicable)
	GenerateStructData(node *models.Node) (*StructData, error)

	// GenerateFuncData generates the function data for this node
	GenerateFuncData(node *models.Node, ctx *GeneratorContext) (*NodeFunctionData, error)

	// GetLaunchArgs returns the launch arguments for this node (db connections, channels, etc.)
	GetLaunchArgs(node *models.Node, channels []channelInfo, dbConnections map[string]string) []string
}

// channelInfo is exposed for generators
type ChannelInfo struct {
	PortID     uint
	FromNodeID int
	ToNodeID   int
	RowType    string
	BufferSize int
}

// GeneratorContext holds context during code generation
type GeneratorContext struct {
	// NodeStructNames maps node ID to generated struct name
	NodeStructNames map[int]string

	// NodeFuncNames maps node ID to generated function name
	NodeFuncNames map[int]string

	// Imports collects all imports needed
	Imports map[string]string // path -> alias (empty string for no alias)
}

// NewGeneratorContext creates a new generator context
func NewGeneratorContext() *GeneratorContext {
	return &GeneratorContext{
		NodeStructNames: make(map[int]string),
		NodeFuncNames:   make(map[int]string),
		Imports:         make(map[string]string),
	}
}

// AddImport adds an import to the context
func (ctx *GeneratorContext) AddImport(path string) {
	if _, exists := ctx.Imports[path]; !exists {
		ctx.Imports[path] = ""
	}
}

// AddImportAlias adds an aliased import
func (ctx *GeneratorContext) AddImportAlias(alias, path string) {
	ctx.Imports[path] = alias
}

// StructName returns or generates a struct name for a node
func (ctx *GeneratorContext) StructName(node *models.Node) string {
	if name, exists := ctx.NodeStructNames[node.ID]; exists {
		return name
	}
	name := fmt.Sprintf("Node%dRow", node.ID)
	ctx.NodeStructNames[node.ID] = name
	return name
}

// FuncName returns or generates a function name for a node
func (ctx *GeneratorContext) FuncName(node *models.Node) string {
	if name, exists := ctx.NodeFuncNames[node.ID]; exists {
		return name
	}
	name := fmt.Sprintf("executeNode%d", node.ID)
	ctx.NodeFuncNames[node.ID] = name
	return name
}

// uniqueFieldNames returns deduplicated Go field names for a list of DataModels.
// If two columns have the same name (e.g. "id" from two joined tables),
// the duplicates get a numeric suffix: Id, Id2, Id3, etc.
func uniqueFieldNames(cols []models.DataModel) []string {
	names := make([]string, len(cols))
	seen := make(map[string]int)

	for i, col := range cols {
		name := col.GoFieldName()
		seen[name]++
		if seen[name] > 1 {
			names[i] = fmt.Sprintf("%s%d", name, seen[name])
		} else {
			names[i] = name
		}
	}
	return names
}

// Registry holds all registered generators
type Registry struct {
	generators map[models.NodeType]NodeGenerator
}

// NewRegistry creates a new generator registry
func NewRegistry() *Registry {
	return &Registry{
		generators: make(map[models.NodeType]NodeGenerator),
	}
}

// Register registers a generator for a node type
func (r *Registry) Register(gen NodeGenerator) {
	r.generators[gen.NodeType()] = gen
}

// Get returns the generator for a node type
func (r *Registry) Get(nodeType models.NodeType) (NodeGenerator, bool) {
	gen, ok := r.generators[nodeType]
	return gen, ok
}

// DefaultRegistry is the default generator registry
var DefaultRegistry = NewRegistry()

// RegisterGenerator registers a generator with the default registry
func RegisterGenerator(gen NodeGenerator) {
	DefaultRegistry.Register(gen)
}

// init registers all built-in generators
func init() {
	RegisterGenerator(&DBInputGenerator{})
	RegisterGenerator(&DBOutputGenerator{})
	RegisterGenerator(&MapGenerator{})
	RegisterGenerator(&LogGenerator{})
	RegisterGenerator(&EmailOutputGenerator{})
}
