# cenv claude and exec commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename `cenv run` to `cenv claude` (keeping `run` as a deprecated alias) and add `cenv exec <name> -- <command> [args...]` for running arbitrary commands under a cenv environment's `CLAUDE_CONFIG_DIR`.

**Architecture:** Extract the env-exists/settings-preflight/auth-preflight checks shared by `run.go` today into a small `preflightEnv` helper. Rename `run.go` → `claude.go`, adding a second `runCmd` that shares the same `RunE` but is marked `Deprecated`. Add a new `exec.go` with the same preflight but generic command resolution (no assumption the target is `claude`).

**Tech Stack:** Go 1.26.1, `github.com/spf13/cobra`, standard library (`os/exec`, `syscall`).

## Global Constraints

- Auth preflight (`internal/auth.Detect`) is **always required** for `cenv claude`/`cenv run` and `cenv exec` — no bypass for custom commands. (Spec: "the point of a cenv environment is a working, authed Claude setup, and that guarantee shouldn't depend on which program you're launching under it.")
- Both commands replace the current process via `syscall.Exec` — no subprocess/wait mode.
- `cenv exec` requires `--` before the command; the first token after `--` is the program (resolved via `exec.LookPath`), the remaining tokens are its args verbatim — this is a full command line, never an implicit-`claude` arg list.
- `cenv run` stays functionally identical to `cenv claude`, marked deprecated via cobra's `Deprecated` field — not removed, not behaviorally different.
- Go module path: `github.com/technicalpickles/cenv`. Run `mise run check` (fmt + vet + test) — or `go fmt ./... && go vet ./... && go test ./...` without mise — before considering any task done. `go test ./...` already excludes the keychain-tagged tests; don't add the `keychain` build tag to anything in this plan.

---

### shared-preflight-helper

**Files:**
- Create: `cmd/cenv/preflight.go`
- Test: `cmd/cenv/preflight_test.go`

**Interfaces:**
- Produces: `preflightEnv(name string) (envDir string, err error)` — used by both `claude.go` (next task) and `exec.go` (task after). Returns the environment's directory path on success; on failure returns one of three errors: `"environment %q not found"` (env missing), `"preflight failed: %w"` (settings.json fails to load), or `"env %q has no auth configured; run 'cenv login %s' first"` (auth.Detect fails).

- [ ] **Step 1: Write the failing tests**

```go
// cmd/cenv/preflight_test.go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreflightEnv_NonexistentEnv(t *testing.T) {
	t.Setenv("CENV_BASE", t.TempDir())

	_, err := preflightEnv("does-not-exist")
	if err == nil {
		t.Fatal("expected error for nonexistent env, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found'", err.Error())
	}
}

func TestPreflightEnv_NoAuth(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "bare-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	_, err := preflightEnv("bare-env")
	if err == nil {
		t.Fatal("expected auth pre-flight error, got nil")
	}
	if !strings.Contains(err.Error(), "cenv login") {
		t.Errorf("error = %q, want it to mention 'cenv login'", err.Error())
	}
}

func TestPreflightEnv_Success(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "authed-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	settingsJSON := `{"awsAuthRefresh": {"region": "us-east-1"}}`
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	gotDir, err := preflightEnv("authed-env")
	if err != nil {
		t.Fatalf("preflightEnv returned error: %v", err)
	}
	if gotDir != envDir {
		t.Errorf("preflightEnv dir = %q, want %q", gotDir, envDir)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/cenv/... -run TestPreflightEnv -v`
Expected: FAIL — `preflightEnv` undefined (build failure), since `preflight.go` doesn't exist yet.

- [ ] **Step 3: Write the implementation**

```go
// cmd/cenv/preflight.go
package main

import (
	"fmt"
	"path/filepath"

	"github.com/technicalpickles/cenv/internal/auth"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/settings"
)

// preflightEnv validates that the named environment exists, has a loadable
// settings.json, and has auth configured. On success it returns the
// environment's directory path.
func preflightEnv(name string) (string, error) {
	if !env.Exists(name) {
		return "", fmt.Errorf("environment %q not found", name)
	}

	envDir := env.Path(name)
	settingsPath := filepath.Join(envDir, "settings.json")
	if _, err := settings.Load(settingsPath); err != nil {
		return "", fmt.Errorf("preflight failed: %w", err)
	}

	if err := auth.Detect(envDir); err != nil {
		return "", fmt.Errorf("env %q has no auth configured; run 'cenv login %s' first", name, name)
	}

	return envDir, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/cenv/... -run TestPreflightEnv -v`
Expected: PASS (3 tests)

- [ ] **Step 5: Commit**

```bash
git add cmd/cenv/preflight.go cmd/cenv/preflight_test.go
git commit -m "feat(cmd): extract preflightEnv helper for env/settings/auth checks"
```

---

### rename-run-to-claude

**Files:**
- Rename (git mv): `cmd/cenv/run.go` → `cmd/cenv/claude.go`
- Rename (git mv): `cmd/cenv/run_test.go` → `cmd/cenv/claude_test.go`
- Modify: `cmd/cenv/claude.go` (full rewrite of contents)
- Modify: `cmd/cenv/claude_test.go` (retarget to `claudeCmd`, add deprecation-alias test)
- Modify: `cmd/cenv/help_examples_test.go`

**Interfaces:**
- Consumes: `preflightEnv(name string) (string, error)` from `shared-preflight-helper`.
- Produces: `claudeCmd *cobra.Command` (primary), `runCmd *cobra.Command` (deprecated alias), `runClaudeE(cmd *cobra.Command, args []string) error` (shared `RunE`, used by `login-cenv` help text updates in a later task by reference only, not by call).

- [ ] **Step 1: Rename the files**

```bash
git mv cmd/cenv/run.go cmd/cenv/claude.go
git mv cmd/cenv/run_test.go cmd/cenv/claude_test.go
```

- [ ] **Step 2: Update the retargeted tests to fail against the new command name**

Replace the contents of `cmd/cenv/claude_test.go`:

```go
// cmd/cenv/claude_test.go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeCmd_NonexistentEnv(t *testing.T) {
	t.Setenv("CENV_BASE", t.TempDir())

	err := claudeCmd.RunE(claudeCmd, []string{"does-not-exist"})
	if err == nil {
		t.Fatal("expected error for nonexistent env, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found'", err.Error())
	}
}

func TestClaudeCmd_NoAuth(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "bare-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	err := claudeCmd.RunE(claudeCmd, []string{"bare-env"})
	if err == nil {
		t.Fatal("expected auth pre-flight error, got nil")
	}
	if !strings.Contains(err.Error(), "cenv login") {
		t.Errorf("error = %q, want it to mention 'cenv login'", err.Error())
	}
}

func TestRunCmd_DeprecatedAlias(t *testing.T) {
	if runCmd.Deprecated == "" {
		t.Error("runCmd.Deprecated is empty, want a deprecation message")
	}

	t.Setenv("CENV_BASE", t.TempDir())

	err := runCmd.RunE(runCmd, []string{"does-not-exist"})
	if err == nil {
		t.Fatal("expected error for nonexistent env, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found' (runCmd should behave like claudeCmd)", err.Error())
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./cmd/cenv/... -run 'TestClaudeCmd|TestRunCmd_DeprecatedAlias' -v`
Expected: FAIL — `claudeCmd` undefined (build failure), since `claude.go` still defines `runCmd` only, with the old single-command shape.

- [ ] **Step 4: Rewrite claude.go**

Replace the contents of `cmd/cenv/claude.go`:

```go
// cmd/cenv/claude.go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/style"
)

var claudeCmd = &cobra.Command{
	Use:   "claude <name> [-- claude-args...]",
	Short: "Launch Claude in an environment",
	Example: `  cenv claude myenv
  cenv claude myenv -- --model opus`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE:               runClaudeE,
}

var runCmd = &cobra.Command{
	Use:   "run <name> [-- claude-args...]",
	Short: "Launch Claude in an environment",
	Example: `  cenv run myenv
  cenv run myenv -- --model opus`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	Deprecated:         "use 'cenv claude' instead",
	RunE:               runClaudeE,
}

func runClaudeE(cmd *cobra.Command, args []string) error {
	name := args[0]

	envDir, err := preflightEnv(name)
	if err != nil {
		return err
	}

	var claudeArgs []string
	if len(args) > 1 {
		if args[1] != "--" {
			return fmt.Errorf("unexpected argument %q (use -- before claude arguments)", args[1])
		}
		claudeArgs = args[2:]
	}

	logf("%s\n", style.Info("Using %q (%s)", name, envDir))

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH")
	}

	environ := os.Environ()
	environ = append(environ, fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", envDir))

	execArgs := append([]string{"claude"}, claudeArgs...)
	return syscall.Exec(claudePath, execArgs, environ)
}

func init() {
	rootCmd.AddCommand(claudeCmd)
	rootCmd.AddCommand(runCmd)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./cmd/cenv/... -run 'TestClaudeCmd|TestRunCmd_DeprecatedAlias' -v`
Expected: PASS (3 tests)

- [ ] **Step 6: Update help_examples_test.go to cover claudeCmd**

In `cmd/cenv/help_examples_test.go`, add `"claude": claudeCmd.Example,` to the `cases` map (keep the existing `"run": runCmd.Example,` entry — `run` still has its own `Example` text even though deprecated):

```go
func TestHelpExamples_Present(t *testing.T) {
	cases := map[string]string{
		"create":         createCmd.Example,
		"claude":         claudeCmd.Example,
		"run":            runCmd.Example,
		"login":          loginCmd.Example,
		"remove":         removeCmd.Example,
		"path":           pathCmd.Example,
		"trust":          trustCmd.Example,
		"settings show":  settingsShowCmd.Example,
		"settings get":   settingsGetCmd.Example,
		"settings merge": settingsMergeCmd.Example,
	}
	for name, example := range cases {
		if example == "" {
			t.Errorf("%s: missing Example text", name)
		}
	}
}
```

- [ ] **Step 7: Run the full cmd/cenv test suite**

Run: `go test ./cmd/cenv/... -v`
Expected: PASS, all tests including the pre-existing ones for other commands.

- [ ] **Step 8: Commit**

```bash
git add cmd/cenv/claude.go cmd/cenv/claude_test.go cmd/cenv/help_examples_test.go
git commit -m "feat(cmd): rename run to claude, keep run as a deprecated alias"
```

---

### exec-command

**Files:**
- Create: `cmd/cenv/exec.go`
- Test: `cmd/cenv/exec_test.go`

**Interfaces:**
- Consumes: `preflightEnv(name string) (string, error)` from `shared-preflight-helper`.
- Produces: `execCmd *cobra.Command`, registered on `rootCmd`.

- [ ] **Step 1: Write the failing tests**

```go
// cmd/cenv/exec_test.go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecCmd_NonexistentEnv(t *testing.T) {
	t.Setenv("CENV_BASE", t.TempDir())

	err := execCmd.RunE(execCmd, []string{"does-not-exist", "--", "true"})
	if err == nil {
		t.Fatal("expected error for nonexistent env, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found'", err.Error())
	}
}

func TestExecCmd_NoAuth(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "bare-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	err := execCmd.RunE(execCmd, []string{"bare-env", "--", "true"})
	if err == nil {
		t.Fatal("expected auth pre-flight error, got nil")
	}
	if !strings.Contains(err.Error(), "cenv login") {
		t.Errorf("error = %q, want it to mention 'cenv login'", err.Error())
	}
}

func TestExecCmd_MissingSeparator(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "authed-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	settingsJSON := `{"awsAuthRefresh": {"region": "us-east-1"}}`
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	err := execCmd.RunE(execCmd, []string{"authed-env"})
	if err == nil {
		t.Fatal("expected missing-command error, got nil")
	}
	if !strings.Contains(err.Error(), "missing command") {
		t.Errorf("error = %q, want it to mention 'missing command'", err.Error())
	}
}

func TestExecCmd_MissingCommandAfterSeparator(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "authed-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	settingsJSON := `{"awsAuthRefresh": {"region": "us-east-1"}}`
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	err := execCmd.RunE(execCmd, []string{"authed-env", "--"})
	if err == nil {
		t.Fatal("expected missing-command error, got nil")
	}
	if !strings.Contains(err.Error(), "missing command") {
		t.Errorf("error = %q, want it to mention 'missing command'", err.Error())
	}
}

func TestExecCmd_CommandNotFound(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	envDir := filepath.Join(base, "authed-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	settingsJSON := `{"awsAuthRefresh": {"region": "us-east-1"}}`
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	err := execCmd.RunE(execCmd, []string{"authed-env", "--", "definitely-not-a-real-binary-xyz"})
	if err == nil {
		t.Fatal("expected command-not-found error, got nil")
	}
	if !strings.Contains(err.Error(), "not found in PATH") {
		t.Errorf("error = %q, want it to mention 'not found in PATH'", err.Error())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/cenv/... -run TestExecCmd -v`
Expected: FAIL — `execCmd` undefined (build failure), since `exec.go` doesn't exist yet.

- [ ] **Step 3: Write the implementation**

```go
// cmd/cenv/exec.go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/style"
)

var execCmd = &cobra.Command{
	Use:   "exec <name> -- <command> [args...]",
	Short: "Run an arbitrary command in an environment",
	Example: `  cenv exec myenv -- a2acode serve
  cenv exec myenv -- npm test`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		envDir, err := preflightEnv(name)
		if err != nil {
			return err
		}

		if len(args) < 3 || args[1] != "--" {
			return fmt.Errorf("missing command (use: cenv exec %s -- <command> [args...])", name)
		}
		command := args[2:]

		logf("%s\n", style.Info("Using %q (%s)", name, envDir))

		cmdPath, err := exec.LookPath(command[0])
		if err != nil {
			return fmt.Errorf("%s not found in PATH", command[0])
		}

		environ := os.Environ()
		environ = append(environ, fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", envDir))

		execArgs := append([]string{command[0]}, command[1:]...)
		return syscall.Exec(cmdPath, execArgs, environ)
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/cenv/... -run TestExecCmd -v`
Expected: PASS (5 tests)

- [ ] **Step 5: Add exec's Example to help_examples_test.go**

In `cmd/cenv/help_examples_test.go`, add `"exec": execCmd.Example,` to the `cases` map.

- [ ] **Step 6: Run the full cmd/cenv test suite**

Run: `go test ./cmd/cenv/... -v`
Expected: PASS, all tests.

- [ ] **Step 7: Commit**

```bash
git add cmd/cenv/exec.go cmd/cenv/exec_test.go cmd/cenv/help_examples_test.go
git commit -m "feat(cmd): add exec command to run arbitrary commands in an environment"
```

---

### docs-and-smoke-updates

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `internal/auth/auth.go` (doc comments only)
- Modify: `.claude/skills/run-cenv/SKILL.md`
- Modify: `.claude/skills/run-cenv/smoke.sh`

No new Go code in this task — text-only changes plus a shell script update, verified by running the smoke script and re-running the full test suite at the end.

- [ ] **Step 1: Update README.md**

In `README.md`, in the "Anthropic OAuth users" section:

Replace:
```
cenv create my-env           # copies OAuth (keychain + oauthAccount) from ~/.claude
cenv run my-env -- -p 'hi'   # env is authenticated
```
with:
```
cenv create my-env              # copies OAuth (keychain + oauthAccount) from ~/.claude
cenv claude my-env -- -p 'hi'   # env is authenticated
```

Replace:
```
For scripts and agents, `cenv run` fails fast with a message pointing at `cenv login` if the target env has never been authenticated.
```
with:
```
For scripts and agents, `cenv claude` fails fast with a message pointing at `cenv login` if the target env has never been authenticated. (`cenv run` still works as a deprecated alias for `cenv claude`.)

To run something other than `claude` itself under an env's config — e.g. a different tool built on the Claude Agent SDK — use `cenv exec`:

```sh
cenv exec my-env -- a2acode serve
```

`cenv exec` runs the same auth pre-flight as `cenv claude`, then execs whatever command you give it with `CLAUDE_CONFIG_DIR` pointed at the env.
```

- [ ] **Step 2: Update CLAUDE.md**

In `CLAUDE.md`, in the `internal/auth` bullet under Packages:

Replace:
```
- **`internal/auth`** — `Detect(configDir)` is the auth predicate used both for `env.Info.HasAuth` and as a preflight check in `cenv run`: nil error means either `settings.json` has a non-empty `awsAuthRefresh` (Bedrock) or `.claude.json` has a non-empty `oauthAccount` (Anthropic OAuth).
```
with:
```
- **`internal/auth`** — `Detect(configDir)` is the auth predicate used for `env.Info.HasAuth` and as a preflight check shared by `cenv claude`/`cenv run` and `cenv exec` (via `preflightEnv` in `cmd/cenv/preflight.go`): nil error means either `settings.json` has a non-empty `awsAuthRefresh` (Bedrock) or `.claude.json` has a non-empty `oauthAccount` (Anthropic OAuth).
```

In the "Process replacement, not subprocess" section, replace:
```
`cenv run` and `cenv login` both use `syscall.Exec` (not `os/exec` + wait) to replace the current process with `claude`, after injecting `CLAUDE_CONFIG_DIR=<envdir>` into the environment. This means signals, TTY, and exit codes pass straight through to the real `claude` process — there's no cenv process left to relay them.
```
with:
```
`cenv claude`/`cenv run` and `cenv login` both use `syscall.Exec` (not `os/exec` + wait) to replace the current process with `claude`, after injecting `CLAUDE_CONFIG_DIR=<envdir>` into the environment. `cenv exec` uses the same mechanism to launch an arbitrary command instead. This means signals, TTY, and exit codes pass straight through to the launched process — there's no cenv process left to relay them.
```

- [ ] **Step 3: Update internal/auth/auth.go doc comments**

In `internal/auth/auth.go`, the package doc comment currently reads (lines 7-8):
```go
// Callers use Detect as a predicate (error == nil means "authenticated")
// for pre-flight checks in cenv run and the HasAuth field in env.Info.
```
Replace with:
```go
// Callers use Detect as a predicate (error == nil means "authenticated")
// for pre-flight checks in cenv claude/run/exec and the HasAuth field in
// env.Info.
```

- [ ] **Step 4: Update .claude/skills/run-cenv/SKILL.md**

In the frontmatter `description`, replace `(create/list/path/remove/settings/trust/run/login)` with `(create/list/path/remove/settings/trust/claude/exec/login)`.

Replace the `--help` output comment:
```
./cenv --help
# → lists: create, list, path, remove, run, settings, trust, login, completion
```
with:
```
./cenv --help
# → lists: create, list, path, remove, claude, exec, settings, trust, login, completion
# (run still works too — it's a deprecated alias for claude, hidden from this list)
```

In the smoke script description bullet:
```
- `run` / `login` pre-flight checks: missing env, and missing-auth
  rejection (it does NOT launch the nested Claude REPL — see Gotchas)
```
replace with:
```
- `claude` / `exec` / `login` pre-flight checks: missing env, and
  missing-auth rejection (it does NOT launch the nested Claude REPL or
  an arbitrary command — see Gotchas)
```

In the "Run (human path)" section, replace:
```
`cenv run <name> -- <claude-args>` and `cenv login <name>` both
`syscall.Exec` into the real `claude` binary (found via `PATH`) with
`CLAUDE_CONFIG_DIR` pointed at the env dir — they replace the current
process, so they only make sense in an interactive terminal against a
real, authenticated env:

```bash
cenv create myenv                  # auto-copies OAuth from ~/.claude if logged in
cenv run myenv -- -p 'hi'          # or: cenv login myenv, then /login inside Claude
```
```
with:
```
`cenv claude <name> -- <claude-args>` and `cenv login <name>` both
`syscall.Exec` into the real `claude` binary (found via `PATH`) with
`CLAUDE_CONFIG_DIR` pointed at the env dir — they replace the current
process, so they only make sense in an interactive terminal against a
real, authenticated env:

```bash
cenv create myenv                  # auto-copies OAuth from ~/.claude if logged in
cenv claude myenv -- -p 'hi'       # or: cenv login myenv, then /login inside Claude
```

`cenv exec <name> -- <command> [args...]` works the same way but for any
command, not just `claude` — useful for other tools built on the Claude
Agent SDK:

```bash
cenv exec myenv -- a2acode serve
```
```

In the Gotchas section, replace:
```
- **`cenv run --help` doesn't work as you'd expect.** `run` uses
  `cobra.MinimumNArgs(1)` + `DisableFlagParsing: true` so it can pass
  `-- <claude-args>` through untouched. That means `--help` is parsed
  as the environment name, so `cenv run --help` fails with `environment
  "--help" does not exist` instead of printing usage. Use `cenv help
  run` instead.
```
with:
```
- **`cenv claude --help` / `cenv exec --help` don't work as you'd
  expect.** Both use `cobra.MinimumNArgs(1)` + `DisableFlagParsing:
  true` so they can pass `-- <args>` through untouched. That means
  `--help` is parsed as the environment name, so `cenv claude --help`
  fails with `environment "--help" does not exist` instead of printing
  usage. Use `cenv help claude` / `cenv help exec` instead.
```

- [ ] **Step 5: Update smoke.sh**

In `.claude/skills/run-cenv/smoke.sh`, update the header comment (line 4):
```
# trust, and run/login's pre-flight error paths (missing env, missing auth).
```
to:
```
# trust, and claude/exec/login's pre-flight error paths (missing env, missing auth).
```

Replace the final section:
```bash
echo "== run / login pre-flight (no nested Claude REPL launch) =="
check_fail "run on missing env fails" "$BIN" run ghost -- -p hi
check_fail "run on unauthenticated env fails" "$BIN" run demo -- -p hi
check_fail "login on missing env fails" "$BIN" login ghost
```
with:
```bash
echo "== claude / exec / login pre-flight (no nested process launch) =="
check_fail "claude on missing env fails" "$BIN" claude ghost -- -p hi
check_fail "claude on unauthenticated env fails" "$BIN" claude demo -- -p hi
check_fail "run (deprecated alias) on missing env fails" "$BIN" run ghost -- -p hi
check_fail "exec on missing env fails" "$BIN" exec ghost -- true
check_fail "exec on unauthenticated env fails" "$BIN" exec demo -- true
check_fail "exec with missing separator fails" "$BIN" exec demo true
check_fail "login on missing env fails" "$BIN" login ghost
```

- [ ] **Step 6: Build and run the smoke script**

```bash
go build -o cenv ./cmd/cenv
bash .claude/skills/run-cenv/smoke.sh ./cenv
```
Expected: `== N passed, 0 failed ==` with N reflecting the new checks added in Step 5.

- [ ] **Step 7: Run the full test suite and checks**

```bash
mise run check
```
Expected: fmt makes no changes, `go vet` is clean, `go test ./...` passes.

- [ ] **Step 8: Commit**

```bash
git add README.md CLAUDE.md internal/auth/auth.go .claude/skills/run-cenv/SKILL.md .claude/skills/run-cenv/smoke.sh
git commit -m "docs: update README/CLAUDE.md/run-cenv skill for claude and exec commands"
```
