package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	tmplpkg "github.com/rylero/robot-creator/v3/internal/template"
)

type FollowerInfo struct {
	Index     int // 2, 3, 4... (variable name suffix)
	DefaultID int // 1, 2, 3... (default CAN ID)
}

type SubsystemContext struct {
	Name       string
	NameLower  string
	Type       string
	Package    string
	TeamNumber int
	MotorCount int
	Aligned    bool
	LeaderID   int
	Followers  []FollowerInfo
}

var templateFiles = []string{
	"Subsystem.java.tmpl",
	"SubsystemIO.java.tmpl",
	"SubsystemIOTalonFX.java.tmpl",
	"SubsystemIOSim.java.tmpl",
	"SubsystemConstants.java.tmpl",
}

// supportFile is a helper class the generated subsystems depend on. It is written only when
// missing; an existing file is kept but must still declare Symbol.
type supportFile struct {
	Template string
	SubPkg   string
	Symbol   string
}

var supportFiles = []supportFile{
	{"PhoenixUtil.java.tmpl", "util", "applySafetyLimits"},
	{"Measures.java.tmpl", "util", "finiteVoltage"},
	{"VelocityPlant.java.tmpl", "sim", "setVelocityRadPerSec"},
}

type Generator struct {
	Source tmplpkg.TemplateSource
}

func New(source tmplpkg.TemplateSource) *Generator {
	return &Generator{Source: source}
}

// JavaDir returns the source directory for a Java package inside the project.
func JavaDir(projectRoot, pkg string) string {
	return filepath.Join(projectRoot, "src", "main", "java", strings.ReplaceAll(pkg, ".", "/"))
}

func (g *Generator) GenerateSubsystem(ctx SubsystemContext, projectRoot string) error {
	outDir := filepath.Join(JavaDir(projectRoot, ctx.Package), "subsystems", ctx.NameLower)
	if _, err := os.Stat(outDir); err == nil {
		return fmt.Errorf("%s already exists; remove it first or pick another name", outDir)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	for _, tmplFile := range templateFiles {
		raw, err := g.Source.GetTemplate(ctx.Type, tmplFile)
		if err != nil {
			return err
		}
		outFile := strings.TrimSuffix(strings.ReplaceAll(tmplFile, "Subsystem", ctx.Name), ".tmpl")
		if err := render(tmplFile, raw, ctx, filepath.Join(outDir, outFile)); err != nil {
			return err
		}
		fmt.Printf("  created %s/%s\n", ctx.NameLower, outFile)
	}
	return nil
}

// EnsureSupportFiles writes the helper classes the templates use if the project lacks them.
// It returns warnings for existing helpers that look incompatible.
func (g *Generator) EnsureSupportFiles(pkg, projectRoot string) ([]string, error) {
	var warnings []string
	for _, f := range supportFiles {
		className := strings.TrimSuffix(f.Template, ".java.tmpl")
		outPath := filepath.Join(JavaDir(projectRoot, pkg), f.SubPkg, className+".java")
		if existing, err := os.ReadFile(outPath); err == nil {
			if !strings.Contains(string(existing), f.Symbol) {
				warnings = append(warnings, fmt.Sprintf(
					"%s.%s.%s exists but has no %s(); generated code expects it", pkg, f.SubPkg, className, f.Symbol))
			}
			continue
		}

		raw, err := g.Source.GetTemplate("support", f.Template)
		if err != nil {
			// Templates exported before support files existed: fall back to the built-ins.
			raw, err = tmplpkg.NewEmbeddedSource().GetTemplate("support", f.Template)
			if err != nil {
				return warnings, err
			}
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return warnings, err
		}
		if err := render(f.Template, raw, struct{ Package string }{pkg}, outPath); err != nil {
			return warnings, err
		}
		fmt.Printf("  created %s/%s.java\n", f.SubPkg, className)
	}
	return warnings, nil
}

func render(name string, raw []byte, data any, outPath string) error {
	t, err := template.New(name).Parse(string(raw))
	if err != nil {
		return fmt.Errorf("parsing template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template %s: %w", name, err)
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", filepath.Base(outPath), err)
	}
	return nil
}
