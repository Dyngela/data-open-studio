package gen

import (
	"api/internal/api/models"
	"database/sql"
	"fmt"
	"time"
)

type JobResult struct {
	Success   bool
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Errors    []error
	Output    map[string]interface{}
}

type DBConnection struct {
	Id   string
	Conn *sql.DB
}

type ExecutionContext struct {
	// Data passed between nodes
	DataFlow map[string]interface{}
	// Errors encountered during execution
	Errors []error
	// Database connection for executing queries
	DBConnections []models.DBConnectionConfig
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
	// GetType returns the node type this generator handles
	GetType() models.NodeType
	// GetNodeID returns the ID of the node this generator was created from
	GetNodeID() int
	// GetNodeName returns the name of the node
	GetNodeName() string
	// GenerateCode generates a Go source file for this generator (legacy)
	GenerateCode(ctx *ExecutionContext, outputPath string) error
	// GenerateFunctionSignature returns the function signature for this node
	GenerateFunctionSignature() string
	// GenerateFunctionBody returns the function body (without signature) for this node
	GenerateFunctionBody() string
	// GenerateHelperFunctions returns any helper functions that should be generated alongside
	// These are written as separate top-level functions, not nested inside the main function
	GenerateHelperFunctions() string
	// GenerateImports returns the list of imports needed for this node
	GenerateImports() []string
}

// BaseGenerator provides common functionality for all generators
type BaseGenerator struct {
	nodeID   int
	nodeType models.NodeType
	nodeName string
}

func (g *BaseGenerator) GetType() models.NodeType {
	return g.nodeType
}

func (g *BaseGenerator) GetNodeID() int {
	return g.nodeID
}

func (g *BaseGenerator) GetNodeName() string {
	return g.nodeName
}

// GenerateHelperFunctions returns an empty string by default (no helpers needed)
func (g *BaseGenerator) GenerateHelperFunctions() string {
	return ""
}

// NewGenerator creates the appropriate generator for a given node
// This is the recommended way to create generators from nodes.
func NewGenerator(node models.Node) (Generator, error) {
	switch node.Type {
	case models.NodeTypeDBInput:
		config, err := node.GetDBInputConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get DB input config for node %d: %w", node.ID, err)
		}
		return NewDBInputGenerator(node.ID, node.Name, config), nil

	case models.NodeTypeDBOutput:
		config, err := node.GetDBOutputConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get DB output config for node %d: %w", node.ID, err)
		}
		return NewDBOutputGenerator(node.ID, node.Name, config), nil

	case models.NodeTypeMap:
		config, err := node.GetMapConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get map config for node %d: %w", node.ID, err)
		}
		return NewMapGenerator(node.ID, node.Name, config), nil

	case models.NodeTypeStart:
		// Start nodes don't have generators, they just mark the beginning
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown node type '%s' for node %d", node.Type, node.ID)
	}
}

// sanitizeNodeName converts a node name to a valid Go identifier
// Removes/replaces special characters and ensures it starts with a letter
func sanitizeNodeName(name string) string {
	if name == "" {
		return fmt.Sprintf("Node_%d", time.Now().UnixNano())
	}

	result := ""
	for i, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9' && i > 0) {
			result += string(r)
		} else if r == '_' || r == '-' || r == ' ' {
			result += "_"
		}
	}

	if result == "" || (result[0] >= '0' && result[0] <= '9') {
		result = "Node_" + result
	}

	return result
}
