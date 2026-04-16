package settings

import (
	"encoding/json"
	"fmt"
	"os"
)

// Load reads and parses a JSON file into a map.
func Load(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return result, nil
}

// Save writes data to path as pretty-printed JSON with a trailing newline.
func Save(path string, data map[string]any) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// MergeInto loads the existing settings at path, deep merges overlay into them, and saves the result.
func MergeInto(path string, overlay map[string]any) error {
	base, err := Load(path)
	if err != nil {
		return err
	}

	merged := DeepMerge(base, overlay)

	return Save(path, merged)
}

// ResolveOverlay parses arg as inline JSON if it looks like JSON, otherwise loads it as a file path.
func ResolveOverlay(arg string) (map[string]any, error) {
	if IsJSON(arg) {
		var result map[string]any
		if err := json.Unmarshal([]byte(arg), &result); err != nil {
			return nil, fmt.Errorf("parsing inline JSON: %w", err)
		}
		return result, nil
	}

	return Load(arg)
}
