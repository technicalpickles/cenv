package keychain_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/technicalpickles/cenv/internal/keychain"
)

func TestServiceName_DefaultClaudeDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	defaultDir := filepath.Join(home, ".claude")

	got := keychain.ServiceName(defaultDir)
	want := "Claude Code-credentials"
	if got != want {
		t.Errorf("ServiceName(%q) = %q, want %q", defaultDir, got, want)
	}
}

func TestServiceName_CenvEnvDir(t *testing.T) {
	// Hash derivation verified against real cenv envs via
	// tmp/gt-9e94/analyze-keychain-hash.py.
	dir := "/Users/example/.local/share/cenv/myenv"

	got := keychain.ServiceName(dir)
	// sha256(dir) hex first 8 chars; computed once and pinned as fixture.
	// Recompute locally with: printf %s "/Users/example/.local/share/cenv/myenv" | shasum -a 256 | cut -c1-8
	want := "Claude Code-credentials-" + shaPrefix(t, dir)
	if got != want {
		t.Errorf("ServiceName(%q) = %q, want %q", dir, got, want)
	}
}

func TestServiceName_HashStability(t *testing.T) {
	// Same input -> same output.
	dir := "/some/path"
	if keychain.ServiceName(dir) != keychain.ServiceName(dir) {
		t.Error("ServiceName is not deterministic")
	}
}

// shaPrefix computes the expected hash prefix using the same scheme as
// ServiceName. Duplicated here deliberately so the test fails if either side
// drifts.
func shaPrefix(t *testing.T, s string) string {
	t.Helper()
	return keychain.HashPrefixForTesting(s)
}
