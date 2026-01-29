package gen

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"text/template"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// TemplateEngine handles code generation using templates
type TemplateEngine struct {
	templates *template.Template
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine() (*TemplateEngine, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &TemplateEngine{
		templates: tmpl,
	}, nil
}

// GenerateMainFile generates the complete main.go file
func (e *TemplateEngine) GenerateMainFile(data *TemplateData) ([]byte, error) {
	var buf bytes.Buffer

	if err := e.templates.ExecuteTemplate(&buf, "main.go.tmpl", data); err != nil {
		return nil, fmt.Errorf("failed to execute main template: %w", err)
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Return unformatted code for debugging
		return buf.Bytes(), fmt.Errorf("failed to format generated code: %w (raw output available)", err)
	}

	return formatted, nil
}

// GenerateNodeFunction generates a node function using the appropriate template
func (e *TemplateEngine) GenerateNodeFunction(templateName string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := e.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return buf.String(), nil
}
