package main

import (
	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/env"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an environment",
	Example: `  cenv remove myenv
  cenv remove myenv --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := env.Remove(name); err != nil {
			return err
		}
		logf("[cenv] Removed environment %q\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
