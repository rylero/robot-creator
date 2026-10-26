package cmd

import (
	"fmt"
	"strings"

	"github.com/rylero/robot-creator/v3/internal/config"
	"github.com/rylero/robot-creator/v3/internal/generator"
	"github.com/rylero/robot-creator/v3/internal/injector"
	tmpl "github.com/rylero/robot-creator/v3/internal/template"
	"github.com/spf13/cobra"
)

var subsystemType string
var motorCount int
var aligned bool
var leaderID int

var addSubsystemCmd = &cobra.Command{
	Use:   "subsystem <Name>",
	Short: "Generate an AdvantageKit Commands v3 mechanism",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddSubsystem,
}

func init() {
	addCmd.AddCommand(addSubsystemCmd)
	addSubsystemCmd.Flags().StringVarP(&subsystemType, "type", "t", "", "subsystem type (flywheel|pivot|roller|arm|elevator|turret|generic|manipulator)")
	addSubsystemCmd.Flags().IntVar(&motorCount, "motors", 0, "number of TalonFX motors (default: 2 for arm/elevator, 1 otherwise)")
	addSubsystemCmd.Flags().BoolVar(&aligned, "aligned", true, "followers are mechanically aligned to the leader (with 2+ motors)")
	addSubsystemCmd.Flags().IntVar(&leaderID, "id", 0, "leader/motor CAN ID (followers get consecutive IDs)")
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

func buildFollowers(n int, leaderID int) []generator.FollowerInfo {
	if n <= 1 {
		return nil
	}
	followers := make([]generator.FollowerInfo, 0, n-1)
	for i := 2; i <= n; i++ {
		followers = append(followers, generator.FollowerInfo{Index: i, DefaultID: leaderID + (i - 1)})
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

	root, cfg, err := loadProject()
	if err != nil {
		return err
	}
	if cfg.HasSubsystem(name) {
		return fmt.Errorf("%s already exists. Run 'robot-creator remove subsystem %s' to regenerate it", name, name)
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
		LeaderID:   leaderID,
		Followers:  buildFollowers(motors, leaderID),
	}

	gen := generator.New(templateSource(root, cfg))
	fmt.Printf("Generating %s mechanism (type: %s)...\n", name, subsystemType)
	if err := gen.GenerateSubsystem(ctx, root); err != nil {
		return err
	}
	warnings, err := gen.EnsureSupportFiles(cfg.Package, root)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		fmt.Printf("\nWarning: %s\n", w)
	}

	si := injector.SubsystemInjection{Name: name, NameLower: ctx.NameLower, Package: cfg.Package}
	inj, hasContainer, hasRobot := projectInjector(root, cfg)
	if hasContainer {
		if err := inj.Inject(si); err != nil {
			fmt.Printf("\nWarning: could not auto-inject into RobotContainer.java: %v\n", err)
			fmt.Println("Add these lines manually:")
			printManualInstructions(si)
		} else {
			fmt.Println("  injected into RobotContainer.java")
		}
	}
	if hasRobot {
		if err := inj.InjectPeriodic(ctx.NameLower); err != nil {
			fmt.Printf("\nWarning: could not auto-inject into Robot.java: %v\n", err)
			fmt.Printf("Call container.%s.periodic(); in robotPeriodic(), before Scheduler.getDefault().run()\n", ctx.NameLower)
		} else {
			fmt.Println("  injected periodic() into Robot.java")
		}
	}

	sub := config.Subsystem{Name: name, Type: subsystemType}
	if motors > 1 {
		sub.Motors = motors
		sub.Aligned = aligned
	}
	cfg.Subsystems = append(cfg.Subsystems, sub)
	if err := config.Save(root, cfg); err != nil {
		return fmt.Errorf("updating robot-creator.yaml: %w", err)
	}

	fmt.Printf("\nDone! %s mechanism created.\n", name)
	if cfg.Superstructure {
		fmt.Printf("Note: the superstructure was generated before %s. Pass it to Superstructure's constructor by hand.\n", name)
	}
	return nil
}

func printManualInstructions(s injector.SubsystemInjection) {
	pkg := s.Package + ".subsystems." + s.NameLower
	fmt.Printf("// Imports:\n")
	fmt.Printf("import %s.%s;\n", pkg, s.Name)
	fmt.Printf("import %s.%sIO;\n", pkg, s.Name)
	fmt.Printf("import %s.%sIOTalonFX;\n", pkg, s.Name)
	fmt.Printf("import %s.%sIOSim;\n\n", pkg, s.Name)
	fmt.Printf("// Field:\npublic final %s %s;\n\n", s.Name, s.NameLower)
	fmt.Printf("// case REAL:\n  %s = new %s(new %sIOTalonFX());\n\n", s.NameLower, s.Name, s.Name)
	fmt.Printf("// case SIM:\n  %s = new %s(new %sIOSim());\n\n", s.NameLower, s.Name, s.Name)
	fmt.Printf("// default:\n  %s = new %s(new %sIO() {});\n", s.NameLower, s.Name, s.Name)
}
