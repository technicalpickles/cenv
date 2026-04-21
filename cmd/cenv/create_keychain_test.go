//go:build keychain

package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/technicalpickles/cenv/internal/keychain"
)

// These tests hit the real macOS keychain. They're guarded by the "keychain"
// build tag so CI skips them by default. Run locally with:
//
//	go test -tags keychain ./cmd/cenv/...
//
// First run will prompt for keychain access (the service names are new).
// Click "Always Allow" to avoid repeated prompts during development.

func TestCreate_CopiesOAuthFromFakeSource_RealKeychain(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("CENV_BASE", t.TempDir())

	// Seed a "source" env that's already authed.
	srcName := "copyauth-src-" + randSuffix(t)
	srcDir := filepath.Join(os.Getenv("CENV_BASE"), srcName)
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeSourceOAuth(t, srcDir, "user@example.com", "test-token-value")

	// Ensure cleanup of the source keychain entry regardless of test outcome.
	t.Cleanup(func() {
		_ = keychain.Default.Delete(keychain.ServiceName(srcDir))
	})

	// Compute dst name and dir, and pre-register cleanup BEFORE calling RunE
	// to ensure keychain entry is cleaned even if RunE fails after writing.
	dstName := "copyauth-dst-" + randSuffix(t)
	dstDir := filepath.Join(os.Getenv("CENV_BASE"), dstName)
	t.Cleanup(func() {
		_ = keychain.Default.Delete(keychain.ServiceName(dstDir))
	})

	// Run cenv create <dst> --from <src>.
	createFrom = srcName
	defer func() { createFrom = "" }()
	if err := createCmd.RunE(createCmd, []string{dstName}); err != nil {
		t.Fatalf("create returned err: %v", err)
	}

	// dst .claude.json has oauth.
	raw, err := os.ReadFile(filepath.Join(dstDir, ".claude.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	acct, ok := got["oauthAccount"].(map[string]any)
	if !ok {
		t.Fatalf("dst .claude.json missing oauthAccount: %v", got)
	}
	if acct["emailAddress"] != "user@example.com" {
		t.Errorf("oauthAccount.emailAddress = %v, want user@example.com", acct["emailAddress"])
	}

	// dst keychain has the token.
	token, notFound, err := keychain.Default.Read(keychain.ServiceName(dstDir))
	if err != nil {
		t.Fatalf("reading dst keychain: %v", err)
	}
	if notFound {
		t.Fatal("dst keychain entry not found")
	}
	if token != "test-token-value" {
		t.Errorf("dst keychain token = %q, want test-token-value", token)
	}
}

func TestCreate_NoOAuthSource_SkipsSilently_RealKeychain(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("CENV_BASE", t.TempDir())

	// Source is a cenv env with no OAuth (no .claude.json auth, no keychain entry).
	srcName := "copyauth-bare-" + randSuffix(t)
	srcDir := filepath.Join(os.Getenv("CENV_BASE"), srcName)
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	// settings.json is required by createCmd's --from branch.
	if err := os.WriteFile(filepath.Join(srcDir, "settings.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(srcDir, ".claude.json"),
		[]byte(`{"hasCompletedOnboarding": true}`),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	dstName := "copyauth-dst2-" + randSuffix(t)
	dstDir := filepath.Join(os.Getenv("CENV_BASE"), dstName)
	t.Cleanup(func() {
		_ = keychain.Default.Delete(keychain.ServiceName(dstDir))
	})

	createFrom = srcName
	defer func() { createFrom = "" }()
	if err := createCmd.RunE(createCmd, []string{dstName}); err != nil {
		t.Fatalf("create returned err: %v", err)
	}

	// dst .claude.json must NOT have oauthAccount.
	raw, err := os.ReadFile(filepath.Join(dstDir, ".claude.json"))
	if err != nil {
		t.Fatalf("reading dst .claude.json: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("parsing dst .claude.json: %v", err)
	}
	if _, has := got["oauthAccount"]; has {
		t.Error("dst gained oauthAccount despite unauthed source")
	}
	// dst keychain must be empty.
	_, notFound, err := keychain.Default.Read(keychain.ServiceName(dstDir))
	if err != nil {
		t.Fatalf("read err: %v", err)
	}
	if !notFound {
		t.Error("dst keychain entry exists despite unauthed source")
	}
}

// --- helpers ---------------------------------------------------------------

// writeSourceOAuth seeds a fully-authed source env: settings.json, .claude.json
// with oauthAccount, and a matching keychain entry.
func writeSourceOAuth(t *testing.T, srcDir, email, token string) {
	t.Helper()
	// settings.json is required by createCmd's --from branch.
	if err := os.WriteFile(filepath.Join(srcDir, "settings.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	body := map[string]any{
		"oauthAccount":             map[string]any{"emailAddress": email},
		"claudeCodeFirstTokenDate": "2026-04-20T12:00:00Z",
		"hasCompletedOnboarding":   true,
	}
	b, _ := json.MarshalIndent(body, "", "  ")
	if err := os.WriteFile(filepath.Join(srcDir, ".claude.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if err := keychain.Default.Write(keychain.ServiceName(srcDir), token); err != nil {
		t.Fatalf("seeding source keychain: %v", err)
	}
}

func randSuffix(t *testing.T) string {
	t.Helper()
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", b)
}
