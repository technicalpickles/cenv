package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/style"
)

var quiet bool
var noColor bool

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "cenv",
	Short: "Manage isolated Claude Code configuration directories",
	Long: `cenv manages isolated Claude Code configuration directories.
Each one gets its own settings, permissions, hooks, plugins, and session
history, completely independent of ~/.claude/. Think virtualenv for Claude Code.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", envBool("CENV_QUIET"), "Suppress [cenv] informational output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		applyColorSetting()
	}
}

// logf writes informational output to stderr unless quiet mode is enabled.
func logf(format string, args ...any) {
	if quiet {
		return
	}
	fmt.Fprintf(os.Stderr, format, args...)
}

// envBool reads a boolean env var. Empty or unparseable values return false.
func envBool(key string) bool {
	v := os.Getenv(key)
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}

// colorEnabled decides whether cenv should emit ANSI color codes. Color is
// enabled only when nothing disables it: no --no-color flag, no NO_COLOR
// env var (per https://no-color.org, presence disables regardless of
// value), and stdout is an actual terminal (not piped/redirected).
func colorEnabled(noColorFlag bool, noColorEnvSet bool, stdoutIsTTY bool) bool {
	return !noColorFlag && !noColorEnvSet && stdoutIsTTY
}

// applyColorSetting sets fatih/color's global switch from the current
// flag/env/TTY state. Called from PersistentPreRun (after flags are
// parsed, before any command's RunE) and again from main's error path
// (in case Execute failed before reaching PersistentPreRun, e.g. on a
// flag-parse error).
func applyColorSetting() {
	color.NoColor = !colorEnabled(noColor, os.Getenv("NO_COLOR") != "", isTerminal(os.Stdout))
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		applyColorSetting()
		fmt.Fprintln(os.Stderr, style.Error("%v", err))
		os.Exit(1)
	}
}
