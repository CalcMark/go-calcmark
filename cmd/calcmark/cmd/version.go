package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information - set by main package from ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("CalcMark %s\n", Version)
		if BuildTime != "unknown" {
			fmt.Printf("  built: %s\n", BuildTime)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
