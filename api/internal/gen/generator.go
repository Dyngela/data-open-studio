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
	// GenerateCode generates a Go source file for this generator
	GenerateCode(ctx *ExecutionContext, outputPath string) error
}

// BaseGenerator provides common functionality for all generators
type BaseGenerator struct {
	nodeID   int
	nodeType models.NodeType
}

func (g *BaseGenerator) GetType() models.NodeType {
	return g.nodeType
}

func (g *BaseGenerator) GetNodeID() int {
	return g.nodeID
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
		return NewDBInputGenerator(node.ID, config), nil

	case models.NodeTypeDBOutput:
		config, err := node.GetDBOutputConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get DB output config for node %d: %w", node.ID, err)
		}
		return NewDBOutputGenerator(node.ID, config), nil

	case models.NodeTypeMap:
		config, err := node.GetMapConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get map config for node %d: %w", node.ID, err)
		}
		return NewMapGenerator(node.ID, config), nil

	case models.NodeTypeStart:
		// Start nodes don't have generators, they just mark the beginning
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown node type '%s' for node %d", node.Type, node.ID)
	}
}
