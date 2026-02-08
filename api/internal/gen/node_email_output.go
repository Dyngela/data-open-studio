package gen

import (
	"api/internal/api/models"
	"fmt"
	"strings"
)

// EmailOutputGenerator generates code for email_output nodes
type EmailOutputGenerator struct{}

func (g *EmailOutputGenerator) NodeType() models.NodeType {
	return models.NodeTypeEmailOutput
}

// GenerateStructData returns nil - email_output consumes data, doesn't produce a new type
func (g *EmailOutputGenerator) GenerateStructData(node *models.Node) (*StructData, error) {
	return nil, nil
}

// GenerateFuncData generates the function data for this email_output node
func (g *EmailOutputGenerator) GenerateFuncData(node *models.Node, ctx *GeneratorContext) (*NodeFunctionData, error) {
	config, err := node.GetEmailOutputConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get email_output config: %w", err)
	}

	ctx.AddImport("context")
	ctx.AddImport("fmt")
	ctx.AddImport("bytes")
	ctx.AddImport("text/template")
	ctx.AddImport("test/lib")
	ctx.AddImport("github.com/wneessen/go-mail")

	funcName := ctx.FuncName(node)
	inputRowType := g.findInputRowType(node, ctx)
	if inputRowType == "" {
		inputRowType = "any"
	}

	engine, err := NewTemplateEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create template engine: %w", err)
	}

	templateData := EmailOutputTemplateData{
		FuncName:        funcName,
		NodeID:          node.ID,
		NodeName:        node.Name,
		InputType:       inputRowType,
		MetadataEmailID: config.MetadataEmailID,
		SmtpHost:        config.SmtpHost,
		SmtpPort:        config.SmtpPort,
		Username:        config.Username,
		Password:        config.Password,
		UseTLS:          config.UseTLS,
		To:              strings.Join(config.To, ", "),
		CC:              strings.Join(config.CC, ", "),
		BCC:             strings.Join(config.BCC, ", "),
		Subject:         config.Subject,
		Body:            config.Body,
		IsHTML:          config.IsHTML,
	}

	body, err := engine.GenerateNodeFunction("node_email_output.go.tmpl", templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate email_output function: %w", err)
	}

	return &NodeFunctionData{
		Name:      funcName,
		NodeID:    node.ID,
		NodeName:  node.Name,
		Signature: "",
		Body:      body,
	}, nil
}

// GetLaunchArgs returns the launch arguments for email_output: [inputChannel]
func (g *EmailOutputGenerator) GetLaunchArgs(node *models.Node, channels []channelInfo, dbConnections map[string]string) []string {
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

func (g *EmailOutputGenerator) findInputRowType(node *models.Node, ctx *GeneratorContext) string {
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
