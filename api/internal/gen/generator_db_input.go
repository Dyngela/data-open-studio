package gen

import (
	"api/internal/api/models"
	"fmt"
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

// GenerateFunctionSignature returns the function signature for this DB input node
func (g *DBInputGenerator) GenerateFunctionSignature() string {
	nodeName := sanitizeNodeName(g.nodeName)
	return fmt.Sprintf("func executeNode_%d_%s(ctx *JobContext) ([]map[string]interface{}, error)",
		g.nodeID, nodeName)
}

// GenerateFunctionBody returns the function body for this DB input node
func (g *DBInputGenerator) GenerateFunctionBody() string {
	schema := g.config.DbSchema
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
