package claudeconfig_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/technicalpickles/cenv/internal/claudeconfig"
)

func TestReadOAuth_FullConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	body := `{
		"oauthAccount": {"emailAddress": "user@example.com", "organization": {"name": "Acme"}},
		"claudeCodeFirstTokenDate": "2026-04-20T12:00:00Z",
		"hasCompletedOnboarding": true
	}`
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := claudeconfig.ReadOAuth(path)
	if err != nil {
		t.Fatalf("ReadOAuth returned err: %v", err)
	}
	if got == nil {
		t.Fatal("got nil, want non-nil OAuth")
	}
	if got.ClaudeCodeFirstDate != "2026-04-20T12:00:00Z" {
		t.Errorf("ClaudeCodeFirstDate = %q, want 2026-04-20T12:00:00Z", got.ClaudeCodeFirstDate)
	}
	if email, _ := got.Account["emailAddress"].(string); email != "user@example.com" {
		t.Errorf("Account.emailAddress = %v, want user@example.com", got.Account["emailAddress"])
	}
}

func TestReadOAuth_NoOAuthField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	body := `{"hasCompletedOnboarding": true}`
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := claudeconfig.ReadOAuth(path)
	if err != nil {
		t.Fatalf("ReadOAuth returned err: %v", err)
	}
	if got != nil {
		t.Errorf("got %+v, want nil", got)
	}
}

func TestReadOAuth_MissingFile(t *testing.T) {
	got, err := claudeconfig.ReadOAuth("/nonexistent/path/.claude.json")
	if err != nil {
		t.Fatalf("ReadOAuth on missing file returned err: %v (want nil)", err)
	}
	if got != nil {
		t.Errorf("got %+v, want nil", got)
	}
}

func TestReadOAuth_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := claudeconfig.ReadOAuth(path)
	if err == nil {
		t.Fatal("ReadOAuth on malformed JSON returned nil err, want error")
	}
}

func TestReadOAuth_EmptyOAuthObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	body := `{"oauthAccount": {}}`
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := claudeconfig.ReadOAuth(path)
	if err != nil {
		t.Fatal(err)
	}
	// Empty object is still technically "no oauth"; treat as nil.
	if got != nil {
		t.Errorf("got %+v, want nil for empty oauthAccount", got)
	}
}

func TestReadOAuth_NullOAuth(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	if err := os.WriteFile(path, []byte(`{"oauthAccount": null}`), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := claudeconfig.ReadOAuth(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("got %+v, want nil for null oauthAccount", got)
	}
}

func TestReadOAuth_NonObjectOAuth(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	if err := os.WriteFile(path, []byte(`{"oauthAccount": "garbage"}`), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := claudeconfig.ReadOAuth(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("got %+v, want nil for non-object oauthAccount", got)
	}
}

func TestMergeOAuth_PreservesOnboardingKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	// This is what bootstrap.WriteOnboarding produces in a fresh env.
	body := `{"hasCompletedOnboarding": true, "hasSeenTasksHint": true, "theme": "dark", "numStartups": 0}`
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}

	oauth := &claudeconfig.OAuth{
		Account:             map[string]any{"emailAddress": "user@example.com"},
		ClaudeCodeFirstDate: "2026-04-20T12:00:00Z",
	}
	if err := claudeconfig.MergeOAuth(path, oauth); err != nil {
		t.Fatalf("MergeOAuth err: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}

	if got["hasCompletedOnboarding"] != true {
		t.Error("hasCompletedOnboarding dropped")
	}
	if got["theme"] != "dark" {
		t.Error("theme dropped")
	}
	if got["claudeCodeFirstTokenDate"] != "2026-04-20T12:00:00Z" {
		t.Errorf("claudeCodeFirstTokenDate = %v, want 2026-04-20T12:00:00Z", got["claudeCodeFirstTokenDate"])
	}
	acct, ok := got["oauthAccount"].(map[string]any)
	if !ok {
		t.Fatalf("oauthAccount missing or wrong type: %T", got["oauthAccount"])
	}
	if acct["emailAddress"] != "user@example.com" {
		t.Errorf("oauthAccount.emailAddress = %v, want user@example.com", acct["emailAddress"])
	}
}

func TestMergeOAuth_OmitsEmptyFirstDate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	if err := os.WriteFile(path, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}

	oauth := &claudeconfig.OAuth{
		Account: map[string]any{"emailAddress": "user@example.com"},
		// ClaudeCodeFirstDate is empty; must not be written.
	}
	if err := claudeconfig.MergeOAuth(path, oauth); err != nil {
		t.Fatal(err)
	}

	raw, _ := os.ReadFile(path)
	var got map[string]any
	json.Unmarshal(raw, &got)
	if _, has := got["claudeCodeFirstTokenDate"]; has {
		t.Error("claudeCodeFirstTokenDate was written when empty; want omitted")
	}
}

func TestMergeOAuth_MissingDestinationFile(t *testing.T) {
	// MergeOAuth only runs after bootstrap.WriteOnboarding, so .claude.json
	// is always expected to exist. But if it doesn't, surface the error.
	path := "/nonexistent/dir/.claude.json"
	err := claudeconfig.MergeOAuth(path, &claudeconfig.OAuth{
		Account: map[string]any{"emailAddress": "user@example.com"},
	})
	if err == nil {
		t.Error("MergeOAuth on missing file returned nil err, want error")
	}
}

func TestMergeTrust_CreatesProjectsKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	if err := os.WriteFile(path, []byte(`{"hasCompletedOnboarding": true}`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := claudeconfig.MergeTrust(path, "/work/dir"); err != nil {
		t.Fatalf("MergeTrust err: %v", err)
	}

	raw, _ := os.ReadFile(path)
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["hasCompletedOnboarding"] != true {
		t.Error("hasCompletedOnboarding dropped")
	}
	projects, ok := got["projects"].(map[string]any)
	if !ok {
		t.Fatalf("projects missing or wrong type: %T", got["projects"])
	}
	entry, ok := projects["/work/dir"].(map[string]any)
	if !ok {
		t.Fatalf("projects[/work/dir] missing: %v", projects)
	}
	if entry["hasTrustDialogAccepted"] != true {
		t.Errorf("hasTrustDialogAccepted = %v, want true", entry["hasTrustDialogAccepted"])
	}
}

func TestMergeTrust_PreservesExistingProjectFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	body := `{
		"projects": {
			"/work/dir": {
				"allowedTools": ["Bash(ls)"],
				"mcpServers": {"foo": {"command": "bar"}},
				"hasTrustDialogAccepted": false
			},
			"/other/dir": {"allowedTools": ["Read"]}
		}
	}`
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}

	if err := claudeconfig.MergeTrust(path, "/work/dir"); err != nil {
		t.Fatal(err)
	}

	raw, _ := os.ReadFile(path)
	var got map[string]any
	json.Unmarshal(raw, &got)
	projects := got["projects"].(map[string]any)

	work := projects["/work/dir"].(map[string]any)
	if work["hasTrustDialogAccepted"] != true {
		t.Error("hasTrustDialogAccepted not flipped to true")
	}
	if tools, _ := work["allowedTools"].([]any); len(tools) != 1 || tools[0] != "Bash(ls)" {
		t.Errorf("allowedTools dropped: %v", work["allowedTools"])
	}
	if _, ok := work["mcpServers"].(map[string]any); !ok {
		t.Error("mcpServers dropped")
	}

	other, ok := projects["/other/dir"].(map[string]any)
	if !ok {
		t.Fatal("/other/dir entry dropped")
	}
	if tools, _ := other["allowedTools"].([]any); len(tools) != 1 || tools[0] != "Read" {
		t.Errorf("/other/dir allowedTools dropped: %v", other["allowedTools"])
	}
}

func TestMergeTrust_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	if err := os.WriteFile(path, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := claudeconfig.MergeTrust(path, "/work/dir"); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(path)

	if err := claudeconfig.MergeTrust(path, "/work/dir"); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)

	if string(first) != string(second) {
		t.Errorf("second call changed file:\nfirst:  %s\nsecond: %s", first, second)
	}
}

func TestMergeTrust_MultiplePaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	if err := os.WriteFile(path, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := claudeconfig.MergeTrust(path, "/a", "/b", "/c"); err != nil {
		t.Fatal(err)
	}

	raw, _ := os.ReadFile(path)
	var got map[string]any
	json.Unmarshal(raw, &got)
	projects := got["projects"].(map[string]any)
	for _, p := range []string{"/a", "/b", "/c"} {
		entry, ok := projects[p].(map[string]any)
		if !ok {
			t.Errorf("%s entry missing", p)
			continue
		}
		if entry["hasTrustDialogAccepted"] != true {
			t.Errorf("%s not trusted", p)
		}
	}
}

func TestMergeTrust_NoPathsIsNoop(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	original := []byte(`{"hasCompletedOnboarding": true}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}

	if err := claudeconfig.MergeTrust(path); err != nil {
		t.Errorf("MergeTrust with no paths returned err: %v", err)
	}

	after, _ := os.ReadFile(path)
	if string(after) != string(original) {
		t.Errorf("file was rewritten when no paths passed:\nbefore: %s\nafter:  %s", original, after)
	}
}

func TestMergeTrust_MissingFile(t *testing.T) {
	err := claudeconfig.MergeTrust("/nonexistent/dir/.claude.json", "/work/dir")
	if err == nil {
		t.Error("MergeTrust on missing file returned nil err, want error")
	}
}

func TestMergeTrust_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude.json")
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	err := claudeconfig.MergeTrust(path, "/work/dir")
	if err == nil {
		t.Error("MergeTrust on malformed JSON returned nil err, want error")
	}
}
