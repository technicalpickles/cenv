package claudeconfig_test

import (
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
	// Empty object is still technically "no oauth" — treat as nil.
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
