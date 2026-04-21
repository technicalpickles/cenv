package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrustCmd_EnvMissing(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	err := trustCmd.RunE(trustCmd, []string{"no-such-env", "/tmp"})
	if err == nil {
		t.Fatal("expected error for missing env, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found'", err.Error())
	}
}

func TestTrustCmd_NoClaudeJSON(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "bare-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatal(err)
	}

	err := trustCmd.RunE(trustCmd, []string{"bare-env", "/tmp"})
	if err == nil {
		t.Fatal("expected error for missing .claude.json, got nil")
	}
	if !strings.Contains(err.Error(), "cenv create") && !strings.Contains(err.Error(), "cenv login") {
		t.Errorf("error = %q, want a hint to run cenv create/login", err.Error())
	}
}

func TestTrustCmd_WritesTrustEntry(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "myenv")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatal(err)
	}
	claudeJSON := filepath.Join(envDir, ".claude.json")
	if err := os.WriteFile(claudeJSON, []byte(`{"hasCompletedOnboarding": true}`), 0600); err != nil {
		t.Fatal(err)
	}

	// Use a real absolute path so we can check it round-trips cleanly.
	workDir := t.TempDir()
	if err := trustCmd.RunE(trustCmd, []string{"myenv", workDir}); err != nil {
		t.Fatalf("trust err: %v", err)
	}

	raw, _ := os.ReadFile(claudeJSON)
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	projects, ok := got["projects"].(map[string]any)
	if !ok {
		t.Fatalf("projects missing: %v", got)
	}
	entry, ok := projects[workDir].(map[string]any)
	if !ok {
		t.Fatalf("projects[%q] missing: %v", workDir, projects)
	}
	if entry["hasTrustDialogAccepted"] != true {
		t.Errorf("hasTrustDialogAccepted = %v, want true", entry["hasTrustDialogAccepted"])
	}
}

func TestTrustCmd_ResolvesRelativePath(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "myenv")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(envDir, ".claude.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}

	// Run the command with a relative path; it should resolve relative to cwd.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := trustCmd.RunE(trustCmd, []string{"myenv", "."}); err != nil {
		t.Fatalf("trust err: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(envDir, ".claude.json"))
	var got map[string]any
	json.Unmarshal(raw, &got)
	projects, _ := got["projects"].(map[string]any)
	if _, ok := projects[cwd]; !ok {
		t.Errorf("expected projects[%q], got keys: %v", cwd, keys(projects))
	}
}

func TestTrustCmd_MultiplePaths(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "myenv")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(envDir, ".claude.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}

	a := t.TempDir()
	b := t.TempDir()
	if err := trustCmd.RunE(trustCmd, []string{"myenv", a, b}); err != nil {
		t.Fatalf("trust err: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(envDir, ".claude.json"))
	var got map[string]any
	json.Unmarshal(raw, &got)
	projects, _ := got["projects"].(map[string]any)
	for _, p := range []string{a, b} {
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

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
