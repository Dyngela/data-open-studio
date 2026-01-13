package gen

import (
	"api/internal/api/models"
	"fmt"
	"os"
	"path"
	"text/template"
)

// DBInputGenerator handles database input operations
type DBInputGenerator struct {
	BaseGenerator
	config models.DBInputConfig
}

// NewDBInputGenerator creates a new DB input generator
func NewDBInputGenerator(nodeID int, nodeName string, config models.DBInputConfig) *DBInputGenerator {
	return &DBInputGenerator{
		BaseGenerator: BaseGenerator{
			nodeID:   nodeID,
			nodeName: nodeName,
			nodeType: models.NodeTypeDBInput,
		},
		config: config,
	}
}

// GenerateCode generates a standalone Go file for this DB input generator
func (g *DBInputGenerator) GenerateCode(ctx *ExecutionContext, outputPath string) error {
	tmpl := template.Must(template.New("dbinput").Parse(dbInputTemplate))

	schema := g.config.Schema
	if schema == "" {
		schema = "public"
	}

	query := g.config.Query
	if query == "" && g.config.Table != "" {
		query = fmt.Sprintf("SELECT * FROM %s.%s", schema, g.config.Table)
	}

	// Build connection string from config
	connStr := g.config.Connection.BuildConnectionString()
	driverName := g.config.Connection.GetDriverName()
	importPath := g.config.Connection.GetImportPath()
	connID := g.config.Connection.GetConnectionID()

	data := map[string]interface{}{
		"NodeID":           g.nodeID,
		"ConnectionID":     connID,
		"Query":            query,
		"Schema":           schema,
		"Table":            g.config.Table,
		"ConnectionString": connStr,
		"DriverName":       driverName,
		"ImportPath":       importPath,
		"Host":             g.config.Connection.Host,
		"Port":             g.config.Connection.Port,
		"Database":         g.config.Connection.Database,
		"Username":         g.config.Connection.Username,
		"SSLMode":          g.config.Connection.SSLMode,
	}

	file, err := os.Create(path.Join(outputPath, fmt.Sprintf("node_%s.go", g.nodeName)))
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// GenerateFunctionSignature returns the function signature for this DB input node
func (g *DBInputGenerator) GenerateFunctionSignature() string {
	nodeName := sanitizeNodeName(g.nodeName)
	return fmt.Sprintf("func executeNode_%d_%s(ctx *JobContext) ([]map[string]interface{}, error)",
		g.nodeID, nodeName)
}

// GenerateFunctionBody returns the function body for this DB input node
func (g *DBInputGenerator) GenerateFunctionBody() string {
	schema := g.config.Schema
	if schema == "" {
		schema = "public"
	}

	query := g.config.Query
	if query == "" && g.config.Table != "" {
		query = fmt.Sprintf("SELECT * FROM %s.%s", schema, g.config.Table)
	}

	connID := g.config.Connection.GetConnectionID()

	return fmt.Sprintf(`	// Get connection from global context
	ctx.mu.RLock()
	db := ctx.Connections[%q]
	ctx.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("connection %%s not found in context", %q)
	}

	// Execute query
	query := %q
	log.Printf("Node %%d: Executing query: %%s", %d, query)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %%w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %%w", err)
	}

	// Read all rows
	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %%w", err)
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			// Convert []byte to string for better JSON output
			if b, ok := values[i].([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = values[i]
			}
		}
		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %%w", err)
	}

	return results, nil`, connID, connID, query, g.nodeID)
}

func (g *DBInputGenerator) GenerateImports() []string {
	return []string{
		`"database/sql"`,
		`"fmt"`,
		`_ "` + g.config.Connection.GetImportPath() + `"`,
	}
}

// Legacy template for standalone file generation (kept for backward compatibility)
const dbInputTemplate = `package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	_ "{{.ImportPath}}"
)

// Generated code for DB Input Node {{.NodeID}}
// Schema: {{.Schema}}
// Table: {{.Table}}
// Database: {{.Host}}:{{.Port}}/{{.Database}}
// Connection ID: {{.ConnectionID}} (shared across nodes with same config)

// GlobalContext holds all database connections for the job
var GlobalContext *JobContext

type JobContext struct {
	Connections map[string]*sql.DB
	mu          sync.RWMutex
}

// InitGlobalContext initializes the global context with all database connections
func InitGlobalContext() error {
	GlobalContext = &JobContext{
		Connections: make(map[string]*sql.DB),
	}

	// Initialize connection for {{.ConnectionID}}
	if err := initConnection_{{.ConnectionID}}(); err != nil {
		return fmt.Errorf("failed to init connection {{.ConnectionID}}: %w", err)
	}

	return nil
}

// initConnection_{{.ConnectionID}} initializes the database connection
func initConnection_{{.ConnectionID}}() error {
	// Database connection string from config
	// Override with DATABASE_URL environment variable if set
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = {{.ConnectionString | printf "%q"}}
	}

	// Connect to database
	db, err := sql.Open({{.DriverName | printf "%q"}}, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	// Store in global context
	GlobalContext.mu.Lock()
	GlobalContext.Connections[{{.ConnectionID | printf "%q"}}] = db
	GlobalContext.mu.Unlock()

	fmt.Printf("Initialized connection pool: {{.ConnectionID}}\n")
	return nil
}

// getConnection retrieves a connection from the global context
func getConnection(connID string) (*sql.DB, error) {
	GlobalContext.mu.RLock()
	defer GlobalContext.mu.RUnlock()

	db, exists := GlobalContext.Connections[connID]
	if !exists {
		return nil, fmt.Errorf("connection %s not found in global context", connID)
	}
	return db, nil
}

// executeNode{{.NodeID}} executes the DB input operation for node {{.NodeID}}
func executeNode{{.NodeID}}() ([]map[string]interface{}, error) {
	// Get connection from global context
	db, err := getConnection({{.ConnectionID | printf "%q"}})
	if err != nil {
		return nil, err
	}

	// Execute query
	query := {{.Query | printf "%q"}}
	fmt.Printf("Executing query: %s\n", query)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Read all rows
	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			// Convert []byte to string for better JSON output
			if b, ok := values[i].([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = values[i]
			}
		}
		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	fmt.Printf("Retrieved %d rows\n", len(results))
	return results, nil
}

func main() {
	// Initialize global context (all DB connections)
	if err := InitGlobalContext(); err != nil {
		log.Fatalf("Failed to initialize global context: %v", err)
	}
	defer closeAllConnections()

	// Execute the node
	results, err := executeNode{{.NodeID}}()
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	// Output results as JSON
	output, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal results: %v", err)
	}

	fmt.Println(string(output))
}

// closeAllConnections closes all database connections in global context
func closeAllConnections() {
	if GlobalContext == nil {
		return
	}

	GlobalContext.mu.Lock()
	defer GlobalContext.mu.Unlock()

	for connID, db := range GlobalContext.Connections {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing connection %s: %v\n", connID, err)
		}
	}
}
`
