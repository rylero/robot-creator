package template

import (
	"fmt"
	"os"
	"path/filepath"
)

type LocalTemplateSource struct {
	dir string
}

func NewLocalSource(dir string) *LocalTemplateSource {
	return &LocalTemplateSource{dir: dir}
}

func (l *LocalTemplateSource) GetTemplate(subsystemType, fileName string) ([]byte, error) {
	path := filepath.Join(l.dir, subsystemType, fileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("local template %s/%s not found: %w", subsystemType, fileName, err)
	}
	return data, nil
}

func (l *LocalTemplateSource) ListTypes() []string {
	return validTypes
}
