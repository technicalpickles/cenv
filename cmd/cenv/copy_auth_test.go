package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/technicalpickles/cenv/internal/keychain"
)

// fakeKeychain implements keychain.Runner for tests. It keeps an in-memory
// map of service -> token so Read/Write round-trip cleanly.
type fakeKeychain struct {
	entries map[string]string
	// readErr, if non-nil, is returned from every Read (non-44).
	readErr error
}

func (f *fakeKeychain) Run(args ...string) ([]byte, error) {
	if len(args) == 0 {
		return nil, nil
	}
	switch args[0] {
	case "find-generic-password":
		// args: find-generic-password -a <acct> -s <svc> -w
		svc := extractFlag(args, "-s")
		if f.readErr != nil {
			return nil, f.readErr
		}
		tok, ok := f.entries[svc]
		if !ok {
			return nil, &keychain.ExitError{Code: 44}
		}
		return []byte(tok + "\n"), nil
	case "delete-generic-password":
		svc := extractFlag(args, "-s")
		if _, ok := f.entries[svc]; !ok {
			return nil, &keychain.ExitError{Code: 44}
		}
		delete(f.entries, svc)
		return nil, nil
	case "add-generic-password":
		svc := extractFlag(args, "-s")
		tok := extractFlag(args, "-w")
		if f.entries == nil {
			f.entries = map[string]string{}
		}
		f.entries[svc] = tok
		return nil, nil
	}
	return nil, nil
}

func extractFlag(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// --- copyAuth tests --------------------------------------------------------

func TestCopyAuth_SourceNotAuthed_NoOp(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	// dst has onboarding written (simulate bootstrap.WriteOnboarding result)
	writeJSON(t, filepath.Join(dstDir, ".claude.json"), map[string]any{
		"hasCompletedOnboarding": true,
	})
	// src has no .claude.json and no keychain entry

	kc := &keychain.Client{Runner: &fakeKeychain{}}
	copied, err := copyAuth(srcDir, filepath.Join(srcDir, ".claude.json"), dstDir, kc)
	if err != nil {
		t.Fatalf("copyAuth err: %v", err)
	}
	if copied {
		t.Error("copied = true, want false for unauthed source")
	}
	// dst .claude.json must be untouched (no oauthAccount)
	got := readJSON(t, filepath.Join(dstDir, ".claude.json"))
	if _, has := got["oauthAccount"]; has {
		t.Error("dst gained oauthAccount despite unauthed source")
	}
}

func TestCopyAuth_HappyPath(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Source has .claude.json with oauth + a matching keychain token.
	writeJSON(t, filepath.Join(srcDir, ".claude.json"), map[string]any{
		"oauthAccount":             map[string]any{"emailAddress": "user@example.com"},
		"claudeCodeFirstTokenDate": "2026-04-20T12:00:00Z",
	})
	srcSvc := keychain.ServiceName(srcDir)
	fk := &fakeKeychain{entries: map[string]string{srcSvc: "token-abc"}}

	// Destination has fresh onboarding.
	writeJSON(t, filepath.Join(dstDir, ".claude.json"), map[string]any{
		"hasCompletedOnboarding": true,
	})

	kc := &keychain.Client{Runner: fk}
	copied, err := copyAuth(srcDir, filepath.Join(srcDir, ".claude.json"), dstDir, kc)
	if err != nil {
		t.Fatalf("copyAuth err: %v", err)
	}
	if !copied {
		t.Error("copied = false, want true for happy path")
	}

	// dst keychain has the token under the destination service name
	dstSvc := keychain.ServiceName(dstDir)
	if fk.entries[dstSvc] != "token-abc" {
		t.Errorf("dst keychain token = %q, want token-abc", fk.entries[dstSvc])
	}

	// dst .claude.json has oauthAccount AND preserved onboarding
	got := readJSON(t, filepath.Join(dstDir, ".claude.json"))
	if got["hasCompletedOnboarding"] != true {
		t.Error("hasCompletedOnboarding lost")
	}
	acct, _ := got["oauthAccount"].(map[string]any)
	if acct["emailAddress"] != "user@example.com" {
		t.Errorf("oauthAccount.emailAddress = %v, want user@example.com", acct["emailAddress"])
	}
	if got["claudeCodeFirstTokenDate"] != "2026-04-20T12:00:00Z" {
		t.Error("firstTokenDate not copied")
	}
}

func TestCopyAuth_HasOAuthButNoKeychain_Skips(t *testing.T) {
	// Half-authed source: oauthAccount present but keychain missing.
	// Spec: skip OAuth copy entirely rather than write a partial state.
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	writeJSON(t, filepath.Join(srcDir, ".claude.json"), map[string]any{
		"oauthAccount": map[string]any{"emailAddress": "user@example.com"},
	})
	writeJSON(t, filepath.Join(dstDir, ".claude.json"), map[string]any{
		"hasCompletedOnboarding": true,
	})

	kc := &keychain.Client{Runner: &fakeKeychain{entries: map[string]string{}}}
	copied, err := copyAuth(srcDir, filepath.Join(srcDir, ".claude.json"), dstDir, kc)
	if err != nil {
		t.Fatalf("copyAuth err: %v", err)
	}
	if copied {
		t.Error("copied = true, want false for half-authed source")
	}
	got := readJSON(t, filepath.Join(dstDir, ".claude.json"))
	if _, has := got["oauthAccount"]; has {
		t.Error("dst gained oauthAccount despite missing keychain token")
	}
}

func TestCopyAuth_MergeFailure_RollsBackKeychain(t *testing.T) {
	// If MergeOAuth fails after the keychain write succeeded, the keychain
	// entry must be rolled back so cenv's has_auth check doesn't report
	// true for an env whose .claude.json lacks oauthAccount.
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	writeJSON(t, filepath.Join(srcDir, ".claude.json"), map[string]any{
		"oauthAccount": map[string]any{"emailAddress": "user@example.com"},
	})
	srcSvc := keychain.ServiceName(srcDir)
	fk := &fakeKeychain{entries: map[string]string{srcSvc: "token-abc"}}

	// Destination has NO .claude.json at all, so MergeOAuth will error on
	// read. This triggers the rollback path.

	kc := &keychain.Client{Runner: fk}
	copied, err := copyAuth(srcDir, filepath.Join(srcDir, ".claude.json"), dstDir, kc)
	if err == nil {
		t.Fatal("copyAuth returned nil err, want error (no dst .claude.json)")
	}
	if copied {
		t.Error("copied = true, want false on merge failure")
	}
	dstSvc := keychain.ServiceName(dstDir)
	if _, present := fk.entries[dstSvc]; present {
		t.Errorf("keychain entry for dst still present after rollback: %v", fk.entries)
	}
}

func TestCopyAuth_UserStyleClaudeJSONLocation(t *testing.T) {
	// User case: srcConfigDir is like ~/.claude but srcClaudeJSON is at
	// HOME root (~/.claude.json), not inside the config dir. Regression
	// test for the asymmetry fix.
	homeLike := t.TempDir()
	srcConfigDir := filepath.Join(homeLike, ".claude")
	srcClaudeJSON := filepath.Join(homeLike, ".claude.json") // at home root
	if err := os.MkdirAll(srcConfigDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, srcClaudeJSON, map[string]any{
		"oauthAccount": map[string]any{"emailAddress": "user@example.com"},
	})

	srcSvc := keychain.ServiceName(srcConfigDir)
	fk := &fakeKeychain{entries: map[string]string{srcSvc: "token-xyz"}}

	dstDir := t.TempDir()
	writeJSON(t, filepath.Join(dstDir, ".claude.json"), map[string]any{
		"hasCompletedOnboarding": true,
	})

	kc := &keychain.Client{Runner: fk}
	copied, err := copyAuth(srcConfigDir, srcClaudeJSON, dstDir, kc)
	if err != nil {
		t.Fatalf("copyAuth err: %v", err)
	}
	if !copied {
		t.Error("copied = false, want true when user-style oauth is present")
	}

	dstSvc := keychain.ServiceName(dstDir)
	if fk.entries[dstSvc] != "token-xyz" {
		t.Errorf("dst keychain = %q, want token-xyz", fk.entries[dstSvc])
	}

	got := readJSON(t, filepath.Join(dstDir, ".claude.json"))
	acct, _ := got["oauthAccount"].(map[string]any)
	if acct["emailAddress"] != "user@example.com" {
		t.Error("dst oauthAccount not merged from user-style claude.json")
	}
}

// --- helpers ---------------------------------------------------------------

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}
