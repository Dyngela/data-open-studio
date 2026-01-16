package main

import (
	"context"
	"database/sql"
	"os/signal"
	_ "github.com/lib/pq"
	"strings"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Generated pipeline for job: Test job (ID: 1)
// Uses channel-based streaming with backpressure support

// ============================================================
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

// ============================================================
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

	if err := initConnection_postgres_localhost_5433_data_open_studio_postgres(jobCtx); err != nil {
		return nil, fmt.Errorf("failed to init connection postgres_localhost_5433_data_open_studio_postgres: %w", err)
	}

	if err := initConnection_postgres_localhost_5433_test_input_postgres(jobCtx); err != nil {
		return nil, fmt.Errorf("failed to init connection postgres_localhost_5433_test_input_postgres: %w", err)
	}

	return jobCtx, nil
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

// ============================================================
// DATABASE CONNECTION INITIALIZATION
// ============================================================

func initConnection_postgres_localhost_5433_data_open_studio_postgres(ctx *JobContext) error {
	dbURL := os.Getenv("DATABASE_URL_POSTGRES_LOCALHOST_5433_DATA_OPEN_STUDIO_POSTGRES")
	if dbURL == "" {
		dbURL = "host=localhost port=5433 user=postgres password=postgres dbname=data-open-studio sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping: %w", err)
	}

	// Optimized connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	ctx.mu.Lock()
	ctx.Connections["postgres_localhost_5433_data_open_studio_postgres"] = db
	ctx.mu.Unlock()

	log.Printf("Initialized connection: postgres_localhost_5433_data_open_studio_postgres")
	return nil
}

func initConnection_postgres_localhost_5433_test_input_postgres(ctx *JobContext) error {
	dbURL := os.Getenv("DATABASE_URL_POSTGRES_LOCALHOST_5433_TEST_INPUT_POSTGRES")
	if dbURL == "" {
		dbURL = "host=localhost port=5433 user=postgres password=postgres dbname=test-input sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping: %w", err)
	}

	// Optimized connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	ctx.mu.Lock()
	ctx.Connections["postgres_localhost_5433_test_input_postgres"] = db
	ctx.mu.Unlock()

	log.Printf("Initialized connection: postgres_localhost_5433_test_input_postgres")
	return nil
}

// ============================================================
// NODE EXECUTION FUNCTIONS
// ============================================================

func executeNode_2_First_DB_Input(ctx *JobContext) *RowStream {
	stream := NewRowStream(1000)

	go func() {
		defer stream.Close()

		// Get connection
		ctx.mu.RLock()
		db := ctx.Connections["postgres_localhost_5433_data_open_studio_postgres"]
		ctx.mu.RUnlock()

		if db == nil {
			stream.SendError(fmt.Errorf("connection %s not found", "postgres_localhost_5433_data_open_studio_postgres"))
			return
		}

		query := "SET search_path TO public; select * from sender"
		log.Printf("Node 2: Executing query: %s", query)

		rows, err := db.Query(query)
		if err != nil {
			stream.SendError(fmt.Errorf("query failed: %w", err))
			return
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			stream.SendError(fmt.Errorf("failed to get columns: %w", err))
			return
		}

		colCount := len(columns)
		// Reuse scan buffer across all rows
		values := make([]interface{}, colCount)
		valuePtrs := make([]interface{}, colCount)
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		rowCount := 0
		for rows.Next() {
			if err := rows.Scan(valuePtrs...); err != nil {
				stream.SendError(fmt.Errorf("scan failed at row %d: %w", rowCount, err))
				return
			}

			// Build row map with pre-allocated capacity
			row := make(map[string]interface{}, colCount)
			for i, col := range columns {
				val := values[i]
				if b, ok := val.([]byte); ok {
					// Copy bytes to avoid reference to reused buffer
					row[col] = string(b)
				} else {
					row[col] = val
				}
				values[i] = nil // Clear for GC
			}

			// Send to channel - blocks if buffer full (backpressure)
			if !stream.Send(row) {
				return // Context cancelled
			}
			rowCount++
		}

		if err := rows.Err(); err != nil {
			stream.SendError(fmt.Errorf("iteration error: %w", err))
			return
		}

		log.Printf("Node 2: Streamed %d rows", rowCount)
	}()

	return stream
}

func executeNode_3_Second_DB_Input(ctx *JobContext) *RowStream {
	stream := NewRowStream(1000)

	go func() {
		defer stream.Close()

		// Get connection
		ctx.mu.RLock()
		db := ctx.Connections["postgres_localhost_5433_data_open_studio_postgres"]
		ctx.mu.RUnlock()

		if db == nil {
			stream.SendError(fmt.Errorf("connection %s not found", "postgres_localhost_5433_data_open_studio_postgres"))
			return
		}

		query := "SET search_path TO public; select * from receiver"
		log.Printf("Node 3: Executing query: %s", query)

		rows, err := db.Query(query)
		if err != nil {
			stream.SendError(fmt.Errorf("query failed: %w", err))
			return
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			stream.SendError(fmt.Errorf("failed to get columns: %w", err))
			return
		}

		colCount := len(columns)
		// Reuse scan buffer across all rows
		values := make([]interface{}, colCount)
		valuePtrs := make([]interface{}, colCount)
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		rowCount := 0
		for rows.Next() {
			if err := rows.Scan(valuePtrs...); err != nil {
				stream.SendError(fmt.Errorf("scan failed at row %d: %w", rowCount, err))
				return
			}

			// Build row map with pre-allocated capacity
			row := make(map[string]interface{}, colCount)
			for i, col := range columns {
				val := values[i]
				if b, ok := val.([]byte); ok {
					// Copy bytes to avoid reference to reused buffer
					row[col] = string(b)
				} else {
					row[col] = val
				}
				values[i] = nil // Clear for GC
			}

			// Send to channel - blocks if buffer full (backpressure)
			if !stream.Send(row) {
				return // Context cancelled
			}
			rowCount++
		}

		if err := rows.Err(); err != nil {
			stream.SendError(fmt.Errorf("iteration error: %w", err))
			return
		}

		log.Printf("Node 3: Streamed %d rows", rowCount)
	}()

	return stream
}

func executeNode_4_Map_Node(ctx *JobContext, inputs ...[]map[string]interface{}) (*RowStream, error) {
	// Calculate total input size
	totalInputRows := 0
	for _, input := range inputs {
		totalInputRows += len(input)
	}
	log.Printf("Node 4: Processing %d input source(s), %d total rows", len(inputs), totalInputRows)

	// Create output stream
	stream := NewRowStream(1000)

	go func() {
		defer stream.Close()

		// Process and transform data
		outputCount := 0

		// Call transformation function
		results := transformData_4(inputs)

		// Stream results
		for _, row := range results {
			if !stream.Send(row) {
				return // Stream closed
			}
			outputCount++
		}

		log.Printf("Node 4: Transformed %d -> %d rows", totalInputRows, outputCount)
	}()

	return stream, nil
}

func executeNode_5_DB_Output_Node(ctx *JobContext, input *RowStream) error {
	// Get connection
	ctx.mu.RLock()
	db := ctx.Connections["postgres_localhost_5433_test_input_postgres"]
	ctx.mu.RUnlock()

	if db == nil {
		return fmt.Errorf("connection %s not found", "postgres_localhost_5433_test_input_postgres")
	}

	log.Printf("Node 5: Starting streaming insert to test (batch size: 500)")

	// Batch buffer
	batch := make([]map[string]interface{}, 0, 500)
	var columns []string
	totalRows := 0
	batchNum := 0

	// Process stream
	for row := range input.Rows() {
		// Capture columns from first row
		if columns == nil {
			columns = make([]string, 0, len(row))
			for col := range row {
				columns = append(columns, col)
			}
		}

		batch = append(batch, row)

		// Flush batch when full
		if len(batch) >= 500 {
			if err := flushBatch_5(db, "test", columns, batch); err != nil {
				return fmt.Errorf("batch %d failed: %w", batchNum, err)
			}
			totalRows += len(batch)
			batchNum++
			batch = batch[:0] // Reuse slice

			if batchNum%10 == 0 {
				log.Printf("Node 5: Wrote %d rows (%d batches)", totalRows, batchNum)
			}
		}
	}

	// Check for stream errors
	if err := input.Err(); err != nil {
		return fmt.Errorf("input stream error: %w", err)
	}

	// Flush remaining rows
	if len(batch) > 0 {
		if err := flushBatch_5(db, "test", columns, batch); err != nil {
			return fmt.Errorf("final batch failed: %w", err)
		}
		totalRows += len(batch)
	}

	log.Printf("Node 5: Completed - wrote %d total rows to test", totalRows)
	return nil
}

// transformData_4 processes multiple input datasets
// Customize this function for your cross-data transformation logic
func transformData_4(inputs [][]map[string]interface{}) []map[string]interface{} {
	// Handle single input - simple passthrough with optional transform
	if len(inputs) == 1 {
		result := make([]map[string]interface{}, 0, len(inputs[0]))
		for _, row := range inputs[0] {
			// Apply transformation here
			result = append(result, row)
		}
		return result
	}

	// Handle multiple inputs - cross-data operations
	// Example: Join, merge, or combine datasets

	// Calculate output capacity
	totalRows := 0
	for _, input := range inputs {
		totalRows += len(input)
	}
	result := make([]map[string]interface{}, 0, totalRows)

	// Default: Concatenate all inputs
	// TODO: Replace with your actual cross-data logic (joins, lookups, etc.)
	for inputIdx, input := range inputs {
		for _, row := range input {
			// Tag source for debugging
			row["_source_input"] = inputIdx
			result = append(result, row)
		}
	}

	// Example cross-data patterns:
	//
	// 1. Inner Join by key:
	// input0Map := make(map[interface{}]map[string]interface{})
	// for _, row := range inputs[0] {
	//     key := row["join_key"]
	//     input0Map[key] = row
	// }
	// for _, row := range inputs[1] {
	//     if matchRow, ok := input0Map[row["join_key"]]; ok {
	//         merged := make(map[string]interface{})
	//         for k, v := range matchRow { merged[k] = v }
	//         for k, v := range row { merged[k] = v }
	//         result = append(result, merged)
	//     }
	// }
	//
	// 2. Lookup enrichment:
	// lookup := make(map[interface{}]map[string]interface{})
	// for _, row := range inputs[1] {
	//     lookup[row["lookup_key"]] = row
	// }
	// for _, row := range inputs[0] {
	//     if extra, ok := lookup[row["lookup_key"]]; ok {
	//         row["extra_field"] = extra["value"]
	//     }
	//     result = append(result, row)
	// }

	return result
}

// flushBatch_5 performs optimized batch insert
func flushBatch_5(db *sql.DB, table string, columns []string, batch []map[string]interface{}) error {
	if len(batch) == 0 {
		return nil
	}

	// Pre-allocate for performance
	numCols := len(columns)
	numRows := len(batch)
	values := make([]interface{}, 0, numRows*numCols)
	placeholders := make([]string, numRows)

	// Build values and placeholders
	for i, row := range batch {
		rowPH := make([]string, numCols)
		for j, col := range columns {
			idx := i*numCols + j + 1
			rowPH[j] = fmt.Sprintf("$%d", idx)
			values = append(values, row[col])
		}
		placeholders[i] = "(" + strings.Join(rowPH, ",") + ")"
	}

	// Build and execute query
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		table,
		strings.Join(columns, ","),
		strings.Join(placeholders, ","))

	_, err := db.Exec(query, values...)
	return err
}

// ============================================================
// MAIN ORCHESTRATION
// ============================================================

func main() {
	log.SetPrefix("[Test job] ")
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

	// Step 0: Pipeline start
	log.Println("Step 0: Starting pipeline")

	// Step 1: 2 node(s)
	log.Printf("Step 1: Launching %d node(s)", 2)
	{
		// Node 2: First DB Input (streaming)
		nodeStreams[2] = executeNode_2_First_DB_Input(ctx)
		// Node 3: Second DB Input (streaming)
		nodeStreams[3] = executeNode_3_Second_DB_Input(ctx)
	}

	// Step 2: 1 node(s)
	log.Printf("Step 2: Launching %d node(s)", 1)
	{
		// Node 4: Map Node (sync point)
		allData_4, err := CollectMultiple(nodeStreams[2], nodeStreams[3])
		if err != nil {
			log.Fatalf("Failed to sync inputs for node 4: %v", err)
		}
		log.Printf("Node 4: Synced %d input streams", len(allData_4))
		nodeStreams[4], err = executeNode_4_Map_Node(ctx, allData_4...)
		if err != nil {
			log.Fatalf("Node 4 (Map Node) failed: %v", err)
		}
	}

	// Step 3: 1 node(s)
	log.Printf("Step 3: Launching %d node(s)", 1)
	{
		// Node 5: DB Output Node (output)
		if err := executeNode_5_DB_Output_Node(ctx, nodeStreams[4]); err != nil {
			log.Fatalf("Node 5 (DB Output Node) failed: %v", err)
		}
	}

	duration := time.Since(startTime)
	log.Printf("Pipeline completed in %v", duration)

	// Suppress unused variable warnings
	_ = nodeStreams
	_ = nodeData
	_ = streamsMu
}
