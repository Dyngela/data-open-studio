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
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(path.Join(outputPath, "main.go"))
	if err != nil {
		return fmt.Errorf("failed to create main.go: %w", err)
	}
	defer file.Close()

	// Generate package and imports
	if err := g.writePackageAndImports(file); err != nil {
		return err
	}

	// Generate JobContext struct and initialization
	if err := g.writeJobContext(file); err != nil {
		return err
	}

	// Generate DB connection initialization functions
	if err := g.writeConnectionInitFunctions(file); err != nil {
		return err
	}

	// Generate all node execution functions
	if err := g.writeNodeFunctions(file); err != nil {
		return err
	}

	// Generate main orchestration function
	if err := g.writeMainFunction(file); err != nil {
		return err
	}

	return nil
}

// writePackageAndImports writes the package declaration and imports
func (g *MainProgramGenerator) writePackageAndImports(file *os.File) error {
	// Collect all imports from generators
	importSet := make(map[string]bool)

	// Add base imports always needed
	importSet[`"database/sql"`] = true
	importSet[`"fmt"`] = true
	importSet[`"log"`] = true
	importSet[`"os"`] = true
	importSet[`"sync"`] = true
	importSet[`"time"`] = true
	importSet[`"strings"`] = true

	// Add imports from each generator
	for _, gen := range g.Generators {
		for _, imp := range gen.GenerateImports() {
			importSet[imp] = true
		}
	}

	// Write package declaration
	if _, err := file.WriteString("package main\n\n"); err != nil {
		return err
	}

	// Write imports
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

	// Write header comment
	comment := fmt.Sprintf("// Generated code for job: %s\n", g.Job.Name)
	comment += fmt.Sprintf("// Job ID: %d\n", g.Job.ID)
	comment += fmt.Sprintf("// Total nodes: %d\n", len(g.Job.Nodes))
	comment += "// This file contains all node execution functions interconnected via data flow\n\n"

	if _, err := file.WriteString(comment); err != nil {
		return err
	}

	return nil
}

// writeJobContext writes the JobContext struct and helper methods
func (g *MainProgramGenerator) writeJobContext(file *os.File) error {
	contextCode := `// ============================================================
// GLOBAL CONTEXT & CONNECTION MANAGEMENT
// ============================================================

// JobContext holds all database connections for the job
type JobContext struct {
	Connections map[string]*sql.DB
	mu          sync.RWMutex
}

// InitJobContext initializes the global context with all database connections
func InitJobContext() (*JobContext, error) {
	ctx := &JobContext{
		Connections: make(map[string]*sql.DB),
	}

`
	if _, err := file.WriteString(contextCode); err != nil {
		return err
	}

	// Add initialization for each unique connection
	for _, connConfig := range g.Execution.Context.DBConnections {
		connID := connConfig.GetConnectionID()
		initCall := fmt.Sprintf("\tif err := initConnection_%s(ctx); err != nil {\n", connID)
		initCall += fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"failed to init connection %s: %%w\", err)\n", connID)
		initCall += "\t}\n\n"

		if _, err := file.WriteString(initCall); err != nil {
			return err
		}
	}

	closeCode := `	return ctx, nil
}

// Close closes all database connections
func (ctx *JobContext) Close() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	for connID, db := range ctx.Connections {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing connection %s: %v\n", connID, err)
		}
	}
}

`
	if _, err := file.WriteString(closeCode); err != nil {
		return err
	}

	return nil
}

// writeConnectionInitFunctions writes the database connection initialization functions
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

		initFunc := fmt.Sprintf(`// initConnection_%s initializes the database connection
func initConnection_%s(ctx *JobContext) error {
	// Database connection string from config
	// Override with DATABASE_URL_%s environment variable if set
	dbURL := os.Getenv("DATABASE_URL_%s")
	if dbURL == "" {
		dbURL = %q
	}

	// Connect to database
	db, err := sql.Open(%q, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %%w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %%w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	// Store in context
	ctx.mu.Lock()
	ctx.Connections[%q] = db
	ctx.mu.Unlock()

	log.Printf("Initialized connection pool: %s")
	return nil
}

`, connID, connID, strings.ToUpper(connID), strings.ToUpper(connID), connStr, driverName, connID, connID)

		if _, err := file.WriteString(initFunc); err != nil {
			return err
		}
	}

	return nil
}

// writeNodeFunctions writes all node execution functions
func (g *MainProgramGenerator) writeNodeFunctions(file *os.File) error {
	header := `// ============================================================
// NODE EXECUTION FUNCTIONS
// ============================================================

`
	if _, err := file.WriteString(header); err != nil {
		return err
	}

	// First, write all main node functions
	for _, gen := range g.Generators {
		// Write function signature and body
		signature := gen.GenerateFunctionSignature()
		body := gen.GenerateFunctionBody()

		nodeFunc := fmt.Sprintf("%s {\n%s\n}\n\n", signature, body)

		if _, err := file.WriteString(nodeFunc); err != nil {
			return err
		}
	}

	// Then, write all helper functions
	helperHeader := `// ============================================================
// HELPER FUNCTIONS
// ============================================================

`
	if _, err := file.WriteString(helperHeader); err != nil {
		return err
	}

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

// writeMainFunction writes the main orchestration function
func (g *MainProgramGenerator) writeMainFunction(file *os.File) error {
	header := `// ============================================================
// MAIN ORCHESTRATION
// ============================================================

func main() {
	log.SetPrefix("[Job: ` + g.Job.Name + `] ")
	startTime := time.Now()

	// Initialize context
	ctx, err := InitJobContext()
	if err != nil {
		log.Fatalf("Failed to initialize context: %v", err)
	}
	defer ctx.Close()

	// Track results from each node
	nodeResults := make(map[int][]map[string]interface{})

`
	if _, err := file.WriteString(header); err != nil {
		return err
	}

	// Generate code for each step
	for stepIdx, step := range g.Execution.Steps {
		// Count non-start nodes in this step
		nonStartNodes := make([]models.Node, 0)
		for _, node := range step.nodes {
			if node.Type != models.NodeTypeStart {
				nonStartNodes = append(nonStartNodes, node)
			}
		}

		if len(nonStartNodes) == 0 {
			// Skip steps with only start nodes
			stepCode := fmt.Sprintf("\t// Step %d: Start node (no execution)\n", stepIdx)
			stepCode += fmt.Sprintf("\tlog.Println(\"Step %d: Starting pipeline\")\n\n", stepIdx)
			if _, err := file.WriteString(stepCode); err != nil {
				return err
			}
			continue
		}

		stepCode := fmt.Sprintf("\t// Step %d: Executing %d node(s)\n", stepIdx, len(nonStartNodes))
		stepCode += fmt.Sprintf("\tlog.Println(\"Step %d: Executing %d node(s)\")\n", stepIdx, len(nonStartNodes))
		stepCode += "\t{\n"

		// Check if we need parallel execution (more than 1 node in step)
		if len(nonStartNodes) > 1 {
			// Parallel execution with goroutines
			stepCode += "\t\tvar wg sync.WaitGroup\n"
			stepCode += "\t\tvar mu sync.Mutex\n"
			stepCode += "\t\terrors := make([]error, 0)\n\n"

			for _, node := range nonStartNodes {
				stepCode += g.generateParallelNodeExecution(node)
			}

			stepCode += "\t\twg.Wait()\n\n"
			stepCode += "\t\tif len(errors) > 0 {\n"
			stepCode += fmt.Sprintf("\t\t\tlog.Fatalf(\"Step %d failed with %%d error(s): %%v\", len(errors), errors)\n", stepIdx)
			stepCode += "\t\t}\n"
		} else {
			// Sequential execution (single node)
			stepCode += g.generateSequentialNodeExecution(nonStartNodes[0])
		}

		stepCode += "\t}\n\n"

		if _, err := file.WriteString(stepCode); err != nil {
			return err
		}
	}

	footer := `	duration := time.Since(startTime)
	log.Printf("Pipeline completed successfully in %v", duration)
}
`
	if _, err := file.WriteString(footer); err != nil {
		return err
	}

	return nil
}

// generateParallelNodeExecution generates code for a node executing in parallel
func (g *MainProgramGenerator) generateParallelNodeExecution(node models.Node) string {
	nodeName := sanitizeNodeName(node.Name)
	code := fmt.Sprintf("\t\t// Node %d: %s\n", node.ID, node.Name)
	code += "\t\twg.Add(1)\n"
	code += "\t\tgo func() {\n"
	code += "\t\t\tdefer wg.Done()\n"

	// Get data inputs for this node
	dataInputs := node.GetDataInputNodes()

	switch node.Type {
	case models.NodeTypeDBInput:
		// DB Input: no input parameters
		code += fmt.Sprintf("\t\t\tresult, err := executeNode_%d_%s(ctx)\n", node.ID, nodeName)
		code += "\t\t\tif err != nil {\n"
		code += "\t\t\t\tmu.Lock()\n"
		code += fmt.Sprintf("\t\t\t\terrors = append(errors, fmt.Errorf(\"node %d (%s) failed: %%w\", err))\n", node.ID, node.Name)
		code += "\t\t\t\tmu.Unlock()\n"
		code += "\t\t\t\treturn\n"
		code += "\t\t\t}\n"
		code += "\t\t\tmu.Lock()\n"
		code += fmt.Sprintf("\t\t\tnodeResults[%d] = result\n", node.ID)
		code += "\t\t\tmu.Unlock()\n"
		code += fmt.Sprintf("\t\t\tlog.Printf(\"Node %d (%s): Processed %%d rows\", len(result))\n", node.ID, node.Name)

	case models.NodeTypeMap:
		// Map: requires input data
		if len(dataInputs) == 0 {
			code += fmt.Sprintf("\t\t\t// ERROR: Node %d (%s) has no data inputs\n", node.ID, node.Name)
			code += "\t\t\tmu.Lock()\n"
			code += fmt.Sprintf("\t\t\terrors = append(errors, fmt.Errorf(\"node %d (%s) has no data inputs\"))\n", node.ID, node.Name)
			code += "\t\t\tmu.Unlock()\n"
		} else if len(dataInputs) == 1 {
			code += fmt.Sprintf("\t\t\tinputData := nodeResults[%d]\n", dataInputs[0].ID)
			code += fmt.Sprintf("\t\t\tresult, err := executeNode_%d_%s(ctx, inputData)\n", node.ID, nodeName)
			code += "\t\t\tif err != nil {\n"
			code += "\t\t\t\tmu.Lock()\n"
			code += fmt.Sprintf("\t\t\t\terrors = append(errors, fmt.Errorf(\"node %d (%s) failed: %%w\", err))\n", node.ID, node.Name)
			code += "\t\t\t\tmu.Unlock()\n"
			code += "\t\t\t\treturn\n"
			code += "\t\t\t}\n"
			code += "\t\t\tmu.Lock()\n"
			code += fmt.Sprintf("\t\t\tnodeResults[%d] = result\n", node.ID)
			code += "\t\t\tmu.Unlock()\n"
			code += fmt.Sprintf("\t\t\tlog.Printf(\"Node %d (%s): Processed %%d rows\", len(result))\n", node.ID, node.Name)
		} else {
			// Multiple inputs: concatenate
			code += "\t\t\tvar inputData []map[string]interface{}\n"
			for _, inputNode := range dataInputs {
				code += fmt.Sprintf("\t\t\tinputData = append(inputData, nodeResults[%d]...)\n", inputNode.ID)
			}
			code += fmt.Sprintf("\t\t\tresult, err := executeNode_%d_%s(ctx, inputData)\n", node.ID, nodeName)
			code += "\t\t\tif err != nil {\n"
			code += "\t\t\t\tmu.Lock()\n"
			code += fmt.Sprintf("\t\t\t\terrors = append(errors, fmt.Errorf(\"node %d (%s) failed: %%w\", err))\n", node.ID, node.Name)
			code += "\t\t\t\tmu.Unlock()\n"
			code += "\t\t\t\treturn\n"
			code += "\t\t\t}\n"
			code += "\t\t\tmu.Lock()\n"
			code += fmt.Sprintf("\t\t\tnodeResults[%d] = result\n", node.ID)
			code += "\t\t\tmu.Unlock()\n"
			code += fmt.Sprintf("\t\t\tlog.Printf(\"Node %d (%s): Processed %%d rows\", len(result))\n", node.ID, node.Name)
		}

	case models.NodeTypeDBOutput:
		// DB Output: requires input data
		if len(dataInputs) == 0 {
			code += fmt.Sprintf("\t\t\t// ERROR: Node %d (%s) has no data inputs\n", node.ID, node.Name)
			code += "\t\t\tmu.Lock()\n"
			code += fmt.Sprintf("\t\t\terrors = append(errors, fmt.Errorf(\"node %d (%s) has no data inputs\"))\n", node.ID, node.Name)
			code += "\t\t\tmu.Unlock()\n"
		} else if len(dataInputs) == 1 {
			code += fmt.Sprintf("\t\t\tinputData := nodeResults[%d]\n", dataInputs[0].ID)
			code += fmt.Sprintf("\t\t\tif err := executeNode_%d_%s(ctx, inputData); err != nil {\n", node.ID, nodeName)
			code += "\t\t\t\tmu.Lock()\n"
			code += fmt.Sprintf("\t\t\t\terrors = append(errors, fmt.Errorf(\"node %d (%s) failed: %%w\", err))\n", node.ID, node.Name)
			code += "\t\t\t\tmu.Unlock()\n"
			code += "\t\t\t\treturn\n"
			code += "\t\t\t}\n"
			code += fmt.Sprintf("\t\t\tlog.Printf(\"Node %d (%s): Completed successfully\")\n", node.ID, node.Name)
		} else {
			// Multiple inputs: concatenate
			code += "\t\t\tvar inputData []map[string]interface{}\n"
			for _, inputNode := range dataInputs {
				code += fmt.Sprintf("\t\t\tinputData = append(inputData, nodeResults[%d]...)\n", inputNode.ID)
			}
			code += fmt.Sprintf("\t\t\tif err := executeNode_%d_%s(ctx, inputData); err != nil {\n", node.ID, nodeName)
			code += "\t\t\t\tmu.Lock()\n"
			code += fmt.Sprintf("\t\t\t\terrors = append(errors, fmt.Errorf(\"node %d (%s) failed: %%w\", err))\n", node.ID, node.Name)
			code += "\t\t\t\tmu.Unlock()\n"
			code += "\t\t\t\treturn\n"
			code += "\t\t\t}\n"
			code += fmt.Sprintf("\t\t\tlog.Printf(\"Node %d (%s): Completed successfully\")\n", node.ID, node.Name)
		}
	}

	code += "\t\t}()\n\n"
	return code
}

// generateSequentialNodeExecution generates code for a node executing sequentially
func (g *MainProgramGenerator) generateSequentialNodeExecution(node models.Node) string {
	nodeName := sanitizeNodeName(node.Name)
	code := fmt.Sprintf("\t\t// Node %d: %s\n", node.ID, node.Name)
	code += "\t\t{\n"

	// Get data inputs for this node
	dataInputs := node.GetDataInputNodes()

	switch node.Type {
	case models.NodeTypeDBInput:
		// DB Input: no input parameters
		code += fmt.Sprintf("\t\t\tresult, err := executeNode_%d_%s(ctx)\n", node.ID, nodeName)
		code += "\t\t\tif err != nil {\n"
		code += fmt.Sprintf("\t\t\t\tlog.Fatalf(\"Node %d (%s) failed: %%v\", err)\n", node.ID, node.Name)
		code += "\t\t\t}\n"
		code += fmt.Sprintf("\t\t\tnodeResults[%d] = result\n", node.ID)
		code += fmt.Sprintf("\t\t\tlog.Printf(\"Node %d (%s): Processed %%d rows\", len(result))\n", node.ID, node.Name)

	case models.NodeTypeMap:
		// Map: requires input data
		if len(dataInputs) == 0 {
			code += fmt.Sprintf("\t\t\tlog.Fatalf(\"Node %d (%s) has no data inputs\")\n", node.ID, node.Name)
		} else if len(dataInputs) == 1 {
			code += fmt.Sprintf("\t\t\tinputData := nodeResults[%d]\n", dataInputs[0].ID)
			code += fmt.Sprintf("\t\t\tresult, err := executeNode_%d_%s(ctx, inputData)\n", node.ID, nodeName)
			code += "\t\t\tif err != nil {\n"
			code += fmt.Sprintf("\t\t\t\tlog.Fatalf(\"Node %d (%s) failed: %%v\", err)\n", node.ID, node.Name)
			code += "\t\t\t}\n"
			code += fmt.Sprintf("\t\t\tnodeResults[%d] = result\n", node.ID)
			code += fmt.Sprintf("\t\t\tlog.Printf(\"Node %d (%s): Processed %%d rows\", len(result))\n", node.ID, node.Name)
		} else {
			// Multiple inputs: concatenate
			code += "\t\t\tvar inputData []map[string]interface{}\n"
			for _, inputNode := range dataInputs {
				code += fmt.Sprintf("\t\t\tinputData = append(inputData, nodeResults[%d]...)\n", inputNode.ID)
			}
			code += fmt.Sprintf("\t\t\tresult, err := executeNode_%d_%s(ctx, inputData)\n", node.ID, nodeName)
			code += "\t\t\tif err != nil {\n"
			code += fmt.Sprintf("\t\t\t\tlog.Fatalf(\"Node %d (%s) failed: %%v\", err)\n", node.ID, node.Name)
			code += "\t\t\t}\n"
			code += fmt.Sprintf("\t\t\tnodeResults[%d] = result\n", node.ID)
			code += fmt.Sprintf("\t\t\tlog.Printf(\"Node %d (%s): Processed %%d rows\", len(result))\n", node.ID, node.Name)
		}

	case models.NodeTypeDBOutput:
		// DB Output: requires input data
		if len(dataInputs) == 0 {
			code += fmt.Sprintf("\t\t\tlog.Fatalf(\"Node %d (%s) has no data inputs\")\n", node.ID, node.Name)
		} else if len(dataInputs) == 1 {
			code += fmt.Sprintf("\t\t\tinputData := nodeResults[%d]\n", dataInputs[0].ID)
			code += fmt.Sprintf("\t\t\tif err := executeNode_%d_%s(ctx, inputData); err != nil {\n", node.ID, nodeName)
			code += fmt.Sprintf("\t\t\t\tlog.Fatalf(\"Node %d (%s) failed: %%v\", err)\n", node.ID, node.Name)
			code += "\t\t\t}\n"
			code += fmt.Sprintf("\t\t\tlog.Printf(\"Node %d (%s): Completed successfully\")\n", node.ID, node.Name)
		} else {
			// Multiple inputs: concatenate
			code += "\t\t\tvar inputData []map[string]interface{}\n"
			for _, inputNode := range dataInputs {
				code += fmt.Sprintf("\t\t\tinputData = append(inputData, nodeResults[%d]...)\n", inputNode.ID)
			}
			code += fmt.Sprintf("\t\t\tif err := executeNode_%d_%s(ctx, inputData); err != nil {\n", node.ID, nodeName)
			code += fmt.Sprintf("\t\t\t\tlog.Fatalf(\"Node %d (%s) failed: %%v\", err)\n", node.ID, node.Name)
			code += "\t\t\t}\n"
			code += fmt.Sprintf("\t\t\tlog.Printf(\"Node %d (%s): Completed successfully\")\n", node.ID, node.Name)
		}
	}

	code += "\t\t}\n"
	return code
}
