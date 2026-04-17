package main

import "os"

// isTerminal reports whether f refers to a terminal (character device).
// Safe to call with a nil receiver; returns false in that case.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
