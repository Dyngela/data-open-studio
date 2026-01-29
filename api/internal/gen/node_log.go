package gen

import (
	"api/internal/api/models"
	"api/internal/gen/ir"
	"fmt"
)

type LogGenerator struct{}

func (g *LogGenerator) NodeType() models.NodeType {
	return models.NodeTypeLog
}

func (g *LogGenerator) GenerateStruct(node *models.Node) (*ir.StructDecl, error) {
	config, err := node.GetLogConfig()
	if err != nil {
		return nil, err
	}

	structName := fmt.Sprintf("Node%dRow", node.ID)
	builder := ir.NewStruct(structName)

	for _, col := range config.Input {
		tag := fmt.Sprintf(`db:"%s"`, col.Name)
		builder.FieldWithTag(col.GoFieldName(), col.GoFieldType(), tag)
	}

	return builder.Build(), nil
}

func (g *LogGenerator) GenerateFunc(node *models.Node, ctx *GeneratorContext) (*ir.FuncDecl, error) {
	config, err := node.GetLogConfig()
	if err != nil {
		return nil, err
	}
	structName := ctx.StructName(node)
	funcName := ctx.FuncName(node)

	fn := ir.NewFunc(funcName).
		Param("ctx", "context.Context").
		Param("data", fmt.Sprintf("%s", structName)).
		Param("progress", "lib.ProgressFunc").
		Body(
			ir.If(ir.Neq(ir.Id("progress"), ir.Nil()),
				ir.ForClassic(
					ir.Define(ir.Id("j"), ir.Lit(0)),
					ir.Lt(ir.Id("j"), ir.Lit(len(config.Input))),
					ir.Assign(ir.Id("j"), ir.Add(ir.Id("j"), ir.Lit(1))),
					ir.ExprStatement(ir.Call("progress", ir.Call("lib.NewProgress",
						ir.Lit(node.ID),
						ir.Lit(node.Name),
						ir.Id("lib.StatusRunning"),
						ir.Lit("j"),
						ir.Lit("data[j]"),
					))),
				),
			),

			ir.If(ir.Eq(ir.Id("progress"), ir.Nil()),
				ir.ExprStatement(ir.Lit("panic(\"progress func not set\")")),
			),
		).
		Build()
	return fn, nil
}
