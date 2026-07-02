package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfirmation(t *testing.T) {
	cases := map[string]bool{
		"y\n":   true,
		"y":     true,
		"Y\n":   true,
		"yes\n": true,
		"YES":   true,
		" y \n": true,
		"n\n":   false,
		"no\n":  false,
		"\n":    false,
		"":      false,
		"maybe": false,
	}
	for input, want := range cases {
		if got := parseConfirmation(input); got != want {
			t.Errorf("parseConfirmation(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestRemoveCmd_ForceFlag_SkipsPrompt(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)
	envDir := filepath.Join(base, "myenv")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}

	removeForce = true
	t.Cleanup(func() { removeForce = false })

	if err := removeCmd.RunE(removeCmd, []string{"myenv"}); err != nil {
		t.Fatalf("remove err: %v", err)
	}
	if _, err := os.Stat(envDir); !os.IsNotExist(err) {
		t.Error("env dir still exists after forced remove")
	}
}

func TestRemoveCmd_NonTTY_ProceedsWithoutPrompt(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)
	envDir := filepath.Join(base, "myenv")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}

	// Redirect stdin to a pipe so isTerminal returns false, simulating a
	// script/CI invocation with no interactive terminal attached.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	t.Cleanup(func() { r.Close(); w.Close() })
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig })

	removeForce = false
	if err := removeCmd.RunE(removeCmd, []string{"myenv"}); err != nil {
		t.Fatalf("remove err: %v", err)
	}
	if _, err := os.Stat(envDir); !os.IsNotExist(err) {
		t.Error("env dir still exists after non-TTY remove")
	}
}

func TestRemoveCmd_SuccessMessageHasSymbol(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)
	envDir := filepath.Join(base, "myenv")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	removeForce = true
	t.Cleanup(func() { removeForce = false })

	out := captureStderr(t, func() {
		if err := removeCmd.RunE(removeCmd, []string{"myenv"}); err != nil {
			t.Fatalf("remove err: %v", err)
		}
	})

	if !strings.Contains(out, `✓ Removed environment "myenv"`) {
		t.Errorf("output = %q, want it to contain the ✓-prefixed success message", out)
	}
}
