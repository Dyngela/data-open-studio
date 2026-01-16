package gen

import (
	"api/internal/api/models"
	"fmt"
)

// DBOutputGenerator handles database output operations with streaming support
type DBOutputGenerator struct {
	BaseGenerator
	config models.DBOutputConfig
}

// NewDBOutputGenerator creates a new DB output generator
func NewDBOutputGenerator(nodeID int, nodeName string, config models.DBOutputConfig) *DBOutputGenerator {
	return &DBOutputGenerator{
		BaseGenerator: BaseGenerator{
			nodeID:   nodeID,
			nodeName: nodeName,
			nodeType: models.NodeTypeDBOutput,
		},
		config: config,
	}
}

// GenerateFunctionSignature returns the function signature accepting a stream
func (g *DBOutputGenerator) GenerateFunctionSignature() string {
	nodeName := sanitizeNodeName(g.nodeName)
	return fmt.Sprintf("func executeNode_%d_%s(ctx *JobContext, input *RowStream) error",
		g.nodeID, nodeName)
}

// GenerateFunctionBody returns the function body with streaming batch insert
func (g *DBOutputGenerator) GenerateFunctionBody() string {
	connID := g.config.Connection.GetConnectionID()
	batchSize := g.config.BatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	return fmt.Sprintf(`	// Get connection
	ctx.mu.RLock()
	db := ctx.Connections[%q]
	ctx.mu.RUnlock()

	if db == nil {
		return fmt.Errorf("connection %%s not found", %q)
	}

	log.Printf("Node %d: Starting streaming insert to %s (batch size: %d)")

	// Batch buffer
	batch := make([]map[string]interface{}, 0, %d)
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
		if len(batch) >= %d {
			if err := flushBatch_%d(db, %q, columns, batch); err != nil {
				return fmt.Errorf("batch %%d failed: %%w", batchNum, err)
			}
			totalRows += len(batch)
			batchNum++
			batch = batch[:0] // Reuse slice

			if batchNum%%10 == 0 {
				log.Printf("Node %d: Wrote %%d rows (%%d batches)", totalRows, batchNum)
			}
		}
	}

	// Check for stream errors
	if err := input.Err(); err != nil {
		return fmt.Errorf("input stream error: %%w", err)
	}

	// Flush remaining rows
	if len(batch) > 0 {
		if err := flushBatch_%d(db, %q, columns, batch); err != nil {
			return fmt.Errorf("final batch failed: %%w", err)
		}
		totalRows += len(batch)
	}

	log.Printf("Node %d: Completed - wrote %%d total rows to %s", totalRows)
	return nil`,
		connID, connID,
		g.nodeID, g.config.Table, batchSize,
		batchSize,
		batchSize,
		g.nodeID, g.config.Table,
		g.nodeID,
		g.nodeID, g.config.Table,
		g.nodeID, g.config.Table)
}

// GenerateHelperFunctions returns optimized batch insert helper
func (g *DBOutputGenerator) GenerateHelperFunctions() string {
	// Determine placeholder style based on DB type
	placeholderStyle := "$%d" // PostgreSQL style
	if g.config.Connection.Type == models.DBTypeSQLServer {
		placeholderStyle = "@p%d"
	} else if g.config.Connection.Type == models.DBTypeMySQL {
		placeholderStyle = "?"
	}

	return fmt.Sprintf(`// flushBatch_%d performs optimized batch insert
func flushBatch_%d(db *sql.DB, table string, columns []string, batch []map[string]interface{}) error {
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
			rowPH[j] = fmt.Sprintf("%s", idx)
			values = append(values, row[col])
		}
		placeholders[i] = "(" + strings.Join(rowPH, ",") + ")"
	}

	// Build and execute query
	query := fmt.Sprintf("INSERT INTO %%s (%%s) VALUES %%s",
		table,
		strings.Join(columns, ","),
		strings.Join(placeholders, ","))

	_, err := db.Exec(query, values...)
	return err
}`, g.nodeID, g.nodeID, placeholderStyle)
}

func (g *DBOutputGenerator) GenerateImports() []string {
	return []string{
		`"database/sql"`,
		`"fmt"`,
		`"strings"`,
		`_ "` + g.config.Connection.GetImportPath() + `"`,
	}
}
