package main

import (
	"os"
	"strings"
	"testing"
)

func TestLoginCmd_NonexistentEnv(t *testing.T) {
	t.Setenv("CENV_BASE", t.TempDir())

	err := loginCmd.RunE(loginCmd, []string{"does-not-exist"})
	if err == nil {
		t.Fatal("expected error for nonexistent env, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found'", err.Error())
	}
}

func TestLoginCmd_RequiresTTY(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)
	if err := os.MkdirAll(base+"/test-env", 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}

	// Redirect os.Stdin to a pipe so isTerminal returns false regardless of
	// whether the test runner has a real TTY attached.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	t.Cleanup(func() { r.Close(); w.Close() })
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig })

	runErr := loginCmd.RunE(loginCmd, []string{"test-env"})
	if runErr == nil {
		t.Fatal("expected TTY error, got nil")
	}
	if !strings.Contains(runErr.Error(), "terminal") {
		t.Errorf("error = %q, want it to mention 'terminal'", runErr.Error())
	}
}
