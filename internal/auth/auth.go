// Package auth detects whether a Claude config dir has authentication
// configured. It recognizes three shapes:
//
//  1. AWS Bedrock: settings.json has an "awsAuthRefresh" object
//  2. Anthropic OAuth: .claude.json has a non-empty "oauthAccount" (string or object)
//  3. Token or API key: a non-empty ANTHROPIC_AUTH_TOKEN or ANTHROPIC_API_KEY,
//     in the process env or in the "env" block of settings.json
//
// Callers use Detect as a predicate (error == nil means "authenticated")
// for pre-flight checks in cenv claude/run/exec and the HasAuth field in
// env.Info.
package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// tokenEnvVars are the env vars that make Claude Code authenticate without
// an OAuth login (AI gateway, API key).
var tokenEnvVars = []string{"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY"}

// Detect returns nil if the process env or configDir carries Bedrock,
// Anthropic OAuth, or token/API-key auth, or an error describing what's
// missing.
func Detect(configDir string) error {
	for _, name := range tokenEnvVars {
		if os.Getenv(name) != "" {
			return nil
		}
	}

	settingsPath := filepath.Join(configDir, "settings.json")
	if settingsData, err := readJSON(settingsPath); err == nil {
		if obj, ok := settingsData["awsAuthRefresh"].(map[string]any); ok && len(obj) > 0 {
			return nil
		}
		if envBlock, ok := settingsData["env"].(map[string]any); ok {
			for _, name := range tokenEnvVars {
				if v, ok := envBlock[name].(string); ok && v != "" {
					return nil
				}
			}
		}
	}

	claudePath := filepath.Join(configDir, ".claude.json")
	if claudeData, err := readJSON(claudePath); err == nil {
		if val, ok := claudeData["oauthAccount"]; ok {
			switch v := val.(type) {
			case string:
				if v != "" {
					return nil
				}
			case map[string]any:
				if len(v) > 0 {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("no auth found in %q: no awsAuthRefresh in settings.json, no oauthAccount in .claude.json, and no ANTHROPIC_AUTH_TOKEN or ANTHROPIC_API_KEY", configDir)
}

func readJSON(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing %q: %w", path, err)
	}
	return result, nil
}
