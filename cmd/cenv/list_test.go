package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestListCmd_PlainOutputShowsAuthStatus(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	authed := filepath.Join(base, "authed")
	if err := os.MkdirAll(authed, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authed, ".claude.json"), []byte(`{"oauthAccount":"user@example.com"}`), 0644); err != nil {
		t.Fatal(err)
	}

	bare := filepath.Join(base, "bare")
	if err := os.MkdirAll(bare, 0755); err != nil {
		t.Fatal(err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })
	origListJSON := listJSON
	listJSON = false
	t.Cleanup(func() { listJSON = origListJSON })

	runErr := listCmd.RunE(listCmd, nil)
	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()

	if runErr != nil {
		t.Fatalf("list err: %v", runErr)
	}

	if !strings.Contains(out, "NAME") || !strings.Contains(out, "AUTH") {
		t.Fatalf("output missing header row: %q", out)
	}

	var authedLine, bareLine string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		switch {
		case strings.HasPrefix(line, "authed"):
			authedLine = line
		case strings.HasPrefix(line, "bare"):
			bareLine = line
		}
	}
	if !strings.Contains(authedLine, "yes") {
		t.Errorf("authed line = %q, want it to contain 'yes'", authedLine)
	}
	if !strings.Contains(bareLine, "no") {
		t.Errorf("bare line = %q, want it to contain 'no'", bareLine)
	}
}

func TestListCmd_ColoredOutputHasNoSymbolClutter(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	authed := filepath.Join(base, "authed")
	if err := os.MkdirAll(authed, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authed, ".claude.json"), []byte(`{"oauthAccount":"user@example.com"}`), 0644); err != nil {
		t.Fatal(err)
	}

	origNoColor := color.NoColor
	color.NoColor = false
	t.Cleanup(func() { color.NoColor = origNoColor })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	listJSON = false

	runErr := listCmd.RunE(listCmd, nil)
	w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()

	if runErr != nil {
		t.Fatalf("list err: %v", runErr)
	}
	if !strings.Contains(out, "\x1b[32m") {
		t.Errorf("output = %q, want it to contain a green ANSI code for the authed row", out)
	}
	if strings.Contains(out, "✓") || strings.Contains(out, "✗") {
		t.Errorf("output = %q, table cells should not have status symbols", out)
	}
}
