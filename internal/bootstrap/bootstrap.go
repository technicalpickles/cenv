package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// onboardingData is written to .claude.json to skip the onboarding flow.
var onboardingData = map[string]any{
	"hasCompletedOnboarding": true,
	"hasSeenTasksHint":       true,
	"theme":                  "dark",
	"numStartups":            0,
}

// authKeys are the keys extracted from a settings map for auth-related config.
var authKeys = []string{"env", "awsAuthRefresh", "statusLine"}

// WriteOnboarding writes .claude.json to envDir with onboarding skipped.
func WriteOnboarding(envDir string) error {
	return writeJSON(filepath.Join(envDir, ".claude.json"), onboardingData)
}

// ExtractAuth returns a new map containing only auth-related keys from input.
// Keys not present in input are omitted from the result.
func ExtractAuth(settings map[string]any) map[string]any {
	result := make(map[string]any)
	for _, key := range authKeys {
		if val, ok := settings[key]; ok {
			result[key] = val
		}
	}
	return result
}

// WriteSettings writes settings.json to envDir as pretty-printed JSON with a trailing newline.
func WriteSettings(envDir string, data map[string]any) error {
	return writeJSON(filepath.Join(envDir, "settings.json"), data)
}

func writeJSON(path string, data any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0600)
}
