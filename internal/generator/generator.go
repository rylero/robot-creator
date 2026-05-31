package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	tmplpkg "github.com/ced4rtree/robot-creator/internal/template"
)

type SubsystemContext struct {
	Name       string
	NameLower  string
	Type       string
	Package    string
	TeamNumber int
}

var templateFiles = []string{
	"Subsystem.java.tmpl",
	"SubsystemIO.java.tmpl",
	"SubsystemIOTalonFX.java.tmpl",
	"SubsystemIOSim.java.tmpl",
	"SubsystemConstants.java.tmpl",
}

type Generator struct {
	Source tmplpkg.TemplateSource
}

func New(source tmplpkg.TemplateSource) *Generator {
	return &Generator{Source: source}
}

func (g *Generator) GenerateSubsystem(ctx SubsystemContext, projectRoot string) error {
	pkgPath := strings.ReplaceAll(ctx.Package, ".", "/")
	outDir := filepath.Join(projectRoot, "src", "main", "java", pkgPath, "subsystems", ctx.NameLower)

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	for _, tmplFile := range templateFiles {
		raw, err := g.Source.GetTemplate(ctx.Type, tmplFile)
		if err != nil {
			return err
		}

		t, err := template.New(tmplFile).Parse(string(raw))
		if err != nil {
			return fmt.Errorf("parsing template %s: %w", tmplFile, err)
		}

		var buf bytes.Buffer
		if err := t.Execute(&buf, ctx); err != nil {
			return fmt.Errorf("executing template %s: %w", tmplFile, err)
		}

		outFile := strings.ReplaceAll(tmplFile, "Subsystem", ctx.Name)
		outFile = strings.TrimSuffix(outFile, ".tmpl")

		if err := os.WriteFile(filepath.Join(outDir, outFile), buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", outFile, err)
		}
		fmt.Printf("  created %s/%s\n", ctx.NameLower, outFile)
	}
	return nil
}
