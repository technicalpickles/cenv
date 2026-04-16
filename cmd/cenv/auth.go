package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/auth"
	"github.com/technicalpickles/cenv/internal/bootstrap"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/settings"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage auth environments",
}

var authCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create or update the auth environment from ~/.claude/",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("finding home directory: %w", err)
		}
		claudeDir := filepath.Join(home, ".claude")

		detected, err := auth.Detect(claudeDir)
		if err != nil {
			return fmt.Errorf("detecting auth: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[cenv] Detected auth type: %s\n", detected.Type)

		userSettingsPath := filepath.Join(claudeDir, "settings.json")
		loaded, err := settings.Load(userSettingsPath)
		if err != nil {
			return fmt.Errorf("loading user settings: %w", err)
		}
		settingsData := bootstrap.ExtractAuth(loaded)

		envName := detected.EnvName
		alreadyExisted := env.Exists(envName)

		envDir := env.Path(envName)
		if err := os.MkdirAll(envDir, 0755); err != nil {
			return fmt.Errorf("creating environment directory: %w", err)
		}

		if err := bootstrap.WriteSettings(envDir, settingsData); err != nil {
			return fmt.Errorf("writing settings: %w", err)
		}

		if err := bootstrap.WriteOnboarding(envDir); err != nil {
			return fmt.Errorf("writing onboarding: %w", err)
		}

		if alreadyExisted {
			fmt.Println("Updated")
		} else {
			fmt.Println("Created")
		}
		return nil
	},
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all auth environments",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := env.List()
		if err != nil {
			return err
		}

		var authNames []string
		for _, name := range names {
			if strings.HasPrefix(name, "auth-") {
				authNames = append(authNames, name)
			}
		}

		if len(authNames) == 0 {
			fmt.Println("No auth environments yet.")
			fmt.Println("Create one: cenv auth create")
			return nil
		}

		for _, name := range authNames {
			fmt.Println(name)
		}
		return nil
	},
}

func init() {
	authCmd.AddCommand(authCreateCmd)
	authCmd.AddCommand(authListCmd)
	rootCmd.AddCommand(authCmd)
}
