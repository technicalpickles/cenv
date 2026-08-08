package main

import (
	"fmt"
	"path/filepath"

	"github.com/technicalpickles/cenv/internal/auth"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/settings"
)

// preflightEnv validates that the named environment exists, has a loadable
// settings.json, and has auth configured. On success it returns the
// environment's directory path.
func preflightEnv(name string) (string, error) {
	if !env.Exists(name) {
		return "", fmt.Errorf("environment %q not found", name)
	}

	envDir := env.Path(name)
	settingsPath := filepath.Join(envDir, "settings.json")
	if _, err := settings.Load(settingsPath); err != nil {
		return "", fmt.Errorf("preflight failed: %w", err)
	}

	if err := auth.Detect(envDir); err != nil {
		return "", fmt.Errorf("env %q has no auth configured; run 'cenv login %s' first", name, name)
	}

	return envDir, nil
}
