package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var Version = "dev"

func resolveVersion() string {
	if Version != "dev" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return Version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the robot-creator version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("robot-creator", resolveVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
