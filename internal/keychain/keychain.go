// Package keychain wraps macOS Security framework operations for storing
// and retrieving Claude Code OAuth tokens. The service-name scheme mirrors
// claude-code itself: "Claude Code-credentials" for ~/.claude, or
// "Claude Code-credentials-<first 8 hex of sha256(configDir)>" for any other
// config dir.
package keychain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

// ServiceName returns the keychain service name claude-code uses for the
// given config dir. The default ~/.claude dir uses an unhashed name; all
// others use a hashed suffix.
func ServiceName(configDir string) string {
	home, err := os.UserHomeDir()
	if err == nil && configDir == filepath.Join(home, ".claude") {
		return "Claude Code-credentials"
	}
	return "Claude Code-credentials-" + hashPrefix(configDir)
}

// HashPrefixForTesting exposes the hash derivation for tests. Not part of the
// public API for callers.
func HashPrefixForTesting(s string) string {
	return hashPrefix(s)
}

func hashPrefix(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:8]
}

// ExitError represents a failed `security` CLI invocation. Code is the
// process exit code (44 = item not found). Stderr contains captured stderr
// bytes for surfacing upstream.
type ExitError struct {
	Code   int
	Stderr []byte
}

func (e *ExitError) Error() string {
	msg := strings.TrimSpace(string(e.Stderr))
	if msg == "" {
		return fmt.Sprintf("security exit %d", e.Code)
	}
	return fmt.Sprintf("security exit %d: %s", e.Code, msg)
}

// Runner abstracts invocation of the `security` CLI so tests can stub it.
// Implementations receive the argument list (without the "security" prefix)
// and return stdout + an error. Non-zero exits must return an *ExitError.
type Runner interface {
	Run(args ...string) ([]byte, error)
}

// execRunner is the default Runner that shells out to /usr/bin/security.
type execRunner struct{}

func (execRunner) Run(args ...string) ([]byte, error) {
	cmd := exec.Command("security", args...)
	out, err := cmd.Output()
	if ee, ok := err.(*exec.ExitError); ok {
		return out, &ExitError{Code: ee.ExitCode(), Stderr: ee.Stderr}
	}
	return out, err
}

// Client performs keychain operations via a Runner. Use Default for normal
// calls; inject a mock Runner in tests.
type Client struct {
	Runner Runner
}

// Default is a Client that runs /usr/bin/security.
var Default = &Client{Runner: execRunner{}}

func (c *Client) run(args ...string) ([]byte, error) {
	r := c.Runner
	if r == nil {
		r = execRunner{}
	}
	return r.Run(args...)
}

// Read returns the token stored under service. If the entry does not exist,
// returns notFound=true with a nil error (this is not a failure — the source
// env simply isn't authed). Any other non-zero exit is surfaced as an error.
func (c *Client) Read(service string) (token string, notFound bool, err error) {
	acct, err := currentUser()
	if err != nil {
		return "", false, err
	}
	out, err := c.run("find-generic-password", "-a", acct, "-s", service, "-w")
	if err != nil {
		if ee, ok := err.(*ExitError); ok && ee.Code == 44 {
			return "", true, nil
		}
		return "", false, fmt.Errorf("reading keychain entry %q: %w", service, err)
	}
	return strings.TrimRight(string(out), "\n"), false, nil
}

// Write stores token under service. Because `security add-generic-password`
// does not upsert, Write first deletes any existing entry (ignoring
// not-found) before adding.
func (c *Client) Write(service, token string) error {
	acct, err := currentUser()
	if err != nil {
		return err
	}
	if _, err := c.run("delete-generic-password", "-a", acct, "-s", service); err != nil {
		if ee, ok := err.(*ExitError); !ok || ee.Code != 44 {
			return fmt.Errorf("clearing existing keychain entry %q: %w", service, err)
		}
	}
	if _, err := c.run("add-generic-password", "-a", acct, "-s", service, "-w", token); err != nil {
		return fmt.Errorf("adding keychain entry %q: %w", service, err)
	}
	return nil
}

// Delete removes the keychain entry for service. Not-found (exit 44) is not
// an error — the goal is "ensure no entry exists."
func (c *Client) Delete(service string) error {
	acct, err := currentUser()
	if err != nil {
		return err
	}
	if _, err := c.run("delete-generic-password", "-a", acct, "-s", service); err != nil {
		if ee, ok := err.(*ExitError); ok && ee.Code == 44 {
			return nil
		}
		return fmt.Errorf("deleting keychain entry %q: %w", service, err)
	}
	return nil
}

func currentUser() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("looking up current user: %w", err)
	}
	return u.Username, nil
}
