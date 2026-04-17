package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthCreateCmd_RefusesOAuth(t *testing.T) {
	// Redirect HOME so Detect reads our fake .claude dir.
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("CENV_BASE", t.TempDir())

	claudeDir := filepath.Join(fakeHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("creating claude dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}
	oauth := `{"oauthAccount": {"emailAddress": "user@example.com"}}`
	if err := os.WriteFile(filepath.Join(claudeDir, ".claude.json"), []byte(oauth), 0644); err != nil {
		t.Fatalf("writing .claude.json: %v", err)
	}

	err := authCreateCmd.RunE(authCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected refusal for OAuth user, got nil")
	}
	if !strings.Contains(err.Error(), "cenv login") {
		t.Errorf("error = %q, want it to mention 'cenv login'", err.Error())
	}
}

func TestAuthCreateCmd_AcceptsBedrock(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("CENV_BASE", t.TempDir())

	claudeDir := filepath.Join(fakeHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("creating claude dir: %v", err)
	}
	bedrock := `{"awsAuthRefresh": {"region": "us-west-2"}}`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(bedrock), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	err := authCreateCmd.RunE(authCreateCmd, []string{})
	if err != nil {
		t.Fatalf("Bedrock auth create unexpectedly failed: %v", err)
	}
}
