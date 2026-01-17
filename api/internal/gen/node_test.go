package gen

import (
	"api/internal/api/models"
	"fmt"
	"testing"
)

func TestDBInputAlone(t *testing.T) {
	conn := models.DBConnectionConfig{
		Type:     models.DBTypePostgres,
		Host:     "localhost",
		Port:     5433,
		Database: "mydb",
		Username: "postgres",
		Password: "postgres",
		SSLMode:  "disable",
	}

	startNode := models.Node{
		ID:    0,
		Type:  models.NodeTypeStart,
		Name:  "Start",
		JobID: 1,
	}

	inputNode := models.Node{
		ID:    1,
		Type:  models.NodeTypeDBInput,
		Name:  "Read Users",
		JobID: 1,
	}
	inputNode.SetData(models.DBInputConfig{
		Query:           "SELECT id, name, email, age FROM users",
		QueryWithSchema: "SET search_path TO public; SELECT id, name, email, age FROM users",
		DbSchema:        "public",
		Connection:      conn,
		DataModels: []models.DataModel{
			{Name: "id", Type: "integer", GoType: "int"},
			{Name: "name", Type: "varchar", GoType: "string"},
			{Name: "email", Type: "varchar", GoType: "string", Nullable: true},
			{Name: "age", Type: "integer", GoType: "int"},
		},
	})

	// Wire ports
	startNode.OutputPort = []models.Port{
		{ID: 1, Type: models.PortNodeFlowOutput, Node: inputNode, NodeID: 0},
	}
	inputNode.InputPort = []models.Port{
		{ID: 2, Type: models.PortNodeFlowInput, Node: startNode, NodeID: 0},
	}
	inputNode.OutputPort = []models.Port{
		{ID: 3, Type: models.PortTypeOutput, NodeID: 1},
	}

	job := models.Job{
		ID:    1,
		Name:  "DB Input Test",
		Nodes: []models.Node{startNode, inputNode},
	}

	exec := NewJobExecution(&job)
	_, err := exec.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	source, err := exec.GenerateSource("main")
	if err != nil {
		t.Fatalf("GenerateSource failed: %v", err)
	}

	fmt.Println("=== DB INPUT GENERATED CODE ===")
	fmt.Println(string(source))
	fmt.Println("=== END ===")
}

func TestDBOutputAlone(t *testing.T) {
	conn := models.DBConnectionConfig{
		Type:     models.DBTypePostgres,
		Host:     "localhost",
		Port:     5433,
		Database: "mydb",
		Username: "postgres",
		Password: "postgres",
		SSLMode:  "disable",
	}

	startNode := models.Node{
		ID:    0,
		Type:  models.NodeTypeStart,
		Name:  "Start",
		JobID: 1,
	}

	// We need an input node to provide the row type
	inputNode := models.Node{
		ID:    1,
		Type:  models.NodeTypeDBInput,
		Name:  "Read Source",
		JobID: 1,
	}
	inputNode.SetData(models.DBInputConfig{
		Query:           "SELECT id, name, amount FROM orders",
		QueryWithSchema: "SET search_path TO public; SELECT id, name, amount FROM orders",
		DbSchema:        "public",
		Connection:      conn,
		DataModels: []models.DataModel{
			{Name: "id", Type: "integer", GoType: "int"},
			{Name: "name", Type: "varchar", GoType: "string"},
			{Name: "amount", Type: "numeric", GoType: "float64"},
		},
	})

	outputNode := models.Node{
		ID:    2,
		Type:  models.NodeTypeDBOutput,
		Name:  "Write Target",
		JobID: 1,
	}
	outputNode.SetData(models.DBOutputConfig{
		Table:     "orders_backup",
		Mode:      "INSERT",
		BatchSize: 250,
		DbSchema:  "archive",
		Connection: conn,
		DataModels: []models.DataModel{
			{Name: "id", Type: "integer", GoType: "int"},
			{Name: "name", Type: "varchar", GoType: "string"},
			{Name: "amount", Type: "numeric", GoType: "float64"},
		},
	})

	// Wire ports
	startNode.OutputPort = []models.Port{
		{ID: 1, Type: models.PortNodeFlowOutput, Node: inputNode, NodeID: 0},
	}
	inputNode.InputPort = []models.Port{
		{ID: 2, Type: models.PortNodeFlowInput, Node: startNode, NodeID: 0},
	}
	inputNode.OutputPort = []models.Port{
		{ID: 3, Type: models.PortNodeFlowOutput, Node: outputNode, NodeID: 1},
		{ID: 4, Type: models.PortTypeOutput, Node: outputNode, NodeID: 1},
	}
	outputNode.InputPort = []models.Port{
		{ID: 5, Type: models.PortNodeFlowInput, Node: inputNode, NodeID: 1},
		{ID: 6, Type: models.PortTypeInput, Node: inputNode, NodeID: 1},
	}

	job := models.Job{
		ID:    1,
		Name:  "DB Output Test",
		Nodes: []models.Node{startNode, inputNode, outputNode},
	}

	exec := NewJobExecution(&job)
	_, err := exec.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	source, err := exec.GenerateSource("main")
	if err != nil {
		t.Fatalf("GenerateSource failed: %v", err)
	}

	fmt.Println("=== DB OUTPUT GENERATED CODE ===")
	fmt.Println(string(source))
	fmt.Println("=== END ===")
}
