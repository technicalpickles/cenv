package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecCmd_NonexistentEnv(t *testing.T) {
	t.Setenv("CENV_BASE", t.TempDir())

	err := execCmd.RunE(execCmd, []string{"does-not-exist", "--", "true"})
	if err == nil {
		t.Fatal("expected error for nonexistent env, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found'", err.Error())
	}
}

func TestExecCmd_NoAuth(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "bare-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	err := execCmd.RunE(execCmd, []string{"bare-env", "--", "true"})
	if err == nil {
		t.Fatal("expected auth pre-flight error, got nil")
	}
	if !strings.Contains(err.Error(), "cenv login") {
		t.Errorf("error = %q, want it to mention 'cenv login'", err.Error())
	}
}

func TestExecCmd_MissingSeparator(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "authed-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	settingsJSON := `{"awsAuthRefresh": {"region": "us-east-1"}}`
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	err := execCmd.RunE(execCmd, []string{"authed-env"})
	if err == nil {
		t.Fatal("expected missing-command error, got nil")
	}
	if !strings.Contains(err.Error(), "missing command") {
		t.Errorf("error = %q, want it to mention 'missing command'", err.Error())
	}
}

func TestExecCmd_MissingCommandAfterSeparator(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "authed-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	settingsJSON := `{"awsAuthRefresh": {"region": "us-east-1"}}`
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	err := execCmd.RunE(execCmd, []string{"authed-env", "--"})
	if err == nil {
		t.Fatal("expected missing-command error, got nil")
	}
	if !strings.Contains(err.Error(), "missing command") {
		t.Errorf("error = %q, want it to mention 'missing command'", err.Error())
	}
}

func TestExecCmd_CommandNotFound(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "authed-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	settingsJSON := `{"awsAuthRefresh": {"region": "us-east-1"}}`
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	err := execCmd.RunE(execCmd, []string{"authed-env", "--", "definitely-not-a-real-binary-xyz"})
	if err == nil {
		t.Fatal("expected command-not-found error, got nil")
	}
	if !strings.Contains(err.Error(), "not found in PATH") {
		t.Errorf("error = %q, want it to mention 'not found in PATH'", err.Error())
	}
}
