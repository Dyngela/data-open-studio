package gen

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"api/internal/api/models"
)

// ExecuteDBOutput executes a DB output node, consuming a stream and writing to database
func ExecuteDBOutput(ctx *ExecutionContext, nodeID int, nodeName string, config models.DBOutputConfig, input *RowStream) error {
	db, err := ctx.GetConnection(config.Connection)
	if err != nil {
		return fmt.Errorf("node %d (%s): %w", nodeID, nodeName, err)
	}

	// Set schema for PostgreSQL
	// TODO - move to connection config instead
	if config.Connection.Type == models.DBTypePostgres {
		schema := config.DbSchema
		if schema == "" {
			schema = "public"
		}
		if _, err := db.Exec("SET search_path TO " + schema); err != nil {
			return fmt.Errorf("node %d (%s): failed to set search_path: %w", nodeID, nodeName, err)
		}
	}

	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	log.Printf("Node %d (%s): Starting insert to %s (batch size: %d)", nodeID, nodeName, config.Table, batchSize)

	batch := make([]map[string]interface{}, 0, batchSize)
	var columns []string
	totalRows := 0
	batchNum := 0

	for row := range input.Rows() {
		// Check for cancellation
		if ctx.IsCancelled() {
			log.Printf("Node %d (%s): Cancelled after %d rows", nodeID, nodeName, totalRows)
			return fmt.Errorf("cancelled")
		}

		// Capture columns from first row
		if columns == nil {
			columns = make([]string, 0, len(row))
			for col := range row {
				columns = append(columns, col)
			}
		}

		batch = append(batch, row)

		// Flush batch when full
		if len(batch) >= batchSize {
			if err := flushBatch(db, config, columns, batch); err != nil {
				return fmt.Errorf("node %d (%s): batch %d failed: %w", nodeID, nodeName, batchNum, err)
			}
			totalRows += len(batch)
			batchNum++
			batch = batch[:0]

			if batchNum%10 == 0 {
				log.Printf("Node %d (%s): Wrote %d rows (%d batches)", nodeID, nodeName, totalRows, batchNum)
			}
		}
	}

	// Check for stream errors
	if err := input.Err(); err != nil {
		return fmt.Errorf("node %d (%s): input stream error: %w", nodeID, nodeName, err)
	}

	// Flush remaining rows
	if len(batch) > 0 {
		if err := flushBatch(db, config, columns, batch); err != nil {
			return fmt.Errorf("node %d (%s): final batch failed: %w", nodeID, nodeName, err)
		}
		totalRows += len(batch)
	}

	log.Printf("Node %d (%s): Completed - wrote %d total rows to %s", nodeID, nodeName, totalRows, config.Table)
	return nil
}

func flushBatch(db interface {
	Exec(string, ...any) (sql.Result, error)
}, config models.DBOutputConfig, columns []string, batch []map[string]interface{}) error {
	if len(batch) == 0 {
		return nil
	}

	numCols := len(columns)
	numRows := len(batch)
	values := make([]interface{}, 0, numRows*numCols)
	placeholders := make([]string, numRows)

	// Build values and placeholders based on DB type
	for i, row := range batch {
		rowPH := make([]string, numCols)
		for j, col := range columns {
			idx := i*numCols + j + 1
			rowPH[j] = getPlaceholder(config.Connection.Type, idx)
			values = append(values, row[col])
		}
		placeholders[i] = "(" + strings.Join(rowPH, ",") + ")"
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		config.Table,
		strings.Join(columns, ","),
		strings.Join(placeholders, ","))

	_, err := db.Exec(query, values...)
	return err
}

// getPlaceholder returns the placeholder syntax for the given DB type
func getPlaceholder(dbType models.DBType, index int) string {
	switch dbType {
	case models.DBTypePostgres:
		return fmt.Sprintf("$%d", index)
	case models.DBTypeSQLServer:
		return fmt.Sprintf("@p%d", index)
	default: // MySQL and others
		return "?"
	}
}
