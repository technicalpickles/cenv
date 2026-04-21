// Package auth detects whether a Claude config dir has authentication
// configured. It recognizes two shapes:
//
//  1. AWS Bedrock: settings.json has an "awsAuthRefresh" object
//  2. Anthropic OAuth: .claude.json has a non-empty "oauthAccount" (string or object)
//
// Callers use Detect as a predicate (error == nil means "authenticated")
// for pre-flight checks in cenv run and the HasAuth field in env.Info.
package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Detect returns nil if configDir contains either Bedrock or Anthropic OAuth
// config, or an error describing what's missing.
func Detect(configDir string) error {
	settingsPath := filepath.Join(configDir, "settings.json")
	if settingsData, err := readJSON(settingsPath); err == nil {
		if obj, ok := settingsData["awsAuthRefresh"].(map[string]any); ok && len(obj) > 0 {
			return nil
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

	return fmt.Errorf("no auth found in %q: no awsAuthRefresh in settings.json and no oauthAccount in .claude.json", configDir)
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
