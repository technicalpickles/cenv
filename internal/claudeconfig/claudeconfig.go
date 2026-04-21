// Package claudeconfig reads and merges OAuth-related fields in a Claude
// Code config file (.claude.json). Only oauthAccount and
// claudeCodeFirstTokenDate are touched; onboarding keys written by cenv's
// bootstrap are preserved.
package claudeconfig

import (
	"encoding/json"
	"fmt"
	"os"
)

// OAuth captures the fields we need to transplant between envs. Account is
// opaque (kept as a map so we round-trip any future fields claude-code adds
// without changes here).
type OAuth struct {
	Account             map[string]any
	ClaudeCodeFirstDate string
}

// ReadOAuth parses claudeJSONPath and returns the OAuth fields, or nil if
// the file does not exist or oauthAccount is missing, null, or empty.
// Missing file is not an error (source may never have been authed).
func ReadOAuth(claudeJSONPath string) (*OAuth, error) {
	data, err := os.ReadFile(claudeJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", claudeJSONPath, err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", claudeJSONPath, err)
	}

	acctRaw, ok := parsed["oauthAccount"]
	if !ok {
		return nil, nil
	}
	acct, ok := acctRaw.(map[string]any)
	if !ok || len(acct) == 0 {
		return nil, nil
	}

	oa := &OAuth{Account: acct}
	if s, ok := parsed["claudeCodeFirstTokenDate"].(string); ok {
		oa.ClaudeCodeFirstDate = s
	}
	return oa, nil
}
