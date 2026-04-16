package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/env"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all environments",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := env.List()
		if err != nil {
			return err
		}

		if len(names) == 0 {
			fmt.Println("No environments yet.")
			fmt.Println("Create one: cenv create <name>")
			return nil
		}

		for _, name := range names {
			fmt.Println(name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
