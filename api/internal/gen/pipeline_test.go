package gen

import (
	"api/internal/api/models"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Test Organization
// ============================================================================
//
// This test suite is organized into the following sections:
//
// 1. Helper Functions & Builders
//    - mustNodeData: Convert values to NodeData
//    - NodeBuilder: Fluent builder for creating node graphs
//    - MockGenerator: Test implementation of Generator interface
//
// 2. Job Execution Basic Tests
//    - TestNewJobExecution: Job execution creation
//    - TestWithDbConnection: Database connection management
//
// 3. Job Node Ordering Tests (TestJob_NodeOrdering)
//    - Table-driven tests for various graph patterns
//    - Tests focus on verifying correct topological ordering
//    - Covers: empty jobs, linear chains, parallel branches, diamonds, etc.
//
// 4. Job Execution Flow Tests (TestJob_ExecutionFlow)
//    - Tests complete job execution setup
//    - Method chaining verification
//    - Node count validation
//
// 5. Job Node Dependency Tests (TestJob_NodeDependencies)
//    - Tests specific dependency resolution scenarios
//    - Multi-dependency nodes
//    - Transitive dependencies
//    - Parallel execution verification
//
// 6. Generator Creation Tests (TestNewGenerator)
//    - Tests NewGenerator(node) factory function
//    - Verifies proper generator type creation with typed configs
//    - Error handling for invalid nodes/configs
//
// 7. Job Build Tests (TestJobExecution_Build)
//    - Tests complete job build process
//    - Generator creation and ordering
//    - Integration of steps and generators
//
// ============================================================================

// mustNodeData is a helper to create NodeData from any value
func mustNodeData(v any) models.NodeData {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// NodeBuilder helps build a graph of connected nodes
type NodeBuilder struct {
	nodes  map[int]*models.Node
	portID uint
}

func NewNodeBuilder() *NodeBuilder {
	return &NodeBuilder{
		nodes:  make(map[int]*models.Node),
		portID: 1,
	}
}

func (nb *NodeBuilder) AddNode(id int, nodeType models.NodeType, name string) *NodeBuilder {
	node := &models.Node{
		ID:   id,
		Type: nodeType,
		Name: name,
		Data: mustNodeData(models.DBInputConfig{}),
	}
	nb.nodes[id] = node
	return nb
}

func (nb *NodeBuilder) Connect(fromID, toID int) *NodeBuilder {
	fromNode := nb.nodes[fromID]
	toNode := nb.nodes[toID]

	if fromNode == nil || toNode == nil {
		panic("nodes must exist before connecting")
	}

	// Create a lightweight copy of toNode for the output port (without ports to avoid circular refs)
	toNodeCopy := models.Node{
		ID:   toNode.ID,
		Type: toNode.Type,
		Name: toNode.Name,
		Data: toNode.Data,
	}

	// Create output port on fromNode that points to toNode
	outputPort := models.Port{
		ID:     nb.portID,
		Type:   models.PortNodeFlowOutput,
		Node:   toNodeCopy,
		NodeID: uint(toNode.ID),
	}
	nb.portID++
	fromNode.OutputPort = append(fromNode.OutputPort, outputPort)

	// Create a lightweight copy of fromNode for the input port (without ports to avoid circular refs)
	fromNodeCopy := models.Node{
		ID:   fromNode.ID,
		Type: fromNode.Type,
		Name: fromNode.Name,
		Data: fromNode.Data,
	}

	// Create input port on toNode that points to fromNode
	inputPort := models.Port{
		ID:     nb.portID,
		Type:   models.PortNodeFlowInput,
		Node:   fromNodeCopy,
		NodeID: uint(fromNode.ID),
	}
	nb.portID++
	toNode.InputPort = append(toNode.InputPort, inputPort)

	return nb
}

func (nb *NodeBuilder) Build() []models.Node {
	result := make([]models.Node, 0, len(nb.nodes))
	for _, node := range nb.nodes {
		result = append(result, *node)
	}
	return result
}

// BuildJob creates a complete Job with the built nodes
func (nb *NodeBuilder) BuildJob(id uint, name string) *models.Job {
	return &models.Job{
		ID:          id,
		Name:        name,
		Description: "Test job",
		CreatorID:   1,
		Active:      true,
		Nodes:       nb.Build(),
	}
}

func TestNewJobExecution(t *testing.T) {
	job := &models.Job{
		ID:          1,
		Name:        "Test Job",
		Description: "Test Description",
		CreatorID:   1,
		Active:      true,
	}

	execution := NewJobExecution(job)

	assert.NotNil(t, execution)
	assert.Equal(t, job, execution.Job)
	assert.NotNil(t, execution.Context)
}

func TestWithDbConnection(t *testing.T) {
	job := &models.Job{ID: 1, Name: "Test Job"}
	execution := NewJobExecution(job)

	conn1 := models.DBConnectionConfig{
		Type:     models.DBTypePostgres,
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		Username: "user",
		Password: "pass",
	}

	conn2 := models.DBConnectionConfig{
		Type:     models.DBTypeMySQL,
		Host:     "localhost",
		Port:     3306,
		Database: "testdb2",
		Username: "user2",
		Password: "pass2",
	}

	// Add first connection
	execution.withDbConnection(conn1)
	assert.Len(t, execution.Context.DBConnections, 1)
	assert.Equal(t, conn1.GetConnectionID(), execution.Context.DBConnections[0].GetConnectionID())

	// Add second connection
	execution.withDbConnection(conn2)
	assert.Len(t, execution.Context.DBConnections, 2)

	// Try to add duplicate connection (should not add)
	execution.withDbConnection(conn1)
	assert.Len(t, execution.Context.DBConnections, 2, "Should not add duplicate connection")
}

// Mock generator for testing
type MockGenerator struct {
	BaseGenerator
	name string
}

func NewMockGenerator(name string, nodeID int, nodeType models.NodeType) *MockGenerator {
	return &MockGenerator{
		BaseGenerator: BaseGenerator{
			nodeID:   nodeID,
			nodeType: nodeType,
		},
		name: name,
	}
}

func (m *MockGenerator) GenerateCode(_ *ExecutionContext, _ string) error {
	return nil
}

// ============================================================================
// Job Node Ordering Tests
// These tests verify that jobs correctly order their nodes for execution
// based on dependencies and graph structure
// ============================================================================

func TestJob_NodeOrdering(t *testing.T) {
	tests := []struct {
		name              string
		buildJob          func() *models.Job
		expectedStepCount int
		expectError       bool
		errorContains     string
		verifySteps       func(t *testing.T, steps []Step)
	}{
		{
			name: "empty job returns error",
			buildJob: func() *models.Job {
				return &models.Job{
					ID:    1,
					Name:  "Empty Job",
					Nodes: []models.Node{},
				}
			},
			expectedStepCount: 0,
			expectError:       true,
			errorContains:     "has no nodes",
			verifySteps:       nil,
		},
		{
			name: "job without start node returns error",
			buildJob: func() *models.Job {
				return NewNodeBuilder().
					AddNode(1, models.NodeTypeDBInput, "Input").
					AddNode(2, models.NodeTypeDBOutput, "Output").
					BuildJob(1, "No Start Job")
			},
			expectedStepCount: 0,
			expectError:       true,
			errorContains:     "has no start node",
			verifySteps:       nil,
		},
		{
			name: "single start node executes in one step",
			buildJob: func() *models.Job {
				return NewNodeBuilder().
					AddNode(1, models.NodeTypeStart, "Start").
					BuildJob(1, "Single Node Job")
			},
			expectedStepCount: 1,
			verifySteps: func(t *testing.T, steps []Step) {
				assert.Len(t, steps[0].nodes, 1)
				assert.Equal(t, "Start", steps[0].nodes[0].Name)
				assert.Equal(t, models.NodeTypeStart, steps[0].nodes[0].Type)
			},
		},
		{
			name: "linear chain maintains sequential order",
			buildJob: func() *models.Job {
				// Start -> Input -> Map -> Output
				return NewNodeBuilder().
					AddNode(1, models.NodeTypeStart, "Start").
					AddNode(2, models.NodeTypeDBInput, "Input").
					AddNode(3, models.NodeTypeMap, "Map").
					AddNode(4, models.NodeTypeDBOutput, "Output").
					Connect(1, 2).
					Connect(2, 3).
					Connect(3, 4).
					BuildJob(1, "Linear Chain Job")
			},
			expectedStepCount: 4,
			verifySteps: func(t *testing.T, steps []Step) {
				// Verify sequential execution order
				nodeOrder := []struct {
					name     string
					nodeType models.NodeType
				}{
					{"Start", models.NodeTypeStart},
					{"Input", models.NodeTypeDBInput},
					{"Map", models.NodeTypeMap},
					{"Output", models.NodeTypeDBOutput},
				}

				for i, expected := range nodeOrder {
					require.Len(t, steps[i].nodes, 1, "Step %d should have exactly 1 node", i)
					assert.Equal(t, expected.name, steps[i].nodes[0].Name, "Step %d node name", i)
					assert.Equal(t, expected.nodeType, steps[i].nodes[0].Type, "Step %d node type", i)
				}
			},
		},
		{
			name: "parallel branches execute at same level",
			buildJob: func() *models.Job {
				// Start -> [Input1, Input2]
				return NewNodeBuilder().
					AddNode(1, models.NodeTypeStart, "Start").
					AddNode(2, models.NodeTypeDBInput, "Input1").
					AddNode(3, models.NodeTypeDBInput, "Input2").
					Connect(1, 2).
					Connect(1, 3).
					BuildJob(1, "Parallel Branches Job")
			},
			expectedStepCount: 2,
			verifySteps: func(t *testing.T, steps []Step) {
				// Level 0: Start
				assert.Len(t, steps[0].nodes, 1)
				assert.Equal(t, "Start", steps[0].nodes[0].Name)

				// Level 1: Both parallel inputs
				assert.Len(t, steps[1].nodes, 2, "Parallel nodes should execute at same level")
				nodeNames := []string{steps[1].nodes[0].Name, steps[1].nodes[1].Name}
				assert.Contains(t, nodeNames, "Input1")
				assert.Contains(t, nodeNames, "Input2")
			},
		},
		{
			name: "diamond pattern waits for all dependencies",
			buildJob: func() *models.Job {
				// Start -> [Input1, Input2] -> Output
				return NewNodeBuilder().
					AddNode(1, models.NodeTypeStart, "Start").
					AddNode(2, models.NodeTypeDBInput, "Input1").
					AddNode(3, models.NodeTypeDBInput, "Input2").
					AddNode(4, models.NodeTypeDBOutput, "Output").
					Connect(1, 2).
					Connect(1, 3).
					Connect(2, 4).
					Connect(3, 4).
					BuildJob(1, "Diamond Pattern Job")
			},
			expectedStepCount: 3,
			verifySteps: func(t *testing.T, steps []Step) {
				// Level 0: Start
				assert.Len(t, steps[0].nodes, 1)
				assert.Equal(t, "Start", steps[0].nodes[0].Name)

				// Level 1: Parallel inputs
				assert.Len(t, steps[1].nodes, 2)

				// Level 2: Output waits for both dependencies
				assert.Len(t, steps[2].nodes, 1)
				assert.Equal(t, "Output", steps[2].nodes[0].Name)
			},
		},
		{
			name: "complex graph with multiple parallel paths",
			buildJob: func() *models.Job {
				//        Start
				//       /     \
				//   Input1   Input2
				//      |       |  \
				//     Map1    Map2 Map3
				//       \      |  /
				//         Output
				return NewNodeBuilder().
					AddNode(1, models.NodeTypeStart, "Start").
					AddNode(2, models.NodeTypeDBInput, "Input1").
					AddNode(3, models.NodeTypeDBInput, "Input2").
					AddNode(4, models.NodeTypeMap, "Map1").
					AddNode(5, models.NodeTypeMap, "Map2").
					AddNode(6, models.NodeTypeMap, "Map3").
					AddNode(7, models.NodeTypeDBOutput, "Output").
					Connect(1, 2).
					Connect(1, 3).
					Connect(2, 4).
					Connect(3, 5).
					Connect(3, 6).
					Connect(4, 7).
					Connect(5, 7).
					Connect(6, 7).
					BuildJob(1, "Complex Graph Job")
			},
			expectedStepCount: 4,
			verifySteps: func(t *testing.T, steps []Step) {
				// Level 0: Start
				assert.Len(t, steps[0].nodes, 1)
				assert.Equal(t, "Start", steps[0].nodes[0].Name)

				// Level 1: Parallel inputs
				assert.Len(t, steps[1].nodes, 2)

				// Level 2: All three map nodes (Map2 and Map3 are parallel from Input2)
				assert.Len(t, steps[2].nodes, 3)

				// Level 3: Output waits for all map nodes
				assert.Len(t, steps[3].nodes, 1)
				assert.Equal(t, "Output", steps[3].nodes[0].Name)
			},
		},
		{
			name: "unequal path depths resolve to longest path",
			buildJob: func() *models.Job {
				// Start -> Input1 -> Map1 -> Output
				//   |
				//   +----> Input2 ---------> Output
				return NewNodeBuilder().
					AddNode(1, models.NodeTypeStart, "Start").
					AddNode(2, models.NodeTypeDBInput, "Input1").
					AddNode(3, models.NodeTypeDBInput, "Input2").
					AddNode(4, models.NodeTypeMap, "Map1").
					AddNode(5, models.NodeTypeDBOutput, "Output").
					Connect(1, 2).
					Connect(1, 3).
					Connect(2, 4).
					Connect(3, 5). // Short path
					Connect(4, 5). // Long path
					BuildJob(1, "Unequal Depth Paths Job")
			},
			expectedStepCount: 4,
			verifySteps: func(t *testing.T, steps []Step) {
				// Output must wait for the longest dependency chain (Map1 at level 2)
				require.Len(t, steps, 4)
				assert.Equal(t, "Output", steps[3].nodes[0].Name, "Output should be at level 3, waiting for Map1")
			},
		},
		{
			name: "isolated nodes are not included in execution",
			buildJob: func() *models.Job {
				return NewNodeBuilder().
					AddNode(1, models.NodeTypeStart, "Start").
					AddNode(2, models.NodeTypeDBInput, "Connected").
					AddNode(3, models.NodeTypeDBInput, "Isolated").
					Connect(1, 2).
					BuildJob(1, "Reachability Test Job")
			},
			expectedStepCount: 2,
			verifySteps: func(t *testing.T, steps []Step) {
				// Collect all node names from steps
				allNodeNames := make([]string, 0)
				for _, step := range steps {
					for _, node := range step.nodes {
						allNodeNames = append(allNodeNames, node.Name)
					}
				}

				assert.Contains(t, allNodeNames, "Start")
				assert.Contains(t, allNodeNames, "Connected")
				assert.NotContains(t, allNodeNames, "Isolated", "Unreachable nodes should not be executed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := tt.buildJob()

			// Execute the job's node ordering
			execution, err := NewJobExecution(job).withStepsSetup()

			// Check for expected error
			if tt.expectError {
				require.Error(t, err, "Expected an error but got none")
				assert.Contains(t, err.Error(), tt.errorContains, "Error message should contain expected text")
				assert.Nil(t, execution, "Execution should be nil when error occurs")
				return
			}

			// No error expected
			require.NoError(t, err, "Unexpected error")
			require.NotNil(t, execution, "Execution should not be nil")

			// Verify the job and execution are properly linked
			assert.Equal(t, job, execution.Job, "Execution should reference the job")
			assert.Equal(t, job.Name, execution.Job.Name, "Job name should be preserved")

			// Verify step count
			assert.Len(t, execution.Steps, tt.expectedStepCount, "Incorrect number of execution steps")

			// Run custom verification
			if tt.verifySteps != nil {
				tt.verifySteps(t, execution.Steps)
			}
		})
	}
}

// ============================================================================
// Job Node Dependency Tests
// These tests specifically verify dependency resolution and level assignment
// ============================================================================

func TestJob_NodeDependencies(t *testing.T) {
	t.Run("node with multiple dependencies waits for all", func(t *testing.T) {
		// Create a node that depends on 3 parallel predecessors
		job := NewNodeBuilder().
			AddNode(1, models.NodeTypeStart, "Start").
			AddNode(2, models.NodeTypeDBInput, "Input1").
			AddNode(3, models.NodeTypeDBInput, "Input2").
			AddNode(4, models.NodeTypeDBInput, "Input3").
			AddNode(5, models.NodeTypeMap, "Merger").
			Connect(1, 2).
			Connect(1, 3).
			Connect(1, 4).
			Connect(2, 5).
			Connect(3, 5).
			Connect(4, 5).
			BuildJob(1, "Multi-Dependency Job")

		execution, err := NewJobExecution(job).withStepsSetup()

		require.NoError(t, err)
		require.NotNil(t, execution)
		require.Len(t, execution.Steps, 3)

		// Level 0: Start
		assert.Len(t, execution.Steps[0].nodes, 1)

		// Level 1: All three inputs in parallel
		assert.Len(t, execution.Steps[1].nodes, 3)

		// Level 2: Merger waits for all inputs
		assert.Len(t, execution.Steps[2].nodes, 1)
		assert.Equal(t, "Merger", execution.Steps[2].nodes[0].Name)
	})

	t.Run("transitive dependencies are respected", func(t *testing.T) {
		// A -> B -> C -> D (linear chain ensures transitive dependencies)
		job := NewNodeBuilder().
			AddNode(1, models.NodeTypeStart, "A").
			AddNode(2, models.NodeTypeDBInput, "B").
			AddNode(3, models.NodeTypeMap, "C").
			AddNode(4, models.NodeTypeDBOutput, "D").
			Connect(1, 2).
			Connect(2, 3).
			Connect(3, 4).
			BuildJob(1, "Transitive Deps Job")

		execution, err := NewJobExecution(job).withStepsSetup()

		require.NoError(t, err)
		require.NotNil(t, execution)

		// Each node should be at progressively higher levels
		require.Len(t, execution.Steps, 4)
		for i := 0; i < 4; i++ {
			assert.Len(t, execution.Steps[i].nodes, 1, "Step %d should have exactly 1 node", i)
		}
	})

	t.Run("nodes at same level have no dependencies on each other", func(t *testing.T) {
		job := NewNodeBuilder().
			AddNode(1, models.NodeTypeStart, "Start").
			AddNode(2, models.NodeTypeDBInput, "Parallel1").
			AddNode(3, models.NodeTypeDBInput, "Parallel2").
			AddNode(4, models.NodeTypeDBInput, "Parallel3").
			Connect(1, 2).
			Connect(1, 3).
			Connect(1, 4).
			BuildJob(1, "Parallel Nodes Job")

		execution, err := NewJobExecution(job).withStepsSetup()

		require.NoError(t, err)
		require.NotNil(t, execution)
		require.Len(t, execution.Steps, 2)

		// All parallel nodes should be at level 1
		parallelNodes := execution.Steps[1].nodes
		assert.Len(t, parallelNodes, 3)

		// Verify they all have the same predecessor (Start)
		for _, node := range parallelNodes {
			prevNodes := node.GetPrevFlowNode()
			require.Len(t, prevNodes, 1, "Parallel nodes should have exactly 1 predecessor")
			assert.Equal(t, "Start", prevNodes[0].Name)
		}
	})
}

func TestJobExecution_Build(t *testing.T) {
	t.Run("builds generators for simple pipeline", func(t *testing.T) {
		// Create a simple pipeline: Start -> Input -> Output
		builder := NewNodeBuilder()
		builder.AddNode(1, models.NodeTypeStart, "Start")

		// Add DB Input node with proper config
		inputNode := &models.Node{
			ID:   2,
			Type: models.NodeTypeDBInput,
			Name: "Input",
			Data: mustNodeData(models.DBInputConfig{
				Connection: models.DBConnectionConfig{
					Type:     models.DBTypePostgres,
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					Username: "user",
					Password: "pass",
				},
				DbSchema: "public",
			}),
		}
		builder.nodes[2] = inputNode

		// Add DB Output node with proper config
		outputNode := &models.Node{
			ID:   3,
			Type: models.NodeTypeDBOutput,
			Name: "Output",
			Data: mustNodeData(models.DBOutputConfig{
				Connection: models.DBConnectionConfig{
					Type:     models.DBTypePostgres,
					Host:     "localhost",
					Port:     5432,
					Database: "testdb",
					Username: "user",
					Password: "pass",
				},
				Table:     "output",
				Mode:      "insert",
				BatchSize: 100,
			}),
		}
		builder.nodes[3] = outputNode

		builder.Connect(1, 2).Connect(2, 3)
		job := builder.BuildJob(1, "Simple Pipeline")

		// Build the execution
		execution, err := NewJobExecution(job).Build()

		require.NoError(t, err)
		require.NotNil(t, execution)

	})

	t.Run("builds generators maintaining step order", func(t *testing.T) {
		// Create pipeline with parallel branches:
		// Start -> [Input1, Input2] -> Output
		builder := NewNodeBuilder()
		builder.AddNode(1, models.NodeTypeStart, "Start")

		// Input1
		input1 := &models.Node{
			ID:   2,
			Type: models.NodeTypeDBInput,
			Name: "Input1",
			Data: mustNodeData(models.DBInputConfig{
				Connection: models.DBConnectionConfig{
					Type: models.DBTypePostgres, Host: "localhost",
					Port: 5432, Database: "testdb", Username: "user", Password: "pass",
				},
				DbSchema: "public",
			}),
		}
		builder.nodes[2] = input1

		// Input2
		input2 := &models.Node{
			ID:   3,
			Type: models.NodeTypeDBInput,
			Name: "Input2",
			Data: mustNodeData(models.DBInputConfig{
				Connection: models.DBConnectionConfig{
					Type: models.DBTypePostgres, Host: "localhost",
					Port: 5432, Database: "testdb", Username: "user", Password: "pass",
				},
				DbSchema: "public",
			}),
		}
		builder.nodes[3] = input2

		// Output
		output := &models.Node{
			ID:   4,
			Type: models.NodeTypeDBOutput,
			Name: "Output",
			Data: mustNodeData(models.DBOutputConfig{
				Connection: models.DBConnectionConfig{
					Type: models.DBTypePostgres, Host: "localhost",
					Port: 5432, Database: "testdb", Username: "user", Password: "pass",
				},
				Table: "output", Mode: "insert", BatchSize: 100,
			}),
		}
		builder.nodes[4] = output

		builder.Connect(1, 2).Connect(1, 3).Connect(2, 4).Connect(3, 4)
		job := builder.BuildJob(1, "Parallel Pipeline")

		// Build the execution
		execution, err := NewJobExecution(job).Build()

		require.NoError(t, err)
		require.NotNil(t, execution)
	})

	t.Run("returns error on invalid job", func(t *testing.T) {
		job := &models.Job{
			ID:    1,
			Name:  "Empty Job",
			Nodes: []models.Node{},
		}

		execution, err := NewJobExecution(job).Build()

		require.Error(t, err)
		assert.Nil(t, execution)
		assert.Contains(t, err.Error(), "has no nodes")
	})
}

func TestJobExecution_Generation(t *testing.T) {
	// Create output directory in the project root for inspection
	// This will persist after the test runs so you can see the generated code
	outputDir := "../../generated_test_output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}
	t.Logf("Output will be written to: %s", outputDir)

	// Create all nodes first without ports to avoid circular dependencies
	dbOutputNode := models.Node{
		ID:         5,
		Type:       models.NodeTypeDBOutput,
		Name:       "DB Output Node",
		InputPort:  nil, // Will be set later
		OutputPort: nil,
		Data:       nil,
		JobID:      1,
	}

	dbOutputNode.SetData(models.DBOutputConfig{
		Table:     "OUTPUT_TABLE",
		Mode:      "INSERT",
		BatchSize: 500,
		Connection: models.DBConnectionConfig{
			Type:     models.DBTypePostgres,
			Host:     "DC-CENTRIC-01",
			Port:     5432,
			Database: "TEST_DB",
			Username: "postgres",
			Password: "postgres",
			SSLMode:  "disable",
			Extra:    nil,
		},
	})

	firstDBInputNode := models.Node{
		ID:         2,
		Type:       models.NodeTypeDBInput,
		Name:       "First DB Input",
		InputPort:  nil, // Will be set later
		OutputPort: nil, // Will be set later
		Data:       nil,
		JobID:      1,
	}

	secondDBInputNode := models.Node{
		ID:         3,
		Type:       models.NodeTypeDBInput,
		Name:       "Second DB Input",
		InputPort:  nil, // Will be set later
		OutputPort: nil, // Will be set later
		Data:       nil,
		JobID:      1,
	}

	mapNode := models.Node{
		ID:         4,
		Type:       models.NodeTypeMap,
		Name:       "Map Node",
		InputPort:  nil, // Will be set later
		OutputPort: nil, // Will be set later
		Data:       nil,
		JobID:      1,
	}
	mapNode.SetData(models.MapConfig{})

	// Now set up the ports with proper node references
	firstDBInputNode.OutputPort = []models.Port{
		// FLOW port
		{
			ID:     3,
			Type:   models.PortNodeFlowOutput,
			Node:   mapNode,
			NodeID: 2,
		},
		// DATA port
		{
			ID:     11,
			Type:   models.PortTypeOutput,
			Node:   mapNode,
			NodeID: 2,
		},
	}

	mapNode.InputPort = []models.Port{
		// FLOW ports (execution order)
		{
			ID:     4,
			Type:   models.PortNodeFlowInput,
			Node:   firstDBInputNode,
			NodeID: 2,
		},
		{
			ID:     5,
			Type:   models.PortNodeFlowInput,
			Node:   secondDBInputNode,
			NodeID: 3,
		},
		// DATA ports (data input)
		{
			ID:     13,
			Type:   models.PortTypeInput,
			Node:   firstDBInputNode,
			NodeID: 2,
		},
		{
			ID:     14,
			Type:   models.PortTypeInput,
			Node:   secondDBInputNode,
			NodeID: 3,
		},
	}
	mapNode.OutputPort = []models.Port{
		// FLOW port
		{
			ID:     6,
			Type:   models.PortNodeFlowOutput,
			Node:   dbOutputNode,
			NodeID: 4,
		},
		// DATA port
		{
			ID:     15,
			Type:   models.PortTypeOutput,
			Node:   dbOutputNode,
			NodeID: 4,
		},
	}
	secondDBInputNode.OutputPort = []models.Port{
		// FLOW port
		{
			ID:     10,
			Type:   models.PortNodeFlowOutput,
			Node:   mapNode,
			NodeID: 3,
		},
		// DATA port
		{
			ID:     12,
			Type:   models.PortTypeOutput,
			Node:   mapNode,
			NodeID: 3,
		},
	}

	firstDBInputNode.SetData(models.DBInputConfig{
		Query:    "select * from tgcliente",
		DbSchema: "public",
		Connection: models.DBConnectionConfig{
			Type:     models.DBTypeSQLServer,
			Host:     "DC-SQL-01",
			Port:     1433,
			Database: "ICarDEMO",
			Username: "sa",
			Password: "sa",
			SSLMode:  "disable",
			Extra:    nil,
		},
	})

	secondDBInputNode.SetData(models.DBInputConfig{
		Query:    "select * from tgclienteProtec",
		DbSchema: "dbo",
		Connection: models.DBConnectionConfig{
			Type:     models.DBTypeSQLServer,
			Host:     "DC-SQL-02",
			Port:     1895,
			Database: "ICarKKKKK",
			Username: "sa",
			Password: "sa",
			SSLMode:  "disable",
			Extra:    nil,
		},
	})

	// Node 1: Start (create first to be able to reference it)
	startNode := models.Node{
		ID:         1,
		Type:       models.NodeTypeStart,
		Name:       "Starter",
		InputPort:  nil,
		OutputPort: nil,
		Data:       nil,
		JobID:      1,
	}

	// Now set up InputPorts for DB Input nodes (they receive from startNode)
	firstDBInputNode.InputPort = []models.Port{
		{
			ID:     2,
			Type:   models.PortNodeFlowInput,
			Node:   startNode,
			NodeID: 1,
		},
	}

	secondDBInputNode.InputPort = []models.Port{
		{
			ID:     9,
			Type:   models.PortNodeFlowInput,
			Node:   startNode,
			NodeID: 1,
		},
	}

	// Set up dbOutputNode InputPort (receives from mapNode)
	dbOutputNode.InputPort = []models.Port{
		// FLOW port
		{
			ID:     7,
			Type:   models.PortNodeFlowInput,
			Node:   mapNode,
			NodeID: 4,
		},
		// DATA port
		{
			ID:     16,
			Type:   models.PortTypeInput,
			Node:   mapNode,
			NodeID: 4,
		},
	}

	// Finally, set startNode OutputPorts (sends to both DB Input nodes)
	startNode.OutputPort = []models.Port{
		{
			ID:     1,
			Type:   models.PortNodeFlowOutput,
			Node:   firstDBInputNode,
			NodeID: 1,
		},
		{
			ID:     8,
			Type:   models.PortNodeFlowOutput,
			Node:   secondDBInputNode,
			NodeID: 1,
		},
	}

	// Liste finale
	nodes := []models.Node{
		startNode,
		firstDBInputNode,
		secondDBInputNode,
		mapNode,
		dbOutputNode,
		models.Node{
			ID:    89,
			Type:  models.NodeTypeDBInput,
			Name:  "unlinked",
			Data:  nil,
			JobID: 1,
		},
	}
	test := models.Job{
		ID:          1,
		Name:        "Test job",
		Description: "Un job de test",
		CreatorID:   1,
		Active:      true,
		Nodes:       nodes,
		OutputPath:  outputDir,
	}

	execution, err := NewJobExecution(&test).Build()

	require.NoError(t, err)
	require.NotNil(t, execution)

	// Verify that main.go was generated
	generatedFile := outputDir + "/main.go"
	_, err = os.Stat(generatedFile)
	require.NoError(t, err, "main.go should have been generated")

	// Read and log the generated content
	content, err := os.ReadFile(generatedFile)
	require.NoError(t, err)
	t.Logf("Generated main.go (%d bytes):\n%s", len(content), string(content))
}
