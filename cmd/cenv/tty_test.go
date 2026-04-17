package main

import (
	"os"
	"testing"
)

func TestIsTerminal_RegularFile(t *testing.T) {
	f, err := os.CreateTemp("", "cenv-tty-test-")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	if isTerminal(f) {
		t.Error("isTerminal returned true for a regular file, want false")
	}
}

func TestIsTerminal_Nil(t *testing.T) {
	if isTerminal(nil) {
		t.Error("isTerminal returned true for nil, want false")
	}
}
