package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/bootstrap"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/settings"
)

var (
	createBare bool
	createAuth string
	createFrom string
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if err := env.ValidateName(name); err != nil {
			return err
		}

		if env.Exists(name) {
			return fmt.Errorf("environment %q already exists", name)
		}

		envDir := env.Path(name)
		if err := os.MkdirAll(envDir, 0755); err != nil {
			return fmt.Errorf("creating environment directory: %w", err)
		}

		// Cleanup on any subsequent error
		var cleanupNeeded = true
		defer func() {
			if cleanupNeeded {
				os.RemoveAll(envDir)
			}
		}()

		var settingsData map[string]any

		switch {
		case createBare:
			settingsData = map[string]any{}

		case createAuth != "":
			authEnvName := "auth-" + createAuth
			if !env.Exists(authEnvName) {
				return fmt.Errorf("auth environment %q not found", authEnvName)
			}
			authSettingsPath := filepath.Join(env.Path(authEnvName), "settings.json")
			loaded, err := settings.Load(authSettingsPath)
			if err != nil {
				return fmt.Errorf("loading auth environment settings: %w", err)
			}
			settingsData = loaded

		case createFrom == "user":
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("finding home directory: %w", err)
			}
			userSettingsPath := filepath.Join(home, ".claude", "settings.json")
			loaded, err := settings.Load(userSettingsPath)
			if err != nil {
				return fmt.Errorf("loading user settings: %w", err)
			}
			settingsData = loaded

		case createFrom != "":
			if !env.Exists(createFrom) {
				return fmt.Errorf("source environment %q not found", createFrom)
			}
			srcSettingsPath := filepath.Join(env.Path(createFrom), "settings.json")
			loaded, err := settings.Load(srcSettingsPath)
			if err != nil {
				return fmt.Errorf("loading source environment settings: %w", err)
			}
			settingsData = loaded

		default:
			// Auto-detect auth from ~/.claude/settings.json
			home, err := os.UserHomeDir()
			if err == nil {
				userSettingsPath := filepath.Join(home, ".claude", "settings.json")
				if loaded, err := settings.Load(userSettingsPath); err == nil {
					settingsData = bootstrap.ExtractAuth(loaded)
				}
				if hasOAuth(home) {
					logf("[cenv] Note: Anthropic OAuth detected. Login tokens don't transfer between envs; run 'cenv login %s' to authenticate this env.\n", name)
				}
			}
			if settingsData == nil {
				settingsData = map[string]any{}
			}
		}

		if err := bootstrap.WriteSettings(envDir, settingsData); err != nil {
			return fmt.Errorf("writing settings: %w", err)
		}

		if err := bootstrap.WriteOnboarding(envDir); err != nil {
			return fmt.Errorf("writing onboarding: %w", err)
		}

		cleanupNeeded = false
		logf("[cenv] Created environment %q\n", name)
		return nil
	},
}

func init() {
	createCmd.Flags().BoolVar(&createBare, "bare", false, "Create with empty settings")
	createCmd.Flags().StringVar(&createAuth, "auth", "", "Use auth from named auth environment")
	createCmd.Flags().StringVar(&createFrom, "from", "", "Clone settings from 'user' or another environment")
	rootCmd.AddCommand(createCmd)
}

// hasOAuth reports whether the user has Anthropic OAuth configured, indicated
// by a non-empty oauthAccount field in ~/.claude.json (home root, not ~/.claude/).
// Claude Code writes oauthAccount as an object; older versions may have used a
// string. Both shapes are accepted.
func hasOAuth(home string) bool {
	data, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		return false
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return false
	}
	switch v := parsed["oauthAccount"].(type) {
	case string:
		return v != ""
	case map[string]any:
		return len(v) > 0
	default:
		return false
	}
}
