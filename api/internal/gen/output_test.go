package gen

import (
	"api/internal/api/models"
	"fmt"
	"testing"
)

func TestDBOutputGeneration(t *testing.T) {
	conn := models.DBConnectionConfig{
		Type:     models.DBTypePostgres,
		Host:     "localhost",
		Port:     5433,
		Database: "test",
		Username: "postgres",
		Password: "postgres",
		SSLMode:  "disable",
	}

	// Start node
	startNode := models.Node{
		ID:    0,
		Type:  models.NodeTypeStart,
		Name:  "Start",
		JobID: 1,
	}

	// DB Input node
	inputNode := models.Node{
		ID:    1,
		Type:  models.NodeTypeDBInput,
		Name:  "Read Users",
		JobID: 1,
	}
	inputNode.SetData(models.DBInputConfig{
		Query:           "SELECT id, name, age FROM users",
		QueryWithSchema: "SET search_path TO public; SELECT id, name, age FROM users",
		DbSchema:        "public",
		Connection:      conn,
		DataModels: []models.DataModel{
			{Name: "id", Type: "integer", GoType: "int"},
			{Name: "name", Type: "varchar", GoType: "string"},
			{Name: "age", Type: "integer", GoType: "int"},
		},
	})

	// DB Output node
	outputNode := models.Node{
		ID:    2,
		Type:  models.NodeTypeDBOutput,
		Name:  "Write Users",
		JobID: 1,
	}
	outputNode.SetData(models.DBOutputConfig{
		Table:      "users_copy",
		Mode:       "INSERT",
		BatchSize:  100,
		DbSchema:   "public",
		Connection: conn,
		DataModels: []models.DataModel{
			{Name: "id", Type: "integer", GoType: "int"},
			{Name: "name", Type: "varchar", GoType: "string"},
			{Name: "age", Type: "integer", GoType: "int"},
		},
	})

	// Wire up ports
	startNode.OutputPort = []models.Port{
		{ID: 1, Type: models.PortNodeFlowOutput, Node: inputNode, NodeID: 0},
	}
	inputNode.InputPort = []models.Port{
		{ID: 2, Type: models.PortNodeFlowInput, Node: startNode, NodeID: 0},
	}
	inputNode.OutputPort = []models.Port{
		{ID: 3, Type: models.PortNodeFlowOutput, Node: outputNode, NodeID: 1},
		{ID: 4, Type: models.PortTypeOutput, Node: outputNode, NodeID: 1}, // Data port
	}
	outputNode.InputPort = []models.Port{
		{ID: 5, Type: models.PortNodeFlowInput, Node: inputNode, NodeID: 1},
		{ID: 6, Type: models.PortTypeInput, Node: inputNode, NodeID: 1}, // Data port
	}

	job := models.Job{
		ID:    1,
		Name:  "Test Job",
		Nodes: []models.Node{startNode, inputNode, outputNode},
	}

	exec := NewJobExecution(&job)
	_, err := exec.build()
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	source, err := exec.generateSource()
	if err != nil {
		t.Fatalf("generateSource failed: %v", err)
	}

	fmt.Println("=== Generated Code ===")
	fmt.Println(string(source))
	fmt.Println("=== End Generated Code ===")
}
