package service

import (
	"api"
	"api/internal/api/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB initializes the database connection for tests
func setupTestDB(t *testing.T) {
	// Initialize config which also sets up DB connection
	api.InitConfig("../../../.env.test")

	// Auto-migrate the Job table for testing
	err := api.DB.AutoMigrate(&models.Job{})
	require.NoError(t, err, "Failed to migrate Job table")
}

// cleanupTestJob removes the test job from the database
func cleanupTestJob(t *testing.T, jobID uint) {
	if jobID > 0 {
		api.DB.Unscoped().Delete(&models.Job{}, jobID)
	}
}

func TestCreateJobWithNodes(t *testing.T) {
	setupTestDB(t)

	jobService := NewJobService()

	// Create test nodes
	nodes := models.NodeList{
		// Start node - DB Input reading from a table
		&models.DBInputConfig{
			BaseNode: models.BaseNode{
				ID:   1,
				Type: models.NodeTypeDBInput,
				Name: "Start - DB Connection",
				Xpos: 100,
				Ypos: 100,
				InputPort: []models.Port{
					{ID: 1, Type: models.PortNodeFlowInput},
				},
				OutputPort: []models.Port{
					{ID: 2, Type: models.PortTypeOutput},
					{ID: 3, Type: models.PortNodeFlowOutput},
				},
			},
			Query:  "SELECT * FROM source_table",
			Schema: "public",
			Table:  "source_table",
		},

		// DB Input node - reading from another source
		&models.DBInputConfig{
			BaseNode: models.BaseNode{
				ID:   2,
				Type: models.NodeTypeDBInput,
				Name: "DB Input Node",
				Xpos: 300,
				Ypos: 150,
				InputPort: []models.Port{
					{ID: 4, Type: models.PortTypeInput},
					{ID: 5, Type: models.PortNodeFlowInput},
				},
				OutputPort: []models.Port{
					{ID: 6, Type: models.PortTypeOutput},
					{ID: 7, Type: models.PortNodeFlowOutput},
				},
			},
			Query:  "SELECT id, name, value FROM input_data WHERE active = true",
			Schema: "public",
			Table:  "input_data",
		},

		// Map node - transformation logic
		&models.MapConfig{
			BaseNode: models.BaseNode{
				ID:   3,
				Type: models.NodeTypeMap,
				Name: "Data Transformation Map",
				Xpos: 500,
				Ypos: 200,
				InputPort: []models.Port{
					{ID: 8, Type: models.PortTypeInput},
					{ID: 9, Type: models.PortNodeFlowInput},
				},
				OutputPort: []models.Port{
					{ID: 10, Type: models.PortTypeOutput},
					{ID: 11, Type: models.PortNodeFlowOutput},
				},
			},
		},

		// DB Output node - writing to destination
		&models.DBOutputConfig{
			BaseNode: models.BaseNode{
				ID:   4,
				Type: models.NodeTypeDBOutput,
				Name: "DB Output Node",
				Xpos: 700,
				Ypos: 250,
				InputPort: []models.Port{
					{ID: 12, Type: models.PortTypeInput},
					{ID: 13, Type: models.PortNodeFlowInput},
				},
				OutputPort: []models.Port{
					{ID: 14, Type: models.PortNodeFlowOutput},
				},
			},
			Table:     "output_table",
			Mode:      "insert",
			BatchSize: 1000,
		},
	}

	// Create job with nodes
	job := models.Job{
		Name:        "Test ETL Pipeline Job",
		Description: "A test job that demonstrates data flow from DB input through transformation to DB output",
		CreatorID:   1,
		Active:      true,
		Nodes:       nodes,
	}

	// Save the job
	createdJob, err := jobService.CreateJob(job)
	require.NoError(t, err, "Failed to create job")
	require.NotZero(t, createdJob.ID, "Job ID should not be zero")

	// Clean up after test
	defer cleanupTestJob(t, createdJob.ID)

	// Verify the job was created correctly
	assert.Equal(t, "Test ETL Pipeline Job", createdJob.Name)
	assert.Equal(t, "A test job that demonstrates data flow from DB input through transformation to DB output", createdJob.Description)
	assert.Equal(t, uint(1), createdJob.CreatorID)
	assert.True(t, createdJob.Active)
	assert.Len(t, createdJob.Nodes, 4, "Should have 4 nodes")

	// Retrieve the job from database to verify persistence
	retrievedJob, err := jobService.FindJobByID(createdJob.ID)
	require.NoError(t, err, "Failed to retrieve job")

	// Verify retrieved job matches created job
	assert.Equal(t, createdJob.ID, retrievedJob.ID)
	assert.Equal(t, createdJob.Name, retrievedJob.Name)
	assert.Equal(t, createdJob.Description, retrievedJob.Description)
	assert.Equal(t, createdJob.CreatorID, retrievedJob.CreatorID)
	assert.Equal(t, createdJob.Active, retrievedJob.Active)
	assert.Len(t, retrievedJob.Nodes, 4, "Retrieved job should have 4 nodes")

	// Verify node details
	dbInputNode, ok := retrievedJob.Nodes[0].(*models.DBInputConfig)
	require.True(t, ok, "First node should be DBInputConfig")
	assert.Equal(t, "Start - DB Connection", dbInputNode.Name)
	assert.Equal(t, "SELECT * FROM source_table", dbInputNode.Query)
	assert.Equal(t, "public", dbInputNode.Schema)
	assert.Equal(t, "source_table", dbInputNode.Table)

	dbOutputNode, ok := retrievedJob.Nodes[3].(*models.DBOutputConfig)
	require.True(t, ok, "Fourth node should be DBOutputConfig")
	assert.Equal(t, "DB Output Node", dbOutputNode.Name)
	assert.Equal(t, "output_table", dbOutputNode.Table)
	assert.Equal(t, "insert", dbOutputNode.Mode)
	assert.Equal(t, 1000, dbOutputNode.BatchSize)
}

func TestCreateJobWithMultipleDBNodes(t *testing.T) {
	setupTestDB(t)

	jobService := NewJobService()

	// Create a more complex pipeline with multiple DB connections
	nodes := models.NodeList{
		// Start DB Connection 1
		&models.DBInputConfig{
			BaseNode: models.BaseNode{
				ID:   1,
				Type: models.NodeTypeDBInput,
				Name: "Start DB Conn - Users",
				Xpos: 50,
				Ypos: 100,
				InputPort: []models.Port{
					{ID: 1, Type: models.PortNodeFlowInput},
				},
				OutputPort: []models.Port{
					{ID: 2, Type: models.PortTypeOutput},
					{ID: 3, Type: models.PortNodeFlowOutput},
				},
			},
			Query:  "SELECT * FROM users WHERE created_at > NOW() - INTERVAL '30 days'",
			Schema: "public",
			Table:  "users",
		},

		// Start DB Connection 2
		&models.DBInputConfig{
			BaseNode: models.BaseNode{
				ID:   2,
				Type: models.NodeTypeDBInput,
				Name: "Start DB Conn - Orders",
				Xpos: 50,
				Ypos: 300,
				InputPort: []models.Port{
					{ID: 4, Type: models.PortNodeFlowInput},
				},
				OutputPort: []models.Port{
					{ID: 5, Type: models.PortTypeOutput},
					{ID: 6, Type: models.PortNodeFlowOutput},
				},
			},
			Query:  "SELECT * FROM orders WHERE status = 'completed'",
			Schema: "sales",
			Table:  "orders",
		},

		// DB Input for additional data
		&models.DBInputConfig{
			BaseNode: models.BaseNode{
				ID:   3,
				Type: models.NodeTypeDBInput,
				Name: "DB Input - Products",
				Xpos: 250,
				Ypos: 200,
				InputPort: []models.Port{
					{ID: 7, Type: models.PortTypeInput},
					{ID: 8, Type: models.PortNodeFlowInput},
				},
				OutputPort: []models.Port{
					{ID: 9, Type: models.PortTypeOutput},
					{ID: 10, Type: models.PortNodeFlowOutput},
				},
			},
			Query:  "SELECT product_id, name, price FROM products",
			Schema: "inventory",
			Table:  "products",
		},

		// Map node for data transformation
		&models.MapConfig{
			BaseNode: models.BaseNode{
				ID:   4,
				Type: models.NodeTypeMap,
				Name: "Join & Transform",
				Xpos: 450,
				Ypos: 200,
				InputPort: []models.Port{
					{ID: 11, Type: models.PortTypeInput},
					{ID: 12, Type: models.PortTypeInput},
					{ID: 13, Type: models.PortNodeFlowInput},
				},
				OutputPort: []models.Port{
					{ID: 14, Type: models.PortTypeOutput},
					{ID: 15, Type: models.PortNodeFlowOutput},
				},
			},
		},

		// DB Output 1 - Main output
		&models.DBOutputConfig{
			BaseNode: models.BaseNode{
				ID:   5,
				Type: models.NodeTypeDBOutput,
				Name: "DB Output - Analytics",
				Xpos: 650,
				Ypos: 150,
				InputPort: []models.Port{
					{ID: 16, Type: models.PortTypeInput},
					{ID: 17, Type: models.PortNodeFlowInput},
				},
				OutputPort: []models.Port{
					{ID: 18, Type: models.PortNodeFlowOutput},
				},
			},
			Table:     "user_analytics",
			Mode:      "upsert",
			BatchSize: 500,
		},

		// DB Output 2 - Secondary output
		&models.DBOutputConfig{
			BaseNode: models.BaseNode{
				ID:   6,
				Type: models.NodeTypeDBOutput,
				Name: "DB Output - Archive",
				Xpos: 650,
				Ypos: 300,
				InputPort: []models.Port{
					{ID: 19, Type: models.PortTypeInput},
					{ID: 20, Type: models.PortNodeFlowInput},
				},
				OutputPort: []models.Port{
					{ID: 21, Type: models.PortNodeFlowOutput},
				},
			},
			Table:     "data_archive",
			Mode:      "append",
			BatchSize: 2000,
		},
	}

	// Create complex job
	job := models.Job{
		Name:        "Complex Multi-DB ETL Pipeline",
		Description: "Complex pipeline with multiple DB inputs, transformation, and multiple outputs",
		CreatorID:   1,
		Active:      true,
		Nodes:       nodes,
	}

	// Save the job
	createdJob, err := jobService.CreateJob(job)
	require.NoError(t, err, "Failed to create complex job")
	require.NotZero(t, createdJob.ID, "Job ID should not be zero")

	// Clean up after test
	defer cleanupTestJob(t, createdJob.ID)

	// Verify the job was created correctly
	assert.Equal(t, "Complex Multi-DB ETL Pipeline", createdJob.Name)
	assert.Len(t, createdJob.Nodes, 6, "Should have 6 nodes")

	// Retrieve and verify
	retrievedJob, err := jobService.FindJobByID(createdJob.ID)
	require.NoError(t, err, "Failed to retrieve complex job")

	// Count node types
	var dbInputCount, dbOutputCount, mapCount int
	for _, node := range retrievedJob.Nodes {
		switch node.(type) {
		case *models.DBInputConfig:
			dbInputCount++
		case *models.DBOutputConfig:
			dbOutputCount++
		case *models.MapConfig:
			mapCount++
		}
	}

	assert.Equal(t, 3, dbInputCount, "Should have 3 DB input nodes")
	assert.Equal(t, 2, dbOutputCount, "Should have 2 DB output nodes")
	assert.Equal(t, 1, mapCount, "Should have 1 map node")

	// Verify specific node details
	firstDBInput, ok := retrievedJob.Nodes[0].(*models.DBInputConfig)
	require.True(t, ok, "First node should be DBInputConfig")
	assert.Equal(t, "Start DB Conn - Users", firstDBInput.Name)
	assert.Equal(t, "users", firstDBInput.Table)

	lastDBOutput, ok := retrievedJob.Nodes[5].(*models.DBOutputConfig)
	require.True(t, ok, "Last node should be DBOutputConfig")
	assert.Equal(t, "DB Output - Archive", lastDBOutput.Name)
	assert.Equal(t, "data_archive", lastDBOutput.Table)
	assert.Equal(t, 2000, lastDBOutput.BatchSize)
}

func TestCreateJobWithSimpleFlow(t *testing.T) {
	setupTestDB(t)

	jobService := NewJobService()

	// Simple linear flow: Start -> Input -> Output
	nodes := models.NodeList{
		&models.DBInputConfig{
			BaseNode: models.BaseNode{
				ID:   1,
				Type: models.NodeTypeDBInput,
				Name: "Start",
				Xpos: 100,
				Ypos: 100,
				InputPort: []models.Port{
					{ID: 1, Type: models.PortNodeFlowInput},
				},
				OutputPort: []models.Port{
					{ID: 2, Type: models.PortTypeOutput},
					{ID: 3, Type: models.PortNodeFlowOutput},
				},
			},
			Query:  "SELECT * FROM source",
			Schema: "public",
			Table:  "source",
		},
		&models.DBOutputConfig{
			BaseNode: models.BaseNode{
				ID:   2,
				Type: models.NodeTypeDBOutput,
				Name: "Output",
				Xpos: 300,
				Ypos: 100,
				InputPort: []models.Port{
					{ID: 4, Type: models.PortTypeInput},
					{ID: 5, Type: models.PortNodeFlowInput},
				},
				OutputPort: []models.Port{
					{ID: 6, Type: models.PortNodeFlowOutput},
				},
			},
			Table:     "destination",
			Mode:      "insert",
			BatchSize: 100,
		},
	}

	job := models.Job{
		Name:        "Simple Flow Test",
		Description: "Simple two-node pipeline for testing",
		CreatorID:   1,
		Active:      true,
		Nodes:       nodes,
	}

	createdJob, err := jobService.CreateJob(job)
	require.NoError(t, err, "Failed to create simple job")
	require.NotZero(t, createdJob.ID, "Job ID should not be zero")

	defer cleanupTestJob(t, createdJob.ID)

	assert.Equal(t, "Simple Flow Test", createdJob.Name)
	assert.Len(t, createdJob.Nodes, 2, "Should have 2 nodes")

	// Verify persistence
	retrievedJob, err := jobService.FindJobByID(createdJob.ID)
	require.NoError(t, err, "Failed to retrieve simple job")
	assert.Len(t, retrievedJob.Nodes, 2, "Retrieved job should have 2 nodes")
}
