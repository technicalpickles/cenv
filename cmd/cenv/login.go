package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/env"
)

var loginCmd = &cobra.Command{
	Use:   "login <name>",
	Short: "Open Claude in an environment so you can run /login",
	Long: `Opens the Claude Code REPL with CLAUDE_CONFIG_DIR pointed at the named
environment. Type /login inside the REPL to authenticate this env.

cenv login requires an interactive terminal. Agents and scripts should
create envs via 'cenv create' and prompt the user to run 'cenv login'.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !env.Exists(name) {
			return fmt.Errorf("environment %q does not exist", name)
		}
		if !isTerminal(os.Stdin) {
			return fmt.Errorf("cenv login requires an interactive terminal")
		}

		envDir := env.Path(name)

		claudePath, err := exec.LookPath("claude")
		if err != nil {
			return fmt.Errorf("claude not found in PATH")
		}

		logf("[cenv] Opening Claude in %q; run /login inside the REPL.\n", name)

		environ := append(os.Environ(), fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", envDir))
		return syscall.Exec(claudePath, []string{"claude"}, environ)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
