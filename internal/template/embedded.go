package template

import (
	"embed"
	"fmt"
)

//go:embed templates
var embeddedFS embed.FS

type EmbeddedTemplateSource struct{}

func NewEmbeddedSource() *EmbeddedTemplateSource {
	return &EmbeddedTemplateSource{}
}

func (e *EmbeddedTemplateSource) GetTemplate(subsystemType, fileName string) ([]byte, error) {
	path := fmt.Sprintf("templates/%s/%s", subsystemType, fileName)
	data, err := embeddedFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("template %s/%s not found: %w", subsystemType, fileName, err)
	}
	return data, nil
}

func (e *EmbeddedTemplateSource) ListTypes() []string {
	return validTypes
}
