package gen

import (
	"api/internal/api/models"
	"api/internal/gen/ir"
	"fmt"
	"strings"
)

// MapGenerator generates code for map nodes
type MapGenerator struct{}

func (g *MapGenerator) NodeType() models.NodeType {
	return models.NodeTypeMap
}

// GenerateStruct generates the output row struct(s) for this map node
// Each output flow gets its own struct type
func (g *MapGenerator) GenerateStruct(node *models.Node) (*ir.StructDecl, error) {
	config, err := node.GetMapConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get map config: %w", err)
	}

	// For now, generate struct for the first (main) output
	// Multi-output support can be added later
	if len(config.Outputs) == 0 {
		return nil, fmt.Errorf("map node %d has no outputs defined", node.ID)
	}

	output := config.Outputs[0]
	structName := fmt.Sprintf("Node%dRow", node.ID)
	builder := ir.NewStruct(structName)

	for _, col := range output.Columns {
		fieldName := toPascalCase(col.Name)
		fieldType := mapDataType(col.DataType)
		tag := fmt.Sprintf(`json:"%s"`, col.Name)
		builder.FieldWithTag(fieldName, fieldType, tag)
	}

	return builder.Build(), nil
}

// GenerateFunc generates the execution function for this map node
func (g *MapGenerator) GenerateFunc(node *models.Node, ctx *GeneratorContext) (*ir.FuncDecl, error) {
	config, err := node.GetMapConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get map config: %w", err)
	}

	// Add required imports
	ctx.AddImport("context")

	funcName := ctx.FuncName(node)
	outputStructName := ctx.StructName(node)

	// Determine input row types from connected nodes
	inputTypes := g.findInputRowTypes(node, &config, ctx)

	if len(config.Inputs) == 1 {
		// Single input - simple transform
		return g.generateSingleInputFunc(node, &config, ctx, funcName, outputStructName, inputTypes)
	}

	// Multiple inputs - need join logic
	return g.generateJoinFunc(node, &config, ctx, funcName, outputStructName, inputTypes)
}

// findInputRowTypes finds the row types for each input flow
func (g *MapGenerator) findInputRowTypes(node *models.Node, config *models.MapConfig, ctx *GeneratorContext) map[string]string {
	result := make(map[string]string)

	for _, input := range config.Inputs {
		// Find the port with matching ID
		for _, port := range node.InputPort {
			if port.Type == models.PortTypeInput && int(port.ID) == input.PortID {
				// port.Node is the source node
				sourceNode := &port.Node
				if sourceNode.ID != 0 {
					result[input.Name] = ctx.StructName(sourceNode)
				}
				break
			}
		}
	}

	return result
}

// generateSingleInputFunc generates a function for single-input map (simple transform)
func (g *MapGenerator) generateSingleInputFunc(node *models.Node, config *models.MapConfig, ctx *GeneratorContext, funcName, outputStructName string, inputTypes map[string]string) (*ir.FuncDecl, error) {
	ctx.AddImport("test/lib")

	input := config.Inputs[0]
	inputType := inputTypes[input.Name]
	if inputType == "" {
		inputType = "any"
	}

	output := config.Outputs[0]
	progressInterval := 1000

	// Build the transformation statements for each output column
	transformStmts := g.buildTransformStatements(input.Name, output.Columns, config, ctx)

	body := []ir.Stmt{
		// var rowCount int64
		ir.Var("rowCount", "int64"),

		// Report start
		ir.If(ir.Neq(ir.Id("progress"), ir.Nil()),
			ir.ExprStatement(ir.Call("progress", ir.Call("lib.NewProgress",
				ir.Lit(node.ID),
				ir.Lit(node.Name),
				ir.Id("lib.StatusRunning"),
				ir.Lit(0),
				ir.Lit("starting transform"),
			))),
		),

		// for row := range in {
		ir.RangeValue("row", ir.Id("in"),
			append([]ir.Stmt{
				// out := &OutputStruct{}
				ir.Define(ir.Id("out"), ir.Addr(ir.Composite(outputStructName))),
			},
				append(transformStmts,
					// rowCount++
					ir.RawStatement("rowCount++"),

					// Report progress every N rows
					ir.If(ir.And(ir.Neq(ir.Id("progress"), ir.Nil()), ir.Eq(ir.Mod(ir.Id("rowCount"), ir.Lit(progressInterval)), ir.Lit(0))),
						ir.ExprStatement(ir.Call("progress", ir.Call("lib.NewProgress",
							ir.Lit(node.ID),
							ir.Lit(node.Name),
							ir.Id("lib.StatusRunning"),
							ir.Id("rowCount"),
							ir.Lit(fmt.Sprintf("transformed %d rows", progressInterval)),
						))),
					),

					// select { case outChan <- out: case <-ctx.Done(): return ctx.Err() }
					ir.RawStatementf(`select {
		case outChan <- out:
		case <-ctx.Done():
			return ctx.Err()
		}`),
				)...,
			)...,
		),

		// Report completion
		ir.If(ir.Neq(ir.Id("progress"), ir.Nil()),
			ir.ExprStatement(ir.Call("progress", ir.Call("lib.NewProgress",
				ir.Lit(node.ID),
				ir.Lit(node.Name),
				ir.Id("lib.StatusCompleted"),
				ir.Id("rowCount"),
				ir.Lit("completed"),
			))),
		),

		ir.Return(ir.Nil()),
	}

	return ir.NewFunc(funcName).
		Param("ctx", "context.Context").
		Param("in", fmt.Sprintf("<-chan *%s", inputType)).
		Param("outChan", fmt.Sprintf("chan<- *%s", outputStructName)).
		Param("progress", "lib.ProgressFunc").
		Returns("error").
		Body(body...).
		Build(), nil
}

// generateJoinFunc generates a function for multi-input map with join
func (g *MapGenerator) generateJoinFunc(node *models.Node, config *models.MapConfig, ctx *GeneratorContext, funcName, outputStructName string, inputTypes map[string]string) (*ir.FuncDecl, error) {
	if config.Join == nil {
		return nil, fmt.Errorf("map node %d has multiple inputs but no join config", node.ID)
	}

	ctx.AddImport("test/lib")

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

	// Get join keys
	leftKey := join.LeftKey
	rightKey := join.RightKey
	if len(join.LeftKeys) > 0 {
		leftKey = join.LeftKeys[0] // For now, use first key for composite
	}
	if len(join.RightKeys) > 0 {
		rightKey = join.RightKeys[0]
	}

	leftKeyField := toPascalCase(leftKey)
	rightKeyField := toPascalCase(rightKey)

	// Build transform statements
	transformStmts := g.buildJoinTransformStatements(join.LeftInput, join.RightInput, output.Columns, config, ctx)

	var body []ir.Stmt

	// Add progress reporting wrapper
	progressStart := []ir.Stmt{
		ir.Var("rowCount", "int64"),
		ir.If(ir.Neq(ir.Id("progress"), ir.Nil()),
			ir.ExprStatement(ir.Call("progress", ir.Call("lib.NewProgress",
				ir.Lit(node.ID),
				ir.Lit(node.Name),
				ir.Id("lib.StatusRunning"),
				ir.Lit(0),
				ir.Lit("starting join"),
			))),
		),
	}

	progressEnd := []ir.Stmt{
		ir.If(ir.Neq(ir.Id("progress"), ir.Nil()),
			ir.ExprStatement(ir.Call("progress", ir.Call("lib.NewProgress",
				ir.Lit(node.ID),
				ir.Lit(node.Name),
				ir.Id("lib.StatusCompleted"),
				ir.Id("rowCount"),
				ir.Lit("completed"),
			))),
		),
	}

	switch join.Type {
	case models.JoinTypeLeft:
		body = g.generateLeftJoinBody(node, leftType, rightType, leftKeyField, rightKeyField, outputStructName, transformStmts, ctx)
	case models.JoinTypeInner:
		body = g.generateInnerJoinBody(node, leftType, rightType, leftKeyField, rightKeyField, outputStructName, transformStmts, ctx)
	case models.JoinTypeRight:
		body = g.generateRightJoinBody(node, leftType, rightType, leftKeyField, rightKeyField, outputStructName, transformStmts, ctx)
	case models.JoinTypeCross:
		body = g.generateCrossJoinBody(node, leftType, rightType, outputStructName, transformStmts, ctx)
	case models.JoinTypeUnion:
		body = g.generateUnionBody(node, leftType, rightType, outputStructName, transformStmts, ctx)
	default:
		// Default to left join
		body = g.generateLeftJoinBody(node, leftType, rightType, leftKeyField, rightKeyField, outputStructName, transformStmts, ctx)
	}

	// Insert progress start at beginning, progress end before return
	body = append(progressStart, body[:len(body)-1]...) // Remove final return
	body = append(body, progressEnd...)
	body = append(body, ir.Return(ir.Nil()))

	return ir.NewFunc(funcName).
		Param("ctx", "context.Context").
		Param("leftIn", fmt.Sprintf("<-chan *%s", leftType)).
		Param("rightIn", fmt.Sprintf("<-chan *%s", rightType)).
		Param("outChan", fmt.Sprintf("chan<- *%s", outputStructName)).
		Param("progress", "lib.ProgressFunc").
		Returns("error").
		Body(body...).
		Build(), nil
}

// generateLeftJoinBody generates the body for a left join
func (g *MapGenerator) generateLeftJoinBody(node *models.Node, leftType, rightType, leftKeyField, rightKeyField, outputStructName string, transformStmts []ir.Stmt, ctx *GeneratorContext) []ir.Stmt {
	ctx.AddImport("fmt")
	progressInterval := 1000

	return []ir.Stmt{
		// Build right index: map[key]*RightRow
		ir.Define(ir.Id("rightIndex"), ir.Call("make", ir.Raw(fmt.Sprintf("map[string]*%s", rightType)))),

		// Drain right channel into index
		ir.RangeValue("r", ir.Id("rightIn"),
			ir.Assign(
				ir.Index(ir.Id("rightIndex"), ir.Call("fmt.Sprintf", ir.Lit("%v"), ir.Sel(ir.Id("r"), rightKeyField))),
				ir.Id("r"),
			),
		),

		// Iterate left, lookup right
		ir.RangeValue("left", ir.Id("leftIn"),
			append([]ir.Stmt{
				ir.Define(ir.Id("key"), ir.Call("fmt.Sprintf", ir.Lit("%v"), ir.Sel(ir.Id("left"), leftKeyField))),
				ir.Define(ir.Id("right"), ir.Index(ir.Id("rightIndex"), ir.Id("key"))),
				ir.Define(ir.Id("out"), ir.Addr(ir.Composite(outputStructName))),
			},
				append(transformStmts,
					ir.RawStatement("rowCount++"),
					ir.If(ir.And(ir.Neq(ir.Id("progress"), ir.Nil()), ir.Eq(ir.Mod(ir.Id("rowCount"), ir.Lit(progressInterval)), ir.Lit(0))),
						ir.ExprStatement(ir.Call("progress", ir.Call("lib.NewProgress",
							ir.Lit(node.ID),
							ir.Lit(node.Name),
							ir.Id("lib.StatusRunning"),
							ir.Id("rowCount"),
							ir.Lit(fmt.Sprintf("joined %d rows", progressInterval)),
						))),
					),
					ir.RawStatementf(`select {
		case outChan <- out:
		case <-ctx.Done():
			return ctx.Err()
		}`),
				)...,
			)...,
		),

		ir.Return(ir.Nil()),
	}
}

// generateInnerJoinBody generates the body for an inner join
func (g *MapGenerator) generateInnerJoinBody(node *models.Node, leftType, rightType, leftKeyField, rightKeyField, outputStructName string, transformStmts []ir.Stmt, ctx *GeneratorContext) []ir.Stmt {
	ctx.AddImport("fmt")
	_ = node // Used for progress reporting in wrapper

	return []ir.Stmt{
		// Build right index
		ir.Define(ir.Id("rightIndex"), ir.Call("make", ir.Raw(fmt.Sprintf("map[string]*%s", rightType)))),
		ir.RangeValue("r", ir.Id("rightIn"),
			ir.Assign(
				ir.Index(ir.Id("rightIndex"), ir.Call("fmt.Sprintf", ir.Lit("%v"), ir.Sel(ir.Id("r"), rightKeyField))),
				ir.Id("r"),
			),
		),

		// Iterate left, only emit when right matches
		ir.RangeValue("left", ir.Id("leftIn"),
			ir.Define(ir.Id("key"), ir.Call("fmt.Sprintf", ir.Lit("%v"), ir.Sel(ir.Id("left"), leftKeyField))),
			ir.DefineMulti(
				[]ir.Expr{ir.Id("right"), ir.Id("ok")},
				[]ir.Expr{ir.Index(ir.Id("rightIndex"), ir.Id("key"))},
			),
			ir.If(ir.Not(ir.Id("ok")),
				ir.RawStatement("continue"),
			),
			ir.Define(ir.Id("out"), ir.Addr(ir.Composite(outputStructName))),
			ir.RawStatementf("%s", strings.Join(stmtsToStrings(transformStmts), "\n")),
			ir.RawStatement("rowCount++"),
			ir.RawStatementf(`select {
		case outChan <- out:
		case <-ctx.Done():
			return ctx.Err()
		}`),
		),

		ir.Return(ir.Nil()),
	}
}

// generateRightJoinBody generates the body for a right join
func (g *MapGenerator) generateRightJoinBody(node *models.Node, leftType, rightType, leftKeyField, rightKeyField, outputStructName string, transformStmts []ir.Stmt, ctx *GeneratorContext) []ir.Stmt {
	ctx.AddImport("fmt")
	_ = node // Used for progress reporting in wrapper

	return []ir.Stmt{
		// Build left index
		ir.Define(ir.Id("leftIndex"), ir.Call("make", ir.Raw(fmt.Sprintf("map[string]*%s", leftType)))),
		ir.RangeValue("l", ir.Id("leftIn"),
			ir.Assign(
				ir.Index(ir.Id("leftIndex"), ir.Call("fmt.Sprintf", ir.Lit("%v"), ir.Sel(ir.Id("l"), leftKeyField))),
				ir.Id("l"),
			),
		),

		// Iterate right, lookup left
		ir.RangeValue("right", ir.Id("rightIn"),
			append([]ir.Stmt{
				ir.Define(ir.Id("key"), ir.Call("fmt.Sprintf", ir.Lit("%v"), ir.Sel(ir.Id("right"), rightKeyField))),
				ir.Define(ir.Id("left"), ir.Index(ir.Id("leftIndex"), ir.Id("key"))),
				ir.Define(ir.Id("out"), ir.Addr(ir.Composite(outputStructName))),
			},
				append(transformStmts,
					ir.RawStatement("rowCount++"),
					ir.RawStatementf(`select {
		case outChan <- out:
		case <-ctx.Done():
			return ctx.Err()
		}`),
				)...,
			)...,
		),

		ir.Return(ir.Nil()),
	}
}

// generateCrossJoinBody generates the body for a cross join
func (g *MapGenerator) generateCrossJoinBody(node *models.Node, leftType, rightType, outputStructName string, transformStmts []ir.Stmt, ctx *GeneratorContext) []ir.Stmt {
	_ = node // Used for progress reporting in wrapper

	return []ir.Stmt{
		// Collect all right rows
		ir.Define(ir.Id("rightRows"), ir.Call("make", ir.Raw(fmt.Sprintf("[]*%s", rightType)), ir.Lit(0))),
		ir.RangeValue("r", ir.Id("rightIn"),
			ir.Assign(ir.Id("rightRows"), ir.Call("append", ir.Id("rightRows"), ir.Id("r"))),
		),

		// Cross product
		ir.RangeValue("left", ir.Id("leftIn"),
			ir.RangeValue("right", ir.Id("rightRows"),
				append([]ir.Stmt{
					ir.Define(ir.Id("out"), ir.Addr(ir.Composite(outputStructName))),
				},
					append(transformStmts,
						ir.RawStatement("rowCount++"),
						ir.RawStatementf(`select {
		case outChan <- out:
		case <-ctx.Done():
			return ctx.Err()
		}`),
					)...,
				)...,
			),
		),

		ir.Return(ir.Nil()),
	}
}

// generateUnionBody generates the body for a union (concatenate rows)
func (g *MapGenerator) generateUnionBody(node *models.Node, leftType, rightType, outputStructName string, transformStmts []ir.Stmt, ctx *GeneratorContext) []ir.Stmt {
	_ = node // Used for progress reporting in wrapper

	// For union, we process both streams independently
	// This requires separate transform logic for each input type
	return []ir.Stmt{
		ir.RawStatement("// Union: process left stream"),
		ir.RangeValue("left", ir.Id("leftIn"),
			ir.Define(ir.Id("out"), ir.Addr(ir.Composite(outputStructName))),
			ir.RawStatement("// TODO: transform left row to output"),
			ir.RawStatement("_ = left"),
			ir.RawStatement("rowCount++"),
			ir.RawStatementf(`select {
		case outChan <- out:
		case <-ctx.Done():
			return ctx.Err()
		}`),
		),

		ir.RawStatement("// Union: process right stream"),
		ir.RangeValue("right", ir.Id("rightIn"),
			ir.Define(ir.Id("out"), ir.Addr(ir.Composite(outputStructName))),
			ir.RawStatement("// TODO: transform right row to output"),
			ir.RawStatement("_ = right"),
			ir.RawStatement("rowCount++"),
			ir.RawStatementf(`select {
		case outChan <- out:
		case <-ctx.Done():
			return ctx.Err()
		}`),
		),

		ir.Return(ir.Nil()),
	}
}

// buildTransformStatements builds the column transformation statements for single input
func (g *MapGenerator) buildTransformStatements(inputName string, columns []models.MapOutputCol, config *models.MapConfig, ctx *GeneratorContext) []ir.Stmt {
	stmts := make([]ir.Stmt, 0, len(columns))

	for _, col := range columns {
		fieldName := toPascalCase(col.Name)
		stmt := g.buildColumnTransform("row", fieldName, col, config, ctx)
		stmts = append(stmts, stmt)
	}

	return stmts
}

// buildJoinTransformStatements builds transformation statements for join (left/right variables)
func (g *MapGenerator) buildJoinTransformStatements(leftInput, rightInput string, columns []models.MapOutputCol, config *models.MapConfig, ctx *GeneratorContext) []ir.Stmt {
	stmts := make([]ir.Stmt, 0, len(columns))

	for _, col := range columns {
		fieldName := toPascalCase(col.Name)
		stmt := g.buildJoinColumnTransform(leftInput, rightInput, fieldName, col, config, ctx)
		stmts = append(stmts, stmt)
	}

	return stmts
}

// buildColumnTransform builds a single column transformation for single input
func (g *MapGenerator) buildColumnTransform(rowVar, fieldName string, col models.MapOutputCol, config *models.MapConfig, ctx *GeneratorContext) ir.Stmt {
	switch col.FuncType {
	case models.FuncTypeDirect:
		// Direct mapping: out.Field = row.SourceField
		sourceField := extractFieldName(col.InputRef)
		return ir.Assign(
			ir.Sel(ir.Id("out"), fieldName),
			ir.Sel(ir.Id(rowVar), toPascalCase(sourceField)),
		)

	case models.FuncTypeLibrary:
		// Library function: out.Field = lib.Func(args...)
		ctx.AddImport("test/lib")
		args := g.buildFuncArgs(rowVar, col.Args)
		return ir.Assign(
			ir.Sel(ir.Id("out"), fieldName),
			ir.Call(fmt.Sprintf("lib.%s", col.LibFunc), args...),
		)

	case models.FuncTypeCustom:
		// Custom expression or function
		if col.CustomType == models.CustomExpr {
			// Expression: parse and substitute
			expr := g.substituteExprVars(rowVar, col.Expression)
			return ir.RawStatementf("out.%s = %s", fieldName, expr)
		}
		// Custom function body
		return ir.RawStatementf("out.%s = func() { %s }()", fieldName, col.FuncBody)

	default:
		return ir.RawStatementf("// Unknown transform type for %s", fieldName)
	}
}

// buildJoinColumnTransform builds a single column transformation for join
func (g *MapGenerator) buildJoinColumnTransform(leftInput, rightInput, fieldName string, col models.MapOutputCol, config *models.MapConfig, ctx *GeneratorContext) ir.Stmt {
	switch col.FuncType {
	case models.FuncTypeDirect:
		// Direct mapping: out.Field = left.SourceField or right.SourceField
		inputName, sourceField := parseInputRef(col.InputRef)
		rowVar := "left"
		if inputName == rightInput {
			rowVar = "right"
		}

		// Handle nil right for left join
		if rowVar == "right" {
			return ir.IfElse(
				ir.Neq(ir.Id("right"), ir.Nil()),
				[]ir.Stmt{ir.Assign(ir.Sel(ir.Id("out"), fieldName), ir.Sel(ir.Id(rowVar), toPascalCase(sourceField)))},
				nil, // zero value already set
			)
		}

		return ir.Assign(
			ir.Sel(ir.Id("out"), fieldName),
			ir.Sel(ir.Id(rowVar), toPascalCase(sourceField)),
		)

	case models.FuncTypeLibrary:
		// Library function with join context
		ctx.AddImport("test/lib")
		args := g.buildJoinFuncArgs(leftInput, rightInput, col.Args)
		return ir.Assign(
			ir.Sel(ir.Id("out"), fieldName),
			ir.Call(fmt.Sprintf("lib.%s", col.LibFunc), args...),
		)

	case models.FuncTypeCustom:
		if col.CustomType == models.CustomExpr {
			expr := g.substituteJoinExprVars(leftInput, rightInput, col.Expression)
			return ir.RawStatementf("out.%s = %s", fieldName, expr)
		}
		return ir.RawStatementf("out.%s = func() { %s }()", fieldName, col.FuncBody)

	default:
		return ir.RawStatementf("// Unknown transform type for %s", fieldName)
	}
}

// buildFuncArgs builds function arguments for single input
func (g *MapGenerator) buildFuncArgs(rowVar string, args []models.FuncArg) []ir.Expr {
	result := make([]ir.Expr, 0, len(args))

	for _, arg := range args {
		switch arg.Type {
		case "column":
			// Column reference: row.Field
			field := extractFieldName(arg.Value)
			result = append(result, ir.Sel(ir.Id(rowVar), toPascalCase(field)))
		case "literal":
			// Literal value
			result = append(result, ir.Lit(arg.Value))
		default:
			result = append(result, ir.Lit(arg.Value))
		}
	}

	return result
}

// buildJoinFuncArgs builds function arguments for join context
func (g *MapGenerator) buildJoinFuncArgs(leftInput, rightInput string, args []models.FuncArg) []ir.Expr {
	result := make([]ir.Expr, 0, len(args))

	for _, arg := range args {
		switch arg.Type {
		case "column":
			inputName, field := parseInputRef(arg.Value)
			rowVar := "left"
			if inputName == rightInput {
				rowVar = "right"
			}
			result = append(result, ir.Sel(ir.Id(rowVar), toPascalCase(field)))
		case "literal":
			result = append(result, ir.Lit(arg.Value))
		default:
			result = append(result, ir.Lit(arg.Value))
		}
	}

	return result
}

// substituteExprVars substitutes variable references in an expression
func (g *MapGenerator) substituteExprVars(rowVar, expr string) string {
	// Replace inputName.column with row.Column
	// This is a simple implementation - could be enhanced with proper parsing
	return expr // TODO: implement proper substitution
}

// substituteJoinExprVars substitutes variable references in join expressions
func (g *MapGenerator) substituteJoinExprVars(leftInput, rightInput, expr string) string {
	// Replace inputName.column with rowVar.Column (PascalCase)
	result := expr

	// Find and replace all occurrences of inputName.fieldName
	for _, input := range []struct {
		name   string
		rowVar string
	}{
		{leftInput, "left"},
		{rightInput, "right"},
	} {
		prefix := input.name + "."
		for {
			idx := strings.Index(result, prefix)
			if idx == -1 {
				break
			}

			// Find the end of the field name
			endIdx := idx + len(prefix)
			for endIdx < len(result) && (result[endIdx] == '_' || (result[endIdx] >= 'a' && result[endIdx] <= 'z') || (result[endIdx] >= 'A' && result[endIdx] <= 'Z') || (result[endIdx] >= '0' && result[endIdx] <= '9')) {
				endIdx++
			}

			fieldName := result[idx+len(prefix) : endIdx]
			pascalField := toPascalCase(fieldName)
			replacement := input.rowVar + "." + pascalField

			result = result[:idx] + replacement + result[endIdx:]
		}
	}

	return result
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

// mapDataType maps config data types to Go types
func mapDataType(dataType string) string {
	switch strings.ToLower(dataType) {
	case "int", "integer":
		return "int"
	case "int64", "bigint":
		return "int64"
	case "float", "float64", "double":
		return "float64"
	case "bool", "boolean":
		return "bool"
	case "string", "varchar", "text":
		return "string"
	case "time", "time.time", "timestamp", "datetime":
		return "time.Time"
	default:
		return "any"
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

// stmtsToStrings is a helper to convert statements to strings (for embedding in raw)
func stmtsToStrings(stmts []ir.Stmt) []string {
	// This is a placeholder - in practice, we'd use the emitter
	result := make([]string, len(stmts))
	for i := range stmts {
		result[i] = fmt.Sprintf("// transform %d", i)
	}
	return result
}
