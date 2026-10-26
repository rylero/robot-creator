package cmd

import (
	"fmt"

	"github.com/rylero/robot-creator/v3/internal/config"
	tmpl "github.com/rylero/robot-creator/v3/internal/template"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available templates and types",
}

var listTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "List supported subsystem types",
	Run: func(cmd *cobra.Command, args []string) {
		descriptions := map[string]string{
			"flywheel":    "Velocity control (VelocityVoltage)",
			"pivot":       "Angle position control (MotionMagicVoltage) with soft limits",
			"roller":      "Voltage control with stall detection",
			"arm":         "Angle position control with gravity feedforward; 2 motors by default",
			"elevator":    "Linear position control in meters; 2 motors by default",
			"turret":      "Angle position control with ContinuousWrap",
			"generic":     "Minimal scaffold: voltage control only",
			"manipulator": "Position control in rotations with soft limits",
		}
		source := tmpl.NewEmbeddedSource()
		fmt.Println("Supported subsystem types:")
		for _, t := range source.ListTypes() {
			fmt.Printf("  %-12s %s\n", t, descriptions[t])
		}
	},
}

var listSubsystemsCmd = &cobra.Command{
	Use:   "subsystems",
	Short: "List subsystems in the current project",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.FindRoot()
		if err != nil {
			return err
		}
		cfg, err := config.Load(root)
		if err != nil {
			return err
		}
		if len(cfg.Subsystems) == 0 && !cfg.Superstructure {
			fmt.Println("No subsystems added yet. Use 'robot-creator add subsystem <Name> --type <type>'")
			return nil
		}
		fmt.Printf("Project: team %d  package: %s\n\n", cfg.Team, cfg.Package)
		if len(cfg.Subsystems) > 0 {
			fmt.Println("Subsystems:")
			for _, s := range cfg.Subsystems {
				fmt.Printf("  %-20s %s\n", s.Name, s.Type)
			}
		}
		if cfg.Superstructure {
			fmt.Println("\nSuperstructure: yes")
		}
		return nil
	},
}

func init() {
	listCmd.AddCommand(listTypesCmd)
	listCmd.AddCommand(listSubsystemsCmd)
}
