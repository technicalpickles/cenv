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

func TestClaudeCmd_LoginBypassesAuthPreflight(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)
	// Empty PATH so we stop at LookPath instead of exec'ing a real claude.
	t.Setenv("PATH", "")

	envDir := filepath.Join(base, "bare-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	err := claudeCmd.RunE(claudeCmd, []string{"bare-env", "--", "/login"})
	if err == nil {
		t.Fatal("expected claude-not-found error, got nil")
	}
	if strings.Contains(err.Error(), "no auth configured") {
		t.Errorf("auth pre-flight blocked /login: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "claude not found") {
		t.Errorf("error = %q, want it to reach the claude lookup", err.Error())
	}
}

func TestClaudeCmd_LoginBypassStillRequiresEnv(t *testing.T) {
	t.Setenv("CENV_BASE", t.TempDir())

	err := claudeCmd.RunE(claudeCmd, []string{"nope", "--", "/login"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error = %v, want env not found", err)
	}
}

func TestIsLoginRecovery(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{[]string{"/login"}, true},
		{nil, false},
		{[]string{"/status"}, false},
		{[]string{"/login", "extra"}, false},
		{[]string{"--model", "opus"}, false},
	}
	for _, c := range cases {
		if got := isLoginRecovery(c.args); got != c.want {
			t.Errorf("isLoginRecovery(%v) = %v, want %v", c.args, got, c.want)
		}
	}
}
