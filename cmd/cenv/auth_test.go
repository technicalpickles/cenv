package main

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/technicalpickles/cenv/internal/claudeauth"
)

type scriptedRunner struct {
	status claudeauth.Result
	probe  claudeauth.Result
	calls  [][]string
}

func (s *scriptedRunner) Run(ctx context.Context, configDir string, args ...string) (claudeauth.Result, error) {
	s.calls = append(s.calls, args)
	if len(args) > 0 && args[0] == "auth" {
		return s.status, nil
	}
	return s.probe, nil
}

func (s *scriptedRunner) probed() bool {
	for _, c := range s.calls {
		if slices.Equal(c, claudeauth.ProbeArgs) {
			return true
		}
	}
	return false
}

func setupAuthTest(t *testing.T, r *scriptedRunner) {
	t.Helper()
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)
	if err := os.MkdirAll(filepath.Join(base, "myenv"), 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}

	origClient := authClient
	authClient = &claudeauth.Client{Runner: r}
	t.Cleanup(func() { authClient = origClient })

	origLive, origText := authStatusLive, authStatusText
	t.Cleanup(func() { authStatusLive, authStatusText = origLive, origText })
	authStatusLive, authStatusText = false, false
}

func TestAuthStatus_NonexistentEnv(t *testing.T) {
	setupAuthTest(t, &scriptedRunner{})

	err := authStatusCmd.RunE(authStatusCmd, []string{"does-not-exist"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error = %v, want 'not found'", err)
	}
}

func TestAuthStatus_LoggedIn(t *testing.T) {
	r := &scriptedRunner{status: claudeauth.Result{Stdout: []byte(`{"loggedIn": true}`)}}
	setupAuthTest(t, r)

	if err := authStatusCmd.RunE(authStatusCmd, []string{"myenv"}); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
	if !slices.Equal(r.calls[0], []string{"auth", "status", "--json"}) {
		t.Errorf("status args = %v, want [auth status --json]", r.calls[0])
	}
	if r.probed() {
		t.Error("probe ran without --live")
	}
}

func TestAuthStatus_TextFlag(t *testing.T) {
	r := &scriptedRunner{status: claudeauth.Result{Stdout: []byte("Login method: Claude account\n")}}
	setupAuthTest(t, r)
	authStatusText = true

	if err := authStatusCmd.RunE(authStatusCmd, []string{"myenv"}); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
	if !slices.Equal(r.calls[0], []string{"auth", "status", "--text"}) {
		t.Errorf("status args = %v, want [auth status --text]", r.calls[0])
	}
}

func TestAuthStatus_NotLoggedIn(t *testing.T) {
	r := &scriptedRunner{status: claudeauth.Result{Stdout: []byte(`{"loggedIn": false}`), ExitCode: 1}}
	setupAuthTest(t, r)
	authStatusLive = true

	err := authStatusCmd.RunE(authStatusCmd, []string{"myenv"})
	if err == nil || !strings.Contains(err.Error(), "cenv login myenv") {
		t.Fatalf("error = %v, want hint to run 'cenv login myenv'", err)
	}
	if r.probed() {
		t.Error("probe ran even though nothing is stored")
	}
}

func TestAuthStatus_LiveOK(t *testing.T) {
	r := &scriptedRunner{
		status: claudeauth.Result{Stdout: []byte(`{"loggedIn": true}`)},
		probe:  claudeauth.Result{Stdout: []byte(`{"is_error":false,"result":"ok","total_cost_usd":0.002}`)},
	}
	setupAuthTest(t, r)
	authStatusLive = true

	if err := authStatusCmd.RunE(authStatusCmd, []string{"myenv"}); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
	if !r.probed() {
		t.Error("probe did not run with --live")
	}
}

func TestAuthStatus_LiveFailure(t *testing.T) {
	r := &scriptedRunner{
		status: claudeauth.Result{Stdout: []byte(`{"loggedIn": true}`)},
		probe: claudeauth.Result{
			Stdout:   []byte(`{"is_error":true,"result":"Login expired · Please run /login"}`),
			ExitCode: 1,
		},
	}
	setupAuthTest(t, r)
	authStatusLive = true

	err := authStatusCmd.RunE(authStatusCmd, []string{"myenv"})
	if err == nil {
		t.Fatal("RunE() error = nil, want live failure")
	}
	for _, want := range []string{"Login expired", "cenv login myenv"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to contain %q", err.Error(), want)
		}
	}
}
