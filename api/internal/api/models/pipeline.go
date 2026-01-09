package models

import (
	"database/sql"
	"fmt"
	"time"
)

// Pipeline represents a sequence of nodes to execute
type Pipeline struct {
	Job        *Job
	Nodes      []Node
	Generators []Generator
	Context    *ExecutionContext
}

// PipelineResult contains the execution result
type PipelineResult struct {
	Success   bool
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Errors    []error
	Output    map[string]interface{}
}

// NewPipeline creates a new pipeline from a job
func NewPipeline(job *Job, dbConn *sql.DB) *Pipeline {
	return &Pipeline{
		Job:        job,
		Nodes:      make([]Node, 0),
		Generators: make([]Generator, 0),
		Context:    NewExecutionContext(dbConn),
	}
}

// AddNode adds a node to the pipeline
func (p *Pipeline) AddNode(node Node) error {
	// Generate the generator for this node
	generator, err := node.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate for node %d: %w", node.ID, err)
	}

	p.Nodes = append(p.Nodes, node)
	p.Generators = append(p.Generators, generator)
	return nil
}

// BuildFromJob builds a pipeline from a job by ordering nodes based on connections
func (p *Pipeline) BuildFromJob(nodes []Node) error {
	if len(nodes) == 0 {
		return fmt.Errorf("no nodes provided")
	}

	// TODO: Implement topological sort based on node connections
	// For now, just add nodes in the order provided
	for _, node := range nodes {
		if err := p.AddNode(node); err != nil {
			return err
		}
	}

	return nil
}

// Execute runs all generators in the pipeline sequentially
func (p *Pipeline) Execute() *PipelineResult {
	result := &PipelineResult{
		Success:   true,
		StartTime: time.Now(),
		Errors:    make([]error, 0),
		Output:    make(map[string]interface{}),
	}

	// Execute each generator in sequence
	for i, generator := range p.Generators {
		nodeID := generator.GetNodeID()
		nodeType := generator.GetType()

		fmt.Printf("Executing node %d (type: %s)\n", nodeID, nodeType)

		if err := generator.Execute(p.Context); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Errorf("node %d failed: %w", nodeID, err))

			// Continue or stop based on error handling strategy
			// For now, we stop on first error
			break
		}

		fmt.Printf("Node %d completed successfully (%d/%d)\n", nodeID, i+1, len(p.Generators))
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Output = p.Context.DataFlow

	return result
}

// Validate checks if the pipeline is valid
func (p *Pipeline) Validate() error {
	if len(p.Generators) == 0 {
		return fmt.Errorf("pipeline has no generators")
	}

	if p.Context == nil {
		return fmt.Errorf("pipeline has no execution context")
	}

	if p.Context.DBConnection == nil {
		return fmt.Errorf("pipeline has no database connection")
	}

	// Check for valid node sequence
	// First node should be an input node
	if len(p.Generators) > 0 {
		firstType := p.Generators[0].GetType()
		if firstType != NodeTypeDBInput {
			return fmt.Errorf("pipeline should start with a DB input node, got %s", firstType)
		}
	}

	// Last node should be an output node
	if len(p.Generators) > 1 {
		lastType := p.Generators[len(p.Generators)-1].GetType()
		if lastType != NodeTypeDBOutput {
			return fmt.Errorf("pipeline should end with a DB output node, got %s", lastType)
		}
	}

	return nil
}

// GetNodeOrder returns the ordered list of node IDs
func (p *Pipeline) GetNodeOrder() []int {
	order := make([]int, len(p.Generators))
	for i, gen := range p.Generators {
		order[i] = gen.GetNodeID()
	}
	return order
}

// PipelineBuilder helps build pipelines with a fluent API
type PipelineBuilder struct {
	pipeline *Pipeline
}

// NewPipelineBuilder creates a new pipeline builder
func NewPipelineBuilder(job *Job, dbConn *sql.DB) *PipelineBuilder {
	return &PipelineBuilder{
		pipeline: NewPipeline(job, dbConn),
	}
}

// WithNode adds a node to the pipeline
func (b *PipelineBuilder) WithNode(node Node) *PipelineBuilder {
	if err := b.pipeline.AddNode(node); err != nil {
		// Store error in context for later retrieval
		b.pipeline.Context.Errors = append(b.pipeline.Context.Errors, err)
	}
	return b
}

// WithNodes adds multiple nodes to the pipeline
func (b *PipelineBuilder) WithNodes(nodes []Node) *PipelineBuilder {
	if err := b.pipeline.BuildFromJob(nodes); err != nil {
		b.pipeline.Context.Errors = append(b.pipeline.Context.Errors, err)
	}
	return b
}

// Build finalizes and returns the pipeline
func (b *PipelineBuilder) Build() (*Pipeline, error) {
	// Check if there were any errors during building
	if len(b.pipeline.Context.Errors) > 0 {
		return nil, fmt.Errorf("pipeline build failed: %v", b.pipeline.Context.Errors)
	}

	// Validate the pipeline
	if err := b.pipeline.Validate(); err != nil {
		return nil, err
	}

	return b.pipeline, nil
}
