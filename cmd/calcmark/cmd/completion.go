package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:     "completion [bash|zsh|fish|powershell]",
	Aliases: []string{"completions"},
	Short:   "Generate shell completion scripts",
	Long: `Generate shell completion scripts for CalcMark.

To load completions:

Bash:
  # Add to ~/.bashrc:
  eval "$(cm completion bash)"

Zsh:
  # Add to ~/.zshrc (after compinit):
  autoload -Uz compinit && compinit
  eval "$(cm completion zsh)"

Fish:
  $ cm completion fish > ~/.config/fish/completions/cm.fish

PowerShell:
  # Add to your PowerShell profile:
  cm completion powershell | Out-String | Invoke-Expression
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
