package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the robot-creator version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("robot-creator", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
