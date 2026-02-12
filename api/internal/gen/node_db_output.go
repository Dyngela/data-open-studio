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
	case models.DbOutputModeUpdate:
		return g.generateUpdateFuncData(node, &config, ctx, funcName, inputRowType)
	case models.DbOutputModeDelete:
		return g.generateDeleteFuncData(node, &config, ctx, funcName, inputRowType)
	case models.DbOutputModeMerge:
		return g.generateMergeFuncData(node, &config, ctx, funcName, inputRowType)
	case models.DbOutputModeTruncate:
		return g.generateTruncateFuncData(node, &config, ctx, funcName)
	default:
		return nil, fmt.Errorf("db_output node %q: unsupported mode %q", node.Name, config.Mode)
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

// generateUpdateFuncData generates a batch UPDATE function using template
func (g *DBOutputGenerator) generateUpdateFuncData(node *models.Node, config *models.DBOutputConfig, ctx *GeneratorContext, funcName, inputRowType string) (*NodeFunctionData, error) {
	if len(config.DataModels) == 0 {
		return nil, fmt.Errorf("db_output node %q: DataModels is empty - cannot generate UPDATE without columns", node.Name)
	}
	if len(config.KeyColumns) == 0 {
		return nil, fmt.Errorf("db_output node %q: KeyColumns is empty - cannot generate UPDATE without key columns", node.Name)
	}

	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	tableName := config.Table
	if config.DbSchema != "" {
		tableName = fmt.Sprintf("%s.%s", config.DbSchema, config.Table)
	}

	keySet := make(map[string]bool, len(config.KeyColumns))
	for _, k := range config.KeyColumns {
		keySet[k] = true
	}

	var setColumns, setAccessors, keyColumns, keyAccessors []string
	for _, col := range config.DataModels {
		if keySet[col.Name] {
			keyColumns = append(keyColumns, col.Name)
			keyAccessors = append(keyAccessors, col.GoFieldName())
		} else {
			setColumns = append(setColumns, col.Name)
			setAccessors = append(setAccessors, col.GoFieldName())
		}
	}

	engine, err := NewTemplateEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create template engine: %w", err)
	}

	templateData := DBOutputUpdateTemplateData{
		FuncName:     funcName,
		NodeID:       node.ID,
		NodeName:     node.Name,
		InputType:    inputRowType,
		TableName:    tableName,
		NumColumns:   len(config.DataModels),
		BatchSize:    batchSize,
		SetColumns:   setColumns,
		SetAccessors: setAccessors,
		KeyColumns:   keyColumns,
		KeyAccessors: keyAccessors,
	}

	body, err := engine.GenerateNodeFunction("node_db_output_update.go.tmpl", templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate db_output update function: %w", err)
	}

	return &NodeFunctionData{
		Name:     funcName,
		NodeID:   node.ID,
		NodeName: node.Name,
		Body:     body,
	}, nil
}

// generateDeleteFuncData generates a batch DELETE function using template
func (g *DBOutputGenerator) generateDeleteFuncData(node *models.Node, config *models.DBOutputConfig, ctx *GeneratorContext, funcName, inputRowType string) (*NodeFunctionData, error) {
	if len(config.KeyColumns) == 0 {
		return nil, fmt.Errorf("db_output node %q: KeyColumns is empty - cannot generate DELETE without key columns", node.Name)
	}

	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	tableName := config.Table
	if config.DbSchema != "" {
		tableName = fmt.Sprintf("%s.%s", config.DbSchema, config.Table)
	}

	var keyColumns, keyAccessors []string
	keySet := make(map[string]bool, len(config.KeyColumns))
	for _, k := range config.KeyColumns {
		keySet[k] = true
	}
	for _, col := range config.DataModels {
		if keySet[col.Name] {
			keyColumns = append(keyColumns, col.Name)
			keyAccessors = append(keyAccessors, col.GoFieldName())
		}
	}

	engine, err := NewTemplateEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create template engine: %w", err)
	}

	templateData := DBOutputDeleteTemplateData{
		FuncName:     funcName,
		NodeID:       node.ID,
		NodeName:     node.Name,
		InputType:    inputRowType,
		TableName:    tableName,
		BatchSize:    batchSize,
		KeyColumns:   keyColumns,
		KeyAccessors: keyAccessors,
	}

	body, err := engine.GenerateNodeFunction("node_db_output_delete.go.tmpl", templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate db_output delete function: %w", err)
	}

	return &NodeFunctionData{
		Name:     funcName,
		NodeID:   node.ID,
		NodeName: node.Name,
		Body:     body,
	}, nil
}

// generateMergeFuncData generates a PostgreSQL UPSERT function using template
func (g *DBOutputGenerator) generateMergeFuncData(node *models.Node, config *models.DBOutputConfig, ctx *GeneratorContext, funcName, inputRowType string) (*NodeFunctionData, error) {
	if len(config.DataModels) == 0 {
		return nil, fmt.Errorf("db_output node %q: DataModels is empty - cannot generate MERGE without columns", node.Name)
	}
	if len(config.KeyColumns) == 0 {
		return nil, fmt.Errorf("db_output node %q: KeyColumns is empty - cannot generate MERGE without key columns", node.Name)
	}

	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	tableName := config.Table
	if config.DbSchema != "" {
		tableName = fmt.Sprintf("%s.%s", config.DbSchema, config.Table)
	}

	keySet := make(map[string]bool, len(config.KeyColumns))
	for _, k := range config.KeyColumns {
		keySet[k] = true
	}

	columns := make([]string, len(config.DataModels))
	fieldAccessors := make([]string, len(config.DataModels))
	var keyColumns, updateColumns []string

	for i, col := range config.DataModels {
		columns[i] = col.Name
		fieldAccessors[i] = col.GoFieldName()
		if keySet[col.Name] {
			keyColumns = append(keyColumns, col.Name)
		} else {
			updateColumns = append(updateColumns, col.Name)
		}
	}

	engine, err := NewTemplateEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create template engine: %w", err)
	}

	templateData := DBOutputMergeTemplateData{
		FuncName:       funcName,
		NodeID:         node.ID,
		NodeName:       node.Name,
		InputType:      inputRowType,
		TableName:      tableName,
		ColumnNames:    strings.Join(columns, ", "),
		NumColumns:     len(config.DataModels),
		BatchSize:      batchSize,
		FieldAccessors: fieldAccessors,
		KeyColumns:     keyColumns,
		UpdateColumns:  updateColumns,
	}

	body, err := engine.GenerateNodeFunction("node_db_output_merge.go.tmpl", templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate db_output merge function: %w", err)
	}

	return &NodeFunctionData{
		Name:     funcName,
		NodeID:   node.ID,
		NodeName: node.Name,
		Body:     body,
	}, nil
}

// generateTruncateFuncData generates a TRUNCATE function using template
func (g *DBOutputGenerator) generateTruncateFuncData(node *models.Node, config *models.DBOutputConfig, ctx *GeneratorContext, funcName string) (*NodeFunctionData, error) {
	tableName := config.Table
	if config.DbSchema != "" {
		tableName = fmt.Sprintf("%s.%s", config.DbSchema, config.Table)
	}

	engine, err := NewTemplateEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create template engine: %w", err)
	}

	templateData := DBOutputTruncateTemplateData{
		FuncName:  funcName,
		NodeID:    node.ID,
		NodeName:  node.Name,
		TableName: tableName,
	}

	body, err := engine.GenerateNodeFunction("node_db_output_truncate.go.tmpl", templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate db_output truncate function: %w", err)
	}

	return &NodeFunctionData{
		Name:     funcName,
		NodeID:   node.ID,
		NodeName: node.Name,
		Body:     body,
	}, nil
}
