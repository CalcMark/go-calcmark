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
		fmt.Printf("  ♪\n")
		fmt.Printf(" ('>    CalcMark %s\n", Version)
		if BuildTime != "unknown" {
			fmt.Printf(" /V\\      built: %s\n", BuildTime)
		} else {
			fmt.Printf(" /V\\\n")
		}
		fmt.Printf("(| |)\n")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
