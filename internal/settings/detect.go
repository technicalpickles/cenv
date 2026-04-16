package settings

import "strings"

// IsJSON returns true if input looks like inline JSON (starts with { or [ after trimming whitespace).
func IsJSON(input string) bool {
	trimmed := strings.TrimSpace(input)
	if len(trimmed) == 0 {
		return false
	}
	return trimmed[0] == '{' || trimmed[0] == '['
}
