package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/style"
)

var execCmd = &cobra.Command{
	Use:   "exec <name> -- <command> [args...]",
	Short: "Run an arbitrary command in an environment",
	Example: `  cenv exec myenv -- a2acode serve
  cenv exec myenv -- npm test`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		envDir, err := preflightEnv(name)
		if err != nil {
			return err
		}

		if len(args) < 3 || args[1] != "--" {
			return fmt.Errorf("missing command (use: cenv exec %s -- <command> [args...])", name)
		}
		command := args[2:]

		logf("%s\n", style.Info("Using %q (%s)", name, envDir))

		cmdPath, err := exec.LookPath(command[0])
		if err != nil {
			return fmt.Errorf("%s not found in PATH", command[0])
		}

		environ := os.Environ()
		environ = append(environ, fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", envDir))

		execArgs := append([]string{command[0]}, command[1:]...)
		return syscall.Exec(cmdPath, execArgs, environ)
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}
