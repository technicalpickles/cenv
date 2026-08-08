// cmd/cenv/claude.go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/style"
)

var claudeCmd = &cobra.Command{
	Use:   "claude <name> [-- claude-args...]",
	Short: "Launch Claude in an environment",
	Example: `  cenv claude myenv
  cenv claude myenv -- --model opus`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE:               runClaudeE,
}

var runCmd = &cobra.Command{
	Use:   "run <name> [-- claude-args...]",
	Short: "Launch Claude in an environment",
	Example: `  cenv run myenv
  cenv run myenv -- --model opus`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	Deprecated:         "use 'cenv claude' instead",
	RunE:               runClaudeE,
}

func runClaudeE(cmd *cobra.Command, args []string) error {
	name := args[0]

	envDir, err := preflightEnv(name)
	if err != nil {
		return err
	}

	var claudeArgs []string
	if len(args) > 1 {
		if args[1] != "--" {
			return fmt.Errorf("unexpected argument %q (use -- before claude arguments)", args[1])
		}
		claudeArgs = args[2:]
	}

	logf("%s\n", style.Info("Using %q (%s)", name, envDir))

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH")
	}

	environ := os.Environ()
	environ = append(environ, fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", envDir))

	execArgs := append([]string{"claude"}, claudeArgs...)
	return syscall.Exec(claudePath, execArgs, environ)
}

func init() {
	rootCmd.AddCommand(claudeCmd)
	rootCmd.AddCommand(runCmd)
}
