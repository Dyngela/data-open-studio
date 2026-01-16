package gen

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"api/internal/api/models"
	// Note: Database drivers must be imported where the executor is used
	// e.g., _ "github.com/lib/pq" for PostgreSQL
)

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

// ExecutionContext holds runtime state for pipeline execution
type ExecutionContext struct {
	Ctx         context.Context
	Cancel      context.CancelFunc
	Connections map[string]*sql.DB
	Streams     map[int]*RowStream      // Node ID -> output stream
	Data        map[int][]map[string]interface{} // Node ID -> collected data
	mu          sync.RWMutex
}

// NewExecutionContext creates a new execution context with graceful shutdown
func NewExecutionContext() *ExecutionContext {
	ctx, cancel := context.WithCancel(context.Background())

	// Handle interrupt signals
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh
		log.Println("Received interrupt, shutting down...")
		cancel()
	}()

	return &ExecutionContext{
		Ctx:         ctx,
		Cancel:      cancel,
		Connections: make(map[string]*sql.DB),
		Streams:     make(map[int]*RowStream),
		Data:        make(map[int][]map[string]interface{}),
	}
}

// InitConnection initializes a database connection if not already present
func (ec *ExecutionContext) InitConnection(config models.DBConnectionConfig) error {
	connID := config.GetConnectionID()

	ec.mu.Lock()
	defer ec.mu.Unlock()

	// Already initialized
	if _, exists := ec.Connections[connID]; exists {
		return nil
	}

	connStr := config.BuildConnectionString()
	driverName := config.GetDriverName()

	db, err := sql.Open(driverName, connStr)
	if err != nil {
		return fmt.Errorf("failed to open connection %s: %w", connID, err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping %s: %w", connID, err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	ec.Connections[connID] = db
	log.Printf("Initialized connection: %s", connID)
	return nil
}

// GetConnection returns a database connection by config
func (ec *ExecutionContext) GetConnection(config models.DBConnectionConfig) (*sql.DB, error) {
	connID := config.GetConnectionID()

	ec.mu.RLock()
	db, exists := ec.Connections[connID]
	ec.mu.RUnlock()

	if !exists {
		// Try to initialize
		if err := ec.InitConnection(config); err != nil {
			return nil, err
		}
		ec.mu.RLock()
		db = ec.Connections[connID]
		ec.mu.RUnlock()
	}

	return db, nil
}

// SetStream stores a node's output stream
func (ec *ExecutionContext) SetStream(nodeID int, stream *RowStream) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.Streams[nodeID] = stream
}

// GetStream retrieves a node's output stream
func (ec *ExecutionContext) GetStream(nodeID int) (*RowStream, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	stream, exists := ec.Streams[nodeID]
	return stream, exists
}

// SetData stores collected data for a node
func (ec *ExecutionContext) SetData(nodeID int, data []map[string]interface{}) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.Data[nodeID] = data
}

// GetData retrieves collected data for a node
func (ec *ExecutionContext) GetData(nodeID int) ([]map[string]interface{}, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	data, exists := ec.Data[nodeID]
	return data, exists
}

// Close releases all resources
func (ec *ExecutionContext) Close() {
	ec.Cancel()
	ec.mu.Lock()
	defer ec.mu.Unlock()

	for connID, db := range ec.Connections {
		if err := db.Close(); err != nil {
			log.Printf("Error closing connection %s: %v", connID, err)
		}
	}
}

// IsCancelled checks if the context has been cancelled
func (ec *ExecutionContext) IsCancelled() bool {
	select {
	case <-ec.Ctx.Done():
		return true
	default:
		return false
	}
}