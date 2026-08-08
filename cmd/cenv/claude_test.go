// cmd/cenv/claude_test.go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeCmd_NonexistentEnv(t *testing.T) {
	t.Setenv("CENV_BASE", t.TempDir())

	err := claudeCmd.RunE(claudeCmd, []string{"does-not-exist"})
	if err == nil {
		t.Fatal("expected error for nonexistent env, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found'", err.Error())
	}
}

func TestClaudeCmd_NoAuth(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "bare-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	err := claudeCmd.RunE(claudeCmd, []string{"bare-env"})
	if err == nil {
		t.Fatal("expected auth pre-flight error, got nil")
	}
	if !strings.Contains(err.Error(), "cenv login") {
		t.Errorf("error = %q, want it to mention 'cenv login'", err.Error())
	}
}

func TestRunCmd_DeprecatedAlias(t *testing.T) {
	if runCmd.Deprecated == "" {
		t.Error("runCmd.Deprecated is empty, want a deprecation message")
	}

	t.Setenv("CENV_BASE", t.TempDir())

	err := runCmd.RunE(runCmd, []string{"does-not-exist"})
	if err == nil {
		t.Fatal("expected error for nonexistent env, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found' (runCmd should behave like claudeCmd)", err.Error())
	}
}
