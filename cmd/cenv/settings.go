package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/settings"
	"github.com/technicalpickles/cenv/internal/style"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage environment settings",
}

var settingsShowCmd = &cobra.Command{
	Use:     "show <name>",
	Short:   "Show settings for an environment as JSON",
	Example: `  cenv settings show myenv`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !env.Exists(name) {
			return fmt.Errorf("environment %q not found", name)
		}

		settingsPath := filepath.Join(env.Path(name), "settings.json")
		data, err := settings.Load(settingsPath)
		if err != nil {
			return err
		}

		out, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		fmt.Println(string(out))
		return nil
	},
}

var settingsGetCmd = &cobra.Command{
	Use:     "get <name> <key>",
	Short:   "Get a value from settings by dot-path key",
	Example: `  cenv settings get myenv statusLine.type`,
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		key := args[1]

		if !env.Exists(name) {
			return fmt.Errorf("environment %q not found", name)
		}

		settingsPath := filepath.Join(env.Path(name), "settings.json")
		data, err := settings.Load(settingsPath)
		if err != nil {
			return err
		}

		val, err := settings.GetByDotPath(data, key)
		if err != nil {
			return err
		}

		switch v := val.(type) {
		case map[string]any, []any:
			out, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling JSON: %w", err)
			}
			fmt.Println(string(out))
		default:
			fmt.Println(v)
		}
		return nil
	},
}

var settingsMergeCmd = &cobra.Command{
	Use:     "merge <name> <json|file>",
	Short:   "Deep merge JSON or a JSON file into environment settings",
	Example: `  cenv settings merge myenv '{"statusLine":{"type":"command"}}'`,
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		arg := args[1]

		if !env.Exists(name) {
			return fmt.Errorf("environment %q not found", name)
		}

		overlay, err := settings.ResolveOverlay(arg)
		if err != nil {
			return err
		}

		settingsPath := filepath.Join(env.Path(name), "settings.json")
		if err := settings.MergeInto(settingsPath, overlay); err != nil {
			return err
		}

		logf("%s\n", style.Success("Merged settings into %q", name))
		return nil
	},
}

func init() {
	settingsCmd.AddCommand(settingsShowCmd)
	settingsCmd.AddCommand(settingsGetCmd)
	settingsCmd.AddCommand(settingsMergeCmd)
	rootCmd.AddCommand(settingsCmd)
}
