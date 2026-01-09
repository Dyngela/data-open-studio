package models

import (
	"fmt"
)

// MapGenerator handles data transformation operations
type MapGenerator struct {
	BaseGenerator
	config MapConfig
}

// NewMapGenerator creates a new map generator
func NewMapGenerator(nodeID int, config MapConfig) *MapGenerator {
	return &MapGenerator{
		BaseGenerator: BaseGenerator{
			nodeID:   nodeID,
			nodeType: NodeTypeMap,
		},
		config: config,
	}
}

// Execute transforms data in the context
func (g *MapGenerator) Execute(ctx *ExecutionContext) error {
	// Get input data from context
	var inputData []map[string]interface{}
	for key, value := range ctx.DataFlow {
		if data, ok := value.([]map[string]interface{}); ok {
			inputData = data
			fmt.Printf("Mapping data from %s\n", key)
			break
		}
	}

	if len(inputData) == 0 {
		return fmt.Errorf("no input data found in context for mapping")
	}

	// TODO: Implement actual mapping logic based on config
	// For now, just pass through the data
	key := fmt.Sprintf("node_%d_output", g.nodeID)
	ctx.DataFlow[key] = inputData

	return nil
}
