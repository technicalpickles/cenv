package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrustCmd_SuccessMessageHasSymbol(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)
	envDir := filepath.Join(base, "myenv")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, ".claude.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("writing .claude.json: %v", err)
	}

	out := captureStderr(t, func() {
		if err := trustCmd.RunE(trustCmd, []string{"myenv", "/tmp/some/path"}); err != nil {
			t.Fatalf("trust err: %v", err)
		}
	})

	if !strings.Contains(out, "✓ Trusted") {
		t.Errorf("output = %q, want it to contain the ✓-prefixed success message", out)
	}
}
