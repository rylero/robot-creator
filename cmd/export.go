package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rylero/robot-creator/internal/config"
	tmplpkg "github.com/rylero/robot-creator/internal/template"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export resources from robot-creator",
}

var exportTemplatesCmd = &cobra.Command{
	Use:   "templates [dir]",
	Short: "Export embedded templates to a local directory for customization",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runExportTemplates,
}

func init() {
	exportCmd.AddCommand(exportTemplatesCmd)
	rootCmd.AddCommand(exportCmd)
}

func runExportTemplates(cmd *cobra.Command, args []string) error {
	outDir := "templates"
	if len(args) == 1 {
		outDir = args[0]
	}

	count := 0
	err := fs.WalkDir(*tmplpkg.EmbeddedFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel("templates", path)
		dest := filepath.Join(outDir, rel)

		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		data, err := tmplpkg.EmbeddedFS.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dest, data, 0644); err != nil {
			return err
		}
		fmt.Printf("  %s\n", dest)
		count++
		return nil
	})
	if err != nil {
		return fmt.Errorf("exporting templates: %w", err)
	}

	fmt.Printf("\nExported %d template files to %s/\n", count, outDir)

	root, rootErr := config.FindRoot()
	if rootErr == nil {
		cfg, cfgErr := config.Load(root)
		if cfgErr == nil && cfg.TemplatesDir == "" {
			rel, relErr := filepath.Rel(root, outDir)
			if relErr != nil {
				rel = outDir
			}
			cfg.TemplatesDir = rel
			if saveErr := config.Save(root, cfg); saveErr == nil {
				fmt.Printf("Set templates_dir: %s in robot-creator.yaml\n", rel)
				fmt.Println("robot-creator will now use your local templates instead of the built-in ones.")
			}
		}
	}

	return nil
}
