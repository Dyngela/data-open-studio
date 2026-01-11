package gen

import (
	"api/internal/api/models"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"text/template"
)

// DBOutputGenerator handles database output operations
type DBOutputGenerator struct {
	BaseGenerator
	config models.DBOutputConfig
}

// NewDBOutputGenerator creates a new DB output generator
func NewDBOutputGenerator(nodeID int, config models.DBOutputConfig) *DBOutputGenerator {
	return &DBOutputGenerator{
		BaseGenerator: BaseGenerator{
			nodeID:   nodeID,
			nodeType: models.NodeTypeDBOutput,
		},
		config: config,
	}
}

func (g *DBOutputGenerator) insertData(tx *sql.Tx, data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	batchSize := g.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	// Process in batches
	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}
		batch := data[i:end]

		// Get column names from first row
		var columns []string
		for col := range batch[0] {
			columns = append(columns, col)
		}

		// Build INSERT query
		placeholders := make([]string, len(batch))
		values := make([]interface{}, 0, len(batch)*len(columns))

		for j, row := range batch {
			rowPlaceholders := make([]string, len(columns))
			for k, col := range columns {
				placeholderNum := j*len(columns) + k + 1
				rowPlaceholders[k] = fmt.Sprintf("$%d", placeholderNum)
				values = append(values, row[col])
			}
			placeholders[j] = fmt.Sprintf("(%s)", strings.Join(rowPlaceholders, ", "))
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
			g.config.Table,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		if _, err := tx.Exec(query, values...); err != nil {
			return fmt.Errorf("failed to insert batch: %w", err)
		}
	}

	return nil
}

func (g *DBOutputGenerator) updateData(tx *sql.Tx, data []map[string]interface{}) error {
	// TODO: Implement update logic
	return fmt.Errorf("update mode not yet implemented")
}

func (g *DBOutputGenerator) upsertData(tx *sql.Tx, data []map[string]interface{}) error {
	// TODO: Implement upsert logic (INSERT ON CONFLICT)
	return fmt.Errorf("upsert mode not yet implemented")
}

// GenerateCode generates a standalone Go file for this DB output generator
func (g *DBOutputGenerator) GenerateCode(ctx *ExecutionContext, outputPath string) error {
	tmpl := template.Must(template.New("dboutput").Parse(dbOutputTemplate))

	batchSize := g.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	// Build connection string from config
	connStr := g.config.Connection.BuildConnectionString()
	driverName := g.config.Connection.GetDriverName()
	importPath := g.config.Connection.GetImportPath()
	connID := g.config.Connection.GetConnectionID()

	data := map[string]interface{}{
		"NodeID":           g.nodeID,
		"ConnectionID":     connID,
		"Table":            g.config.Table,
		"Mode":             g.config.Mode,
		"BatchSize":        batchSize,
		"ConnectionString": connStr,
		"DriverName":       driverName,
		"ImportPath":       importPath,
		"Host":             g.config.Connection.Host,
		"Port":             g.config.Connection.Port,
		"Database":         g.config.Connection.Database,
		"Username":         g.config.Connection.Username,
		"SSLMode":          g.config.Connection.SSLMode,
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

const dbOutputTemplate = `package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	_ "{{.ImportPath}}"
)

// Generated code for DB Output Node {{.NodeID}}
// Table: {{.Table}}
// Mode: {{.Mode}}
// Batch Size: {{.BatchSize}}
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

// executeNode{{.NodeID}} executes the DB output operation for node {{.NodeID}}
func executeNode{{.NodeID}}(data []map[string]interface{}) error {
	// Get connection from global context
	db, err := getConnection({{.ConnectionID | printf "%q"}})
	if err != nil {
		return err
	}

	fmt.Printf("Processing %d rows for table {{.Table}}\n", len(data))

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Process data based on mode
	mode := "{{.Mode}}"
	switch mode {
	case "insert":
		if err := insertDataNode{{.NodeID}}(tx, data, "{{.Table}}", {{.BatchSize}}); err != nil {
			return fmt.Errorf("failed to insert data: %w", err)
		}
	case "update":
		return fmt.Errorf("update mode not yet implemented")
	case "upsert":
		return fmt.Errorf("upsert mode not yet implemented")
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("Successfully wrote %d rows to %s\n", len(data), "{{.Table}}")
	return nil
}

func insertDataNode{{.NodeID}}(tx *sql.Tx, data []map[string]interface{}, table string, batchSize int) error {
	if len(data) == 0 {
		return nil
	}

	// Process in batches
	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}
		batch := data[i:end]

		// Get column names from first row
		var columns []string
		for col := range batch[0] {
			columns = append(columns, col)
		}

		// Build INSERT query
		placeholders := make([]string, len(batch))
		values := make([]interface{}, 0, len(batch)*len(columns))

		for j, row := range batch {
			rowPlaceholders := make([]string, len(columns))
			for k, col := range columns {
				placeholderNum := j*len(columns) + k + 1
				rowPlaceholders[k] = fmt.Sprintf("$%d", placeholderNum)
				values = append(values, row[col])
			}
			placeholders[j] = fmt.Sprintf("(%s)", strings.Join(rowPlaceholders, ", "))
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
			table,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		if _, err := tx.Exec(query, values...); err != nil {
			return fmt.Errorf("failed to insert batch starting at row %d: %w", i, err)
		}

		fmt.Printf("Inserted batch %d/%d\n", end, len(data))
	}

	return nil
}

func main() {
	// Initialize global context (all DB connections)
	if err := InitGlobalContext(); err != nil {
		log.Fatalf("Failed to initialize global context: %v", err)
	}
	defer closeAllConnections()

	// Read input data from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Failed to read input: %v", err)
	}

	// Parse JSON input
	var data []map[string]interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		log.Fatalf("Failed to parse JSON input: %v", err)
	}

	// Execute the node
	if err := executeNode{{.NodeID}}(data); err != nil {
		log.Fatalf("Execution failed: %v", err)
	}
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
