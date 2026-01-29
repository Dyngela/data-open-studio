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
			{Name: "total_amount", Type: "numeric", GoType: "float32"},
			{Name: "amount_time_twelve", Type: "numeric", GoType: "float32"},
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
		Query:      "SELECT id, status, shipping_address, total_amount from orders",
		DbSchema:   "public",
		Connection: dataOpenStudioConn,
		DataModels: []models.DataModel{
			{Name: "id", Type: "integer", GoType: "int"},
			{Name: "status", Type: "varchar", GoType: "string"},
			{Name: "shipping_address", Type: "varchar", GoType: "string"},
			{Name: "total_amount", Type: "numeric", GoType: "float32"},
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
		Query:      "SELECT id, name, supplier FROM products",
		DbSchema:   "public",
		Connection: testInputConn,
		DataModels: []models.DataModel{
			{Name: "id", Type: "integer", GoType: "int"},
			{Name: "name", Type: "varchar", GoType: "string"},
			{Name: "supplier", Type: "varchar", GoType: "string"},
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
				Name:   "orders",
				PortID: 13,
				Schema: []models.DataModel{
					{Name: "id", Type: "integer", GoType: "int"},
					{Name: "status", Type: "varchar", GoType: "string"},
					{Name: "shipping_address", Type: "varchar", GoType: "string"},
					{Name: "total_amount", Type: "numeric", GoType: "float32"},
				},
			},
			{
				Name:   "products",
				PortID: 14,
				Schema: []models.DataModel{
					{Name: "id", Type: "integer", GoType: "int"},
					{Name: "name", Type: "varchar", GoType: "string"},
					{Name: "supplier", Type: "varchar", GoType: "string"},
				},
			},
		},
		Join: &models.JoinConfig{
			Type:       models.JoinTypeLeft,
			LeftInput:  "orders",
			RightInput: "products",
			LeftKey:    "status",
			RightKey:   "supplier",
		},
		Outputs: []models.OutputFlow{
			{
				Name:   "main",
				PortID: 15,
				Columns: []models.MapOutputCol{
					// Direct mapping of age for receiver table
					{
						Name:     "total_amount",
						DataType: "float32",
						FuncType: models.FuncTypeDirect,
						InputRef: "orders.total_amount",
					},
					// Example: Custom expression
					{
						Name:       "amountTimeTwelve",
						DataType:   "float32",
						FuncType:   models.FuncTypeCustom,
						CustomType: models.CustomExpr,
						Expression: "orders.total_amount * 12",
					},
					// Pass through hobby (may be nil for non-matching)
					{
						Name:     "hobby",
						DataType: "string",
						FuncType: models.FuncTypeDirect,
						InputRef: "products.supplier",
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

	// Start -> Read Users, Read Hobbies
	startNode.OutputPort = []models.Port{
		{ID: 1, Type: models.PortNodeFlowOutput, NodeID: 1, ConnectedNodeID: 2, Node: readUsersNode},
		{ID: 8, Type: models.PortNodeFlowOutput, NodeID: 1, ConnectedNodeID: 3, Node: readHobbiesNode},
	}

	readUsersNode.InputPort = []models.Port{
		{ID: 2, Type: models.PortNodeFlowInput, NodeID: 2, ConnectedNodeID: 1, Node: startNode},
	}
	readUsersNode.OutputPort = []models.Port{
		{ID: 3, Type: models.PortNodeFlowOutput, NodeID: 2, ConnectedNodeID: 4, Node: mapNode},
		{ID: 11, Type: models.PortTypeOutput, NodeID: 2, ConnectedNodeID: 4, Node: mapNode}, // DATA port
	}

	readHobbiesNode.InputPort = []models.Port{
		{ID: 9, Type: models.PortNodeFlowInput, NodeID: 3, ConnectedNodeID: 1, Node: startNode},
	}
	readHobbiesNode.OutputPort = []models.Port{
		{ID: 10, Type: models.PortNodeFlowOutput, NodeID: 3, ConnectedNodeID: 4, Node: mapNode},
		{ID: 12, Type: models.PortTypeOutput, NodeID: 3, ConnectedNodeID: 4, Node: mapNode}, // DATA port
	}

	mapNode.InputPort = []models.Port{
		// Flow ports
		{ID: 4, Type: models.PortNodeFlowInput, NodeID: 4, ConnectedNodeID: 2, Node: readUsersNode},
		{ID: 5, Type: models.PortNodeFlowInput, NodeID: 4, ConnectedNodeID: 3, Node: readHobbiesNode},
		// Data ports
		{ID: 13, Type: models.PortTypeInput, NodeID: 4, ConnectedNodeID: 2, Node: readUsersNode},
		{ID: 14, Type: models.PortTypeInput, NodeID: 4, ConnectedNodeID: 3, Node: readHobbiesNode},
	}
	mapNode.OutputPort = []models.Port{
		{ID: 6, Type: models.PortNodeFlowOutput, NodeID: 4, ConnectedNodeID: 5, Node: dbOutputNode},
		{ID: 15, Type: models.PortTypeOutput, NodeID: 4, ConnectedNodeID: 5, Node: dbOutputNode}, // DATA port
	}

	dbOutputNode.InputPort = []models.Port{
		{ID: 7, Type: models.PortNodeFlowInput, NodeID: 5, ConnectedNodeID: 4, Node: mapNode},
		{ID: 16, Type: models.PortTypeInput, NodeID: 5, ConnectedNodeID: 4, Node: mapNode}, // DATA port
	}

	// ==========================================================================
	// build and execute
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
	err := exec.Run()
	if err != nil {
		t.Fatalf("Pipeline build failed: %v", err)
	}
	require.NoError(t, err)

}
