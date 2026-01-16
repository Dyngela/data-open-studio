package main

import (
	"log"
	"time"
	"strings"
	_ "github.com/denisenkom/go-mssqldb"
	"database/sql"
	"os"
	"sync"
	_ "github.com/lib/pq"
	"fmt"
)

// Generated code for job: Test job
// Job ID: 1
// Total nodes: 6
// This file contains all node execution functions interconnected via data flow

// ============================================================
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

	if err := initConnection_sqlserver_DC_SQL_02_1895_ICarKKKKK_sa(ctx); err != nil {
		return nil, fmt.Errorf("failed to init connection sqlserver_DC_SQL_02_1895_ICarKKKKK_sa: %w", err)
	}

	if err := initConnection_sqlserver_DC_SQL_01_1433_ICarDEMO_sa(ctx); err != nil {
		return nil, fmt.Errorf("failed to init connection sqlserver_DC_SQL_01_1433_ICarDEMO_sa: %w", err)
	}

	if err := initConnection_postgres_DC_CENTRIC_01_5432_TEST_DB_postgres(ctx); err != nil {
		return nil, fmt.Errorf("failed to init connection postgres_DC_CENTRIC_01_5432_TEST_DB_postgres: %w", err)
	}

	return ctx, nil
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

// ============================================================
// DATABASE CONNECTION INITIALIZATION
// ============================================================

// initConnection_sqlserver_DC_SQL_02_1895_ICarKKKKK_sa initializes the database connection
func initConnection_sqlserver_DC_SQL_02_1895_ICarKKKKK_sa(ctx *JobContext) error {
	// Database connection string from config
	// Override with DATABASE_URL_SQLSERVER_DC_SQL_02_1895_ICARKKKKK_SA environment variable if set
	dbURL := os.Getenv("DATABASE_URL_SQLSERVER_DC_SQL_02_1895_ICARKKKKK_SA")
	if dbURL == "" {
		dbURL = "sqlserver://sa:sa@DC-SQL-02:1895?database=ICarKKKKK&encrypt=disable"
	}

	// Connect to database
	db, err := sql.Open("sqlserver", dbURL)
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

	// Store in context
	ctx.mu.Lock()
	ctx.Connections["sqlserver_DC_SQL_02_1895_ICarKKKKK_sa"] = db
	ctx.mu.Unlock()

	log.Printf("Initialized connection pool: sqlserver_DC_SQL_02_1895_ICarKKKKK_sa")
	return nil
}

// initConnection_sqlserver_DC_SQL_01_1433_ICarDEMO_sa initializes the database connection
func initConnection_sqlserver_DC_SQL_01_1433_ICarDEMO_sa(ctx *JobContext) error {
	// Database connection string from config
	// Override with DATABASE_URL_SQLSERVER_DC_SQL_01_1433_ICARDEMO_SA environment variable if set
	dbURL := os.Getenv("DATABASE_URL_SQLSERVER_DC_SQL_01_1433_ICARDEMO_SA")
	if dbURL == "" {
		dbURL = "sqlserver://sa:sa@DC-SQL-01:1433?database=ICarDEMO&encrypt=disable"
	}

	// Connect to database
	db, err := sql.Open("sqlserver", dbURL)
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

	// Store in context
	ctx.mu.Lock()
	ctx.Connections["sqlserver_DC_SQL_01_1433_ICarDEMO_sa"] = db
	ctx.mu.Unlock()

	log.Printf("Initialized connection pool: sqlserver_DC_SQL_01_1433_ICarDEMO_sa")
	return nil
}

// initConnection_postgres_DC_CENTRIC_01_5432_TEST_DB_postgres initializes the database connection
func initConnection_postgres_DC_CENTRIC_01_5432_TEST_DB_postgres(ctx *JobContext) error {
	// Database connection string from config
	// Override with DATABASE_URL_POSTGRES_DC_CENTRIC_01_5432_TEST_DB_POSTGRES environment variable if set
	dbURL := os.Getenv("DATABASE_URL_POSTGRES_DC_CENTRIC_01_5432_TEST_DB_POSTGRES")
	if dbURL == "" {
		dbURL = "host=DC-CENTRIC-01 port=5432 user=postgres password=postgres dbname=TEST_DB sslmode=disable"
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
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

	// Store in context
	ctx.mu.Lock()
	ctx.Connections["postgres_DC_CENTRIC_01_5432_TEST_DB_postgres"] = db
	ctx.mu.Unlock()

	log.Printf("Initialized connection pool: postgres_DC_CENTRIC_01_5432_TEST_DB_postgres")
	return nil
}

// ============================================================
// NODE EXECUTION FUNCTIONS
// ============================================================

func executeNode_3_Second_DB_Input(ctx *JobContext) ([]map[string]interface{}, error) {
	// Get connection from global context
	ctx.mu.RLock()
	db := ctx.Connections["sqlserver_DC_SQL_02_1895_ICarKKKKK_sa"]
	ctx.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("connection %s not found in context", "sqlserver_DC_SQL_02_1895_ICarKKKKK_sa")
	}

	// Execute query
	query := "/* dbo */ select * from tgclienteProtec"
	log.Printf("Node %d: Executing query: %s", 3, query)

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

	return results, nil
}

func executeNode_2_First_DB_Input(ctx *JobContext) ([]map[string]interface{}, error) {
	// Get connection from global context
	ctx.mu.RLock()
	db := ctx.Connections["sqlserver_DC_SQL_01_1433_ICarDEMO_sa"]
	ctx.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("connection %s not found in context", "sqlserver_DC_SQL_01_1433_ICarDEMO_sa")
	}

	// Execute query
	query := "/* public */ select * from tgcliente"
	log.Printf("Node %d: Executing query: %s", 2, query)

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

	return results, nil
}

func executeNode_4_Map_Node(ctx *JobContext, input []map[string]interface{}) ([]map[string]interface{}, error) {
	log.Printf("Node %d: Processing %d rows through map node", 4, len(input))

	// Transform data
	transformedData := transformData_4(input)
 
	return transformedData, nil
}

func executeNode_5_DB_Output_Node(ctx *JobContext, input []map[string]interface{}) error {
	// Get connection from global context
	ctx.mu.RLock()
	db := ctx.Connections["postgres_DC_CENTRIC_01_5432_TEST_DB_postgres"]
	ctx.mu.RUnlock()

	if db == nil {
		return fmt.Errorf("connection %s not found in context", "postgres_DC_CENTRIC_01_5432_TEST_DB_postgres")
	}

	log.Printf("Node %d: Processing %d rows for table OUTPUT_TABLE", 5, len(input))

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Process data based on mode
	mode := "INSERT"
	switch mode {
	case "insert":
		if err := insertData_5(tx, input, "OUTPUT_TABLE", 500); err != nil {
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

	log.Printf("Node %d: Successfully wrote %d rows to OUTPUT_TABLE", 5, len(input))
	return nil
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

// transformData_4 applies transformations to the input data
func transformData_4(data []map[string]interface{}) []map[string]interface{} {
	// TODO: Implement your transformation logic here
	// For now, this is a passthrough that returns data as-is

	// Example transformations you could implement:
	// - Filter rows based on conditions
	// - Add/remove fields
	// - Rename fields
	// - Calculate derived fields
	// - Aggregate data

	result := make([]map[string]interface{}, 0, len(data))

	for _, row := range data {
		// Example: Add a processed timestamp
		// row["processed_at"] = time.Now().Format(time.RFC3339)

		// Example: Transform field values
		// if val, ok := row["status"]; ok {
		//     row["status_upper"] = strings.ToUpper(val.(string))
		// }

		result = append(result, row)
	}

	return result
}

// insertData_5 is a helper function for batch insertion
func insertData_5(tx *sql.Tx, data []map[string]interface{}, table string, batchSize int) error {
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
			return fmt.Errorf("failed to insert batch: %w", err)
		}
	}

	return nil
}

// ============================================================
// MAIN ORCHESTRATION
// ============================================================

func main() {
	log.SetPrefix("[Job: Test job] ")
	startTime := time.Now()

	// Initialize context
	ctx, err := InitJobContext()
	if err != nil {
		log.Fatalf("Failed to initialize context: %v", err)
	}
	defer ctx.Close()

	// Track results from each node
	nodeResults := make(map[int][]map[string]interface{})

	// Step 0: Start node (no execution)
	log.Println("Step 0: Starting pipeline")

	// Step 1: Executing 2 node(s)
	log.Println("Step 1: Executing 2 node(s)")
	{
		var wg sync.WaitGroup
		var mu sync.Mutex
		errors := make([]error, 0)

		// Node 3: Second DB Input
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := executeNode_3_Second_DB_Input(ctx)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("node 3 (Second DB Input) failed: %w", err))
				mu.Unlock()
				return
			}
			mu.Lock()
			nodeResults[3] = result
			mu.Unlock()
			log.Printf("Node 3 (Second DB Input): Processed %d rows", len(result))
		}()

		// Node 2: First DB Input
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := executeNode_2_First_DB_Input(ctx)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("node 2 (First DB Input) failed: %w", err))
				mu.Unlock()
				return
			}
			mu.Lock()
			nodeResults[2] = result
			mu.Unlock()
			log.Printf("Node 2 (First DB Input): Processed %d rows", len(result))
		}()

		wg.Wait()

		if len(errors) > 0 {
			log.Fatalf("Step 1 failed with %d error(s): %v", len(errors), errors)
		}
	}

	// Step 2: Executing 1 node(s)
	log.Println("Step 2: Executing 1 node(s)")
	{
		// Node 4: Map Node
		{
			var inputData []map[string]interface{}
			inputData = append(inputData, nodeResults[2]...)
			inputData = append(inputData, nodeResults[3]...)
			result, err := executeNode_4_Map_Node(ctx, inputData)
			if err != nil {
				log.Fatalf("Node 4 (Map Node) failed: %v", err)
			}
			nodeResults[4] = result
			log.Printf("Node 4 (Map Node): Processed %d rows", len(result))
		}
	}

	// Step 3: Executing 1 node(s)
	log.Println("Step 3: Executing 1 node(s)")
	{
		// Node 5: DB Output Node
		{
			inputData := nodeResults[4]
			if err := executeNode_5_DB_Output_Node(ctx, inputData); err != nil {
				log.Fatalf("Node 5 (DB Output Node) failed: %v", err)
			}
			log.Printf("Node 5 (DB Output Node): Completed successfully")
		}
	}

	duration := time.Since(startTime)
	log.Printf("Pipeline completed successfully in %v", duration)
}
