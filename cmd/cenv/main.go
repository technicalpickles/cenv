package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var quiet bool

var rootCmd = &cobra.Command{
	Use:   "cenv",
	Short: "Manage isolated Claude Code configuration directories",
	Long: `cenv manages isolated Claude Code configuration directories.
Each one gets its own settings, permissions, hooks, plugins, and session
history, completely independent of ~/.claude/. Think virtualenv for Claude Code.`,
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", envBool("CENV_QUIET"), "Suppress [cenv] informational output")
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		// cobra already prints "Error: <err>" to stderr; just exit nonzero.
		os.Exit(1)
	}
}
