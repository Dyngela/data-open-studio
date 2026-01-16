package gen

import (
	"api/internal/api/models"
	"fmt"
)

// MapGenerator handles data transformation with sync point support for cross-data operations
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

// GenerateFunctionSignature returns signature accepting variadic input datasets for cross-data
func (g *MapGenerator) GenerateFunctionSignature() string {
	nodeName := sanitizeNodeName(g.nodeName)
	// Accepts variadic slices for multiple input sources (sync point)
	return fmt.Sprintf("func executeNode_%d_%s(ctx *JobContext, inputs ...[]map[string]interface{}) (*RowStream, error)",
		g.nodeID, nodeName)
}

// GenerateFunctionBody returns the function body with streaming output
func (g *MapGenerator) GenerateFunctionBody() string {
	bufferSize := 1000 // Default buffer for output stream

	return fmt.Sprintf(`	// Calculate total input size
	totalInputRows := 0
	for _, input := range inputs {
		totalInputRows += len(input)
	}
	log.Printf("Node %d: Processing %%d input source(s), %%d total rows", len(inputs), totalInputRows)

	// Create output stream
	stream := NewRowStream(%d)

	go func() {
		defer stream.Close()

		// Process and transform data
		outputCount := 0

		// Call transformation function
		results := transformData_%d(inputs)

		// Stream results
		for _, row := range results {
			if !stream.Send(row) {
				return // Stream closed
			}
			outputCount++
		}

		log.Printf("Node %d: Transformed %%d -> %%d rows", totalInputRows, outputCount)
	}()

	return stream, nil`, g.nodeID, bufferSize, g.nodeID, g.nodeID)
}

// GenerateHelperFunctions returns the transformation helper
func (g *MapGenerator) GenerateHelperFunctions() string {
	return fmt.Sprintf(`// transformData_%d processes multiple input datasets
// Customize this function for your cross-data transformation logic
func transformData_%d(inputs [][]map[string]interface{}) []map[string]interface{} {
	// Handle single input - simple passthrough with optional transform
	if len(inputs) == 1 {
		result := make([]map[string]interface{}, 0, len(inputs[0]))
		for _, row := range inputs[0] {
			// Apply transformation here
			result = append(result, row)
		}
		return result
	}

	// Handle multiple inputs - cross-data operations
	// Example: Join, merge, or combine datasets

	// Calculate output capacity
	totalRows := 0
	for _, input := range inputs {
		totalRows += len(input)
	}
	result := make([]map[string]interface{}, 0, totalRows)

	// Default: Concatenate all inputs
	// TODO: Replace with your actual cross-data logic (joins, lookups, etc.)
	for inputIdx, input := range inputs {
		for _, row := range input {
			// Tag source for debugging
			row["_source_input"] = inputIdx
			result = append(result, row)
		}
	}

	// Example cross-data patterns:
	//
	// 1. Inner Join by key:
	// input0Map := make(map[interface{}]map[string]interface{})
	// for _, row := range inputs[0] {
	//     key := row["join_key"]
	//     input0Map[key] = row
	// }
	// for _, row := range inputs[1] {
	//     if matchRow, ok := input0Map[row["join_key"]]; ok {
	//         merged := make(map[string]interface{})
	//         for k, v := range matchRow { merged[k] = v }
	//         for k, v := range row { merged[k] = v }
	//         result = append(result, merged)
	//     }
	// }
	//
	// 2. Lookup enrichment:
	// lookup := make(map[interface{}]map[string]interface{})
	// for _, row := range inputs[1] {
	//     lookup[row["lookup_key"]] = row
	// }
	// for _, row := range inputs[0] {
	//     if extra, ok := lookup[row["lookup_key"]]; ok {
	//         row["extra_field"] = extra["value"]
	//     }
	//     result = append(result, row)
	// }

	return result
}`, g.nodeID, g.nodeID)
}

func (g *MapGenerator) GenerateImports() []string {
	return []string{
		`"fmt"`,
	}
}
