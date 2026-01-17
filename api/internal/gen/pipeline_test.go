package gen

import (
	"api/internal/api/models"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJobExecution_Generation(t *testing.T) {
	// Connection configs
	dataOpenStudioConn := models.DBConnectionConfig{
		Type:     models.DBTypePostgres,
		Host:     "localhost",
		Port:     5433,
		Database: "data-open-studio",
		Username: "postgres",
		Password: "postgres",
		SSLMode:  "disable",
	}

	testInputConn := models.DBConnectionConfig{
		Type:     models.DBTypePostgres,
		Host:     "localhost",
		Port:     5433,
		Database: "test-input",
		Username: "postgres",
		Password: "postgres",
		SSLMode:  "disable",
	}

	// ==========================================================================
	// Node 5: DB Output - Write to test-input.receiver
	// ==========================================================================
	dbOutputNode := models.Node{
		ID:    5,
		Type:  models.NodeTypeDBOutput,
		Name:  "Write to Receiver",
		JobID: 1,
	}
	dbOutputNode.SetData(models.DBOutputConfig{
		Table:      "receiver",
		Mode:       models.DbOutputModeInsert,
		BatchSize:  500,
		Connection: testInputConn,
		DataModels: []models.DataModel{
			{Name: "age", Type: "integer", GoType: "int"},
			{Name: "full_name", Type: "varchar", GoType: "string"},
			{Name: "age_in_months", Type: "integer", GoType: "int"},
			{Name: "hobby", Type: "varchar", GoType: "string", Nullable: true},
		},
	})

	// ==========================================================================
	// Node 2: DB Input - Read users from data-open-studio.test
	// Schema: id, nom, prenom, age
	// ==========================================================================
	readUsersNode := models.Node{
		ID:    2,
		Type:  models.NodeTypeDBInput,
		Name:  "Read Users",
		JobID: 1,
	}
	readUsersNode.SetData(models.DBInputConfig{
		Query:      "SELECT id, nom, prenom, age FROM test",
		DbSchema:   "public",
		Connection: dataOpenStudioConn,
		DataModels: []models.DataModel{
			{Name: "id", Type: "integer", GoType: "int"},
			{Name: "nom", Type: "varchar", GoType: "string"},
			{Name: "prenom", Type: "varchar", GoType: "string"},
			{Name: "age", Type: "integer", GoType: "int"},
		},
	})

	// ==========================================================================
	// Node 3: DB Input - Read hobbies from test-input.sender
	// Schema: id, hobby, nom
	// ==========================================================================
	readHobbiesNode := models.Node{
		ID:    3,
		Type:  models.NodeTypeDBInput,
		Name:  "Read Hobbies",
		JobID: 1,
	}
	readHobbiesNode.SetData(models.DBInputConfig{
		Query:      "SELECT id, hobby, nom FROM sender",
		DbSchema:   "public",
		Connection: testInputConn,
		DataModels: []models.DataModel{
			{Name: "id", Type: "integer", GoType: "int"},
			{Name: "hobby", Type: "varchar", GoType: "string"},
			{Name: "nom", Type: "varchar", GoType: "string"},
		},
	})

	// ==========================================================================
	// Node 4: Map - Join users with hobbies, transform columns
	// - Left join on nom
	// - Create full_name using lib.Concat
	// - Output age for receiver table
	// ==========================================================================
	mapNode := models.Node{
		ID:    4,
		Type:  models.NodeTypeMap,
		Name:  "Transform Join",
		JobID: 1,
	}
	mapNode.SetData(models.MapConfig{
		Inputs: []models.InputFlow{
			{
				Name:   "users",
				PortID: 13,
				Schema: []models.DataModel{
					{Name: "id", Type: "integer", GoType: "int"},
					{Name: "nom", Type: "varchar", GoType: "string"},
					{Name: "prenom", Type: "varchar", GoType: "string"},
					{Name: "age", Type: "integer", GoType: "int"},
				},
			},
			{
				Name:   "hobbies",
				PortID: 14,
				Schema: []models.DataModel{
					{Name: "id", Type: "integer", GoType: "int"},
					{Name: "hobby", Type: "varchar", GoType: "string"},
					{Name: "nom", Type: "varchar", GoType: "string"},
				},
			},
		},
		Join: &models.JoinConfig{
			Type:       models.JoinTypeLeft,
			LeftInput:  "users",
			RightInput: "hobbies",
			LeftKey:    "nom",
			RightKey:   "nom",
		},
		Outputs: []models.OutputFlow{
			{
				Name:   "main",
				PortID: 15,
				Columns: []models.MapOutputCol{
					// Direct mapping of age for receiver table
					{
						Name:     "age",
						DataType: "int",
						FuncType: models.FuncTypeDirect,
						InputRef: "users.age",
					},
					// Example: Create full_name using library function
					{
						Name:     "full_name",
						DataType: "string",
						FuncType: models.FuncTypeLibrary,
						LibFunc:  "Concat",
						Args: []models.FuncArg{
							{Type: "literal", Value: " "},
							{Type: "column", Value: "users.prenom"},
							{Type: "column", Value: "users.nom"},
						},
					},
					// Example: Custom expression
					{
						Name:       "age_in_months",
						DataType:   "int",
						FuncType:   models.FuncTypeCustom,
						CustomType: models.CustomExpr,
						Expression: "users.age * 12",
					},
					// Pass through hobby (may be nil for non-matching)
					{
						Name:     "hobby",
						DataType: "string",
						FuncType: models.FuncTypeDirect,
						InputRef: "hobbies.hobby",
					},
				},
			},
		},
	})

	// ==========================================================================
	// Node 1: Start
	// ==========================================================================
	startNode := models.Node{
		ID:    1,
		Type:  models.NodeTypeStart,
		Name:  "Pipeline Start",
		JobID: 1,
	}

	// ==========================================================================
	// Wire up ports - Flow connections (execution order)
	// ==========================================================================

	// Start -> Read Users
	startNode.OutputPort = []models.Port{
		{ID: 1, Type: models.PortNodeFlowOutput, Node: readUsersNode, NodeID: 1},
		{ID: 8, Type: models.PortNodeFlowOutput, Node: readHobbiesNode, NodeID: 1},
	}

	readUsersNode.InputPort = []models.Port{
		{ID: 2, Type: models.PortNodeFlowInput, Node: startNode, NodeID: 1},
	}
	readUsersNode.OutputPort = []models.Port{
		{ID: 3, Type: models.PortNodeFlowOutput, Node: mapNode, NodeID: 2},
		{ID: 11, Type: models.PortTypeOutput, Node: mapNode, NodeID: 2}, // DATA port
	}

	readHobbiesNode.InputPort = []models.Port{
		{ID: 9, Type: models.PortNodeFlowInput, Node: startNode, NodeID: 1},
	}
	readHobbiesNode.OutputPort = []models.Port{
		{ID: 10, Type: models.PortNodeFlowOutput, Node: mapNode, NodeID: 3},
		{ID: 12, Type: models.PortTypeOutput, Node: mapNode, NodeID: 3}, // DATA port
	}

	mapNode.InputPort = []models.Port{
		// Flow ports
		{ID: 4, Type: models.PortNodeFlowInput, Node: readUsersNode, NodeID: 2},
		{ID: 5, Type: models.PortNodeFlowInput, Node: readHobbiesNode, NodeID: 3},
		// Data ports
		{ID: 13, Type: models.PortTypeInput, Node: readUsersNode, NodeID: 2},
		{ID: 14, Type: models.PortTypeInput, Node: readHobbiesNode, NodeID: 3},
	}
	mapNode.OutputPort = []models.Port{
		{ID: 6, Type: models.PortNodeFlowOutput, Node: dbOutputNode, NodeID: 4},
		{ID: 15, Type: models.PortTypeOutput, Node: dbOutputNode, NodeID: 4}, // DATA port
	}

	dbOutputNode.InputPort = []models.Port{
		{ID: 7, Type: models.PortNodeFlowInput, Node: mapNode, NodeID: 4},
		{ID: 16, Type: models.PortTypeInput, Node: mapNode, NodeID: 4}, // DATA port
	}

	// ==========================================================================
	// Build and execute
	// ==========================================================================
	nodes := []models.Node{
		startNode,
		readUsersNode,
		readHobbiesNode,
		mapNode,
		dbOutputNode,
		// Orphan node (should be ignored)
		{ID: 89, Type: models.NodeTypeDBInput, Name: "unlinked", JobID: 1},
	}

	job := models.Job{
		ID:          1,
		Name:        "ETL Users with Hobbies",
		Description: "Join users from data-open-studio with hobbies from test-input, output ages to receiver",
		CreatorID:   1,
		Active:      true,
		Nodes:       nodes,
		OutputPath:  "../../bin",
	}

	exec := NewPipelineExecutor(&job)

	// Set progress config - bakes API URL and job ID into the executable
	exec.SetProgressConfig("http://localhost:8080", job.ID)

	_, err := exec.Build()
	if err != nil {
		t.Fatalf("Pipeline build failed: %v", err)
	}
	require.NoError(t, err)

	// Write generated code to file
	err = exec.WriteToFile("../../../bin/generated_job.go", "main")
	require.NoError(t, err)
}
