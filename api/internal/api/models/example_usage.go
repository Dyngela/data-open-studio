package models

import (
	"database/sql"
	"fmt"
)

// Example usage demonstrating how to use the generator system

// ExecuteJob executes a job by building and running its pipeline
func ExecuteJob(job *Job, nodes []Node, dbConn *sql.DB) (*PipelineResult, error) {
	// Create pipeline builder
	builder := NewPipelineBuilder(job, dbConn)

	// Add all nodes to the pipeline
	builder.WithNodes(nodes)

	// Build the pipeline
	pipeline, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build pipeline: %w", err)
	}

	// Execute the pipeline
	result := pipeline.Execute()

	if !result.Success {
		return result, fmt.Errorf("pipeline execution failed: %v", result.Errors)
	}

	return result, nil
}

// Example: How to use the system

/*
func exampleUsage() {
	// 1. Create or load a job
	job := &Job{
		ID:   1,
		Name: "ETL Pipeline",
	}

	// 2. Create nodes with configurations
	inputNode := Node{
		ID:   1,
		Type: NodeTypeDBInput,
		Name: "Source Database",
	}
	inputNode.SetData(DBInputConfig{
		Schema: "public",
		Table:  "users",
		Query:  "SELECT id, name, email FROM public.users WHERE active = true",
	})

	mapNode := Node{
		ID:   2,
		Type: NodeTypeMap,
		Name: "Transform Data",
	}
	mapNode.SetData(MapConfig{})

	outputNode := Node{
		ID:   3,
		Type: NodeTypeDBOutput,
		Name: "Target Database",
	}
	outputNode.SetData(DBOutputConfig{
		Table:     "users_processed",
		Mode:      "insert",
		BatchSize: 100,
	})

	nodes := []Node{inputNode, mapNode, outputNode}

	// 3. Get database connection
	dbConn, err := sql.Open("postgres", "connection_string_here")
	if err != nil {
		panic(err)
	}
	defer dbConn.Close()

	// 4. Execute the job
	result, err := ExecuteJob(job, nodes, dbConn)
	if err != nil {
		fmt.Printf("Job failed: %v\n", err)
		return
	}

	// 5. Check results
	fmt.Printf("Job completed successfully in %v\n", result.Duration)
	fmt.Printf("Processed nodes: %v\n", len(nodes))
	fmt.Printf("Output data keys: %v\n", getKeys(result.Output))
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
*/

// Alternative usage: Direct generator creation and execution

/*
func exampleDirectGeneratorUsage() {
	// 1. Create execution context
	dbConn, _ := sql.Open("postgres", "connection_string")
	ctx := NewExecutionContext(dbConn)

	// 2. Create generators directly
	inputGen := NewDBInputGenerator(1, DBInputConfig{
		Schema: "public",
		Table:  "users",
	})

	mapGen := NewMapGenerator(2, MapConfig{})

	outputGen := NewDBOutputGenerator(3, DBOutputConfig{
		Table:     "users_processed",
		Mode:      "insert",
		BatchSize: 100,
	})

	// 3. Execute generators in sequence
	if err := inputGen.Execute(ctx); err != nil {
		fmt.Printf("Input failed: %v\n", err)
		return
	}

	if err := mapGen.Execute(ctx); err != nil {
		fmt.Printf("Map failed: %v\n", err)
		return
	}

	if err := outputGen.Execute(ctx); err != nil {
		fmt.Printf("Output failed: %v\n", err)
		return
	}

	fmt.Println("Pipeline completed successfully")
}
*/
