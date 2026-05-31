package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/rylero/robot-creator/internal/config"
	"github.com/rylero/robot-creator/internal/injector"
)

var removeSubsystemCmd = &cobra.Command{
	Use:   "subsystem <Name>",
	Short: "Remove a generated subsystem",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemoveSubsystem,
}

func init() {
	removeCmd.AddCommand(removeSubsystemCmd)
}

func runRemoveSubsystem(cmd *cobra.Command, args []string) error {
	if err := validateSubsystemName(args[0]); err != nil {
		return err
	}
	name := strings.ToUpper(args[0][:1]) + args[0][1:]

	root, err := config.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if !cfg.HasSubsystem(name) {
		return fmt.Errorf("%s not found in robot-creator.yaml", name)
	}

	// Delete subsystem directory
	pkgPath := strings.ReplaceAll(cfg.Package, ".", "/")
	subsystemDir := filepath.Join(root, "src", "main", "java", pkgPath, "subsystems", strings.ToLower(name))
	if err := os.RemoveAll(subsystemDir); err != nil {
		return fmt.Errorf("deleting subsystem files: %w", err)
	}
	fmt.Printf("Deleted %s\n", subsystemDir)

	// Best-effort un-injection from RobotContainer
	rcPath := filepath.Join(root, "src", "main", "java", pkgPath, "RobotContainer.java")
	if _, err := os.Stat(rcPath); err == nil {
		inj := injector.New(rcPath)
		si := injector.SubsystemInjection{
			Name:      name,
			NameLower: strings.ToLower(name),
			Package:   cfg.Package,
		}
		if err := inj.Eject(si); err != nil {
			fmt.Printf("\nWarning: could not auto-remove from RobotContainer.java: %v\n", err)
			fmt.Println("Remove these lines manually:")
			printManualRemoveInstructions(si)
		} else {
			fmt.Println("  removed from RobotContainer.java")
		}
	}

	// Remove from config
	updated := cfg.Subsystems[:0]
	for _, s := range cfg.Subsystems {
		if s.Name != name {
			updated = append(updated, s)
		}
	}
	cfg.Subsystems = updated
	if err := config.Save(root, cfg); err != nil {
		return fmt.Errorf("updating robot-creator.yaml: %w", err)
	}

	fmt.Printf("\nDone! %s subsystem removed.\n", name)
	return nil
}

func printManualRemoveInstructions(s injector.SubsystemInjection) {
	pkg := s.Package + ".subsystems." + s.NameLower
	fmt.Printf("// Remove imports:\n")
	fmt.Printf("import %s.%s;\n", pkg, s.Name)
	fmt.Printf("import %s.%sIO;\n", pkg, s.Name)
	fmt.Printf("import %s.%sIOTalonFX;\n", pkg, s.Name)
	fmt.Printf("import %s.%sIOSim;\n\n", pkg, s.Name)
	fmt.Printf("// Remove field:\nprivate final %s %s;\n\n", s.Name, s.NameLower)
	fmt.Printf("// Remove from each switch case:\n%s = new %s(new %sIOTalonFX());\n", s.NameLower, s.Name, s.Name)
	fmt.Printf("%s = new %s(new %sIOSim());\n", s.NameLower, s.Name, s.Name)
	fmt.Printf("%s = new %s(new %sIO() {});\n", s.NameLower, s.Name, s.Name)
}
