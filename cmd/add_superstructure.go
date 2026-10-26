package cmd

import (
	"fmt"
	"strings"

	"github.com/rylero/robot-creator/v3/internal/config"
	"github.com/rylero/robot-creator/v3/internal/generator"
	"github.com/spf13/cobra"
)

var wantedStatesFlag string
var activeStatesFlag string

var addSuperstructureCmd = &cobra.Command{
	Use:   "superstructure",
	Short: "Generate a Superstructure state-machine coordinator",
	RunE:  runAddSuperstructure,
}

func init() {
	addCmd.AddCommand(addSuperstructureCmd)
	addSuperstructureCmd.Flags().StringVar(&wantedStatesFlag, "wanted", "IDLE", "comma-separated Wanted enum values")
	addSuperstructureCmd.Flags().StringVar(&activeStatesFlag, "active", "IDLE", "comma-separated Current enum values")
}

func parseStates(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			result = append(result, strings.ToUpper(s))
		}
	}
	return result
}

func buildTransitions(wanted, active []string) []generator.Transition {
	transitions := make([]generator.Transition, len(wanted))
	for i, w := range wanted {
		a := active[0]
		if i < len(active) {
			a = active[i]
		}
		transitions[i] = generator.Transition{Wanted: w, Active: a}
	}
	return transitions
}

func runAddSuperstructure(cmd *cobra.Command, args []string) error {
	root, cfg, err := loadProject()
	if err != nil {
		return err
	}
	if cfg.Superstructure {
		return fmt.Errorf("superstructure already generated. Run 'robot-creator remove superstructure' to regenerate it")
	}

	wanted := parseStates(wantedStatesFlag)
	active := parseStates(activeStatesFlag)
	if len(wanted) == 0 {
		return fmt.Errorf("--wanted must contain at least one state")
	}
	if len(active) == 0 {
		return fmt.Errorf("--active must contain at least one state")
	}

	ctx := generator.SuperstructureContext{
		Package:       cfg.Package,
		WantedStates:  wanted,
		ActiveStates:  active,
		Transitions:   buildTransitions(wanted, active),
		DefaultActive: active[0],
	}
	var mechanismNames []string
	for _, s := range cfg.Subsystems {
		m := generator.Mechanism{Name: s.Name, NameLower: strings.ToLower(s.Name)}
		ctx.Mechanisms = append(ctx.Mechanisms, m)
		mechanismNames = append(mechanismNames, m.NameLower)
	}

	outPath, err := generator.New(templateSource(root, cfg)).GenerateSuperstructure(ctx, root)
	if err != nil {
		return err
	}
	fmt.Printf("Created %s\n", outPath)

	inj, hasContainer, hasRobot := projectInjector(root, cfg)
	if hasContainer {
		if err := inj.InjectSuperstructure(cfg.Package, mechanismNames); err != nil {
			fmt.Printf("\nWarning: could not auto-inject into RobotContainer.java: %v\n", err)
			fmt.Printf("Add 'public final Superstructure superstructure;' and, after the currentMode switch:\n")
			fmt.Printf("  superstructure = new Superstructure(%s);\n", strings.Join(mechanismNames, ", "))
		} else {
			fmt.Println("  injected into RobotContainer.java")
		}
	}
	if hasRobot {
		if err := inj.InjectPeriodic("superstructure"); err != nil {
			fmt.Printf("\nWarning: could not auto-inject into Robot.java: %v\n", err)
			fmt.Println("Call container.superstructure.periodic(); after the mechanisms' periodic() calls")
		} else {
			fmt.Println("  injected periodic() into Robot.java")
		}
	}

	cfg.Superstructure = true
	return config.Save(root, cfg)
}
