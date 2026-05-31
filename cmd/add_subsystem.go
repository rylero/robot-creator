package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/rylero/robot-creator/internal/config"
	"github.com/rylero/robot-creator/internal/generator"
	"github.com/rylero/robot-creator/internal/injector"
	tmpl "github.com/rylero/robot-creator/internal/template"
)

var subsystemType string

var addSubsystemCmd = &cobra.Command{
	Use:   "subsystem <Name>",
	Short: "Generate an AdvantageKit subsystem",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddSubsystem,
}

func init() {
	addCmd.AddCommand(addSubsystemCmd)
	addSubsystemCmd.Flags().StringVarP(&subsystemType, "type", "t", "", "subsystem type (flywheel|pivot|roller|arm|elevator|turret|generic)")
	addSubsystemCmd.MarkFlagRequired("type")
}

func runAddSubsystem(cmd *cobra.Command, args []string) error {
	name := args[0]

	if !tmpl.IsValidType(subsystemType) {
		return fmt.Errorf("unknown type %q. Run 'robot-creator list types' for valid types", subsystemType)
	}

	root, err := config.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if cfg.HasSubsystem(name) {
		return fmt.Errorf("%s already exists. Delete it from robot-creator.yaml to regenerate", name)
	}

	ctx := generator.SubsystemContext{
		Name:       name,
		NameLower:  strings.ToLower(name),
		Type:       subsystemType,
		Package:    cfg.Package,
		TeamNumber: cfg.Team,
	}

	gen := generator.New(tmpl.NewEmbeddedSource())
	fmt.Printf("Generating %s subsystem (type: %s)...\n", name, subsystemType)
	if err := gen.GenerateSubsystem(ctx, root); err != nil {
		return err
	}

	pkgPath := strings.ReplaceAll(cfg.Package, ".", "/")
	rcPath := filepath.Join(root, "src", "main", "java", pkgPath, "RobotContainer.java")
	if _, err := os.Stat(rcPath); err == nil {
		inj := injector.New(rcPath)
		si := injector.SubsystemInjection{
			Name:      name,
			NameLower: strings.ToLower(name),
			Package:   cfg.Package,
		}
		if err := inj.Inject(si); err != nil {
			fmt.Printf("\nWarning: could not auto-inject into RobotContainer.java: %v\n", err)
			fmt.Println("Add these lines manually:")
			printManualInstructions(si)
		} else {
			fmt.Println("  injected into RobotContainer.java")
		}
	}

	cfg.Subsystems = append(cfg.Subsystems, config.Subsystem{Name: name, Type: subsystemType})
	if err := config.Save(root, cfg); err != nil {
		return fmt.Errorf("updating robot-creator.yaml: %w", err)
	}

	fmt.Printf("\nDone! %s subsystem created.\n", name)
	return nil
}

func printManualInstructions(s injector.SubsystemInjection) {
	pkg := s.Package + ".subsystems." + s.NameLower
	fmt.Printf("// Imports:\n")
	fmt.Printf("import %s.%s;\n", pkg, s.Name)
	fmt.Printf("import %s.%sIO;\n", pkg, s.Name)
	fmt.Printf("import %s.%sIOTalonFX;\n", pkg, s.Name)
	fmt.Printf("import %s.%sIOSim;\n\n", pkg, s.Name)
	fmt.Printf("// Field:\nprivate final %s %s;\n\n", s.Name, s.NameLower)
	fmt.Printf("// case REAL:\n  %s = new %s(new %sIOTalonFX());\n\n", s.NameLower, s.Name, s.Name)
	fmt.Printf("// case SIM:\n  %s = new %s(new %sIOSim());\n\n", s.NameLower, s.Name, s.Name)
	fmt.Printf("// default:\n  %s = new %s(new %sIO() {});\n", s.NameLower, s.Name, s.Name)
}
