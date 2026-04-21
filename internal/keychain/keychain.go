// Package keychain wraps macOS Security framework operations for storing
// and retrieving Claude Code OAuth tokens. The service-name scheme mirrors
// claude-code itself: "Claude Code-credentials" for ~/.claude, or
// "Claude Code-credentials-<first 8 hex of sha256(configDir)>" for any other
// config dir.
package keychain

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
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
