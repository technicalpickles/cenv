// Package claudeconfig reads and merges specific fields in a Claude Code
// config file (.claude.json). OAuth fields (oauthAccount,
// claudeCodeFirstTokenDate) and workspace trust entries under "projects"
// are touched; onboarding keys and unrelated fields are preserved.
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

// MergeOAuth merges oauth fields into the JSON object at claudeJSONPath,
// preserving all existing keys (onboarding flags, etc.). The file must
// already exist -- this is called after bootstrap.WriteOnboarding.
func MergeOAuth(claudeJSONPath string, oauth *OAuth) error {
	data, err := os.ReadFile(claudeJSONPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", claudeJSONPath, err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("parsing %s: %w", claudeJSONPath, err)
	}
	if parsed == nil {
		parsed = map[string]any{}
	}

	parsed["oauthAccount"] = oauth.Account
	if oauth.ClaudeCodeFirstDate != "" {
		parsed["claudeCodeFirstTokenDate"] = oauth.ClaudeCodeFirstDate
	}

	out, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling %s: %w", claudeJSONPath, err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(claudeJSONPath, out, 0600); err != nil {
		return fmt.Errorf("writing %s: %w", claudeJSONPath, err)
	}
	return nil
}

// MergeTrust marks workspacePaths as trusted in the Claude config at
// claudeJSONPath. Each path becomes an entry under "projects" with
// hasTrustDialogAccepted set to true; existing entries for the same path
// keep their sibling fields (allowedTools, mcpServers, etc). The file must
// already exist. Paths are used as-is; callers should resolve to absolute
// and clean before passing in.
func MergeTrust(claudeJSONPath string, workspacePaths ...string) error {
	if len(workspacePaths) == 0 {
		return nil
	}

	data, err := os.ReadFile(claudeJSONPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", claudeJSONPath, err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("parsing %s: %w", claudeJSONPath, err)
	}
	if parsed == nil {
		parsed = map[string]any{}
	}

	projects, _ := parsed["projects"].(map[string]any)
	if projects == nil {
		projects = map[string]any{}
	}

	for _, path := range workspacePaths {
		entry, _ := projects[path].(map[string]any)
		if entry == nil {
			entry = map[string]any{}
		}
		entry["hasTrustDialogAccepted"] = true
		projects[path] = entry
	}
	parsed["projects"] = projects

	out, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling %s: %w", claudeJSONPath, err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(claudeJSONPath, out, 0600); err != nil {
		return fmt.Errorf("writing %s: %w", claudeJSONPath, err)
	}
	return nil
}
