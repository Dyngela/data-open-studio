package models

type ExecutionContext struct {
	// Data passed between nodes
	DataFlow map[string]interface{}
	// Errors encountered during execution
	Errors []error
}

// NewExecutionContext creates a new execution context
func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{
		DataFlow: make(map[string]interface{}),
		Errors:   make([]error, 0),
	}
}

// Generator is the interface that all node generators must implement
type Generator interface {
	// Execute runs the generator logic with the given context
	Execute(ctx *ExecutionContext) error
	// GetType returns the node type this generator handles
	GetType() NodeType
	// GetNodeID returns the ID of the node this generator was created from
	GetNodeID() int
}

// BaseGenerator provides common functionality for all generators
type BaseGenerator struct {
	nodeID   int
	nodeType NodeType
}

func (g *BaseGenerator) GetType() NodeType {
	return g.nodeType
}

func (g *BaseGenerator) GetNodeID() int {
	return g.nodeID
}
