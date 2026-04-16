package settings

import (
	"fmt"
	"strings"
)

// GetByDotPath traverses a nested map using a dot-separated key path.
// Returns the value at the path, or an error if the path doesn't exist
// or tries to traverse a non-object node.
func GetByDotPath(data map[string]any, path string) (any, error) {
	parts := strings.Split(path, ".")
	var current any = data

	for i, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			traversedSoFar := strings.Join(parts[:i], ".")
			return nil, fmt.Errorf("cannot traverse into non-object at %q", traversedSoFar)
		}
		val, exists := m[part]
		if !exists {
			traversedSoFar := strings.Join(parts[:i+1], ".")
			return nil, fmt.Errorf("key not found: %q", traversedSoFar)
		}
		current = val
	}

	return current, nil
}
