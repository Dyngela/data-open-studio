package gen

import (
	"api/internal/api/models"
)

// ExecutionContext holds shared state during code generation and execution
type ExecutionContext struct {
	// DBConnections holds all unique database connections needed by the pipeline
	DBConnections []models.DBConnectionConfig

	// GeneratorCtx holds code generation context
	GeneratorCtx *GeneratorContext
}

// NewExecutionContext creates a new execution context
func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{
		DBConnections: make([]models.DBConnectionConfig, 0),
		GeneratorCtx:  NewGeneratorContext(),
	}
}
