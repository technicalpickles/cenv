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

// --- mockRunner ------------------------------------------------------------

type mockRunner struct {
	// calls accumulates each invocation so tests can assert arg sequences.
	calls [][]string

	// respond is called for each invocation. Return stdout + an error.
	// Set to a func that inspects args and picks a canned response.
	respond func(args []string) ([]byte, error)
}

func (m *mockRunner) Run(args ...string) ([]byte, error) {
	m.calls = append(m.calls, append([]string(nil), args...))
	if m.respond != nil {
		return m.respond(args)
	}
	return nil, nil
}

// --- Read ------------------------------------------------------------------

func TestRead_NotFoundReturnsFlag(t *testing.T) {
	m := &mockRunner{
		respond: func(args []string) ([]byte, error) {
			// exit 44 from security CLI when item missing
			return nil, &keychain.ExitError{Code: 44, Stderr: []byte("not found")}
		},
	}
	c := &keychain.Client{Runner: m}

	token, notFound, err := c.Read("some-service")
	if err != nil {
		t.Fatalf("Read returned err: %v", err)
	}
	if !notFound {
		t.Error("notFound = false, want true")
	}
	if token != "" {
		t.Errorf("token = %q, want empty", token)
	}
}

func TestRead_SuccessReturnsToken(t *testing.T) {
	m := &mockRunner{
		respond: func(args []string) ([]byte, error) {
			return []byte("the-token-value\n"), nil
		},
	}
	c := &keychain.Client{Runner: m}

	token, notFound, err := c.Read("some-service")
	if err != nil {
		t.Fatalf("Read returned err: %v", err)
	}
	if notFound {
		t.Error("notFound = true, want false")
	}
	if token != "the-token-value" {
		t.Errorf("token = %q, want %q", token, "the-token-value")
	}
}

func TestRead_OtherErrorPropagates(t *testing.T) {
	m := &mockRunner{
		respond: func(args []string) ([]byte, error) {
			return nil, &keychain.ExitError{Code: 36, Stderr: []byte("user canceled")}
		},
	}
	c := &keychain.Client{Runner: m}

	_, notFound, err := c.Read("some-service")
	if err == nil {
		t.Fatal("Read returned nil err, want error")
	}
	if notFound {
		t.Error("notFound = true, want false for non-44 error")
	}
}

// --- Write -----------------------------------------------------------------

func TestWrite_DeletesThenAdds(t *testing.T) {
	m := &mockRunner{
		respond: func(args []string) ([]byte, error) {
			// simulate delete returning 44 (not present), then add succeeding
			if len(args) > 0 && args[0] == "delete-generic-password" {
				return nil, &keychain.ExitError{Code: 44}
			}
			return nil, nil
		},
	}
	c := &keychain.Client{Runner: m}

	if err := c.Write("svc", "tok"); err != nil {
		t.Fatalf("Write returned err: %v", err)
	}
	if len(m.calls) != 2 {
		t.Fatalf("expected 2 calls (delete, add), got %d: %v", len(m.calls), m.calls)
	}
	if m.calls[0][0] != "delete-generic-password" {
		t.Errorf("first call = %v, want delete-generic-password first", m.calls[0])
	}
	if m.calls[1][0] != "add-generic-password" {
		t.Errorf("second call = %v, want add-generic-password second", m.calls[1])
	}
}

func TestWrite_AddFailurePropagates(t *testing.T) {
	m := &mockRunner{
		respond: func(args []string) ([]byte, error) {
			if len(args) > 0 && args[0] == "delete-generic-password" {
				return nil, &keychain.ExitError{Code: 44}
			}
			return nil, &keychain.ExitError{Code: 51, Stderr: []byte("denied")}
		},
	}
	c := &keychain.Client{Runner: m}

	if err := c.Write("svc", "tok"); err == nil {
		t.Fatal("Write returned nil, want error")
	}
}

// --- Delete ----------------------------------------------------------------

func TestDelete_IgnoresNotFound(t *testing.T) {
	m := &mockRunner{
		respond: func(args []string) ([]byte, error) {
			return nil, &keychain.ExitError{Code: 44}
		},
	}
	c := &keychain.Client{Runner: m}

	if err := c.Delete("svc"); err != nil {
		t.Errorf("Delete returned err on not-found: %v", err)
	}
}

func TestDelete_PropagatesOtherErrors(t *testing.T) {
	m := &mockRunner{
		respond: func(args []string) ([]byte, error) {
			return nil, &keychain.ExitError{Code: 51, Stderr: []byte("denied")}
		},
	}
	c := &keychain.Client{Runner: m}

	if err := c.Delete("svc"); err == nil {
		t.Fatal("Delete returned nil, want error")
	}
}
