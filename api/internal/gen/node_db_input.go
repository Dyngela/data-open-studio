package gen

import (
	"api/internal/api/models"
	"fmt"
)

// DBInputGenerator generates code for db_input nodes
type DBInputGenerator struct{}

func (g *DBInputGenerator) NodeType() models.NodeType {
	return models.NodeTypeDBInput
}

// GenerateStructData generates the struct data for this db_input node
func (g *DBInputGenerator) GenerateStructData(node *models.Node) (*StructData, error) {
	config, err := node.GetDBInputConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get db_input config: %w", err)
	}

	structName := fmt.Sprintf("Node%dRow", node.ID)
	fieldNames := uniqueFieldNames(config.DataModels)
	fields := make([]FieldData, len(config.DataModels))

	for i, col := range config.DataModels {
		fields[i] = FieldData{
			Name: fieldNames[i],
			Type: col.GoFieldType(),
			Tag:  fmt.Sprintf(`db:"%s"`, col.Name),
		}
	}

	return &StructData{
		Name:   structName,
		NodeID: node.ID,
		Fields: fields,
	}, nil
}

// GetLaunchArgs returns the launch arguments for db_input: [db, outputChannel]
func (g *DBInputGenerator) GetLaunchArgs(node *models.Node, channels []channelInfo, dbConnections map[string]string) []string {
	config, err := node.GetDBInputConfig()
	if err != nil {
		return nil
	}

	args := make([]string, 0, 2)

	// Add DB connection
	connID := config.Connection.GetConnectionID()
	if dbVar, ok := dbConnections[connID]; ok {
		args = append(args, dbVar)
	}

	// Add output channel
	for _, ch := range channels {
		if ch.fromNodeID == node.ID {
			args = append(args, fmt.Sprintf("ch_%d", ch.portID))
			break
		}
	}

	return args
}

// GenerateFuncData generates the function data for this db_input node
func (g *DBInputGenerator) GenerateFuncData(node *models.Node, ctx *GeneratorContext) (*NodeFunctionData, error) {
	config, err := node.GetDBInputConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get db_input config: %w", err)
	}
	config.EnforceSchema()

	// Add required imports
	ctx.AddImport("context")
	ctx.AddImport("database/sql")
	ctx.AddImport("fmt")
	ctx.AddImport("test/lib")
	ctx.AddImportAlias("_", config.Connection.GetImportPath())

	structName := ctx.StructName(node)
	funcName := ctx.FuncName(node)

	// Build scan fields list (must match struct field names)
	scanFields := uniqueFieldNames(config.DataModels)

	// Use template engine
	engine, err := NewTemplateEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create template engine: %w", err)
	}

	// Prepare template data
	templateData := struct {
		FuncName         string
		StructName       string
		NodeID           int
		NodeName         string
		Query            string
		ScanFields       []string
		ProgressInterval int
	}{
		FuncName:         funcName,
		StructName:       structName,
		NodeID:           node.ID,
		NodeName:         node.Name,
		Query:            config.QueryWithSchema,
		ScanFields:       scanFields,
		ProgressInterval: 1000,
	}

	// Generate body using template
	body, err := engine.GenerateNodeFunction("node_db_input.go.tmpl", templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate db_input function: %w", err)
	}

	return &NodeFunctionData{
		Name:      funcName,
		NodeID:    node.ID,
		NodeName:  node.Name,
		Signature: "", // Not used - template generates complete function
		Body:      body,
	}, nil
}
