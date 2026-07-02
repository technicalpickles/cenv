package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/auth"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/settings"
)

var runCmd = &cobra.Command{
	Use:   "run <name> [-- claude-args...]",
	Short: "Launch Claude in an environment",
	Example: `  cenv run myenv
  cenv run myenv -- --model opus`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !env.Exists(name) {
			return fmt.Errorf("environment %q not found", name)
		}

		envDir := env.Path(name)
		settingsPath := filepath.Join(envDir, "settings.json")
		if _, err := settings.Load(settingsPath); err != nil {
			return fmt.Errorf("preflight failed: %w", err)
		}

		if err := auth.Detect(envDir); err != nil {
			return fmt.Errorf("env %q has no auth configured; run 'cenv login %s' first", name, name)
		}

		var claudeArgs []string
		if len(args) > 1 {
			if args[1] != "--" {
				return fmt.Errorf("unexpected argument %q (use -- before claude arguments)", args[1])
			}
			claudeArgs = args[2:]
		}

		logf("[cenv] Using %q (%s)\n", name, envDir)

		claudePath, err := exec.LookPath("claude")
		if err != nil {
			return fmt.Errorf("claude not found in PATH")
		}

		environ := os.Environ()
		environ = append(environ, fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", envDir))

		execArgs := append([]string{"claude"}, claudeArgs...)
		return syscall.Exec(claudePath, execArgs, environ)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
