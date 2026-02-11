package gen

import (
	"api/internal/api/models"
	"fmt"
)

type LogGenerator struct{}

func (g *LogGenerator) NodeType() models.NodeType {
	return models.NodeTypeLog
}

// GenerateStructData returns nil — log is a sink, it uses the input node's row type
func (g *LogGenerator) GenerateStructData(node *models.Node) (*StructData, error) {
	return nil, nil
}

// GetLaunchArgs returns the launch arguments for log: [inputChannel]
func (g *LogGenerator) GetLaunchArgs(node *models.Node, channels []channelInfo, dbConnections map[string]string) []string {
	args := make([]string, 0, 1)

	// Add input channel
	for _, ch := range channels {
		if ch.toNodeID == node.ID {
			args = append(args, fmt.Sprintf("ch_%d", ch.portID))
			break
		}
	}

	return args
}

func (g *LogGenerator) GenerateFuncData(node *models.Node, ctx *GeneratorContext) (*NodeFunctionData, error) {
	ctx.AddImport("context")
	ctx.AddImport("fmt")
	ctx.AddImport("test/lib")

	funcName := ctx.FuncName(node)
	inputRowType := g.findInputRowType(node, ctx)
	if inputRowType == "" {
		inputRowType = "any"
	}

	separator := " | "
	if config, err := node.GetLogConfig(); err == nil && config.Separator != "" {
		separator = config.Separator
	}

	// Use template engine
	engine, err := NewTemplateEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create template engine: %w", err)
	}

	templateData := LogTemplateData{
		FuncName:  funcName,
		NodeID:    node.ID,
		NodeName:  node.Name,
		InputType: inputRowType,
		Separator: separator,
	}

	body, err := engine.GenerateNodeFunction("node_log.go.tmpl", templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate log function: %w", err)
	}

	return &NodeFunctionData{
		Name:      funcName,
		NodeID:    node.ID,
		NodeName:  node.Name,
		Signature: "", // Not used - template generates complete function
		Body:      body,
	}, nil
}

func (g *LogGenerator) findInputRowType(node *models.Node, ctx *GeneratorContext) string {
	for _, port := range node.InputPort {
		if port.Type == models.PortTypeInput {
			sourceNodeID := int(port.ConnectedNodeID)
			if sourceNodeID == 0 {
				continue
			}
			if structName, exists := ctx.NodeStructNames[sourceNodeID]; exists {
				return structName
			}
		}
	}
	return ""
}
