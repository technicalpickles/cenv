package main

import (
	"os"

	"golang.org/x/term"
)

// isTerminal reports whether f is attached to a real terminal. A plain
// character-device check is not enough: /dev/null is a character device too.
// Safe to call with a nil receiver; returns false in that case.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
