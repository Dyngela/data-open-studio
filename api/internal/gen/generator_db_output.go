package gen

import (
	"api/internal/api/models"
	"database/sql"
	"fmt"
	"strings"
)

// DBOutputGenerator handles database output operations
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

// GenerateFunctionSignature returns the function signature for this DB output node
func (g *DBOutputGenerator) GenerateFunctionSignature() string {
	nodeName := sanitizeNodeName(g.nodeName)
	return fmt.Sprintf("func executeNode_%d_%s(ctx *JobContext, input []map[string]interface{}) error",
		g.nodeID, nodeName)
}

// GenerateFunctionBody returns the function body for this DB output node
func (g *DBOutputGenerator) GenerateFunctionBody() string {
	connID := g.config.Connection.GetConnectionID()
	batchSize := g.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	return fmt.Sprintf(`	// Get connection from global context
	ctx.mu.RLock()
	db := ctx.Connections[%q]
	ctx.mu.RUnlock()

	if db == nil {
		return fmt.Errorf("connection %%s not found in context", %q)
	}

	log.Printf("Node %%d: Processing %%d rows for table %s", %d, len(input))

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %%w", err)
	}
	defer tx.Rollback()

	// Process data based on mode
	mode := %q
	switch mode {
	case "insert":
		if err := insertData_%d(tx, input, %q, %d); err != nil {
			return fmt.Errorf("failed to insert data: %%w", err)
		}
	case "update":
		return fmt.Errorf("update mode not yet implemented")
	case "upsert":
		return fmt.Errorf("upsert mode not yet implemented")
	default:
		return fmt.Errorf("unknown mode: %%s", mode)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %%w", err)
	}

	log.Printf("Node %%d: Successfully wrote %%d rows to %s", %d, len(input))
	return nil`, connID, connID, g.config.Table, g.nodeID, g.config.Mode, g.nodeID, g.config.Table, batchSize, g.config.Table, g.nodeID)
}

// GenerateHelperFunctions returns helper functions for this DB output node
func (g *DBOutputGenerator) GenerateHelperFunctions() string {
	return fmt.Sprintf(`// insertData_%d is a helper function for batch insertion
func insertData_%d(tx *sql.Tx, data []map[string]interface{}, table string, batchSize int) error {
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
				rowPlaceholders[k] = fmt.Sprintf("$%%d", placeholderNum)
				values = append(values, row[col])
			}
			placeholders[j] = fmt.Sprintf("(%%s)", strings.Join(rowPlaceholders, ", "))
		}

		query := fmt.Sprintf("INSERT INTO %%s (%%s) VALUES %%s",
			table,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		if _, err := tx.Exec(query, values...); err != nil {
			return fmt.Errorf("failed to insert batch: %%w", err)
		}
	}

	return nil
}`, g.nodeID, g.nodeID)
}

// GenerateImports returns the list of imports needed for this DB output node
func (g *DBOutputGenerator) GenerateImports() []string {
	return []string{
		`"database/sql"`,
		`"fmt"`,
		`"strings"`,
		`_ "` + g.config.Connection.GetImportPath() + `"`,
	}
}
