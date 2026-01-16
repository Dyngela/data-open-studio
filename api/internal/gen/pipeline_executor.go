package gen

import (
	"fmt"
	"log"
	"time"

	"api/internal/api/models"
)

type Step struct {
	nodes []models.Node
}

// PipelineExecutor runs a job pipeline using the runtime interpreter
type PipelineExecutor struct {
	Job   *models.Job
	Ctx   *ExecutionContext
	Steps []Step
}

// NewPipelineExecutor creates a new executor for a job
func NewPipelineExecutor(job *models.Job) *PipelineExecutor {
	return &PipelineExecutor{
		Job: job,
		Ctx: NewExecutionContext(),
	}
}

// Run executes the entire pipeline
func (pe *PipelineExecutor) Run() error {
	startTime := time.Now()
	log.Printf("[%s] Starting pipeline execution", pe.Job.Name)

	defer pe.Ctx.Close()

	// Build execution steps (topological order)
	if err := pe.buildSteps(); err != nil {
		return fmt.Errorf("failed to build execution steps: %w", err)
	}

	// Initialize all database connections upfront
	if err := pe.initConnections(); err != nil {
		return fmt.Errorf("failed to initialize connections: %w", err)
	}

	// Execute each step
	for stepIdx, step := range pe.Steps {
		if pe.Ctx.IsCancelled() {
			return fmt.Errorf("pipeline cancelled at step %d", stepIdx)
		}

		if err := pe.executeStep(stepIdx, step); err != nil {
			return fmt.Errorf("step %d failed: %w", stepIdx, err)
		}
	}

	duration := time.Since(startTime)
	log.Printf("[%s] Pipeline completed in %v", pe.Job.Name, duration)
	return nil
}

// buildSteps creates topologically ordered execution steps
func (pe *PipelineExecutor) buildSteps() error {
	nodeMap := make(map[int]models.Node)
	for _, node := range pe.Job.Nodes {
		nodeMap[node.ID] = node
	}

	// Find start nodes
	startNodes := make([]models.Node, 0)
	for _, node := range pe.Job.Nodes {
		if node.Type == models.NodeTypeStart {
			startNodes = append(startNodes, node)
		}
	}

	if len(startNodes) == 0 {
		return fmt.Errorf("no start node found")
	}

	// BFS to assign levels
	levels := make(map[int]int)
	visited := make(map[int]bool)
	queue := make([]models.Node, 0)

	for _, start := range startNodes {
		queue = append(queue, start)
		levels[start.ID] = 0
		visited[start.ID] = true
	}

	maxLevel := 0
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		currentLevel := levels[current.ID]

		// Find next nodes via flow ports
		for _, port := range current.OutputPort {
			if port.Type == models.PortNodeFlowOutput && port.Node.ID != 0 {
				nextID := port.Node.ID
				nextNode, exists := nodeMap[nextID]
				if !exists {
					continue
				}

				newLevel := currentLevel + 1
				if existingLevel, seen := levels[nextID]; !seen || newLevel > existingLevel {
					levels[nextID] = newLevel
					if newLevel > maxLevel {
						maxLevel = newLevel
					}
				}

				if !visited[nextID] {
					visited[nextID] = true
					queue = append(queue, nextNode)
				}
			}
		}
	}

	// Group nodes by level
	pe.Steps = make([]Step, maxLevel+1)
	for i := range pe.Steps {
		pe.Steps[i] = Step{nodes: make([]models.Node, 0)}
	}

	for _, node := range pe.Job.Nodes {
		if level, ok := levels[node.ID]; ok {
			pe.Steps[level].nodes = append(pe.Steps[level].nodes, node)
		}
	}

	return nil
}

// initConnections initializes all database connections needed
func (pe *PipelineExecutor) initConnections() error {
	for _, node := range pe.Job.Nodes {
		switch node.Type {
		case models.NodeTypeDBInput:
			config, err := node.GetDBInputConfig()
			if err != nil {
				continue
			}
			if err := pe.Ctx.InitConnection(config.Connection); err != nil {
				return err
			}

		case models.NodeTypeDBOutput:
			config, err := node.GetDBOutputConfig()
			if err != nil {
				continue
			}
			if err := pe.Ctx.InitConnection(config.Connection); err != nil {
				return err
			}
		}
	}
	return nil
}

// executeStep executes all nodes in a step
func (pe *PipelineExecutor) executeStep(stepIdx int, step Step) error {
	// Filter out start nodes
	executableNodes := make([]models.Node, 0)
	for _, node := range step.nodes {
		if node.Type != models.NodeTypeStart {
			executableNodes = append(executableNodes, node)
		}
	}

	if len(executableNodes) == 0 {
		log.Printf("Step %d: Pipeline start", stepIdx)
		return nil
	}

	log.Printf("Step %d: Executing %d node(s)", stepIdx, len(executableNodes))

	for _, node := range executableNodes {
		if err := pe.executeNode(node); err != nil {
			return fmt.Errorf("node %d (%s) failed: %w", node.ID, node.Name, err)
		}
	}

	return nil
}

// executeNode executes a single node
func (pe *PipelineExecutor) executeNode(node models.Node) error {
	switch node.Type {
	case models.NodeTypeDBInput:
		return pe.executeDBInput(node)
	case models.NodeTypeMap:
		return pe.executeMap(node)
	case models.NodeTypeDBOutput:
		return pe.executeDBOutput(node)
	default:
		return fmt.Errorf("unknown node type: %s", node.Type)
	}
}

// executeDBInput executes a DB input node
func (pe *PipelineExecutor) executeDBInput(node models.Node) error {
	config, err := node.GetDBInputConfig()
	if err != nil {
		return err
	}

	stream, err := ExecuteDBInput(pe.Ctx, node.ID, node.Name, config)
	if err != nil {
		return err
	}

	pe.Ctx.SetStream(node.ID, stream)
	return nil
}

// executeMap executes a map/transform node
func (pe *PipelineExecutor) executeMap(node models.Node) error {
	config, err := node.GetMapConfig()
	if err != nil {
		return err
	}

	// Get data input nodes
	dataInputNodes := node.GetDataInputNodes()
	if len(dataInputNodes) == 0 {
		return fmt.Errorf("map node has no data inputs")
	}

	// Collect data from all input streams
	inputStreams := make([]*RowStream, len(dataInputNodes))
	for i, inputNode := range dataInputNodes {
		stream, exists := pe.Ctx.GetStream(inputNode.ID)
		if !exists {
			return fmt.Errorf("input stream for node %d not found", inputNode.ID)
		}
		inputStreams[i] = stream
	}

	// Collect all inputs (sync point)
	log.Printf("Node %d (%s): Collecting %d input streams", node.ID, node.Name, len(inputStreams))
	allData, err := CollectMultiple(inputStreams...)
	if err != nil {
		return fmt.Errorf("failed to collect inputs: %w", err)
	}

	// Store collected data
	for i, inputNode := range dataInputNodes {
		pe.Ctx.SetData(inputNode.ID, allData[i])
	}

	// Execute transformation
	if config.HasMultipleOutputs() {
		outputs, err := ExecuteMapMultiOutput(pe.Ctx, node.ID, node.Name, config, allData...)
		if err != nil {
			return err
		}
		// Store each output stream (using port ID or output name as key)
		// For now, store the first output as the main stream
		for name, stream := range outputs {
			if name == config.Outputs[0].Name {
				pe.Ctx.SetStream(node.ID, stream)
			}
			// TODO: Handle multiple outputs properly with port mapping
		}
	} else {
		stream, err := ExecuteMap(pe.Ctx, node.ID, node.Name, config, allData...)
		if err != nil {
			return err
		}
		pe.Ctx.SetStream(node.ID, stream)
	}

	return nil
}

// executeDBOutput executes a DB output node
func (pe *PipelineExecutor) executeDBOutput(node models.Node) error {
	config, err := node.GetDBOutputConfig()
	if err != nil {
		return err
	}

	// Get data input node
	dataInputNodes := node.GetDataInputNodes()
	if len(dataInputNodes) == 0 {
		return fmt.Errorf("output node has no data inputs")
	}

	// Get input stream (or merge if multiple)
	var inputStream *RowStream
	if len(dataInputNodes) == 1 {
		stream, exists := pe.Ctx.GetStream(dataInputNodes[0].ID)
		if !exists {
			return fmt.Errorf("input stream for node %d not found", dataInputNodes[0].ID)
		}
		inputStream = stream
	} else {
		// Merge multiple input streams
		streams := make([]*RowStream, len(dataInputNodes))
		for i, inputNode := range dataInputNodes {
			stream, exists := pe.Ctx.GetStream(inputNode.ID)
			if !exists {
				return fmt.Errorf("input stream for node %d not found", inputNode.ID)
			}
			streams[i] = stream
		}
		inputStream = MergeStreams(1000, streams...)
	}

	return ExecuteDBOutput(pe.Ctx, node.ID, node.Name, config, inputStream)
}
