package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCmd_NoAuth(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	// Create an env with settings.json but no auth configured.
	envDir := filepath.Join(base, "bare-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	err := runCmd.RunE(runCmd, []string{"bare-env"})
	if err == nil {
		t.Fatal("expected auth pre-flight error, got nil")
	}
	if !strings.Contains(err.Error(), "cenv login") {
		t.Errorf("error = %q, want it to mention 'cenv login'", err.Error())
	}
}
