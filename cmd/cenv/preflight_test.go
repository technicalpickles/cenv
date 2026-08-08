package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreflightEnv_NonexistentEnv(t *testing.T) {
	t.Setenv("CENV_BASE", t.TempDir())

	_, err := preflightEnv("does-not-exist")
	if err == nil {
		t.Fatal("expected error for nonexistent env, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found'", err.Error())
	}
}

func TestPreflightEnv_NoAuth(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "bare-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	_, err := preflightEnv("bare-env")
	if err == nil {
		t.Fatal("expected auth pre-flight error, got nil")
	}
	if !strings.Contains(err.Error(), "cenv login") {
		t.Errorf("error = %q, want it to mention 'cenv login'", err.Error())
	}
}

func TestPreflightEnv_Success(t *testing.T) {
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

	gotDir, err := preflightEnv("authed-env")
	if err != nil {
		t.Fatalf("preflightEnv returned error: %v", err)
	}
	if gotDir != envDir {
		t.Errorf("preflightEnv dir = %q, want %q", gotDir, envDir)
	}
}
