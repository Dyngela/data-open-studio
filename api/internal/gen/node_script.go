package gen

import (
	"api/internal/api/models"
	"fmt"
	"strings"
)

type ScriptGenerator struct{}

func (g *ScriptGenerator) NodeType() models.NodeType {
	return models.NodeTypeScript
}

// GenerateStructData generates the output row struct for this script node
func (g *ScriptGenerator) GenerateStructData(node *models.Node) (*StructData, error) {
	config, err := node.GetScriptConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get script config: %w", err)
	}

	if len(config.DataModels) == 0 {
		return nil, nil
	}

	structName := fmt.Sprintf("Node%dRow", node.ID)
	fields := make([]FieldData, len(config.DataModels))

	for i, col := range config.DataModels {
		fields[i] = FieldData{
			Name: col.GoFieldName(),
			Type: col.GoFieldType(),
			Tag:  fmt.Sprintf(`json:"%s"`, col.Name),
		}
	}

	return &StructData{
		Name:   structName,
		NodeID: node.ID,
		Fields: fields,
	}, nil
}

func (g *ScriptGenerator) GenerateFuncData(node *models.Node, ctx *GeneratorContext) (*NodeFunctionData, error) {
	config, err := node.GetScriptConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get script config: %w", err)
	}

	ctx.AddImport("context")
	ctx.AddImport("fmt")
	ctx.AddImport("os")
	ctx.AddImport("os/exec")
	ctx.AddImport("encoding/json")
	ctx.AddImport("bufio")
	ctx.AddImport("bytes")
	ctx.AddImport("test/lib")

	funcName := ctx.FuncName(node)
	outputType := ctx.StructName(node)

	inputs := make([]ScriptInputData, 0, len(config.Inputs))
	for _, sel := range config.Inputs {
		rowType := g.findRowTypeForPort(sel.PortID, node, ctx)
		inputs = append(inputs, ScriptInputData{
			Name:       sel.Name,
			ChannelVar: fmt.Sprintf("ch_%d", sel.PortID),
			RowType:    rowType,
		})
	}

	fullScript := buildPythonScript(config.Code, inputs)

	engine, err := NewTemplateEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create template engine: %w", err)
	}

	templateData := ScriptTemplateData{
		FuncName:   funcName,
		NodeID:     node.ID,
		NodeName:   node.Name,
		OutputType: outputType,
		Inputs:     inputs,
		FullScript: fullScript,
	}

	body, err := engine.GenerateNodeFunction("node_script.go.tmpl", templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate script function: %w", err)
	}

	return &NodeFunctionData{
		Name:     funcName,
		NodeID:   node.ID,
		NodeName: node.Name,
		Body:     body,
	}, nil
}

// buildPythonScript assembles the complete Python script with the runner boilerplate
func buildPythonScript(userCode string, inputs []ScriptInputData) string {
	// Build the transform call: transform(A, B, ...)
	argNames := make([]string, len(inputs))
	for i, inp := range inputs {
		argNames[i] = inp.Name
	}
	transformCall := "transform(" + strings.Join(argNames, ", ") + ")"

	// Build the data extraction lines: A = data.get("A", [])
	var extractLines strings.Builder
	for _, inp := range inputs {
		fmt.Fprintf(&extractLines, "    %s = data.get(%q, [])\n", inp.Name, inp.Name)
	}

	runner := fmt.Sprintf(`

import sys, json

if __name__ == "__main__":
    data = json.loads(sys.stdin.read())
%s    results = %s
    for row in results:
        print(json.dumps(row))
`, extractLines.String(), transformCall)

	return userCode + runner
}

// GetLaunchArgs returns [inputChannels..., outputChannel]
func (g *ScriptGenerator) GetLaunchArgs(node *models.Node, channels []channelInfo, dbConnections map[string]string) []string {
	config, err := node.GetScriptConfig()
	if err != nil {
		return nil
	}

	args := make([]string, 0, len(config.Inputs)+1)

	// Add input channels IN ORDER of config.Inputs
	// Each input port is an incoming channel: find by matching toNodeID + portID
	for _, sel := range config.Inputs {
		for _, ch := range channels {
			if ch.toNodeID == node.ID && ch.portID == sel.PortID {
				args = append(args, fmt.Sprintf("ch_%d", ch.portID))
				break
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

// findRowTypeForPort resolves the Go struct name for a given input port ID
func (g *ScriptGenerator) findRowTypeForPort(portID uint, node *models.Node, ctx *GeneratorContext) string {
	for _, port := range node.InputPort {
		if port.ID == portID && port.Type == models.PortTypeInput {
			sourceNodeID := int(port.ConnectedNodeID)
			if sourceNodeID == 0 {
				continue
			}
			if structName, exists := ctx.NodeStructNames[sourceNodeID]; exists {
				return structName
			}
		}
	}
	return "any"
}
