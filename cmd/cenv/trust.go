package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/claudeconfig"
	"github.com/technicalpickles/cenv/internal/env"
)

var trustCmd = &cobra.Command{
	Use:   "trust <name> <path> [path...]",
	Short: "Mark workspace path(s) as trusted in an environment",
	Example: `  cenv trust myenv ~/projects/foo
  cenv trust myenv ~/projects/foo ~/projects/bar`,
	Long: `Mark one or more workspace paths as trusted in an environment's Claude
config. This pre-accepts the "Do you trust this directory?" dialog that
Claude Code shows on first launch in a new workspace, which is useful for
automated setups (e.g., test harnesses) where interactive prompts break
the flow.

Paths are resolved to absolute paths before being written.`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		paths := args[1:]

		if !env.Exists(name) {
			return fmt.Errorf("environment %q not found", name)
		}

		claudeJSONPath := filepath.Join(env.Path(name), ".claude.json")
		if _, err := os.Stat(claudeJSONPath); os.IsNotExist(err) {
			return fmt.Errorf("env %q has no .claude.json; run 'cenv create %s' or 'cenv login %s' first", name, name, name)
		}

		absPaths := make([]string, 0, len(paths))
		for _, p := range paths {
			abs, err := filepath.Abs(p)
			if err != nil {
				return fmt.Errorf("resolving %q: %w", p, err)
			}
			absPaths = append(absPaths, filepath.Clean(abs))
		}

		if err := claudeconfig.MergeTrust(claudeJSONPath, absPaths...); err != nil {
			return err
		}

		for _, p := range absPaths {
			logf("[cenv] Trusted %q in %q\n", p, name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(trustCmd)
}
