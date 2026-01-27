package gen

import (
	"api/internal/api/models"
	"api/internal/gen/ir"
	"fmt"
)

// DBInputGenerator generates code for db_input nodes
type DBInputGenerator struct{}

func (g *DBInputGenerator) NodeType() models.NodeType {
	return models.NodeTypeDBInput
}

// GenerateStruct generates the row struct for this db_input node
func (g *DBInputGenerator) GenerateStruct(node *models.Node) (*ir.StructDecl, error) {
	config, err := node.GetDBInputConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get db_input config: %w", err)
	}

	structName := fmt.Sprintf("Node%dRow", node.ID)
	builder := ir.NewStruct(structName)

	for _, col := range config.DataModels {
		tag := fmt.Sprintf(`db:"%s"`, col.Name)
		builder.FieldWithTag(col.GoFieldName(), col.GoFieldType(), tag)
	}

	return builder.Build(), nil
}

// GenerateFunc generates the execution function for this db_input node
func (g *DBInputGenerator) GenerateFunc(node *models.Node, ctx *GeneratorContext) (*ir.FuncDecl, error) {
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
	connID := config.Connection.GetConnectionID()

	// build scan arguments: &row.Field1, &row.Field2, ...
	scanArgs := make([]ir.Expr, len(config.DataModels))
	for i, col := range config.DataModels {
		scanArgs[i] = ir.Addr(ir.Sel(ir.Id("row"), col.GoFieldName()))
	}

	// Get output port ID for the channel
	var outputPortID uint
	for _, port := range node.OutputPort {
		if port.Type == models.PortTypeOutput {
			outputPortID = port.ID
			break
		}
	}

	// Progress report interval (every 1000 rows)
	progressInterval := 1000

	fn := ir.NewFunc(funcName).
		Param("ctx", "context.Context").
		Param("db", "*sql.DB").
		Param("out", fmt.Sprintf("chan<- *%s", structName)).
		Param("progress", "lib.ProgressFunc").
		Returns("error").
		Body(
			// query := "SELECT ..."
			ir.Define(ir.Id("query"), ir.Lit(config.QueryWithSchema)),

			// var rowCount int64
			ir.Var("rowCount", "int64"),

			// Report start
			ir.If(ir.Neq(ir.Id("progress"), ir.Nil()),
				ir.ExprStatement(ir.Call("progress", ir.Call("lib.NewProgress",
					ir.Lit(node.ID),
					ir.Lit(node.Name),
					ir.Id("lib.StatusRunning"),
					ir.Lit(0),
					ir.Lit("starting query"),
				))),
			),

			// rows, err := db.QueryContext(ctx, query)
			ir.DefineMulti(
				[]ir.Expr{ir.Id("rows"), ir.Id("err")},
				[]ir.Expr{ir.Call("db.QueryContext", ir.Id("ctx"), ir.Id("query"))},
			),

			// if err != nil { return fmt.Errorf(...) }
			ir.If(ir.Neq(ir.Id("err"), ir.Nil()),
				ir.Return(ir.Call("fmt.Errorf",
					ir.Lit(fmt.Sprintf("node %d query failed: %%w", node.ID)),
					ir.Id("err"),
				)),
			),

			// defer rows.Close()
			ir.Defer(ir.Call("rows.Close")),

			// for rows.Next() { ... }
			ir.For(ir.Call("rows.Next"),
				// var row NodeXRow
				ir.Var("row", structName),

				// err := rows.Scan(&row.Field1, &row.Field2, ...)
				ir.Define(ir.Id("err"), ir.Call("rows.Scan", scanArgs...)),

				// if err != nil { return fmt.Errorf(...) }
				ir.If(ir.Neq(ir.Id("err"), ir.Nil()),
					ir.Return(ir.Call("fmt.Errorf",
						ir.Lit(fmt.Sprintf("node %d scan failed: %%w", node.ID)),
						ir.Id("err"),
					)),
				),

				// rowCount++
				ir.RawStatement("rowCount++"),

				// Report progress every N rows
				ir.If(ir.And(ir.Neq(ir.Id("progress"), ir.Nil()), ir.Eq(ir.Mod(ir.Id("rowCount"), ir.Lit(progressInterval)), ir.Lit(0))),
					ir.ExprStatement(ir.Call("progress", ir.Call("lib.NewProgress",
						ir.Lit(node.ID),
						ir.Lit(node.Name),
						ir.Id("lib.StatusRunning"),
						ir.Id("rowCount"),
						ir.Lit(fmt.Sprintf("read %d rows", progressInterval)),
					))),
				),

				// select { case out <- &row: case <-ctx.Done(): return ctx.Err() }
				ir.RawStatementf(`select {
		case out <- &row:
		case <-ctx.Done():
			return ctx.Err()
		}`),
			),

			// if err := rows.Err(); err != nil { return err }
			ir.IfInit(
				ir.Define(ir.Id("err"), ir.Call("rows.Err")),
				ir.Neq(ir.Id("err"), ir.Nil()),
				ir.Return(ir.Id("err")),
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

			// return nil
			ir.Return(ir.Nil()),
		).
		Build()

	// Store the output port ID for later wiring
	_ = outputPortID
	_ = connID

	return fn, nil
}

// GenerateConnInit generates the connection initialization code
func (g *DBInputGenerator) GenerateConnInit(config *models.DBInputConfig) []ir.Stmt {
	connID := config.Connection.GetConnectionID()
	driverName := config.Connection.GetDriverName()
	connString := config.Connection.BuildConnectionString()

	return []ir.Stmt{
		// db_<connID>, err := sql.Open("<driver>", "<connString>")
		ir.DefineMulti(
			[]ir.Expr{ir.Id(fmt.Sprintf("db_%s", connID)), ir.Id("err")},
			[]ir.Expr{ir.Call("sql.Open", ir.Lit(driverName), ir.Lit(connString))},
		),
		// if err != nil { return fmt.Errorf("...") }
		ir.If(ir.Neq(ir.Id("err"), ir.Nil()),
			ir.Return(ir.Call("fmt.Errorf",
				ir.Lit(fmt.Sprintf("failed to connect to %s: %%w", connID)),
				ir.Id("err"),
			)),
		),
		// defer db_<connID>.Close()
		ir.Defer(ir.Call(fmt.Sprintf("db_%s.Close", connID))),
	}
}
