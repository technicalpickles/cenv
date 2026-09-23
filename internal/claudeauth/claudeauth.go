// Package claudeauth asks claude itself about an env's auth. Status wraps
// `claude auth status`, which only reports stored credentials and never checks
// expiry, so an expired OAuth token still reads as logged in. Probe makes a
// minimal real request, the only way to tell the credentials still work.
package claudeauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ProbeArgs is tuned to cost about $0.002 per run. Don't swap --safe-mode for
// --bare: --bare skips keychain reads, so OAuth can't work under it.
var ProbeArgs = []string{
	"-p", "ok",
	"--safe-mode",
	"--model", "haiku",
	"--tools", "",
	"--system-prompt", "Reply with ok.",
	"--effort", "low",
	"--max-turns", "1",
	"--no-session-persistence",
	"--max-budget-usd", "0.05",
	"--output-format", "json",
}

type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Runner runs claude with CLAUDE_CONFIG_DIR set to configDir. A nonzero exit
// is reported through Result.ExitCode, not as an error; errors mean claude
// couldn't be run at all.
type Runner interface {
	Run(ctx context.Context, configDir string, args ...string) (Result, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, configDir string, args ...string) (Result, error) {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return Result{}, fmt.Errorf("claude not found in PATH")
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, claudePath, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", configDir))
	// Keep the caller's project .claude/ settings out of the check.
	cmd.Dir = os.TempDir()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	res := Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if ctx.Err() != nil {
			return res, fmt.Errorf("claude timed out: %w", ctx.Err())
		}
		res.ExitCode = ee.ExitCode()
		return res, nil
	}
	if err != nil {
		return res, fmt.Errorf("running claude: %w", err)
	}
	return res, nil
}

type Client struct {
	Runner Runner
}

var Default = &Client{Runner: execRunner{}}

type StatusResult struct {
	// LoggedIn mirrors claude's exit code; true means stored, not valid.
	LoggedIn bool
	Output   []byte
}

// Status runs `claude auth status` with format "--json" or "--text".
func (c *Client) Status(ctx context.Context, configDir, format string) (StatusResult, error) {
	res, err := c.Runner.Run(ctx, configDir, "auth", "status", format)
	if err != nil {
		return StatusResult{}, err
	}
	return StatusResult{LoggedIn: res.ExitCode == 0, Output: res.Stdout}, nil
}

type ProbeResult struct {
	OK bool
	// Message is claude's result text, or stderr when the output wasn't JSON.
	Message string
	CostUSD float64
}

type probeOutput struct {
	IsError      bool    `json:"is_error"`
	Result       string  `json:"result"`
	TotalCostUSD float64 `json:"total_cost_usd"`
}

// Probe reports a rejected request as OK=false; an error means the probe
// couldn't run at all.
func (c *Client) Probe(ctx context.Context, configDir string) (ProbeResult, error) {
	res, err := c.Runner.Run(ctx, configDir, ProbeArgs...)
	if err != nil {
		return ProbeResult{}, err
	}

	var out probeOutput
	if jsonErr := json.Unmarshal(res.Stdout, &out); jsonErr != nil {
		msg := strings.TrimSpace(string(res.Stderr))
		if msg == "" {
			msg = strings.TrimSpace(string(res.Stdout))
		}
		if msg == "" {
			msg = fmt.Sprintf("claude exited %d with no output", res.ExitCode)
		}
		return ProbeResult{OK: false, Message: msg}, nil
	}

	return ProbeResult{
		OK:      !out.IsError && res.ExitCode == 0,
		Message: strings.TrimSpace(out.Result),
		CostUSD: out.TotalCostUSD,
	}, nil
}
