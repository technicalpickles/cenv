package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/bootstrap"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/settings"
)

var rootCmd = &cobra.Command{
	Use:   "cenv",
	Short: "Manage isolated Claude Code configuration directories",
	Long: `cenv manages isolated Claude Code configuration directories.
Each one gets its own settings, permissions, hooks, plugins, and session
history, completely independent of ~/.claude/. Think virtualenv for Claude Code.`,
}

// create command

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
		fmt.Fprintf(os.Stderr, "[cenv] Created environment %q\n", name)
		return nil
	},
}

func init() {
	createCmd.Flags().BoolVar(&createBare, "bare", false, "Create with empty settings")
	createCmd.Flags().StringVar(&createAuth, "auth", "", "Use auth from named auth environment")
	createCmd.Flags().StringVar(&createFrom, "from", "", "Clone settings from 'user' or another environment")
	rootCmd.AddCommand(createCmd)
}

// list command

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

// remove command

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

// path command

var pathCmd = &cobra.Command{
	Use:   "path <name>",
	Short: "Print the directory path of an environment",
	Args:  cobra.ExactArgs(1),
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

// settings commands

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage environment settings",
}

var settingsShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show settings for an environment as JSON",
	Args:  cobra.ExactArgs(1),
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
	Use:   "get <name> <key>",
	Short: "Get a value from settings by dot-path key",
	Args:  cobra.ExactArgs(2),
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
	Use:   "merge <name> <json|file>",
	Short: "Deep merge JSON or a JSON file into environment settings",
	Args:  cobra.ExactArgs(2),
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

		fmt.Fprintf(os.Stderr, "[cenv] Merged settings into %q\n", name)
		return nil
	},
}

func init() {
	settingsCmd.AddCommand(settingsShowCmd)
	settingsCmd.AddCommand(settingsGetCmd)
	settingsCmd.AddCommand(settingsMergeCmd)
	rootCmd.AddCommand(settingsCmd)
}

// run command

var runCmd = &cobra.Command{
	Use:                "run <name> [-- claude-args...]",
	Short:              "Launch Claude in an environment",
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !env.Exists(name) {
			return fmt.Errorf("environment %q does not exist", name)
		}

		envDir := env.Path(name)
		settingsPath := filepath.Join(envDir, "settings.json")
		if _, err := settings.Load(settingsPath); err != nil {
			return fmt.Errorf("preflight failed: %w", err)
		}

		var claudeArgs []string
		if len(args) > 1 {
			if args[1] != "--" {
				return fmt.Errorf("unexpected argument %q (use -- before claude arguments)", args[1])
			}
			claudeArgs = args[2:]
		}

		fmt.Fprintf(os.Stderr, "[cenv] Using %q (%s)\n", name, envDir)

		claudePath, err := exec.LookPath("claude")
		if err != nil {
			return fmt.Errorf("claude not found in PATH")
		}

		environ := os.Environ()
		environ = append(environ, fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", envDir))

		execArgs := append([]string{"claude"}, claudeArgs...)
		return syscall.Exec(claudePath, execArgs, environ)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
