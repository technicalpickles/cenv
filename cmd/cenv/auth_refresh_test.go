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

type envResponses struct {
	status claudeauth.Result
	probe  claudeauth.Result
}

type perEnvRunner struct {
	envs   map[string]envResponses
	probed []string
}

func (p *perEnvRunner) Run(ctx context.Context, configDir string, args ...string) (claudeauth.Result, error) {
	name := filepath.Base(configDir)
	resp := p.envs[name]
	if len(args) > 0 && args[0] == "auth" {
		return resp.status, nil
	}
	p.probed = append(p.probed, name)
	return resp.probe, nil
}

var (
	loggedIn    = claudeauth.Result{Stdout: []byte(`{"loggedIn": true}`)}
	notLoggedIn = claudeauth.Result{Stdout: []byte(`{"loggedIn": false}`), ExitCode: 1}
	probeOK     = claudeauth.Result{Stdout: []byte(`{"is_error":false,"result":"ok","total_cost_usd":0.002}`)}
	probeDead   = claudeauth.Result{Stdout: []byte(`{"is_error":true,"result":"OAuth access token is invalid"}`), ExitCode: 1}
)

func setupRefreshTest(t *testing.T, r *perEnvRunner) {
	t.Helper()
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)
	for name := range r.envs {
		if err := os.MkdirAll(filepath.Join(base, name), 0755); err != nil {
			t.Fatalf("creating env dir: %v", err)
		}
	}

	origClient := authClient
	authClient = &claudeauth.Client{Runner: r}
	t.Cleanup(func() { authClient = origClient })

	origAll := authRefreshAll
	t.Cleanup(func() { authRefreshAll = origAll })
	authRefreshAll = false
}

func TestAuthRefresh_RequiresNamesOrAll(t *testing.T) {
	setupRefreshTest(t, &perEnvRunner{})

	err := authRefreshCmd.RunE(authRefreshCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--all") {
		t.Fatalf("error = %v, want hint about env names or --all", err)
	}
}

func TestAuthRefresh_RejectsNamesWithAll(t *testing.T) {
	setupRefreshTest(t, &perEnvRunner{envs: map[string]envResponses{"a": {loggedIn, probeOK}}})
	authRefreshAll = true

	if err := authRefreshCmd.RunE(authRefreshCmd, []string{"a"}); err == nil {
		t.Fatal("error = nil, want error for names combined with --all")
	}
}

func TestAuthRefresh_NamedEnvs(t *testing.T) {
	r := &perEnvRunner{envs: map[string]envResponses{
		"a": {loggedIn, probeOK},
		"b": {loggedIn, probeOK},
		"c": {loggedIn, probeOK},
	}}
	setupRefreshTest(t, r)

	if err := authRefreshCmd.RunE(authRefreshCmd, []string{"a", "c"}); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
	if !slices.Equal(r.probed, []string{"a", "c"}) {
		t.Errorf("probed = %v, want [a c]", r.probed)
	}
}

func TestAuthRefresh_MissingEnv(t *testing.T) {
	setupRefreshTest(t, &perEnvRunner{envs: map[string]envResponses{"a": {loggedIn, probeOK}}})

	err := authRefreshCmd.RunE(authRefreshCmd, []string{"nope"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error = %v, want 'not found'", err)
	}
}

func TestAuthRefresh_AllSkipsNotLoggedIn(t *testing.T) {
	r := &perEnvRunner{envs: map[string]envResponses{
		"authed": {loggedIn, probeOK},
		"fresh":  {notLoggedIn, probeOK},
	}}
	setupRefreshTest(t, r)
	authRefreshAll = true

	if err := authRefreshCmd.RunE(authRefreshCmd, nil); err != nil {
		t.Fatalf("RunE() error = %v, want never-logged-in envs skipped, not failed", err)
	}
	if !slices.Equal(r.probed, []string{"authed"}) {
		t.Errorf("probed = %v, want only [authed]", r.probed)
	}
}

func TestAuthRefresh_FailureKeepsGoingAndExitsNonzero(t *testing.T) {
	r := &perEnvRunner{envs: map[string]envResponses{
		"a": {loggedIn, probeDead},
		"b": {loggedIn, probeOK},
	}}
	setupRefreshTest(t, r)
	authRefreshAll = true

	err := authRefreshCmd.RunE(authRefreshCmd, nil)
	if err == nil {
		t.Fatal("RunE() error = nil, want failure reported")
	}
	if !strings.Contains(err.Error(), "cenv login a") {
		t.Errorf("error = %q, want it to name the dead env with a login hint", err.Error())
	}
	if !slices.Equal(r.probed, []string{"a", "b"}) {
		t.Errorf("probed = %v, want every env probed despite the failure", r.probed)
	}
}
