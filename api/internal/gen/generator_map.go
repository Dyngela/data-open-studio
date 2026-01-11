package gen

import (
	"api/internal/api/models"
	"fmt"
	"os"
	"text/template"
)

// MapGenerator handles data transformation operations
type MapGenerator struct {
	BaseGenerator
	config models.MapConfig
}

// NewMapGenerator creates a new map generator
func NewMapGenerator(nodeID int, config models.MapConfig) *MapGenerator {
	return &MapGenerator{
		BaseGenerator: BaseGenerator{
			nodeID:   nodeID,
			nodeType: models.NodeTypeMap,
		},
		config: config,
	}
}

// GenerateCode generates a standalone Go file for this map generator
func (g *MapGenerator) GenerateCode(ctx *ExecutionContext, outputPath string) error {
	tmpl := template.Must(template.New("map").Parse(mapTemplate))

	data := map[string]interface{}{
		"NodeID": g.nodeID,
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

const mapTemplate = `package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

// Generated code for Map Node {{.NodeID}}
// This is a passthrough transformer that can be extended with custom logic

func main() {
	// Read input data from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Failed to read input: %v", err)
	}

	// Parse JSON input
	var data []map[string]interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		log.Fatalf("Failed to parse JSON input: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Processing %d rows through map node\n", len(data))

	// Transform data
	transformedData := transformData(data)

	// Output transformed data as JSON
	output, err := json.MarshalIndent(transformedData, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal output: %v", err)
	}

	fmt.Println(string(output))
}

func transformData(data []map[string]interface{}) []map[string]interface{} {
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
}
`
