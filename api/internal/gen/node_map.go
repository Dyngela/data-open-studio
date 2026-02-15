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

	config, err := node.GetMapConfig()
	if err == nil && config.Join != nil {
		// Join node: order input channels to match left/right from join config.
		// Build a map from absolute port array index to channel name.
		portIndexToChan := make(map[int]string)
		for idx, port := range node.InputPort {
			if port.Type != models.PortTypeInput {
				continue
			}
			sourceNodeID := int(port.ConnectedNodeID)
			if sourceNodeID == 0 {
				continue
			}
			for _, ch := range channels {
				if ch.toNodeID == node.ID && ch.fromNodeID == sourceNodeID {
					portIndexToChan[idx] = fmt.Sprintf("ch_%d", ch.portID)
					break
				}
			}
		}
		// Use config inputs to map input names to channels via port index.
		inputNameToChan := make(map[string]string)
		for _, input := range config.Inputs {
			if ch, ok := portIndexToChan[input.PortID]; ok {
				inputNameToChan[input.Name] = ch
			}
		}
		// Add left first, then right
		if ch, ok := inputNameToChan[config.Join.LeftInput]; ok {
			args = append(args, ch)
		}
		if ch, ok := inputNameToChan[config.Join.RightInput]; ok {
			args = append(args, ch)
		}
	} else {
		// Single input or no join config: add input channels in order
		for _, ch := range channels {
			if ch.toNodeID == node.ID {
				args = append(args, fmt.Sprintf("ch_%d", ch.portID))
			}
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

	// Build variables code (new system) and legacy filter
	variables := g.normalizeVariables(config)
	variablesCode := g.buildVariablesCode(variables, "row")

	// Build global filter expression (legacy fallback, used only when no variables)
	filterExpr := ""
	if len(variables) == 0 && config.GlobalFilter != "" {
		filterExpr = g.substituteExprVars(config.GlobalFilter, "row")
	}

	// Get template engine
	engine, err := NewTemplateEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create template engine: %w", err)
	}

	// Prepare template data
	templateData := MapTransformTemplateData{
		FuncName:      funcName,
		NodeID:        node.ID,
		NodeName:      node.Name,
		InputType:     inputType,
		OutputType:    outputStructName,
		Transforms:    transforms,
		FilterExpr:    filterExpr,
		VariablesCode: variablesCode,
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

	// Build variables code and legacy filter
	variables := g.normalizeVariables(config)

	filterExpr := ""
	variablesCode := ""
	if len(variables) > 0 {
		if join.Type == models.JoinTypeUnion {
			// Union handles variables per-stream, not here
		} else {
			variablesCode = g.buildJoinVariablesCode(variables, join.LeftInput, join.RightInput)
		}
	} else if config.GlobalFilter != "" {
		if join.Type == models.JoinTypeUnion {
			filterExpr = config.GlobalFilter
		} else {
			filterExpr = g.substituteJoinExprVars(config.GlobalFilter, join.LeftInput, join.RightInput)
		}
	}

	switch join.Type {
	case models.JoinTypeInner:
		body.WriteString(g.generateInnerJoinBody(node, config, leftType, rightType, outputStructName, output.Columns, ctx, filterExpr, variablesCode))
	case models.JoinTypeLeft:
		body.WriteString(g.generateLeftJoinBody(node, config, leftType, rightType, outputStructName, output.Columns, ctx, filterExpr, variablesCode))
	case models.JoinTypeRight:
		body.WriteString(g.generateRightJoinBody(node, config, leftType, rightType, outputStructName, output.Columns, ctx, filterExpr, variablesCode))
	case models.JoinTypeCross:
		body.WriteString(g.generateCrossJoinBody(node, config, leftType, rightType, outputStructName, output.Columns, ctx, filterExpr, variablesCode))
	case models.JoinTypeUnion:
		body.WriteString(g.generateUnionBody(node, config, leftType, rightType, outputStructName, output.Columns, ctx, filterExpr, variables))
	default:
		body.WriteString(g.generateLeftJoinBody(node, config, leftType, rightType, outputStructName, output.Columns, ctx, filterExpr, variablesCode))
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
func (g *MapGenerator) generateLeftJoinBody(node *models.Node, config *models.MapConfig, leftType, rightType, outputStructName string, columns []models.MapOutputCol, ctx *GeneratorContext, filterExpr string, variablesCode string) string {
	join := config.Join
	leftKeys := g.getJoinKeys(join.LeftKey, join.LeftKeys)
	rightKeys := g.getJoinKeys(join.RightKey, join.RightKeys)

	transforms := g.buildJoinTransformCode(columns, join.LeftInput, join.RightInput, ctx)

	engine, err := NewTemplateEngine()
	if err != nil {
		return fmt.Sprintf("// Error: %v", err)
	}

	templateData := MapJoinTemplateData{
		FuncName:      ctx.FuncName(node),
		NodeID:        node.ID,
		NodeName:      node.Name,
		LeftType:      leftType,
		RightType:     rightType,
		OutputType:    outputStructName,
		LeftKeys:      leftKeys,
		RightKeys:     rightKeys,
		Transforms:    transforms,
		FilterExpr:    filterExpr,
		VariablesCode: variablesCode,
	}

	body, err := engine.GenerateNodeFunction("node_map_left_join.go.tmpl", templateData)
	if err != nil {
		return fmt.Sprintf("// Error: %v", err)
	}

	return body
}

// generateInnerJoinBody generates the body for an inner join using template
func (g *MapGenerator) generateInnerJoinBody(node *models.Node, config *models.MapConfig, leftType, rightType, outputStructName string, columns []models.MapOutputCol, ctx *GeneratorContext, filterExpr string, variablesCode string) string {
	join := config.Join
	leftKeys := g.getJoinKeys(join.LeftKey, join.LeftKeys)
	rightKeys := g.getJoinKeys(join.RightKey, join.RightKeys)
	transforms := g.buildJoinTransformCode(columns, join.LeftInput, join.RightInput, ctx)

	engine, _ := NewTemplateEngine()
	templateData := MapJoinTemplateData{
		FuncName: ctx.FuncName(node), NodeID: node.ID, NodeName: node.Name,
		LeftType: leftType, RightType: rightType, OutputType: outputStructName,
		LeftKeys: leftKeys, RightKeys: rightKeys, Transforms: transforms,
		FilterExpr: filterExpr, VariablesCode: variablesCode,
	}
	body, _ := engine.GenerateNodeFunction("node_map_inner_join.go.tmpl", templateData)
	return body
}

// generateRightJoinBody generates the body for a right join using template
func (g *MapGenerator) generateRightJoinBody(node *models.Node, config *models.MapConfig, leftType, rightType, outputStructName string, columns []models.MapOutputCol, ctx *GeneratorContext, filterExpr string, variablesCode string) string {
	join := config.Join
	leftKeys := g.getJoinKeys(join.LeftKey, join.LeftKeys)
	rightKeys := g.getJoinKeys(join.RightKey, join.RightKeys)
	transforms := g.buildJoinTransformCode(columns, join.LeftInput, join.RightInput, ctx)

	engine, _ := NewTemplateEngine()
	templateData := MapJoinTemplateData{
		FuncName: ctx.FuncName(node), NodeID: node.ID, NodeName: node.Name,
		LeftType: leftType, RightType: rightType, OutputType: outputStructName,
		LeftKeys: leftKeys, RightKeys: rightKeys, Transforms: transforms,
		FilterExpr: filterExpr, VariablesCode: variablesCode,
	}
	body, _ := engine.GenerateNodeFunction("node_map_right_join.go.tmpl", templateData)
	return body
}

// generateCrossJoinBody generates the body for a cross join using template
func (g *MapGenerator) generateCrossJoinBody(node *models.Node, config *models.MapConfig, leftType, rightType, outputStructName string, columns []models.MapOutputCol, ctx *GeneratorContext, filterExpr string, variablesCode string) string {
	transforms := g.buildJoinTransformCode(columns, config.Join.LeftInput, config.Join.RightInput, ctx)

	engine, _ := NewTemplateEngine()
	templateData := MapJoinTemplateData{
		FuncName: ctx.FuncName(node), NodeID: node.ID, NodeName: node.Name,
		LeftType: leftType, RightType: rightType, OutputType: outputStructName,
		Transforms: transforms, FilterExpr: filterExpr, VariablesCode: variablesCode,
	}
	body, _ := engine.GenerateNodeFunction("node_map_cross_join.go.tmpl", templateData)
	return body
}

// generateUnionBody generates the body for a union using template
func (g *MapGenerator) generateUnionBody(node *models.Node, config *models.MapConfig, leftType, rightType, outputStructName string, columns []models.MapOutputCol, ctx *GeneratorContext, filterExpr string, variables []models.MapVariable) string {
	leftTransforms := g.buildTransformCodeWithPrefix(columns, "left", config.Join.LeftInput)
	rightTransforms := g.buildTransformCodeWithPrefix(columns, "right", config.Join.RightInput)

	engine, _ := NewTemplateEngine()

	// Build per-stream variables code or legacy filter
	leftVariablesCode := ""
	rightVariablesCode := ""
	leftFilterExpr := ""
	rightFilterExpr := ""
	if len(variables) > 0 {
		leftVariablesCode = g.buildUnionVariablesCode(variables, config.Join.LeftInput, "left")
		rightVariablesCode = g.buildUnionVariablesCode(variables, config.Join.RightInput, "right")
	} else if filterExpr != "" {
		leftFilterExpr = g.substituteJoinInputRefs(filterExpr, config.Join.LeftInput, "left")
		rightFilterExpr = g.substituteJoinInputRefs(filterExpr, config.Join.RightInput, "right")
	}

	templateData := MapUnionTemplateData{
		FuncName: ctx.FuncName(node), NodeID: node.ID, NodeName: node.Name,
		LeftType: leftType, RightType: rightType, OutputType: outputStructName,
		LeftTransforms: leftTransforms, RightTransforms: rightTransforms,
		LeftFilterExpr: leftFilterExpr, RightFilterExpr: rightFilterExpr,
		LeftVariablesCode: leftVariablesCode, RightVariablesCode: rightVariablesCode,
	}
	body, _ := engine.GenerateNodeFunction("node_map_union.go.tmpl", templateData)
	return body
}

// normalizeVariables returns Variables if present, else migrates GlobalFilter into a single filter variable
func (g *MapGenerator) normalizeVariables(config *models.MapConfig) []models.MapVariable {
	if len(config.Variables) > 0 {
		return config.Variables
	}
	if config.GlobalFilter != "" {
		return []models.MapVariable{{
			Name:       "filter_1",
			Kind:       models.VarKindFilter,
			Expression: config.GlobalFilter,
			DataType:   "bool",
		}}
	}
	return nil
}

// buildVariablesCode generates Go code for variables (computed + filter) for single input
func (g *MapGenerator) buildVariablesCode(variables []models.MapVariable, rowVar string) string {
	if len(variables) == 0 {
		return ""
	}
	var result strings.Builder
	result.WriteString("\t\t// Variables\n")
	for _, v := range variables {
		expr := g.substituteExprVars(v.Expression, rowVar)
		expr = g.substituteVarRefs(expr)
		switch v.Kind {
		case models.VarKindComputed:
			result.WriteString(fmt.Sprintf("\t\tvar_%s := %s\n", v.Name, expr))
		case models.VarKindFilter:
			result.WriteString(fmt.Sprintf("\t\tif !(%s) {\n\t\t\tcontinue\n\t\t}\n", expr))
		}
	}
	return result.String()
}

// buildJoinVariablesCode generates Go code for variables in join context
func (g *MapGenerator) buildJoinVariablesCode(variables []models.MapVariable, leftInput, rightInput string) string {
	if len(variables) == 0 {
		return ""
	}
	var result strings.Builder
	result.WriteString("\t\t// Variables\n")
	for _, v := range variables {
		expr := g.substituteJoinExprVars(v.Expression, leftInput, rightInput)
		expr = g.substituteVarRefs(expr)
		switch v.Kind {
		case models.VarKindComputed:
			result.WriteString(fmt.Sprintf("\t\tvar_%s := %s\n", v.Name, expr))
		case models.VarKindFilter:
			result.WriteString(fmt.Sprintf("\t\tif !(%s) {\n\t\t\tcontinue\n\t\t}\n", expr))
		}
	}
	return result.String()
}

// buildUnionVariablesCode generates Go code for variables in union context (per-stream)
func (g *MapGenerator) buildUnionVariablesCode(variables []models.MapVariable, inputName, rowVar string) string {
	if len(variables) == 0 {
		return ""
	}
	var result strings.Builder
	result.WriteString("\t\t// Variables\n")
	for _, v := range variables {
		expr := g.substituteJoinInputRefs(v.Expression, inputName, rowVar)
		expr = g.substituteVarRefs(expr)
		switch v.Kind {
		case models.VarKindComputed:
			result.WriteString(fmt.Sprintf("\t\tvar_%s := %s\n", v.Name, expr))
		case models.VarKindFilter:
			result.WriteString(fmt.Sprintf("\t\tif !(%s) {\n\t\t\tcontinue\n\t\t}\n", expr))
		}
	}
	return result.String()
}

// substituteVarRefs replaces $var.name references with var_name Go identifiers
func (g *MapGenerator) substituteVarRefs(expr string) string {
	result := expr
	prefix := "$var."
	for {
		idx := strings.Index(result, prefix)
		if idx == -1 {
			break
		}

		endIdx := idx + len(prefix)
		varName := ""
		for endIdx < len(result) {
			c := result[endIdx]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
				varName += string(c)
				endIdx++
			} else {
				break
			}
		}

		if varName != "" {
			replacement := "var_" + varName
			result = result[:idx] + replacement + result[endIdx:]
		} else {
			break
		}
	}
	return result
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
			var expr string
			if singleRowVar != "" {
				expr = g.substituteExprVars(col.Expression, singleRowVar)
			} else {
				expr = g.substituteJoinExprVars(col.Expression, leftInput, rightInput)
			}
			return g.substituteVarRefs(expr)
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

	// Build a map from absolute port array index to source node ID (data ports only)
	portIndexToSource := make(map[int]int)
	for idx, port := range node.InputPort {
		if port.Type != models.PortTypeInput {
			continue
		}
		portIndexToSource[idx] = int(port.ConnectedNodeID)
	}

	// Use config inputs to map input names to source types via port index
	for _, input := range config.Inputs {
		sourceNodeID := portIndexToSource[input.PortID]
		if sourceNodeID == 0 {
			continue
		}
		if structName, exists := ctx.NodeStructNames[sourceNodeID]; exists {
			result[input.Name] = structName
		}
	}

	return result
}

// getJoinKeys returns all keys as PascalCase field names
func (g *MapGenerator) getJoinKeys(singleKey string, multiKeys []string) []string {
	if len(multiKeys) > 0 {
		keys := make([]string, len(multiKeys))
		for i, k := range multiKeys {
			keys[i] = toPascalCase(k)
		}
		return keys
	}
	return []string{toPascalCase(singleKey)}
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
