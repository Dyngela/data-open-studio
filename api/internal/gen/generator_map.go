package gen

import (
	"api/internal/api/models"
	"fmt"
)

// MapGenerator handles data transformation operations
type MapGenerator struct {
	BaseGenerator
	config models.MapConfig
}

// NewMapGenerator creates a new map generator
func NewMapGenerator(nodeID int, nodeName string, config models.MapConfig) *MapGenerator {
	return &MapGenerator{
		BaseGenerator: BaseGenerator{
			nodeID:   nodeID,
			nodeName: nodeName,
			nodeType: models.NodeTypeMap,
		},
		config: config,
	}
}

// GenerateFunctionSignature returns the function signature for this map node
func (g *MapGenerator) GenerateFunctionSignature() string {
	nodeName := sanitizeNodeName(g.nodeName)
	return fmt.Sprintf("func executeNode_%d_%s(ctx *JobContext, input []map[string]interface{}) ([]map[string]interface{}, error)",
		g.nodeID, nodeName)
}

// GenerateFunctionBody returns the function body for this map node
func (g *MapGenerator) GenerateFunctionBody() string {
	return fmt.Sprintf(`	log.Printf("Node %%d: Processing %%d rows through map node", %d, len(input))

	// Transform data
	transformedData := transformData_%d(input)
 
	return transformedData, nil`, g.nodeID, g.nodeID)
}

// GenerateHelperFunctions returns helper functions for this map node
func (g *MapGenerator) GenerateHelperFunctions() string {
	return fmt.Sprintf(`// transformData_%d applies transformations to the input data
func transformData_%d(data []map[string]interface{}) []map[string]interface{} {
	// TODO: Implement your transformation logic here
	// For now, this is a passthrough that returns data as-is

	// Example transformations you could implement:
	// - Filter rows based on conditions
	// - Add/remove fields
	// - Rename fields
	// - Calculate derived fields
	// - Aggregate data

	result := make([]map[string]interface{}, 0, len(data))

	for _, row := range data {
		// Example: Add a processed timestamp
		// row["processed_at"] = time.Now().Format(time.RFC3339)

		// Example: Transform field values
		// if val, ok := row["status"]; ok {
		//     row["status_upper"] = strings.ToUpper(val.(string))
		// }

		result = append(result, row)
	}

	return result
}`, g.nodeID, g.nodeID)
}

// GenerateImports returns the list of imports needed for this map node
func (g *MapGenerator) GenerateImports() []string {
	return []string{
		`"fmt"`,
	}
}
