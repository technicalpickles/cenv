package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsMergeCmd_SuccessMessageHasSymbol(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)
	envDir := filepath.Join(base, "myenv")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("writing settings.json: %v", err)
	}

	out := captureStderr(t, func() {
		if err := settingsMergeCmd.RunE(settingsMergeCmd, []string{"myenv", `{"foo":"bar"}`}); err != nil {
			t.Fatalf("settings merge err: %v", err)
		}
	})

	if !strings.Contains(out, `✓ Merged settings into "myenv"`) {
		t.Errorf("output = %q, want it to contain the ✓-prefixed success message", out)
	}
}
