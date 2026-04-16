package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DetectResult holds the outcome of auth detection.
type DetectResult struct {
	Type    string // "anthropic" or "aws-bedrock"
	EnvName string // "auth-anthropic" or "auth-aws-bedrock"
	Detail  string // human-readable detail (email, region, etc.)
}

// Detect examines a Claude config directory and returns the active auth method.
//
// Detection order:
//  1. settings.json: if "awsAuthRefresh" key exists as an object, it's AWS Bedrock.
//  2. .claude.json: if "oauthAccount" key exists as a non-empty string, it's Anthropic.
//
// Returns an error if neither is found.
func Detect(configDir string) (*DetectResult, error) {
	// Step 1: check settings.json for awsAuthRefresh
	settingsPath := filepath.Join(configDir, "settings.json")
	settingsData, err := readJSON(settingsPath)
	if err == nil {
		if obj, ok := settingsData["awsAuthRefresh"]; ok {
			if _, isMap := obj.(map[string]any); isMap {
				detail := ""
				if m, ok := obj.(map[string]any); ok {
					if region, ok := m["region"].(string); ok && region != "" {
						detail = region
					}
				}
				return &DetectResult{
					Type:    "aws-bedrock",
					EnvName: "auth-aws-bedrock",
					Detail:  detail,
				}, nil
			}
		}
	}

	// Step 2: check .claude.json for oauthAccount
	claudePath := filepath.Join(configDir, ".claude.json")
	claudeData, err := readJSON(claudePath)
	if err == nil {
		if val, ok := claudeData["oauthAccount"]; ok {
			if email, ok := val.(string); ok && email != "" {
				return &DetectResult{
					Type:    "anthropic",
					EnvName: "auth-anthropic",
					Detail:  email,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("no auth found in %q: no awsAuthRefresh in settings.json and no oauthAccount in .claude.json", configDir)
}

// readJSON reads a JSON file and returns the top-level map.
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
