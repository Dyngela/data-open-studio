package gen

import (
	"api"
	"api/internal/api/models"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Step struct {
	nodes []models.Node
}

// JobExecution represents a sequence of nodes to execute
type JobExecution struct {
	Job         *models.Job
	Context     *ExecutionContext
	Steps       []Step
	FileBuilder *FileBuilder
	logger      zerolog.Logger
}

// NewJobExecution creates a new pipeline from a job
func NewJobExecution(job *models.Job) *JobExecution {
	return &JobExecution{
		Job:         job,
		Context:     NewExecutionContext(),
		FileBuilder: NewFileBuilder(job),
		logger:      api.Logger,
	}
}

// NewPipelineExecutor is an alias for NewJobExecution (used in tests)
func NewPipelineExecutor(job *models.Job) *JobExecution {
	return NewJobExecution(job)
}

// Run builds and executes the pipeline
func (j *JobExecution) Run() error {
	if _, err := j.build(); err != nil {
		return err
	}
	// TODO assainir le nom du job
	binRepo := os.Getenv("BIN_REPO")
	jobFile := j.Job.Name + ".go"
	jobPath := filepath.Join(binRepo, jobFile)
	j.logger.Info().Msgf("Writing job to %s", jobPath)
	if err := j.writeToFile(jobPath); err != nil {
		return err
	}

	modName := fmt.Sprintf("job_%s", uuid.NewString())
	if err := runCmd(binRepo, "go", "mod", "init", modName); err != nil {
		return err
	}

	if err := runCmd(binRepo, "go", "mod", "tidy"); err != nil {
		return err
	}

	if err := runCmd(binRepo, "go", "run", jobFile); err != nil {
		return err
	}

	return nil
}

func runCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
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

		prevIDs := node.GetPrevFlowNodeIDs()
		if len(prevIDs) == 0 {
			levels[node.ID] = 0
			return 0
		}

		maxPredLevel := -1
		for _, prevID := range prevIDs {
			if prev := nodeByID[prevID]; prev != nil {
				predLevel := calculateLevel(prev)
				if predLevel > maxPredLevel {
					maxPredLevel = predLevel
				}
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

			for _, nextID := range current.GetNextFlowNodeIDs() {
				if !visited[nextID] {
					visited[nextID] = true
					if n := nodeByID[nextID]; n != nil {
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

	// build execution steps
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
func (j *JobExecution) withGlobalVariables(node models.Node) (*JobExecution, error) {
	switch node.Type {
	case models.NodeTypeDBInput:
		dbInputConfig, err := node.GetDBInputConfig()
		if err != nil {
			return nil, err
		}
		j.withDbConnection(dbInputConfig.Connection)
	case models.NodeTypeDBOutput:
		dbOutputConfig, err := node.GetDBOutputConfig()
		if err != nil {
			return nil, err
		}
		j.withDbConnection(dbOutputConfig.Connection)
	}

	return j, nil
}

// build builds the job file for compilation
func (j *JobExecution) build() (*JobExecution, error) {
	// Setup execution steps
	j, err := j.withStepsSetup()
	if err != nil {
		return nil, err
	}

	j.FileBuilder.SetProgressConfig(api.GetEnv("API_HOST", "localhost:8080"), j.Job.ID)

	// Collect global variables and node IDs
	nodeIDs := make([]int, 0)
	for _, step := range j.Steps {
		for _, node := range step.nodes {
			nodeIDs = append(nodeIDs, node.ID)
			if _, err := j.withGlobalVariables(node); err != nil {
				return nil, fmt.Errorf("failed to collect globals for node %d: %w", node.ID, err)
			}
		}
	}

	// Set nodes to generate code for (excludes orphan nodes)
	j.FileBuilder.SetNodes(nodeIDs)

	// Pass DB connections and steps to FileBuilder
	j.FileBuilder.SetDBConnections(j.Context.DBConnections)
	j.FileBuilder.SetSteps(j.Steps)

	// Generate code using FileBuilder
	if err := j.FileBuilder.Build(); err != nil {
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	log.Printf("Successfully generated code with %d structs and %d functions",
		len(j.FileBuilder.GetStructs()),
		len(j.FileBuilder.GetFuncs()))

	return j, nil
}

// generateSource generates the Go source code for this job
func (j *JobExecution) generateSource() ([]byte, error) {
	return j.FileBuilder.EmitFile()
}

// writeToFile writes the generated code to a file
func (j *JobExecution) writeToFile(outputPath string) error {
	source, err := j.generateSource()
	if err != nil {
		return fmt.Errorf("failed to generate source: %w", err)
	}

	if err := os.WriteFile(outputPath, source, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("Generated code written to %s", outputPath)
	return nil
}
