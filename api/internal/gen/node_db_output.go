package gen

import (
	"api/internal/api/models"
	"fmt"
	"strings"
)

// DBOutputGenerator generates code for db_output nodes
type DBOutputGenerator struct{}

func (g *DBOutputGenerator) NodeType() models.NodeType {
	return models.NodeTypeDBOutput
}

// GenerateStructData returns nil - db_output consumes data, doesn't produce a new type
func (g *DBOutputGenerator) GenerateStructData(node *models.Node) (*StructData, error) {
	return nil, nil
}

// GenerateFuncData generates the function data for this db_output node
func (g *DBOutputGenerator) GenerateFuncData(node *models.Node, ctx *GeneratorContext) (*NodeFunctionData, error) {
	config, err := node.GetDBOutputConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get db_output config: %w", err)
	}

	// Add required imports
	ctx.AddImport("context")
	ctx.AddImport("database/sql")
	ctx.AddImport("fmt")
	ctx.AddImport("strings")
	ctx.AddImport("test/lib")
	ctx.AddImportAlias("_", config.Connection.GetImportPath())

	funcName := ctx.FuncName(node)

	// Find the input source node to get its row type
	inputRowType := g.findInputRowType(node, ctx)
	if inputRowType == "" {
		inputRowType = "any" // fallback
	}

	// build the function based on mode
	switch config.Mode {
	case models.DbOutputModeInsert:
		return g.generateInsertFuncData(node, &config, ctx, funcName, inputRowType)
	default:
		// For now, only implement INSERT - others can be added later
		return &NodeFunctionData{
			Name:      funcName,
			NodeID:    node.ID,
			NodeName:  node.Name,
			Signature: fmt.Sprintf("func %s(ctx context.Context, db *sql.DB, in <-chan *%s, progress lib.ProgressFunc) error", funcName, inputRowType),
			Body:      fmt.Sprintf("	// TODO: Implement %s mode\n	for row := range in {\n		_ = row\n	}\n	return nil", config.Mode),
		}, nil
	}
}

// GetLaunchArgs returns the launch arguments for db_output: [db, inputChannel]
func (g *DBOutputGenerator) GetLaunchArgs(node *models.Node, channels []channelInfo, dbConnections map[string]string) []string {
	config, err := node.GetDBOutputConfig()
	if err != nil {
		return nil
	}

	args := make([]string, 0, 2)

	// Add DB connection
	connID := config.Connection.GetConnectionID()
	if dbVar, ok := dbConnections[connID]; ok {
		args = append(args, dbVar)
	}

	// Add input channel
	for _, ch := range channels {
		if ch.toNodeID == node.ID {
			args = append(args, fmt.Sprintf("ch_%d", ch.portID))
			break
		}
	}

	return args
}

// findInputRowType finds the row type from the connected input node
func (g *DBOutputGenerator) findInputRowType(node *models.Node, ctx *GeneratorContext) string {
	for _, port := range node.InputPort {
		if port.Type == models.PortTypeInput {
			// port.Node is the source node
			sourceNode := &port.Node
			if sourceNode.ID != 0 {
				return ctx.StructName(sourceNode)
			}
		}
	}
	return ""
}

// generateInsertFuncData generates a batch insert function data using template
func (g *DBOutputGenerator) generateInsertFuncData(node *models.Node, config *models.DBOutputConfig, ctx *GeneratorContext, funcName, inputRowType string) (*NodeFunctionData, error) {
	if len(config.DataModels) == 0 {
		return nil, fmt.Errorf("db_output node %q: DataModels is empty - cannot generate INSERT without columns", node.Name)
	}

	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	// build column names and field accessors
	columns := make([]string, len(config.DataModels))
	fieldAccessors := make([]string, len(config.DataModels))
	for i, col := range config.DataModels {
		columns[i] = col.Name
		fieldAccessors[i] = col.GoFieldName()
	}

	// build schema-qualified table name
	tableName := config.Table
	if config.DbSchema != "" {
		tableName = fmt.Sprintf("%s.%s", config.DbSchema, config.Table)
	}

	// Use template engine
	engine, err := NewTemplateEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create template engine: %w", err)
	}

	templateData := DBOutputInsertTemplateData{
		FuncName:       funcName,
		NodeID:         node.ID,
		NodeName:       node.Name,
		InputType:      inputRowType,
		TableName:      tableName,
		ColumnNames:    strings.Join(columns, ", "),
		NumColumns:     len(config.DataModels),
		FieldAccessors: fieldAccessors,
		BatchSize:      batchSize,
	}

	body, err := engine.GenerateNodeFunction("node_db_output_insert.go.tmpl", templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate db_output insert function: %w", err)
	}

	return &NodeFunctionData{
		Name:      funcName,
		NodeID:    node.ID,
		NodeName:  node.Name,
		Signature: "", // Not used - template generates complete function
		Body:      body,
	}, nil
}
