// Package cmd provides the CLI commands for Humantime.
//
// This software is a derivative work based on Zeit (https://github.com/mrusme/zeit)
// Original work copyright (c) マリウス (mrusme)
// Modifications copyright (c) Manav Panchal
//
// Licensed under the SEGV License, Version 1.0
// See LICENSE file for full license text.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command.
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for humantime.

To load completions:

Bash:
  $ source <(humantime completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ humantime completion bash > /etc/bash_completion.d/humantime
  # macOS:
  $ humantime completion bash > $(brew --prefix)/etc/bash_completion.d/humantime

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ humantime completion zsh > "${fpath[1]}/_humantime"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ humantime completion fish | source

  # To load completions for each session, execute once:
  $ humantime completion fish > ~/.config/fish/completions/humantime.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
