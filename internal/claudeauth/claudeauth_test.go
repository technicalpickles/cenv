package claudeauth_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/technicalpickles/cenv/internal/claudeauth"
)

type call struct {
	configDir string
	args      []string
}

type fakeRunner struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
	calls    []call
}

func (f *fakeRunner) Run(ctx context.Context, configDir string, args ...string) (claudeauth.Result, error) {
	f.calls = append(f.calls, call{configDir: configDir, args: args})
	if f.err != nil {
		return claudeauth.Result{}, f.err
	}
	return claudeauth.Result{Stdout: []byte(f.stdout), Stderr: []byte(f.stderr), ExitCode: f.exitCode}, nil
}

func TestStatus_LoggedIn(t *testing.T) {
	r := &fakeRunner{stdout: `{"loggedIn": true, "authMethod": "claude.ai"}`}
	c := &claudeauth.Client{Runner: r}

	st, err := c.Status(context.Background(), "/envs/foo", "--json")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !st.LoggedIn {
		t.Error("LoggedIn = false, want true")
	}
	if !strings.Contains(string(st.Output), `"authMethod"`) {
		t.Errorf("Output = %q, want raw claude output passed through", st.Output)
	}
	if got := r.calls[0]; got.configDir != "/envs/foo" || !slices.Equal(got.args, []string{"auth", "status", "--json"}) {
		t.Errorf("call = %+v, want configDir /envs/foo and args [auth status --json]", got)
	}
}

func TestStatus_NotLoggedInExitCode(t *testing.T) {
	r := &fakeRunner{stdout: `{"loggedIn": false}`, exitCode: 1}
	c := &claudeauth.Client{Runner: r}

	st, err := c.Status(context.Background(), "/envs/foo", "--json")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if st.LoggedIn {
		t.Error("LoggedIn = true, want false")
	}
}

func TestStatus_TextFormatUsesExitCode(t *testing.T) {
	r := &fakeRunner{stdout: "Login method: Claude Enterprise account\n"}
	c := &claudeauth.Client{Runner: r}

	st, err := c.Status(context.Background(), "/envs/foo", "--text")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !st.LoggedIn {
		t.Error("LoggedIn = false, want true for exit 0")
	}
}

func TestStatus_RunnerError(t *testing.T) {
	c := &claudeauth.Client{Runner: &fakeRunner{err: errors.New("claude not found in PATH")}}

	if _, err := c.Status(context.Background(), "/envs/foo", "--json"); err == nil {
		t.Fatal("Status() error = nil, want runner error surfaced")
	}
}

func TestProbe_Success(t *testing.T) {
	r := &fakeRunner{stdout: `{"type":"result","is_error":false,"result":"ok","total_cost_usd":0.002}`}
	c := &claudeauth.Client{Runner: r}

	res, err := c.Probe(context.Background(), "/envs/foo")
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if !res.OK {
		t.Errorf("OK = false, want true (message %q)", res.Message)
	}
	if res.CostUSD != 0.002 {
		t.Errorf("CostUSD = %v, want 0.002", res.CostUSD)
	}
	if !slices.Equal(r.calls[0].args, claudeauth.ProbeArgs) {
		t.Errorf("args = %v, want ProbeArgs", r.calls[0].args)
	}
}

func TestProbeArgs_MinimizeCostAndSideEffects(t *testing.T) {
	for _, want := range []string{"-p", "--safe-mode", "--model", "haiku", "--no-session-persistence", "--output-format", "json", "--max-budget-usd"} {
		if !slices.Contains(claudeauth.ProbeArgs, want) {
			t.Errorf("ProbeArgs missing %q: %v", want, claudeauth.ProbeArgs)
		}
	}
	if slices.Contains(claudeauth.ProbeArgs, "--bare") {
		t.Error("ProbeArgs must not use --bare: it skips keychain reads, so OAuth can't work")
	}
}

func TestProbe_ErrorResult(t *testing.T) {
	r := &fakeRunner{
		stdout:   `{"type":"result","is_error":true,"result":"Login expired · Please run /login"}`,
		exitCode: 1,
	}
	c := &claudeauth.Client{Runner: r}

	res, err := c.Probe(context.Background(), "/envs/foo")
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if res.OK {
		t.Error("OK = true, want false")
	}
	if res.Message != "Login expired · Please run /login" {
		t.Errorf("Message = %q, want claude's result text", res.Message)
	}
}

func TestProbe_NonJSONFailure(t *testing.T) {
	r := &fakeRunner{stdout: "", stderr: "Invalid API key\n", exitCode: 1}
	c := &claudeauth.Client{Runner: r}

	res, err := c.Probe(context.Background(), "/envs/foo")
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if res.OK {
		t.Error("OK = true, want false")
	}
	if res.Message != "Invalid API key" {
		t.Errorf("Message = %q, want trimmed stderr", res.Message)
	}
}

func TestProbe_NonZeroExitOverridesJSON(t *testing.T) {
	r := &fakeRunner{stdout: `{"is_error":false,"result":"ok"}`, exitCode: 1}
	c := &claudeauth.Client{Runner: r}

	res, err := c.Probe(context.Background(), "/envs/foo")
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if res.OK {
		t.Error("OK = true, want false when claude exits nonzero")
	}
}
