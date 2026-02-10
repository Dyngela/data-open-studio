package gen

import (
	"api/internal/api/models"
	"fmt"
	"strings"
)

// MapGenerator generates code for map nodes
type MapGenerator struct{}

func (g *MapGenerator) NodeType() models.NodeType {
	return models.NodeTypeMap
}

// GenerateStructData generates the output row struct data for this map node
func (g *MapGenerator) GenerateStructData(node *models.Node) (*StructData, error) {
	config, err := node.GetMapConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get map config: %w", err)
	}

	if len(config.Outputs) == 0 {
		return nil, fmt.Errorf("map node %d has no outputs defined", node.ID)
	}

	output := config.Outputs[0]
	structName := fmt.Sprintf("Node%dRow", node.ID)
	fields := make([]FieldData, len(output.Columns))
	isJoin := config.Join != nil

	for i, col := range output.Columns {
		fieldName := toPascalCase(col.Name)
		tagName := col.Name
		if isJoin && col.FuncType == models.FuncTypeDirect && col.InputRef != "" {
			fieldName = joinFieldName(col.InputRef)
			tagName = joinTagName(col.InputRef)
		}
		fields[i] = FieldData{
			Name: fieldName,
			Type: mapDataType(col.DataType),
			Tag:  fmt.Sprintf(`json:"%s"`, tagName),
		}
	}

	return &StructData{
		Name:   structName,
		NodeID: node.ID,
		Fields: fields,
	}, nil
}

// GenerateFuncData generates the function data for this map node
func (g *MapGenerator) GenerateFuncData(node *models.Node, ctx *GeneratorContext) (*NodeFunctionData, error) {
	config, err := node.GetMapConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get map config: %w", err)
	}

	// Add required imports
	ctx.AddImport("context")
	ctx.AddImport("fmt")
	ctx.AddImport("test/lib")

	funcName := ctx.FuncName(node)
	outputStructName := ctx.StructName(node)

	// Determine input row types from connected nodes
	inputTypes := g.findInputRowTypes(node, &config, ctx)

	if len(config.Inputs) == 1 {
		// Single input - simple transform
		return g.generateSingleInputFuncData(node, &config, ctx, funcName, outputStructName, inputTypes)
	}

	// Multiple inputs - join logic
	return g.generateJoinFuncData(node, &config, ctx, funcName, outputStructName, inputTypes)
}

// GetLaunchArgs returns the launch arguments for map: [inputChannel(s), outputChannel]
func (g *MapGenerator) GetLaunchArgs(node *models.Node, channels []channelInfo, dbConnections map[string]string) []string {
	args := make([]string, 0)

	// Add input channels (in order)
	for _, ch := range channels {
		if ch.toNodeID == node.ID {
			args = append(args, fmt.Sprintf("ch_%d", ch.portID))
		}
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

// generateSingleInputFuncData generates function data for single-input map (transform)
func (g *MapGenerator) generateSingleInputFuncData(node *models.Node, config *models.MapConfig, ctx *GeneratorContext, funcName, outputStructName string, inputTypes map[string]string) (*NodeFunctionData, error) {
	input := config.Inputs[0]
	inputType := inputTypes[input.Name]
	if inputType == "" {
		inputType = "any"
	}

	output := config.Outputs[0]

	// Build transformation statements
	transforms := g.buildTransformCode(output.Columns, "row", ctx)

	// Get template engine
	engine, err := NewTemplateEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create template engine: %w", err)
	}

	// Prepare template data
	templateData := MapTransformTemplateData{
		FuncName:   funcName,
		NodeID:     node.ID,
		NodeName:   node.Name,
		InputType:  inputType,
		OutputType: outputStructName,
		Transforms: transforms,
	}

	// Generate body using template
	body, err := engine.GenerateNodeFunction("node_map_transform.go.tmpl", templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate transform function: %w", err)
	}

	signature := fmt.Sprintf("func %s(ctx context.Context, in <-chan *%s, outChan chan<- *%s, progress lib.ProgressFunc) error",
		funcName, inputType, outputStructName)

	return &NodeFunctionData{
		Name:      funcName,
		NodeID:    node.ID,
		NodeName:  node.Name,
		Signature: signature,
		Body:      body,
	}, nil
}

// generateJoinFuncData generates function data for multi-input map (join)
func (g *MapGenerator) generateJoinFuncData(node *models.Node, config *models.MapConfig, ctx *GeneratorContext, funcName, outputStructName string, inputTypes map[string]string) (*NodeFunctionData, error) {
	if config.Join == nil {
		return nil, fmt.Errorf("map node %d has multiple inputs but no join config", node.ID)
	}

	join := config.Join
	leftType := inputTypes[join.LeftInput]
	rightType := inputTypes[join.RightInput]

	if leftType == "" {
		leftType = "any"
	}
	if rightType == "" {
		rightType = "any"
	}

	output := config.Outputs[0]

	var body strings.Builder

	switch join.Type {
	case models.JoinTypeInner:
		body.WriteString(g.generateInnerJoinBody(node, config, leftType, rightType, outputStructName, output.Columns, ctx))
	case models.JoinTypeLeft:
		body.WriteString(g.generateLeftJoinBody(node, config, leftType, rightType, outputStructName, output.Columns, ctx))
	case models.JoinTypeRight:
		body.WriteString(g.generateRightJoinBody(node, config, leftType, rightType, outputStructName, output.Columns, ctx))
	case models.JoinTypeCross:
		body.WriteString(g.generateCrossJoinBody(node, config, leftType, rightType, outputStructName, output.Columns, ctx))
	case models.JoinTypeUnion:
		body.WriteString(g.generateUnionBody(node, config, leftType, rightType, outputStructName, output.Columns, ctx))
	default:
		// Default to left join
		body.WriteString(g.generateLeftJoinBody(node, config, leftType, rightType, outputStructName, output.Columns, ctx))
	}

	signature := fmt.Sprintf("func %s(ctx context.Context, leftIn <-chan *%s, rightIn <-chan *%s, outChan chan<- *%s, progress lib.ProgressFunc) error",
		funcName, leftType, rightType, outputStructName)

	return &NodeFunctionData{
		Name:      funcName,
		NodeID:    node.ID,
		NodeName:  node.Name,
		Signature: signature,
		Body:      body.String(),
	}, nil
}

// generateLeftJoinBody generates the body for a left join using template
func (g *MapGenerator) generateLeftJoinBody(node *models.Node, config *models.MapConfig, leftType, rightType, outputStructName string, columns []models.MapOutputCol, ctx *GeneratorContext) string {
	join := config.Join
	leftKey := g.getJoinKey(join.LeftKey, join.LeftKeys)
	rightKey := g.getJoinKey(join.RightKey, join.RightKeys)

	transforms := g.buildJoinTransformCode(columns, join.LeftInput, join.RightInput, ctx)

	engine, err := NewTemplateEngine()
	if err != nil {
		return fmt.Sprintf("// Error: %v", err)
	}

	templateData := MapJoinTemplateData{
		FuncName:   ctx.FuncName(node),
		NodeID:     node.ID,
		NodeName:   node.Name,
		LeftType:   leftType,
		RightType:  rightType,
		OutputType: outputStructName,
		LeftKey:    toPascalCase(leftKey),
		RightKey:   toPascalCase(rightKey),
		Transforms: transforms,
	}

	body, err := engine.GenerateNodeFunction("node_map_left_join.go.tmpl", templateData)
	if err != nil {
		return fmt.Sprintf("// Error: %v", err)
	}

	return body
}

// generateInnerJoinBody generates the body for an inner join using template
func (g *MapGenerator) generateInnerJoinBody(node *models.Node, config *models.MapConfig, leftType, rightType, outputStructName string, columns []models.MapOutputCol, ctx *GeneratorContext) string {
	join := config.Join
	leftKey := g.getJoinKey(join.LeftKey, join.LeftKeys)
	rightKey := g.getJoinKey(join.RightKey, join.RightKeys)
	transforms := g.buildJoinTransformCode(columns, join.LeftInput, join.RightInput, ctx)

	engine, _ := NewTemplateEngine()
	templateData := MapJoinTemplateData{
		FuncName: ctx.FuncName(node), NodeID: node.ID, NodeName: node.Name,
		LeftType: leftType, RightType: rightType, OutputType: outputStructName,
		LeftKey: toPascalCase(leftKey), RightKey: toPascalCase(rightKey), Transforms: transforms,
	}
	body, _ := engine.GenerateNodeFunction("node_map_inner_join.go.tmpl", templateData)
	return body
}

// generateRightJoinBody generates the body for a right join using template
func (g *MapGenerator) generateRightJoinBody(node *models.Node, config *models.MapConfig, leftType, rightType, outputStructName string, columns []models.MapOutputCol, ctx *GeneratorContext) string {
	join := config.Join
	leftKey := g.getJoinKey(join.LeftKey, join.LeftKeys)
	rightKey := g.getJoinKey(join.RightKey, join.RightKeys)
	transforms := g.buildJoinTransformCode(columns, join.LeftInput, join.RightInput, ctx)

	engine, _ := NewTemplateEngine()
	templateData := MapJoinTemplateData{
		FuncName: ctx.FuncName(node), NodeID: node.ID, NodeName: node.Name,
		LeftType: leftType, RightType: rightType, OutputType: outputStructName,
		LeftKey: toPascalCase(leftKey), RightKey: toPascalCase(rightKey), Transforms: transforms,
	}
	body, _ := engine.GenerateNodeFunction("node_map_right_join.go.tmpl", templateData)
	return body
}

// generateCrossJoinBody generates the body for a cross join using template
func (g *MapGenerator) generateCrossJoinBody(node *models.Node, config *models.MapConfig, leftType, rightType, outputStructName string, columns []models.MapOutputCol, ctx *GeneratorContext) string {
	transforms := g.buildJoinTransformCode(columns, config.Join.LeftInput, config.Join.RightInput, ctx)

	engine, _ := NewTemplateEngine()
	templateData := MapJoinTemplateData{
		FuncName: ctx.FuncName(node), NodeID: node.ID, NodeName: node.Name,
		LeftType: leftType, RightType: rightType, OutputType: outputStructName,
		Transforms: transforms,
	}
	body, _ := engine.GenerateNodeFunction("node_map_cross_join.go.tmpl", templateData)
	return body
}

// generateUnionBody generates the body for a union using template
func (g *MapGenerator) generateUnionBody(node *models.Node, config *models.MapConfig, leftType, rightType, outputStructName string, columns []models.MapOutputCol, ctx *GeneratorContext) string {
	leftTransforms := g.buildTransformCodeWithPrefix(columns, "left", config.Join.LeftInput)
	rightTransforms := g.buildTransformCodeWithPrefix(columns, "right", config.Join.RightInput)

	engine, _ := NewTemplateEngine()
	templateData := MapUnionTemplateData{
		FuncName: ctx.FuncName(node), NodeID: node.ID, NodeName: node.Name,
		LeftType: leftType, RightType: rightType, OutputType: outputStructName,
		LeftTransforms: leftTransforms, RightTransforms: rightTransforms,
	}
	body, _ := engine.GenerateNodeFunction("node_map_union.go.tmpl", templateData)
	return body
}

// buildTransformCode builds transformation code for single input
func (g *MapGenerator) buildTransformCode(columns []models.MapOutputCol, rowVar string, ctx *GeneratorContext) string {
	var result strings.Builder
	for _, col := range columns {
		fieldName := toPascalCase(col.Name)
		result.WriteString(fmt.Sprintf("\t\tout.%s = ", fieldName))
		result.WriteString(g.buildColumnExpression(col, rowVar, "", "", ctx))
		result.WriteString("\n")
	}
	return result.String()
}

// buildJoinTransformCode builds transformation code for join (with left/right)
func (g *MapGenerator) buildJoinTransformCode(columns []models.MapOutputCol, leftInput, rightInput string, ctx *GeneratorContext) string {
	var result strings.Builder
	for _, col := range columns {
		fieldName := toPascalCase(col.Name)
		if col.FuncType == models.FuncTypeDirect && col.InputRef != "" {
			fieldName = joinFieldName(col.InputRef)
		}
		result.WriteString(fmt.Sprintf("\t\tout.%s = ", fieldName))
		result.WriteString(g.buildColumnExpression(col, "", leftInput, rightInput, ctx))
		result.WriteString("\n")
	}
	return result.String()
}

// buildTransformCodeWithPrefix builds transformation code with input prefix for union
func (g *MapGenerator) buildTransformCodeWithPrefix(columns []models.MapOutputCol, rowVar, inputName string) string {
	var result strings.Builder
	for _, col := range columns {
		fieldName := toPascalCase(col.Name)
		if col.FuncType == models.FuncTypeDirect && col.InputRef != "" {
			fieldName = joinFieldName(col.InputRef)
		}
		result.WriteString(fmt.Sprintf("\t\tout.%s = ", fieldName))

		// For union, try to match column from the specific input
		if col.FuncType == models.FuncTypeDirect {
			inputRef, fieldRef := parseInputRef(col.InputRef)
			if inputRef == inputName || inputRef == "" {
				result.WriteString(fmt.Sprintf("%s.%s", rowVar, toPascalCase(fieldRef)))
			} else {
				result.WriteString(g.getZeroValue(col.DataType))
			}
		} else {
			result.WriteString(g.getZeroValue(col.DataType))
		}
		result.WriteString("\n")
	}
	return result.String()
}

// buildColumnExpression builds the expression for a single column
func (g *MapGenerator) buildColumnExpression(col models.MapOutputCol, singleRowVar, leftInput, rightInput string, ctx *GeneratorContext) string {
	switch col.FuncType {
	case models.FuncTypeDirect:
		if singleRowVar != "" {
			// Single input: row.Field
			sourceField := extractFieldName(col.InputRef)
			return fmt.Sprintf("%s.%s", singleRowVar, toPascalCase(sourceField))
		}
		// Join: left.Field or right.Field (with nil check for left join)
		inputName, sourceField := parseInputRef(col.InputRef)
		rowVar := "left"
		if inputName == rightInput {
			rowVar = "right"
		}

		// Add nil check for nullable side
		if rowVar == "right" {
			return fmt.Sprintf("func() %s { if right != nil { return right.%s }; return %s }()",
				mapDataType(col.DataType), toPascalCase(sourceField), g.getZeroValue(col.DataType))
		}
		return fmt.Sprintf("%s.%s", rowVar, toPascalCase(sourceField))

	case models.FuncTypeLibrary:
		ctx.AddImport("test/lib")
		args := g.buildFuncArgs(col.Args, singleRowVar, leftInput, rightInput)
		return fmt.Sprintf("lib.%s(%s)", col.LibFunc, strings.Join(args, ", "))

	case models.FuncTypeCustom:
		if col.CustomType == models.CustomExpr {
			// Substitute variables in expression
			if singleRowVar != "" {
				return g.substituteExprVars(col.Expression, singleRowVar)
			}
			return g.substituteJoinExprVars(col.Expression, leftInput, rightInput)
		}
		// Custom function
		return fmt.Sprintf("func() %s { %s }()", mapDataType(col.DataType), col.FuncBody)

	default:
		return g.getZeroValue(col.DataType)
	}
}

// buildFuncArgs builds function arguments list
func (g *MapGenerator) buildFuncArgs(args []models.FuncArg, singleRowVar, leftInput, rightInput string) []string {
	result := make([]string, len(args))
	for i, arg := range args {
		switch arg.Type {
		case "column":
			if singleRowVar != "" {
				field := extractFieldName(arg.Value)
				result[i] = fmt.Sprintf("%s.%s", singleRowVar, toPascalCase(field))
			} else {
				inputName, field := parseInputRef(arg.Value)
				rowVar := "left"
				if inputName == rightInput {
					rowVar = "right"
				}
				result[i] = fmt.Sprintf("%s.%s", rowVar, toPascalCase(field))
			}
		case "literal":
			result[i] = fmt.Sprintf("%q", arg.Value)
		default:
			result[i] = fmt.Sprintf("%q", arg.Value)
		}
	}
	return result
}

// substituteExprVars substitutes variables in expressions for single input
func (g *MapGenerator) substituteExprVars(expr, rowVar string) string {
	// Replace input.field_name with row.FieldName (PascalCase)
	result := expr

	// Find all occurrences of input.field_name and replace with row.FieldName
	parts := strings.Split(result, ".")
	if len(parts) > 1 {
		var newParts []string
		for i, part := range parts {
			if i == 0 {
				// Replace "input" with rowVar
				if part == "input" {
					newParts = append(newParts, rowVar)
				} else {
					newParts = append(newParts, part)
				}
			} else {
				// Convert field name to PascalCase
				// Extract field name (might have spaces or operators after it)
				fieldName := ""
				suffix := ""
				for j, c := range part {
					if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
						fieldName += string(c)
					} else {
						suffix = part[j:]
						break
					}
				}
				if fieldName != "" {
					newParts = append(newParts, toPascalCase(fieldName)+suffix)
				} else {
					newParts = append(newParts, part)
				}
			}
		}
		result = strings.Join(newParts, ".")
	}

	return result
}

// substituteJoinExprVars substitutes variables in expressions for joins
func (g *MapGenerator) substituteJoinExprVars(expr, leftInput, rightInput string) string {
	result := expr

	// Process left input references
	result = g.substituteJoinInputRefs(result, leftInput, "left")
	// Process right input references
	result = g.substituteJoinInputRefs(result, rightInput, "right")

	return result
}

// substituteJoinInputRefs replaces inputName.field_name with rowVar.FieldName
func (g *MapGenerator) substituteJoinInputRefs(expr, inputName, rowVar string) string {
	result := expr
	prefix := inputName + "."

	// Find all occurrences of inputName.field
	for {
		idx := strings.Index(result, prefix)
		if idx == -1 {
			break
		}

		// Find the end of the field name
		endIdx := idx + len(prefix)
		fieldName := ""
		for endIdx < len(result) {
			c := result[endIdx]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
				fieldName += string(c)
				endIdx++
			} else {
				break
			}
		}

		if fieldName != "" {
			pascalField := toPascalCase(fieldName)
			replacement := rowVar + "." + pascalField
			result = result[:idx] + replacement + result[endIdx:]
		} else {
			break
		}
	}

	return result
}

// findInputRowTypes finds the row types for each input flow
func (g *MapGenerator) findInputRowTypes(node *models.Node, config *models.MapConfig, ctx *GeneratorContext) map[string]string {
	result := make(map[string]string)

	// Collect input ports in order — their index maps to input name (A=0, B=1, ...)
	inputIndex := 0
	for _, port := range node.InputPort {
		if port.Type != models.PortTypeInput {
			continue
		}

		sourceNodeID := int(port.ConnectedNodeID)
		if sourceNodeID == 0 {
			inputIndex++
			continue
		}

		inputName := string(rune('A' + inputIndex))
		if structName, exists := ctx.NodeStructNames[sourceNodeID]; exists {
			result[inputName] = structName
		}
		inputIndex++
	}

	return result
}

// getJoinKey returns the first key (handles both single and composite keys)
func (g *MapGenerator) getJoinKey(singleKey string, multiKeys []string) string {
	if len(multiKeys) > 0 {
		return multiKeys[0]
	}
	return singleKey
}

// getZeroValue returns the zero value for a data type (sql.Null* zero = {Valid: false})
func (g *MapGenerator) getZeroValue(dataType string) string {
	goType := mapDataType(dataType)
	switch goType {
	case "sql.NullInt64":
		return "sql.NullInt64{}"
	case "sql.NullFloat64":
		return "sql.NullFloat64{}"
	case "sql.NullBool":
		return "sql.NullBool{}"
	case "sql.NullString":
		return "sql.NullString{}"
	case "sql.NullTime":
		return "sql.NullTime{}"
	default:
		return "nil"
	}
}

// joinFieldName derives a prefixed struct field name from an InputRef like "A.id" → "AId"
func joinFieldName(inputRef string) string {
	inputName, fieldName := parseInputRef(inputRef)
	if inputName == "" {
		return toPascalCase(fieldName)
	}
	return toPascalCase(inputName) + toPascalCase(fieldName)
}

// joinTagName derives a prefixed JSON tag from an InputRef like "A.id" → "a_id"
func joinTagName(inputRef string) string {
	inputName, fieldName := parseInputRef(inputRef)
	if inputName == "" {
		return fieldName
	}
	return strings.ToLower(inputName) + "_" + fieldName
}

// Helper functions

// toPascalCase converts snake_case to PascalCase
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// mapDataType maps config data types to sql.Null* Go types
func mapDataType(dataType string) string {
	switch strings.ToLower(dataType) {
	case "int", "integer", "int4", "int2", "smallint", "serial":
		return "sql.NullInt64"
	case "int64", "bigint", "int8", "bigserial":
		return "sql.NullInt64"
	case "float", "float64", "float8", "double", "double precision",
		"numeric", "decimal", "real", "float4", "money":
		return "sql.NullFloat64"
	case "bool", "boolean":
		return "sql.NullBool"
	case "string", "varchar", "text", "char", "character varying",
		"character", "bpchar", "uuid", "json", "jsonb", "xml",
		"name", "citext":
		return "sql.NullString"
	case "time", "time.time", "timestamp", "datetime",
		"timestamptz", "timestamp with time zone",
		"timestamp without time zone", "date":
		return "sql.NullTime"
	case "bytea", "[]byte":
		return "[]byte"
	default:
		return "sql.NullString"
	}
}

// extractFieldName extracts the field name from "input.field" reference
func extractFieldName(ref string) string {
	parts := strings.Split(ref, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return ref
}

// parseInputRef parses "input.field" into (input, field)
func parseInputRef(ref string) (string, string) {
	parts := strings.Split(ref, ".")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return "", ref
}
