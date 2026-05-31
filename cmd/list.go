package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	tmpl "github.com/ced4rtree/robot-creator/internal/template"
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
			"flywheel": "Single motor, velocity control (MotionMagicVelocityVoltage)",
			"pivot":    "Single motor, position control with soft limits",
			"roller":   "Single motor, voltage/on-off control with velocity monitoring",
			"arm":      "Dual motor + follower, position control with soft limits",
			"elevator": "Dual motor + follower, linear position control in meters",
			"turret":   "Single motor, continuous rotation with ContinuousWrap",
			"generic":  "Minimal scaffold — IO interface + voltage setpoint only",
		}
		source := tmpl.NewEmbeddedSource()
		fmt.Println("Supported subsystem types:")
		for _, t := range source.ListTypes() {
			fmt.Printf("  %-10s %s\n", t, descriptions[t])
		}
	},
}

func init() {
	listCmd.AddCommand(listTypesCmd)
}
