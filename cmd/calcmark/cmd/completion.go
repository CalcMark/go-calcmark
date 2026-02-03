package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for CalcMark.

To load completions:

Bash:
  $ source <(cm completion bash)
  # Or add to ~/.bashrc for persistence:
  $ cm completion bash >> ~/.bashrc

Zsh:
  $ cm completion zsh > "${fpath[1]}/_cm"
  # Then restart shell or run: compinit

Fish:
  $ cm completion fish > ~/.config/fish/completions/cm.fish

PowerShell:
  PS> cm completion powershell | Out-String | Invoke-Expression
  # Or add to your profile for persistence
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
