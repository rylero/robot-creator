package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rylero/robot-creator/internal/config"
	"github.com/rylero/robot-creator/internal/generator"
	"github.com/rylero/robot-creator/internal/injector"
	tmpl "github.com/rylero/robot-creator/internal/template"
	"github.com/spf13/cobra"
)

var subsystemType string
var motorCount int
var aligned bool

var addSubsystemCmd = &cobra.Command{
	Use:   "subsystem <Name>",
	Short: "Generate an AdvantageKit subsystem",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddSubsystem,
}

func init() {
	addCmd.AddCommand(addSubsystemCmd)
	addSubsystemCmd.Flags().StringVarP(&subsystemType, "type", "t", "", "subsystem type (flywheel|pivot|roller|arm|elevator|turret|generic|manipulator)")
	addSubsystemCmd.Flags().IntVar(&motorCount, "motors", 0, "number of motors for arm/elevator/manipulator (default: 2 for arm/elevator, 1 for manipulator)")
	addSubsystemCmd.Flags().BoolVar(&aligned, "aligned", true, "followers are mechanically aligned to leader (arm/elevator/manipulator with 2+ motors)")
	addSubsystemCmd.MarkFlagRequired("type")
}

func resolveMotorCount(t string, flag int) int {
	if flag > 0 {
		return flag
	}
	switch t {
	case "arm", "elevator":
		return 2
	default:
		return 1
	}
}

func buildFollowers(n int) []generator.FollowerInfo {
	if n <= 1 {
		return nil
	}
	followers := make([]generator.FollowerInfo, 0, n-1)
	for i := 2; i <= n; i++ {
		followers = append(followers, generator.FollowerInfo{Index: i, DefaultID: i - 1})
	}
	return followers
}

func runAddSubsystem(cmd *cobra.Command, args []string) error {
	if err := validateSubsystemName(args[0]); err != nil {
		return err
	}
	name := strings.ToUpper(args[0][:1]) + args[0][1:]

	if !tmpl.IsValidType(subsystemType) {
		return fmt.Errorf("unknown type %q. Run 'robot-creator list types' for valid types", subsystemType)
	}
	if motorCount < 0 {
		return fmt.Errorf("--motors must be >= 1")
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

	motors := resolveMotorCount(subsystemType, motorCount)
	ctx := generator.SubsystemContext{
		Name:       name,
		NameLower:  strings.ToLower(name),
		Type:       subsystemType,
		Package:    cfg.Package,
		TeamNumber: cfg.Team,
		MotorCount: motors,
		Aligned:    aligned,
		Followers:  buildFollowers(motors),
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

	sub := config.Subsystem{Name: name, Type: subsystemType}
	switch subsystemType {
	case "arm", "elevator", "manipulator":
		sub.Motors = motors
		sub.Aligned = aligned
	}
	cfg.Subsystems = append(cfg.Subsystems, sub)
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
