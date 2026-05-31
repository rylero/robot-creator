package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/rylero/robot-creator/internal/config"
)

var (
	projectName string
	teamNumber  int
	repoURL     string
	packageName string
	noClone     bool
)

const defaultRepo = "https://github.com/rylero/akit-robot-template"

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Clone the AKit starter repo and initialize robot-creator.yaml",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().StringVarP(&projectName, "name", "n", "", "project directory name (clones into this dir; omit with --no-clone)")
	initCmd.Flags().IntVarP(&teamNumber, "team", "t", 0, "FRC team number (required)")
	initCmd.Flags().StringVar(&repoURL, "repo", defaultRepo, "AKit template repo URL")
	initCmd.Flags().StringVar(&packageName, "package", "frc.robot", "Java package name")
	initCmd.Flags().BoolVar(&noClone, "no-clone", false, "skip git clone; write robot-creator.yaml into the current directory")
	initCmd.MarkFlagRequired("team")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if noClone {
		if _, err := os.Stat(config.FileName); err == nil {
			return fmt.Errorf("robot-creator.yaml already exists in current directory")
		}
		cfg := &config.Config{Team: teamNumber, Package: packageName}
		if err := config.Save(".", cfg); err != nil {
			return fmt.Errorf("writing robot-creator.yaml: %w", err)
		}
		fmt.Printf("Initialized robot-creator.yaml (no clone)\nRun: robot-creator add subsystem <Name> --type <type>\n")
		return nil
	}

	if projectName == "" {
		return fmt.Errorf("--name is required unless --no-clone is set")
	}
	if _, err := os.Stat(projectName); err == nil {
		return fmt.Errorf("directory %q already exists", projectName)
	}

	fmt.Printf("Cloning %s into %s/...\n", repoURL, projectName)
	gitCmd := exec.Command("git", "clone", repoURL, projectName)
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr
	if err := gitCmd.Run(); err != nil {
		os.RemoveAll(projectName)
		return fmt.Errorf("git clone failed: %w", err)
	}

	cfg := &config.Config{Team: teamNumber, Package: packageName}
	if err := config.Save(projectName, cfg); err != nil {
		return fmt.Errorf("writing robot-creator.yaml: %w", err)
	}

	fmt.Printf("\nProject %s created!\ncd %s && robot-creator add subsystem <Name> --type <type>\n", projectName, projectName)
	return nil
}
