package cmd

import (
	"github.com/CalcMark/go-calcmark/v2/lsp"
	"github.com/spf13/cobra"
)

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Start the CalcMark language server",
	Long: `Start the CalcMark language server over stdio.

The LSP server provides diagnostics, autocomplete, hover, go-to-definition,
and document symbols for .cm files. Editor extensions spawn this command
to provide language intelligence.

Example:
  cm lsp     Start the language server (typically called by an editor extension)`,
	Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		return runLsp()
	},
}

func init() {
	rootCmd.AddCommand(lspCmd)
}

func runLsp() error {
	server := lsp.NewServer()
	return server.RunStdio()
}
