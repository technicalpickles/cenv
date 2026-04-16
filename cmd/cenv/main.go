package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cenv",
	Short: "Manage isolated Claude Code configuration directories",
	Long: `cenv manages isolated Claude Code configuration directories.
Each one gets its own settings, permissions, hooks, plugins, and session
history, completely independent of ~/.claude/. Think virtualenv for Claude Code.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
