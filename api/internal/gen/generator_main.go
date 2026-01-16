package gen

import (
	"api/internal/api/models"
	"fmt"
	"os"
	"path"
	"strings"
)

// MainProgramGenerator generates the complete main.go file with all node functions
type MainProgramGenerator struct {
	Job        *models.Job
	Execution  *JobExecution
	Generators []Generator
}

// NewMainProgramGenerator creates a new main program generator
func NewMainProgramGenerator(execution *JobExecution) *MainProgramGenerator {
	return &MainProgramGenerator{
		Job:       execution.Job,
		Execution: execution,
	}
}

// Generate creates the main.go file with all interconnected node functions
func (g *MainProgramGenerator) Generate(outputPath string) error {
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(path.Join(outputPath, "main.go"))
	if err != nil {
		return fmt.Errorf("failed to create main.go: %w", err)
	}
	defer file.Close()

	if err := g.writePackageAndImports(file); err != nil {
		return err
	}

	if err := g.writeRowStreamType(file); err != nil {
		return err
	}

	if err := g.writeJobContext(file); err != nil {
		return err
	}

	if err := g.writeConnectionInitFunctions(file); err != nil {
		return err
	}

	if err := g.writeNodeFunctions(file); err != nil {
		return err
	}

	if err := g.writeMainFunction(file); err != nil {
		return err
	}

	return nil
}

func (g *MainProgramGenerator) writePackageAndImports(file *os.File) error {
	importSet := make(map[string]bool)

	// Base imports for pipeline pattern
	importSet[`"context"`] = true
	importSet[`"database/sql"`] = true
	importSet[`"fmt"`] = true
	importSet[`"log"`] = true
	importSet[`"os"`] = true
	importSet[`"os/signal"`] = true
	importSet[`"sync"`] = true
	importSet[`"sync/atomic"`] = true
	importSet[`"time"`] = true

	for _, gen := range g.Generators {
		for _, imp := range gen.GenerateImports() {
			importSet[imp] = true
		}
	}

	if _, err := file.WriteString("package main\n\n"); err != nil {
		return err
	}

	if _, err := file.WriteString("import (\n"); err != nil {
		return err
	}
	for imp := range importSet {
		if _, err := file.WriteString(fmt.Sprintf("\t%s\n", imp)); err != nil {
			return err
		}
	}
	if _, err := file.WriteString(")\n\n"); err != nil {
		return err
	}

	comment := fmt.Sprintf("// Generated pipeline for job: %s (ID: %d)\n", g.Job.Name, g.Job.ID)
	comment += "// Uses channel-based streaming with backpressure support\n\n"

	_, err := file.WriteString(comment)
	return err
}

// writeRowStreamType writes the RowStream type and helper functions
func (g *MainProgramGenerator) writeRowStreamType(file *os.File) error {
	code := `// ============================================================
// PIPELINE STREAMING INFRASTRUCTURE
// ============================================================

// RowStream provides channel-based row streaming with error handling
type RowStream struct {
	rows     chan map[string]interface{}
	err      chan error
	done     chan struct{}
	closed   atomic.Bool
	rowCount atomic.Int64
}

// NewRowStream creates a new stream with the specified buffer size
func NewRowStream(bufferSize int) *RowStream {
	return &RowStream{
		rows: make(chan map[string]interface{}, bufferSize),
		err:  make(chan error, 1),
		done: make(chan struct{}),
	}
}

// Send sends a row to the stream. Returns false if stream is closed.
func (s *RowStream) Send(row map[string]interface{}) bool {
	if s.closed.Load() {
		return false
	}
	select {
	case s.rows <- row:
		s.rowCount.Add(1)
		return true
	case <-s.done:
		return false
	}
}

// SendError sends an error and closes the stream
func (s *RowStream) SendError(err error) {
	if s.closed.Load() {
		return
	}
	select {
	case s.err <- err:
	default:
	}
	s.Close()
}

// Close closes the stream channels
func (s *RowStream) Close() {
	if s.closed.CompareAndSwap(false, true) {
		close(s.rows)
		close(s.done)
	}
}

// Rows returns the row channel for range iteration
func (s *RowStream) Rows() <-chan map[string]interface{} {
	return s.rows
}

// Err returns any error that occurred
func (s *RowStream) Err() error {
	select {
	case err := <-s.err:
		return err
	default:
		return nil
	}
}

// Count returns the number of rows sent
func (s *RowStream) Count() int64 {
	return s.rowCount.Load()
}

// Collect waits for all rows and returns them as a slice (sync point)
func (s *RowStream) Collect() ([]map[string]interface{}, error) {
	results := make([]map[string]interface{}, 0, 1000)
	for row := range s.rows {
		results = append(results, row)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// CollectMultiple collects from multiple streams in parallel (sync point for joins)
func CollectMultiple(streams ...*RowStream) ([][]map[string]interface{}, error) {
	results := make([][]map[string]interface{}, len(streams))
	errors := make([]error, len(streams))
	var wg sync.WaitGroup

	for i, stream := range streams {
		wg.Add(1)
		go func(idx int, s *RowStream) {
			defer wg.Done()
			data, err := s.Collect()
			results[idx] = data
			errors[idx] = err
		}(i, stream)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("stream %d failed: %w", i, err)
		}
	}

	return results, nil
}

// MergeStreams merges multiple input streams into a single output stream
func MergeStreams(bufferSize int, streams ...*RowStream) *RowStream {
	out := NewRowStream(bufferSize)
	var wg sync.WaitGroup

	for _, stream := range streams {
		wg.Add(1)
		go func(s *RowStream) {
			defer wg.Done()
			for row := range s.Rows() {
				if !out.Send(row) {
					return
				}
			}
			if err := s.Err(); err != nil {
				out.SendError(err)
			}
		}(stream)
	}

	go func() {
		wg.Wait()
		out.Close()
	}()

	return out
}

`
	_, err := file.WriteString(code)
	return err
}

func (g *MainProgramGenerator) writeJobContext(file *os.File) error {
	contextCode := `// ============================================================
// JOB CONTEXT & CONNECTION MANAGEMENT
// ============================================================

// JobContext holds database connections and cancellation context
type JobContext struct {
	Ctx         context.Context
	Cancel      context.CancelFunc
	Connections map[string]*sql.DB
	mu          sync.RWMutex
}

// InitJobContext initializes the context with graceful shutdown support
func InitJobContext() (*JobContext, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Handle interrupt signals for graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh
		log.Println("Received interrupt, shutting down...")
		cancel()
	}()

	jobCtx := &JobContext{
		Ctx:         ctx,
		Cancel:      cancel,
		Connections: make(map[string]*sql.DB),
	}

`
	if _, err := file.WriteString(contextCode); err != nil {
		return err
	}

	for _, connConfig := range g.Execution.Context.DBConnections {
		connID := connConfig.GetConnectionID()
		initCall := fmt.Sprintf("\tif err := initConnection_%s(jobCtx); err != nil {\n", connID)
		initCall += fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"failed to init connection %s: %%w\", err)\n", connID)
		initCall += "\t}\n\n"

		if _, err := file.WriteString(initCall); err != nil {
			return err
		}
	}

	closeCode := `	return jobCtx, nil
}

// Close releases all resources
func (ctx *JobContext) Close() {
	ctx.Cancel()
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	for connID, db := range ctx.Connections {
		if err := db.Close(); err != nil {
			log.Printf("Error closing connection %s: %v", connID, err)
		}
	}
}

`
	_, err := file.WriteString(closeCode)
	return err
}

func (g *MainProgramGenerator) writeConnectionInitFunctions(file *os.File) error {
	header := `// ============================================================
// DATABASE CONNECTION INITIALIZATION
// ============================================================

`
	if _, err := file.WriteString(header); err != nil {
		return err
	}

	for _, connConfig := range g.Execution.Context.DBConnections {
		connID := connConfig.GetConnectionID()
		connStr := connConfig.BuildConnectionString()
		driverName := connConfig.GetDriverName()

		initFunc := fmt.Sprintf(`func initConnection_%s(ctx *JobContext) error {
	dbURL := os.Getenv("DATABASE_URL_%s")
	if dbURL == "" {
		dbURL = %q
	}

	db, err := sql.Open(%q, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %%w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping: %%w", err)
	}

	// Optimized connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	ctx.mu.Lock()
	ctx.Connections[%q] = db
	ctx.mu.Unlock()

	log.Printf("Initialized connection: %s")
	return nil
}

`, connID, strings.ToUpper(connID), connStr, driverName, connID, connID)

		if _, err := file.WriteString(initFunc); err != nil {
			return err
		}
	}

	return nil
}

func (g *MainProgramGenerator) writeNodeFunctions(file *os.File) error {
	header := `// ============================================================
// NODE EXECUTION FUNCTIONS
// ============================================================

`
	if _, err := file.WriteString(header); err != nil {
		return err
	}

	for _, gen := range g.Generators {
		signature := gen.GenerateFunctionSignature()
		body := gen.GenerateFunctionBody()

		nodeFunc := fmt.Sprintf("%s {\n%s\n}\n\n", signature, body)

		if _, err := file.WriteString(nodeFunc); err != nil {
			return err
		}
	}

	// Write helper functions from generators
	for _, gen := range g.Generators {
		helpers := gen.GenerateHelperFunctions()
		if helpers != "" {
			if _, err := file.WriteString(helpers + "\n\n"); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *MainProgramGenerator) writeMainFunction(file *os.File) error {
	header := `// ============================================================
// MAIN ORCHESTRATION
// ============================================================

func main() {
	log.SetPrefix("[` + g.Job.Name + `] ")
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	startTime := time.Now()

	ctx, err := InitJobContext()
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}
	defer ctx.Close()

	// Track streams for each node
	nodeStreams := make(map[int]*RowStream)
	// Track collected data for nodes that need sync
	nodeData := make(map[int][]map[string]interface{})
	var streamsMu sync.Mutex

`
	if _, err := file.WriteString(header); err != nil {
		return err
	}

	// Generate code for each step
	for stepIdx, step := range g.Execution.Steps {
		nonStartNodes := make([]models.Node, 0)
		for _, node := range step.nodes {
			if node.Type != models.NodeTypeStart {
				nonStartNodes = append(nonStartNodes, node)
			}
		}

		if len(nonStartNodes) == 0 {
			stepCode := fmt.Sprintf("\t// Step %d: Pipeline start\n", stepIdx)
			stepCode += fmt.Sprintf("\tlog.Println(\"Step %d: Starting pipeline\")\n\n", stepIdx)
			if _, err := file.WriteString(stepCode); err != nil {
				return err
			}
			continue
		}

		stepCode := fmt.Sprintf("\t// Step %d: %d node(s)\n", stepIdx, len(nonStartNodes))
		stepCode += fmt.Sprintf("\tlog.Printf(\"Step %d: Launching %%d node(s)\", %d)\n", stepIdx, len(nonStartNodes))
		stepCode += "\t{\n"

		if len(nonStartNodes) > 1 {
			// Parallel launch
			for _, node := range nonStartNodes {
				stepCode += g.generateStreamingNodeLaunch(node, true)
			}
		} else {
			stepCode += g.generateStreamingNodeLaunch(nonStartNodes[0], false)
		}

		stepCode += "\t}\n\n"

		if _, err := file.WriteString(stepCode); err != nil {
			return err
		}
	}

	footer := `	duration := time.Since(startTime)
	log.Printf("Pipeline completed in %v", duration)

	// Suppress unused variable warnings
	_ = nodeStreams
	_ = nodeData
	_ = streamsMu
}
`
	_, err := file.WriteString(footer)
	return err
}

// generateStreamingNodeLaunch generates code to launch a node with streaming
func (g *MainProgramGenerator) generateStreamingNodeLaunch(node models.Node, parallel bool) string {
	nodeName := sanitizeNodeName(node.Name)
	dataInputs := node.GetDataInputNodes()

	var code string

	switch node.Type {
	case models.NodeTypeDBInput:
		// DB Input: Launch streaming immediately
		code = fmt.Sprintf("\t\t// Node %d: %s (streaming)\n", node.ID, node.Name)
		code += fmt.Sprintf("\t\tnodeStreams[%d] = executeNode_%d_%s(ctx)\n", node.ID, node.ID, nodeName)

	case models.NodeTypeMap:
		// Map: Need to sync/collect inputs first
		code = fmt.Sprintf("\t\t// Node %d: %s (sync point)\n", node.ID, node.Name)

		if len(dataInputs) == 0 {
			code += fmt.Sprintf("\t\tlog.Fatalf(\"Node %d (%s) has no data inputs\")\n", node.ID, node.Name)
		} else if len(dataInputs) == 1 {
			// Single input - collect it
			code += fmt.Sprintf("\t\tdata_%d, err := nodeStreams[%d].Collect()\n", node.ID, dataInputs[0].ID)
			code += "\t\tif err != nil {\n"
			code += fmt.Sprintf("\t\t\tlog.Fatalf(\"Failed to collect input for node %d: %%v\", err)\n", node.ID)
			code += "\t\t}\n"
			code += fmt.Sprintf("\t\tnodeData[%d] = data_%d\n", dataInputs[0].ID, node.ID)
			code += fmt.Sprintf("\t\tlog.Printf(\"Node %d: Collected %%d rows from input\", len(data_%d))\n", node.ID, node.ID)
			code += fmt.Sprintf("\t\tnodeStreams[%d], err = executeNode_%d_%s(ctx, data_%d)\n", node.ID, node.ID, nodeName, node.ID)
			code += "\t\tif err != nil {\n"
			code += fmt.Sprintf("\t\t\tlog.Fatalf(\"Node %d (%s) failed: %%v\", err)\n", node.ID, node.Name)
			code += "\t\t}\n"
		} else {
			// Multiple inputs - collect all (sync point for cross-data)
			streamVars := make([]string, len(dataInputs))
			for i, input := range dataInputs {
				streamVars[i] = fmt.Sprintf("nodeStreams[%d]", input.ID)
			}
			code += fmt.Sprintf("\t\tallData_%d, err := CollectMultiple(%s)\n", node.ID, strings.Join(streamVars, ", "))
			code += "\t\tif err != nil {\n"
			code += fmt.Sprintf("\t\t\tlog.Fatalf(\"Failed to sync inputs for node %d: %%v\", err)\n", node.ID)
			code += "\t\t}\n"
			code += fmt.Sprintf("\t\tlog.Printf(\"Node %d: Synced %%d input streams\", len(allData_%d))\n", node.ID, node.ID)
			code += fmt.Sprintf("\t\tnodeStreams[%d], err = executeNode_%d_%s(ctx, allData_%d...)\n", node.ID, node.ID, nodeName, node.ID)
			code += "\t\tif err != nil {\n"
			code += fmt.Sprintf("\t\t\tlog.Fatalf(\"Node %d (%s) failed: %%v\", err)\n", node.ID, node.Name)
			code += "\t\t}\n"
		}

	case models.NodeTypeDBOutput:
		// DB Output: Stream or collect depending on input count
		code = fmt.Sprintf("\t\t// Node %d: %s (output)\n", node.ID, node.Name)

		if len(dataInputs) == 0 {
			code += fmt.Sprintf("\t\tlog.Fatalf(\"Node %d (%s) has no data inputs\")\n", node.ID, node.Name)
		} else if len(dataInputs) == 1 {
			// Single input - can stream directly
			code += fmt.Sprintf("\t\tif err := executeNode_%d_%s(ctx, nodeStreams[%d]); err != nil {\n", node.ID, nodeName, dataInputs[0].ID)
			code += fmt.Sprintf("\t\t\tlog.Fatalf(\"Node %d (%s) failed: %%v\", err)\n", node.ID, node.Name)
			code += "\t\t}\n"
		} else {
			// Multiple inputs - merge streams
			streamVars := make([]string, len(dataInputs))
			for i, input := range dataInputs {
				streamVars[i] = fmt.Sprintf("nodeStreams[%d]", input.ID)
			}
			code += fmt.Sprintf("\t\tmerged_%d := MergeStreams(1000, %s)\n", node.ID, strings.Join(streamVars, ", "))
			code += fmt.Sprintf("\t\tif err := executeNode_%d_%s(ctx, merged_%d); err != nil {\n", node.ID, nodeName, node.ID)
			code += fmt.Sprintf("\t\t\tlog.Fatalf(\"Node %d (%s) failed: %%v\", err)\n", node.ID, node.Name)
			code += "\t\t}\n"
		}
	}

	return code
}
