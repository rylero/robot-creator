package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rylero/robot-creator/internal/config"
	"github.com/spf13/cobra"
)

var removeSuperstructureCmd = &cobra.Command{
	Use:   "superstructure",
	Short: "Remove the generated superstructure",
	Args:  cobra.NoArgs,
	RunE:  runRemoveSuperstructure,
}

func init() {
	removeCmd.AddCommand(removeSuperstructureCmd)
}

func runRemoveSuperstructure(cmd *cobra.Command, args []string) error {
	root, err := config.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if !cfg.Superstructure {
		return fmt.Errorf("no superstructure found in robot-creator.yaml")
	}

	pkgPath := strings.ReplaceAll(cfg.Package, ".", "/")
	superDir := filepath.Join(root, "src", "main", "java", pkgPath, "subsystems", "superstructure")
	if err := os.RemoveAll(superDir); err != nil {
		return fmt.Errorf("deleting superstructure files: %w", err)
	}
	fmt.Printf("Deleted %s\n", superDir)

	cfg.Superstructure = false
	if err := config.Save(root, cfg); err != nil {
		return fmt.Errorf("updating robot-creator.yaml: %w", err)
	}

	fmt.Println("Done! Superstructure removed.")
	return nil
}
