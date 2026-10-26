package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rylero/robot-creator/v3/internal/config"
	"github.com/rylero/robot-creator/v3/internal/generator"
	"github.com/rylero/robot-creator/v3/internal/injector"
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
	root, cfg, err := loadProject()
	if err != nil {
		return err
	}
	if !cfg.Superstructure {
		return fmt.Errorf("no superstructure found in robot-creator.yaml")
	}

	superDir := filepath.Join(generator.JavaDir(root, cfg.Package), "subsystems", "superstructure")
	if err := os.RemoveAll(superDir); err != nil {
		return fmt.Errorf("deleting superstructure files: %w", err)
	}
	fmt.Printf("Deleted %s\n", superDir)

	inj, hasContainer, hasRobot := projectInjector(root, cfg)
	if hasContainer {
		si := injector.SubsystemInjection{Name: "Superstructure", NameLower: "superstructure", Package: cfg.Package}
		if err := inj.Eject(si); err != nil {
			fmt.Printf("\nWarning: could not auto-remove from RobotContainer.java: %v\n", err)
		} else {
			fmt.Println("  removed from RobotContainer.java")
		}
	}
	if hasRobot {
		if err := inj.EjectPeriodic("superstructure"); err != nil {
			fmt.Printf("\nWarning: could not auto-remove from Robot.java: %v\n", err)
		} else {
			fmt.Println("  removed periodic() from Robot.java")
		}
	}

	cfg.Superstructure = false
	if err := config.Save(root, cfg); err != nil {
		return fmt.Errorf("updating robot-creator.yaml: %w", err)
	}

	fmt.Println("Done! Superstructure removed.")
	return nil
}
