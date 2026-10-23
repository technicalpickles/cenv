package main

import (
	"slices"
	"strings"
	"testing"
)

func TestLoginCmd_NonexistentEnv(t *testing.T) {
	t.Setenv("CENV_BASE", t.TempDir())

	err := loginCmd.RunE(loginCmd, []string{"does-not-exist"})
	if err == nil {
		t.Fatal("expected error for nonexistent env, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found'", err.Error())
	}
}

func TestLoginExec_DefaultsToPlainLogin(t *testing.T) {
	t.Setenv("CENV_LOGIN_FLAGS", "")
	argv, environ := loginExec("/envs/foo", nil, []string{"A=1"})

	if want := []string{"claude", "auth", "login"}; !slices.Equal(argv, want) {
		t.Errorf("argv = %v, want %v", argv, want)
	}
	if want := []string{"A=1", "CLAUDE_CONFIG_DIR=/envs/foo"}; !slices.Equal(environ, want) {
		t.Errorf("environ = %v, want %v", environ, want)
	}
}

func TestLoginExec_DefaultFlagsFromEnv(t *testing.T) {
	t.Setenv("CENV_LOGIN_FLAGS", "--sso --email me@example.com")
	argv, _ := loginExec("/envs/foo", nil, nil)

	if want := []string{"claude", "auth", "login", "--sso", "--email", "me@example.com"}; !slices.Equal(argv, want) {
		t.Errorf("argv = %v, want %v", argv, want)
	}
}

func TestLoginExec_ExplicitFlagsReplaceEnvDefault(t *testing.T) {
	t.Setenv("CENV_LOGIN_FLAGS", "--sso")
	argv, _ := loginExec("/envs/foo", []string{"--console"}, nil)

	if want := []string{"claude", "auth", "login", "--console"}; !slices.Equal(argv, want) {
		t.Errorf("argv = %v, want %v", argv, want)
	}
}

func TestLoginExec_PassesFlagsThrough(t *testing.T) {
	argv, _ := loginExec("/envs/foo", []string{"--console"}, nil)

	if want := []string{"claude", "auth", "login", "--console"}; !slices.Equal(argv, want) {
		t.Errorf("argv = %v, want %v", argv, want)
	}
}

func TestLoginExec_DoesNotMutateInputEnviron(t *testing.T) {
	in := make([]string, 1, 10)
	in[0] = "A=1"
	loginExec("/envs/foo", nil, in)
	if len(in) != 1 || in[:2][1] != "" {
		t.Errorf("input environ was modified: %v", in[:2])
	}
}
