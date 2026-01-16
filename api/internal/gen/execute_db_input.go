package gen

import (
	"fmt"
	"log"

	"api/internal/api/models"

	_ "github.com/lib/pq"
)

// ExecuteDBInput executes a DB input node and returns a stream of rows
func ExecuteDBInput(ctx *ExecutionContext, nodeID int, nodeName string, config models.DBInputConfig) (*RowStream, error) {
	db, err := ctx.GetConnection(config.Connection)
	if err != nil {
		return nil, fmt.Errorf("node %d (%s): %w", nodeID, nodeName, err)
	}

	// Set schema for PostgreSQL
	if config.Connection.Type == models.DBTypePostgres {
		schema := config.DbSchema
		if schema == "" {
			schema = "public"
		}
		if _, err := db.Exec("SET search_path TO " + schema); err != nil {
			return nil, fmt.Errorf("node %d (%s): failed to set search_path: %w", nodeID, nodeName, err)
		}
	}

	bufferSize := config.BatchSize
	if bufferSize <= 0 {
		bufferSize = 1000
	}

	stream := NewRowStream(bufferSize)

	go func() {
		defer stream.Close()

		query := config.Query
		log.Printf("Node %d (%s): Executing query: %s", nodeID, nodeName, query)

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
		values := make([]interface{}, colCount)
		valuePtrs := make([]interface{}, colCount)
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		rowCount := 0
		for rows.Next() {
			// Check for cancellation
			if ctx.IsCancelled() {
				log.Printf("Node %d (%s): Cancelled after %d rows", nodeID, nodeName, rowCount)
				return
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				stream.SendError(fmt.Errorf("scan failed at row %d: %w", rowCount, err))
				return
			}

			row := make(map[string]interface{}, colCount)
			for i, col := range columns {
				val := values[i]
				if b, ok := val.([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = val
				}
				values[i] = nil
			}

			if !stream.Send(row) {
				return
			}
			rowCount++
		}

		if err := rows.Err(); err != nil {
			stream.SendError(fmt.Errorf("iteration error: %w", err))
			return
		}

		log.Printf("Node %d (%s): Streamed %d rows", nodeID, nodeName, rowCount)
	}()

	return stream, nil
}
