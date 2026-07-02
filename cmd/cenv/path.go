package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/env"
)

var pathCmd = &cobra.Command{
	Use:     "path <name>",
	Short:   "Print the directory path of an environment",
	Example: `  cenv path myenv`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !env.Exists(name) {
			return fmt.Errorf("environment %q not found", name)
		}
		fmt.Println(env.Path(name))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pathCmd)
}
