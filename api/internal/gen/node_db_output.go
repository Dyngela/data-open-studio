package gen

import (
	"api/internal/api/models"
	"api/internal/gen/ir"
	"fmt"
	"strings"
)

// DBOutputGenerator generates code for db_output nodes
type DBOutputGenerator struct{}

func (g *DBOutputGenerator) NodeType() models.NodeType {
	return models.NodeTypeDBOutput
}

// GenerateStruct returns nil - db_output consumes data, doesn't produce a new type
func (g *DBOutputGenerator) GenerateStruct(node *models.Node) (*ir.StructDecl, error) {
	return nil, nil
}

// GenerateFunc generates the execution function for this db_output node
func (g *DBOutputGenerator) GenerateFunc(node *models.Node, ctx *GeneratorContext) (*ir.FuncDecl, error) {
	config, err := node.GetDBOutputConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get db_output config: %w", err)
	}

	// Add required imports
	ctx.AddImport("context")
	ctx.AddImport("database/sql")
	ctx.AddImport("fmt")
	ctx.AddImport("strings")
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
		return g.generateInsertFunc(node, &config, ctx, funcName, inputRowType)
	case models.DbOutputModeUpdate:
		return g.generateUpdateFunc(node, &config, ctx, funcName, inputRowType)
	case models.DbOutputModeDelete:
		return g.generateDeleteFunc(node, &config, ctx, funcName, inputRowType)
	case models.DbOutputModeTruncate:
		return g.generateTruncateFunc(node, &config, ctx, funcName, inputRowType)
	default:
		// Default to insert
		panic("invalid db output mode: " + string(config.Mode))
	}
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

// generateInsertFunc generates a batch insert function
func (g *DBOutputGenerator) generateInsertFunc(node *models.Node, config *models.DBOutputConfig, ctx *GeneratorContext, funcName, inputRowType string) (*ir.FuncDecl, error) {
	if len(config.DataModels) == 0 {
		return nil, fmt.Errorf("db_output node %q: DataModels is empty - cannot generate INSERT without columns", node.Name)
	}

	ctx.AddImport("test/lib")

	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	// build column names for INSERT
	columns := make([]string, len(config.DataModels))
	for i, col := range config.DataModels {
		columns[i] = col.Name
	}
	columnsStr := strings.Join(columns, ", ")

	// build schema-qualified table name
	tableName := config.Table
	if config.DbSchema != "" {
		tableName = fmt.Sprintf("%s.%s", config.DbSchema, config.Table)
	}

	// Generate the function body
	body := []ir.Stmt{
		// batch := make([]*InputRowType, 0, batchSize)
		ir.Define(
			ir.Id("batch"),
			ir.Call("make", ir.Raw(fmt.Sprintf("[]*%s", inputRowType)), ir.Lit(0), ir.Lit(batchSize)),
		),

		// var totalRows int64
		ir.Var("totalRows", "int64"),

		// Report start
		ir.If(ir.Neq(ir.Id("progress"), ir.Nil()),
			ir.ExprStatement(ir.Call("progress", ir.Call("lib.NewProgress",
				ir.Lit(node.ID),
				ir.Lit(node.Name),
				ir.Id("lib.StatusRunning"),
				ir.Lit(0),
				ir.Lit("starting insert"),
			))),
		),

		// flushBatch function
		ir.Define(ir.Id("flushBatch"), ir.Closure(
			nil,
			[]ir.Param{{Type: "error"}},
			g.generateFlushBatchBody(node, config, tableName, columnsStr, inputRowType)...,
		)),

		// Main loop: for row := range in
		ir.RangeValue("row", ir.Id("in"),
			// batch = append(batch, row)
			ir.Assign(ir.Id("batch"), ir.Call("append", ir.Id("batch"), ir.Id("row"))),

			// if len(batch) >= batchSize { if err := flushBatch(); err != nil { return err } }
			ir.If(ir.Gte(ir.Call("len", ir.Id("batch")), ir.Lit(batchSize)),
				ir.IfInit(
					ir.Define(ir.Id("err"), ir.Call("flushBatch")),
					ir.Neq(ir.Id("err"), ir.Nil()),
					ir.Return(ir.Id("err")),
				),
			),
		),

		// Flush remaining batch
		ir.If(ir.Gt(ir.Call("len", ir.Id("batch")), ir.Lit(0)),
			ir.IfInit(
				ir.Define(ir.Id("err"), ir.Call("flushBatch")),
				ir.Neq(ir.Id("err"), ir.Nil()),
				ir.Return(ir.Id("err")),
			),
		),

		// Report completion
		ir.If(ir.Neq(ir.Id("progress"), ir.Nil()),
			ir.ExprStatement(ir.Call("progress", ir.Call("lib.NewProgress",
				ir.Lit(node.ID),
				ir.Lit(node.Name),
				ir.Id("lib.StatusCompleted"),
				ir.Id("totalRows"),
				ir.Lit("completed"),
			))),
		),

		ir.Return(ir.Nil()),
	}

	return ir.NewFunc(funcName).
		Param("ctx", "context.Context").
		Param("db", "*sql.DB").
		Param("in", fmt.Sprintf("<-chan *%s", inputRowType)).
		Param("progress", "lib.ProgressFunc").
		Returns("error").
		Body(body...).
		Build(), nil
}

// generateFlushBatchBody generates the body of the flushBatch closure
func (g *DBOutputGenerator) generateFlushBatchBody(node *models.Node, config *models.DBOutputConfig, tableName, columnsStr, inputRowType string) []ir.Stmt {
	numCols := len(config.DataModels)

	// build field accessors for the row
	fieldAccessors := make([]string, numCols)
	for i, col := range config.DataModels {
		fieldAccessors[i] = fmt.Sprintf("row.%s", col.GoFieldName())
	}

	return []ir.Stmt{
		// if len(batch) == 0 { return nil }
		ir.If(ir.Eq(ir.Call("len", ir.Id("batch")), ir.Lit(0)),
			ir.Return(ir.Nil()),
		),

		// batchLen := int64(len(batch))
		ir.Define(ir.Id("batchLen"), ir.Call("int64", ir.Call("len", ir.Id("batch")))),

		// build VALUES placeholders
		// var placeholders []string
		ir.Var("placeholders", "[]string"),

		// var args []any
		ir.Var("args", "[]any"),

		// for i, row := range batch { ... }
		ir.Range("i", "row", ir.Id("batch"),
			// build placeholder like ($1, $2, $3) for each row
			ir.Define(ir.Id("offset"), ir.Mul(ir.Id("i"), ir.Lit(numCols))),
			ir.Define(ir.Id("ph"), ir.Call("make", ir.Raw("[]string"), ir.Lit(0), ir.Lit(numCols))),

			// for j := 0; j < numCols; j++ { ph = append(ph, fmt.Sprintf("$%d", offset+j+1)) }
			ir.ForClassic(
				ir.Define(ir.Id("j"), ir.Lit(0)),
				ir.Lt(ir.Id("j"), ir.Lit(numCols)),
				ir.Assign(ir.Id("j"), ir.Add(ir.Id("j"), ir.Lit(1))),
				ir.Assign(ir.Id("ph"), ir.Call("append", ir.Id("ph"),
					ir.Call("fmt.Sprintf", ir.Lit("$%d"), ir.Add(ir.Add(ir.Id("offset"), ir.Id("j")), ir.Lit(1))),
				)),
			),

			// placeholders = append(placeholders, "("+strings.Join(ph, ", ")+")")
			ir.Assign(ir.Id("placeholders"), ir.Call("append", ir.Id("placeholders"),
				ir.Add(ir.Add(ir.Lit("("), ir.Call("strings.Join", ir.Id("ph"), ir.Lit(", "))), ir.Lit(")")),
			)),

			// args = append(args, row.Field1, row.Field2, ...)
			ir.RawStatementf("args = append(args, %s)", strings.Join(fieldAccessors, ", ")),
		),

		// query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", table, columns, strings.Join(placeholders, ", "))
		ir.Define(ir.Id("query"), ir.Call("fmt.Sprintf",
			ir.Lit(fmt.Sprintf("INSERT INTO %s (%s) VALUES %%s", tableName, columnsStr)),
			ir.Call("strings.Join", ir.Id("placeholders"), ir.Lit(", ")),
		)),

		// _, err := db.ExecContext(ctx, query, args...)
		ir.DefineMulti(
			[]ir.Expr{ir.Id("_"), ir.Id("err")},
			[]ir.Expr{ir.CallVariadic("db.ExecContext", ir.Id("ctx"), ir.Id("query"), ir.Id("args"))},
		),

		// if err != nil { return fmt.Errorf("batch insert failed: %w", err) }
		ir.If(ir.Neq(ir.Id("err"), ir.Nil()),
			ir.Return(ir.Call("fmt.Errorf", ir.Lit("batch insert failed: %w"), ir.Id("err"))),
		),

		// totalRows += batchLen
		ir.RawStatement("totalRows += batchLen"),

		// Report progress after batch insert
		ir.If(ir.Neq(ir.Id("progress"), ir.Nil()),
			ir.ExprStatement(ir.Call("progress", ir.Call("lib.NewProgress",
				ir.Lit(node.ID),
				ir.Lit(node.Name),
				ir.Id("lib.StatusRunning"),
				ir.Id("totalRows"),
				ir.Lit("batch inserted"),
			))),
		),

		// batch = batch[:0] (reset batch)
		ir.Assign(ir.Id("batch"), ir.Slice(ir.Id("batch"), nil, ir.Lit(0))),

		ir.Return(ir.Nil()),
	}
}

// generateUpdateFunc generates an update function (simplified - updates by primary key)
func (g *DBOutputGenerator) generateUpdateFunc(node *models.Node, config *models.DBOutputConfig, ctx *GeneratorContext, funcName, inputRowType string) (*ir.FuncDecl, error) {
	ctx.AddImport("test/lib")
	// For now, generate a placeholder that can be expanded later
	return ir.NewFunc(funcName).
		Param("ctx", "context.Context").
		Param("db", "*sql.DB").
		Param("in", fmt.Sprintf("<-chan *%s", inputRowType)).
		Param("progress", "lib.ProgressFunc").
		Returns("error").
		Body(
			ir.RawStatement("// TODO: Implement UPDATE logic"),
			ir.RawStatement("_ = progress"),
			ir.RangeValue("row", ir.Id("in"),
				ir.RawStatement("_ = row // consume input"),
			),
			ir.Return(ir.Nil()),
		).
		Build(), nil
}

// generateDeleteFunc generates a delete function
func (g *DBOutputGenerator) generateDeleteFunc(node *models.Node, config *models.DBOutputConfig, ctx *GeneratorContext, funcName, inputRowType string) (*ir.FuncDecl, error) {
	ctx.AddImport("test/lib")
	return ir.NewFunc(funcName).
		Param("ctx", "context.Context").
		Param("db", "*sql.DB").
		Param("in", fmt.Sprintf("<-chan *%s", inputRowType)).
		Param("progress", "lib.ProgressFunc").
		Returns("error").
		Body(
			ir.RawStatement("// TODO: Implement DELETE logic"),
			ir.RawStatement("_ = progress"),
			ir.RangeValue("row", ir.Id("in"),
				ir.RawStatement("_ = row // consume input"),
			),
			ir.Return(ir.Nil()),
		).
		Build(), nil
}

// generateTruncateFunc generates a truncate + insert function
func (g *DBOutputGenerator) generateTruncateFunc(node *models.Node, config *models.DBOutputConfig, ctx *GeneratorContext, funcName, inputRowType string) (*ir.FuncDecl, error) {
	ctx.AddImport("test/lib")
	tableName := config.Table
	if config.DbSchema != "" {
		tableName = fmt.Sprintf("%s.%s", config.DbSchema, config.Table)
	}

	return ir.NewFunc(funcName).
		Param("ctx", "context.Context").
		Param("db", "*sql.DB").
		Param("in", fmt.Sprintf("<-chan *%s", inputRowType)).
		Param("progress", "lib.ProgressFunc").
		Returns("error").
		Body(
			ir.RawStatement("_ = progress"),
			// TRUNCATE TABLE
			ir.DefineMulti(
				[]ir.Expr{ir.Id("_"), ir.Id("err")},
				[]ir.Expr{ir.Call("db.ExecContext", ir.Id("ctx"), ir.Lit(fmt.Sprintf("TRUNCATE TABLE %s", tableName)))},
			),
			ir.If(ir.Neq(ir.Id("err"), ir.Nil()),
				ir.Return(ir.Call("fmt.Errorf", ir.Lit("truncate failed: %w"), ir.Id("err"))),
			),

			// Then do normal insert
			ir.RawStatement("// TODO: Follow with INSERT logic"),
			ir.RangeValue("row", ir.Id("in"),
				ir.RawStatement("_ = row // consume input"),
			),
			ir.Return(ir.Nil()),
		).
		Build(), nil
}
