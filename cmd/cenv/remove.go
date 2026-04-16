package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/env"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := env.Remove(name); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "[cenv] Removed environment %q\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
