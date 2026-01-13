package gen

import (
	"api/internal/api/models"
	"fmt"
	"log"
)

type Step struct {
	nodes []models.Node
}

// JobExecution represents a sequence of nodes to execute
type JobExecution struct {
	Job     *models.Job
	Context *ExecutionContext
	Steps   []Step
}

// PipelineResult contains the execution result

// NewJobExecution creates a new pipeline from a job
func NewJobExecution(job *models.Job) *JobExecution {
	return &JobExecution{
		Job:     job,
		Context: NewExecutionContext(),
	}
}

// withDbConnection adds a database connection to the execution context if not already present
func (j *JobExecution) withDbConnection(conn models.DBConnectionConfig) *JobExecution {
	existingConn := j.Context.DBConnections
	for _, v := range existingConn {
		if v.GetConnectionID() == conn.GetConnectionID() {
			return j
		}
	}
	j.Context.DBConnections = append(j.Context.DBConnections, conn)
	return j
}

// withStepsSetup sets up the execution steps based on the job's node structure
// Supports multiple start nodes for independent parallel flows
func (j *JobExecution) withStepsSetup() (*JobExecution, error) {
	if len(j.Job.Nodes) == 0 {
		return nil, fmt.Errorf("job '%s' (ID: %d) has no nodes", j.Job.Name, j.Job.ID)
	}

	nodeByID := make(map[int]*models.Node)
	for i := range j.Job.Nodes {
		nodeByID[j.Job.Nodes[i].ID] = &j.Job.Nodes[i]
	}

	// Find ALL start nodes (support multiple independent flows)
	startNodes := make([]*models.Node, 0)
	for i := range j.Job.Nodes {
		if j.Job.Nodes[i].Type == models.NodeTypeStart {
			startNodes = append(startNodes, &j.Job.Nodes[i])
		}
	}

	if len(startNodes) == 0 {
		return nil, fmt.Errorf("job '%s' (ID: %d) has no start nodes", j.Job.Name, j.Job.ID)
	}

	levels := make(map[int]int)
	var calculateLevel func(node *models.Node) int
	calculateLevel = func(node *models.Node) int {
		if level, exists := levels[node.ID]; exists {
			return level
		}

		prevNodes := node.GetPrevFlowNode()
		if len(prevNodes) == 0 {
			levels[node.ID] = 0
			return 0
		}

		maxPredLevel := -1
		for i := range prevNodes {
			predLevel := calculateLevel(&prevNodes[i])
			if predLevel > maxPredLevel {
				maxPredLevel = predLevel
			}
		}

		levels[node.ID] = maxPredLevel + 1
		return maxPredLevel + 1
	}

	// BFS traversal from each start node
	visited := make(map[int]bool)
	for _, startNode := range startNodes {
		queue := []*models.Node{startNode}
		if !visited[startNode.ID] {
			visited[startNode.ID] = true
		}

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]

			calculateLevel(current)

			for _, next := range current.GetNextFlowNode() {
				if !visited[next.ID] {
					visited[next.ID] = true
					if n := nodeByID[next.ID]; n != nil {
						queue = append(queue, n)
					}
				}
			}
		}
	}

	// Group nodes by level
	maxLevel := 0
	levelNodes := make(map[int][]models.Node)
	for nodeID, level := range levels {
		if node := nodeByID[nodeID]; node != nil {
			levelNodes[level] = append(levelNodes[level], *node)
			if level > maxLevel {
				maxLevel = level
			}
		}
	}

	// Debug: Log levels for each node
	log.Printf("Node levels calculated:")
	for nodeID, level := range levels {
		if node := nodeByID[nodeID]; node != nil {
			log.Printf("  Node %d (%s): Level %d", nodeID, node.Name, level)
		}
	}

	// Build execution steps
	j.Steps = make([]Step, 0, maxLevel+1)
	for level := 0; level <= maxLevel; level++ {
		if nodes := levelNodes[level]; len(nodes) > 0 {
			log.Printf("Step %d: %d node(s)", len(j.Steps), len(nodes))
			j.Steps = append(j.Steps, Step{nodes: nodes})
		}
	}

	log.Printf("Total steps created: %d", len(j.Steps))

	// Log unlinked nodes (not visited - these are orphaned/test nodes)
	for _, node := range j.Job.Nodes {
		if !visited[node.ID] {
			log.Printf("Warning: Node %d (%s) is not reachable from any start node", node.ID, node.Name)
		}
	}

	return j, nil
}

// withGlobalVariables Fill global variables in the execution context like db connections or file path for certificates
func (j *JobExecution) withGlobalVariables() *JobExecution {
	for _, node := range j.Job.Nodes {
		switch node.Type {
		case models.NodeTypeDBInput:
			dbInputConfig, _ := node.GetDBInputConfig()
			j.withDbConnection(dbInputConfig.Connection)
		case models.NodeTypeDBOutput:
			dbOutputConfig, _ := node.GetDBOutputConfig()
			j.withDbConnection(dbOutputConfig.Connection)
		}
	}

	return j
}

// Build builds the job file for compilation
func (j *JobExecution) Build() (*JobExecution, error) {
	// Setup execution steps and global variables
	j, err := j.withStepsSetup()
	if err != nil {
		return nil, err
	}
	j.withGlobalVariables()

	// Collect all generators for non-start nodes
	generators := make([]Generator, 0)
	for _, step := range j.Steps {
		for _, node := range step.nodes {
			// Skip start nodes as they don't have generators
			if node.Type == models.NodeTypeStart {
				continue
			}

			generator, err := NewGenerator(node)
			if err != nil {
				return nil, fmt.Errorf("failed to create generator for node ID %d: %w", node.ID, err)
			}
			generators = append(generators, generator)
			log.Printf("Collected generator for node %d (%s)", node.ID, node.Name)
		}
	}

	// Generate single main.go with all interconnected node functions
	mainGen := NewMainProgramGenerator(j)
	mainGen.Generators = generators

	if err := mainGen.Generate(j.Job.OutputPath); err != nil {
		return nil, fmt.Errorf("failed to generate main program: %w", err)
	}

	log.Printf("Successfully generated main.go with %d node functions", len(generators))
	return j, nil
}

func (j *JobExecution) Execute() error {
	return nil
}
