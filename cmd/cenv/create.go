package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/bootstrap"
	"github.com/technicalpickles/cenv/internal/claudeconfig"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/keychain"
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

		// Cleanup on any subsequent error: remove envDir AND any keychain entry
		// we may have written for it.
		var cleanupNeeded = true
		defer func() {
			if !cleanupNeeded {
				return
			}
			os.RemoveAll(envDir)
			_ = keychain.Default.Delete(keychain.ServiceName(envDir))
		}()

		var settingsData map[string]any
		var sourceDir string        // empty = no auth copy (bare or legacy auth-env)
		var sourceClaudeJSON string // path to source's .claude.json (asymmetric, see copyAuth doc)

		switch {
		case createBare:
			settingsData = map[string]any{}

		case createAuth != "":
			// Legacy auth-<name> env, settings only, no OAuth copy.
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
			userClaudeDir := filepath.Join(home, ".claude")
			loaded, err := settings.Load(filepath.Join(userClaudeDir, "settings.json"))
			if err != nil {
				return fmt.Errorf("loading user settings: %w", err)
			}
			settingsData = loaded
			sourceDir = userClaudeDir
			sourceClaudeJSON = filepath.Join(home, ".claude.json")

		case createFrom != "":
			if !env.Exists(createFrom) {
				return fmt.Errorf("source environment %q not found", createFrom)
			}
			srcEnvDir := env.Path(createFrom)
			loaded, err := settings.Load(filepath.Join(srcEnvDir, "settings.json"))
			if err != nil {
				return fmt.Errorf("loading source environment settings: %w", err)
			}
			settingsData = loaded
			sourceDir = srcEnvDir
			sourceClaudeJSON = filepath.Join(srcEnvDir, ".claude.json")

		default:
			// Auto-detect from ~/.claude.
			home, err := os.UserHomeDir()
			if err == nil {
				userClaudeDir := filepath.Join(home, ".claude")
				if loaded, err := settings.Load(filepath.Join(userClaudeDir, "settings.json")); err == nil {
					settingsData = bootstrap.ExtractAuth(loaded)
				}
				sourceDir = userClaudeDir
				sourceClaudeJSON = filepath.Join(home, ".claude.json")
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

		if sourceDir != "" {
			copied, err := copyAuth(sourceDir, sourceClaudeJSON, envDir, keychain.Default)
			if err != nil {
				return fmt.Errorf("copying auth: %w", err)
			}
			if copied {
				logf("[cenv] Copied OAuth login from %s\n", displaySourceName(sourceDir))
			}
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

// copyAuth moves OAuth auth (keychain token + oauthAccount + firstTokenDate)
// from source to dstEnvDir. Returns copied=true iff something was written.
//
// srcConfigDir is the source's claude config dir: either ~/.claude (user) or
// a cenv env dir. It's used to derive the keychain service name.
//
// srcClaudeJSON is the source's .claude.json path. Note the asymmetry:
//   - For ~/.claude, this is ~/.claude.json (at HOME root, not inside the dir).
//   - For cenv env dirs, this is <envdir>/.claude.json (inside the dir).
//
// Claude Code writes .claude.json outside the config dir for the default
// (~/.claude) case but inside it when CLAUDE_CONFIG_DIR is set.
//
// dstEnvDir is the new cenv env dir. Its .claude.json is at dstEnvDir/.claude.json
// (cenv sets CLAUDE_CONFIG_DIR for its envs so Claude Code writes there).
//
// Source missing either the keychain token OR oauthAccount means the source
// isn't fully OAuth-authed; returns copied=false, nil. This is the common
// case for Bedrock-only or fresh users.
func copyAuth(srcConfigDir, srcClaudeJSON, dstEnvDir string, kc *keychain.Client) (copied bool, err error) {
	srcOAuth, err := claudeconfig.ReadOAuth(srcClaudeJSON)
	if err != nil {
		return false, fmt.Errorf("reading source OAuth config: %w", err)
	}
	if srcOAuth == nil {
		return false, nil
	}

	srcSvc := keychain.ServiceName(srcConfigDir)
	token, notFound, err := kc.Read(srcSvc)
	if err != nil {
		return false, fmt.Errorf("reading source keychain: %w", err)
	}
	if notFound {
		// Partial state: .claude.json claims auth but keychain is empty.
		// Skip rather than write a half-copied state.
		return false, nil
	}

	// Destination writes: keychain first, then config. Cleanup on subsequent
	// failure is the caller's job (via the existing defer in createCmd).
	dstSvc := keychain.ServiceName(dstEnvDir)
	if err := kc.Write(dstSvc, token); err != nil {
		return false, fmt.Errorf("writing destination keychain: %w", err)
	}
	if err := claudeconfig.MergeOAuth(filepath.Join(dstEnvDir, ".claude.json"), srcOAuth); err != nil {
		if delErr := kc.Delete(dstSvc); delErr != nil {
			logf("[cenv] Warning: failed to roll back keychain entry %q: %v\n", dstSvc, delErr)
		}
		return false, fmt.Errorf("merging OAuth into destination config: %w", err)
	}
	return true, nil
}

// displaySourceName formats a source dir for log messages.
// The default user dir is shown as "~/.claude"; other paths as-is.
func displaySourceName(dir string) string {
	home, err := os.UserHomeDir()
	if err == nil && dir == filepath.Join(home, ".claude") {
		return "~/.claude"
	}
	return dir
}
