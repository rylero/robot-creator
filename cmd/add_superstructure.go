package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/ced4rtree/robot-creator/internal/config"
	tmplpkg "github.com/ced4rtree/robot-creator/internal/template"
)

var addSuperstructureCmd = &cobra.Command{
	Use:   "superstructure",
	Short: "Generate a Superstructure state-machine scaffold",
	RunE:  runAddSuperstructure,
}

func init() {
	addCmd.AddCommand(addSuperstructureCmd)
}

func runAddSuperstructure(cmd *cobra.Command, args []string) error {
	root, err := config.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if cfg.Superstructure {
		return fmt.Errorf("superstructure already generated. Remove 'superstructure: true' from robot-creator.yaml to regenerate")
	}

	pkgPath := strings.ReplaceAll(cfg.Package, ".", "/")
	outDir := filepath.Join(root, "src", "main", "java", pkgPath, "subsystems", "superstructure")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	source := tmplpkg.NewEmbeddedSource()
	raw, err := source.GetTemplate("superstructure", "Superstructure.java.tmpl")
	if err != nil {
		return err
	}

	type ctx struct{ Package string }
	t, err := template.New("superstructure").Parse(string(raw))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx{Package: cfg.Package}); err != nil {
		return err
	}

	outPath := filepath.Join(outDir, "Superstructure.java")
	if err := os.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
		return err
	}

	cfg.Superstructure = true
	if err := config.Save(root, cfg); err != nil {
		return err
	}

	fmt.Printf("Created %s\n", outPath)
	return nil
}
