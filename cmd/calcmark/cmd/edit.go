package cmd

import (
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit [file.cm]",
	Short: "Open the CalcMark document editor",
	Long: `Open the split-pane document editor for working with CalcMark files.

The editor shows source on the left and computed results on the right.

Examples:
  cm edit                   Open editor with file picker
  cm edit budget.cm         Open specific file in editor`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			runEdit(args[0])
		} else {
			runEdit("")
		}
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
