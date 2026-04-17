# cenv OAuth Auth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make OAuth-based Anthropic auth work for cenv environments via a deliberate interactive `cenv login` step, without reverse-engineering Claude Code's keychain hash scheme.

**Architecture:** Fix the `oauthAccount` object-vs-string type assertion (gt-5zn7), then layer on a new `cenv login` command, a pre-flight auth check in `cenv run`, and OAuth-specific rejections/messaging in `cenv auth create` and `cenv create`.

**Tech Stack:** Go 1.26, cobra, Go's stdlib for JSON parsing and TTY detection (via `os.File.Stat` + `os.ModeCharDevice`). No new dependencies.

**Related:** Spec at `docs/plans/2026-04-17-oauth-auth-design.md`. Beans: gt-wl86 (this work), gt-5zn7 (prerequisite bug), gt-8tv9 (downstream blocker).

---

## File Structure

| File | Role |
|---|---|
| `internal/auth/auth.go` | Auth detection. Fix `oauthAccount` type handling. |
| `internal/auth/auth_test.go` | Add object-shape test cases. |
| `cmd/cenv/create.go` | Fix `hasOAuth` type handling. Replace placeholder warning with `cenv login` next-step. |
| `cmd/cenv/auth.go` | Refuse `auth create` for OAuth users with clear message. |
| `cmd/cenv/run.go` | Pre-flight `auth.Detect`; hard error on "no auth found". |
| `cmd/cenv/login.go` *(new)* | New `cenv login <name>` command. TTY-gated exec of `claude` with `CLAUDE_CONFIG_DIR` set. |
| `cmd/cenv/tty.go` *(new)* | Small helper `isTerminal(*os.File) bool` using `os.ModeCharDevice`. |
| `README.md` | Document the OAuth flow. |

---

## Task 1: Fix `auth.Detect` for object-shaped `oauthAccount` (gt-5zn7)

Real OAuth users have `oauthAccount` as a JSON object, not a string. The current string assertion silently fails.

**Files:**
- Modify: `internal/auth/auth.go:50-57`
- Test: `internal/auth/auth_test.go`

- [ ] **Step 1: Add failing test for object-shaped `oauthAccount`**

Append to `internal/auth/auth_test.go`:

```go
func TestDetect_Anthropic_ObjectShape(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{
		"oauthAccount": {
			"accountUuid": "abc-123",
			"emailAddress": "user@example.com",
			"organizationUuid": "org-456"
		}
	}`)

	result, err := auth.Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != "anthropic" {
		t.Errorf("Type = %q, want %q", result.Type, "anthropic")
	}
	if result.EnvName != "auth-anthropic" {
		t.Errorf("EnvName = %q, want %q", result.EnvName, "auth-anthropic")
	}
	want := "user@example.com"
	if result.Detail != want {
		t.Errorf("Detail = %q, want %q", result.Detail, want)
	}
}

func TestDetect_Anthropic_ObjectShape_NoEmail(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{
		"oauthAccount": {
			"accountUuid": "abc-123"
		}
	}`)

	result, err := auth.Detect(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != "anthropic" {
		t.Errorf("Type = %q, want %q", result.Type, "anthropic")
	}
}

func TestDetect_Anthropic_ObjectShape_EmptyObject(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "settings.json", `{}`)
	writeJSON(t, dir, ".claude.json", `{
		"oauthAccount": {}
	}`)

	_, err := auth.Detect(dir)
	if err == nil {
		t.Error("expected error for empty oauthAccount object, got nil")
	}
}
```

- [ ] **Step 2: Run new tests to verify their states**

Run: `go test ./internal/auth/ -run TestDetect_Anthropic_ObjectShape -v`

Expected:
- `TestDetect_Anthropic_ObjectShape` FAILS — `val.(string)` returns `"", false` for a map, so Detect falls through and returns "no auth found" instead of the expected anthropic result.
- `TestDetect_Anthropic_ObjectShape_NoEmail` FAILS — same reason.
- `TestDetect_Anthropic_ObjectShape_EmptyObject` PASSES already — empty object fails the string assertion, Detect returns "no auth found", test expects that error. Kept for regression coverage of the post-fix empty-object handling.

- [ ] **Step 3: Replace the string-only assertion in `auth.Detect`**

In `internal/auth/auth.go`, replace lines 49-59 (the `.claude.json` detection block):

```go
	// Step 2: check .claude.json for oauthAccount
	claudePath := filepath.Join(configDir, ".claude.json")
	claudeData, err := readJSON(claudePath)
	if err == nil {
		if val, ok := claudeData["oauthAccount"]; ok {
			switch v := val.(type) {
			case string:
				if v != "" {
					return &DetectResult{
						Type:    "anthropic",
						EnvName: "auth-anthropic",
						Detail:  v,
					}, nil
				}
			case map[string]any:
				email, _ := v["emailAddress"].(string)
				// Real OAuth objects always have at least one populated field.
				// Treat an empty object as "no auth".
				if len(v) > 0 {
					return &DetectResult{
						Type:    "anthropic",
						EnvName: "auth-anthropic",
						Detail:  email,
					}, nil
				}
			}
		}
	}
```

Also update the doc comment above `Detect` (lines 17-23):

```go
// Detect examines a Claude config directory and returns the active auth method.
//
// Detection order:
//  1. settings.json: if "awsAuthRefresh" key exists as an object, it's AWS Bedrock.
//  2. .claude.json: if "oauthAccount" key exists as a non-empty string OR
//     a non-empty object, it's Anthropic. The object shape is what Claude
//     Code writes today (fields: accountUuid, emailAddress, organizationUuid,
//     hasExtraUsageEnabled, billingType).
//
// Returns an error if neither is found.
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/auth/ -v`

Expected: All tests pass, including `TestDetect_Anthropic` (string shape, existing) and the three new object-shape tests.

- [ ] **Step 5: Commit**

```bash
git add internal/auth/auth.go internal/auth/auth_test.go
git commit -m "fix(auth): handle object-shaped oauthAccount (gt-5zn7)

Real Claude Code OAuth writes oauthAccount as a JSON object with
fields like accountUuid, emailAddress, organizationUuid. The previous
string type assertion silently failed, so Detect returned 'no auth'
for OAuth users. Accept both string and map[string]any shapes.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Fix `hasOAuth` in create.go

Same object-vs-string bug lives in `cmd/cenv/create.go`. The warning for OAuth users never fires because of it.

**Files:**
- Modify: `cmd/cenv/create.go:128-141`

- [ ] **Step 1: Inspect current `hasOAuth` signature**

Confirm lines 128-141 look like the current code (from the project's `git show HEAD:cmd/cenv/create.go` if unsure):

```go
// hasOAuth reports whether the user has Anthropic OAuth configured, indicated
// by a non-empty oauthAccount field in ~/.claude.json (home root, not ~/.claude/).
func hasOAuth(home string) bool {
	data, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		return false
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return false
	}
	account, _ := parsed["oauthAccount"].(string)
	return account != ""
}
```

- [ ] **Step 2: Replace with object-aware version**

Replace the `hasOAuth` function in `cmd/cenv/create.go` with:

```go
// hasOAuth reports whether the user has Anthropic OAuth configured, indicated
// by a non-empty oauthAccount field in ~/.claude.json (home root, not ~/.claude/).
// Claude Code writes oauthAccount as an object; older versions may have used a
// string. Both shapes are accepted.
func hasOAuth(home string) bool {
	data, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		return false
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return false
	}
	switch v := parsed["oauthAccount"].(type) {
	case string:
		return v != ""
	case map[string]any:
		return len(v) > 0
	default:
		return false
	}
}
```

- [ ] **Step 3: Build to verify no compile error**

Run: `go build ./...`

Expected: no output, exit 0.

- [ ] **Step 4: Commit**

```bash
git add cmd/cenv/create.go
git commit -m "fix(cmd): hasOAuth accepts object-shaped oauthAccount

Mirrors the auth.Detect fix: Claude Code writes oauthAccount as a
JSON object, not a string. Without this the OAuth warning on
'cenv create' never fires for real users.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Update the OAuth warning message in `cenv create`

Current warning points at the (now-being-fixed) bean gt-wl86. Replace with a concrete next-step.

**Files:**
- Modify: `cmd/cenv/create.go:98-100`

- [ ] **Step 1: Replace the warning text**

In the `default:` branch of the `cenv create` switch (around line 98-100), change:

```go
if hasOAuth(home) {
    logf("[cenv] Warning: Anthropic OAuth detected in %s; login tokens won't transfer to the new env. You'll need to run 'claude /login' on first use. (see gt-wl86)\n", filepath.Join(home, ".claude.json"))
}
```

to:

```go
if hasOAuth(home) {
    logf("[cenv] Note: Anthropic OAuth detected. Login tokens don't transfer between envs; run 'cenv login %s' to authenticate this env.\n", name)
}
```

- [ ] **Step 2: Build to verify**

Run: `go build ./...`

Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add cmd/cenv/create.go
git commit -m "feat(cmd): point OAuth users at cenv login on create

Replaces the placeholder 'see gt-wl86' warning with a concrete next
step. The name substitution tells the user exactly which env to log
in to.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Add TTY detection helper

Needed by `cenv login` to reject non-interactive invocations.

**Files:**
- Create: `cmd/cenv/tty.go`
- Test: `cmd/cenv/tty_test.go`

- [ ] **Step 1: Write the helper and a failing test**

Create `cmd/cenv/tty_test.go`:

```go
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
```

Run: `go test ./cmd/cenv/ -run TestIsTerminal -v`

Expected: FAIL — `isTerminal` not defined.

- [ ] **Step 2: Implement `isTerminal`**

Create `cmd/cenv/tty.go`:

```go
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
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `go test ./cmd/cenv/ -run TestIsTerminal -v`

Expected: both tests PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/cenv/tty.go cmd/cenv/tty_test.go
git commit -m "feat(cmd): add isTerminal helper

Detects whether a file descriptor is a character device. Used by
'cenv login' to reject non-interactive invocations.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Add `cenv login <name>` command

New command that execs `claude` with `CLAUDE_CONFIG_DIR` set, after verifying stdin is a terminal.

**Files:**
- Create: `cmd/cenv/login.go`

- [ ] **Step 1: Implement the command**

Create `cmd/cenv/login.go`:

```go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/env"
)

var loginCmd = &cobra.Command{
	Use:   "login <name>",
	Short: "Open Claude in an environment so you can run /login",
	Long: `Opens the Claude Code REPL with CLAUDE_CONFIG_DIR pointed at the named
environment. Type /login inside the REPL to authenticate this env.

cenv login requires an interactive terminal. Agents and scripts should
create envs via 'cenv create' and prompt the user to run 'cenv login'.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !env.Exists(name) {
			return fmt.Errorf("environment %q does not exist", name)
		}
		if !isTerminal(os.Stdin) {
			return fmt.Errorf("cenv login requires an interactive terminal")
		}

		envDir := env.Path(name)

		claudePath, err := exec.LookPath("claude")
		if err != nil {
			return fmt.Errorf("claude not found in PATH")
		}

		logf("[cenv] Opening Claude in %q; run /login inside the REPL.\n", name)

		environ := append(os.Environ(), fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", envDir))
		return syscall.Exec(claudePath, []string{"claude"}, environ)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
```

- [ ] **Step 2: Build to verify**

Run: `go build ./...`

Expected: no output, exit 0.

- [ ] **Step 3: Write a test for the non-existent-env and non-TTY error paths**

Create `cmd/cenv/login_test.go`:

```go
package main

import (
	"os"
	"strings"
	"testing"
)

func TestLoginCmd_NonexistentEnv(t *testing.T) {
	t.Setenv("CENV_BASE", t.TempDir())

	err := loginCmd.RunE(loginCmd, []string{"does-not-exist"})
	if err == nil {
		t.Fatal("expected error for nonexistent env, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error = %q, want it to mention 'does not exist'", err.Error())
	}
}

func TestLoginCmd_RequiresTTY(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)
	if err := os.MkdirAll(base+"/test-env", 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}

	// go test pipes stdin, so isTerminal(os.Stdin) is false.
	err := loginCmd.RunE(loginCmd, []string{"test-env"})
	if err == nil {
		t.Fatal("expected TTY error, got nil")
	}
	if !strings.Contains(err.Error(), "terminal") {
		t.Errorf("error = %q, want it to mention 'terminal'", err.Error())
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/cenv/ -run TestLoginCmd -v`

Expected: both tests PASS. `TestLoginCmd_RequiresTTY` passes because `go test` attaches a pipe to stdin, so `isTerminal` returns false before we even reach the exec call.

- [ ] **Step 5: Commit**

```bash
git add cmd/cenv/login.go cmd/cenv/login_test.go
git commit -m "feat(cmd): add 'cenv login' for interactive OAuth auth

Opens Claude with CLAUDE_CONFIG_DIR set so the user can run /login
inside the REPL. Requires a TTY; errors clearly when invoked from a
script or agent.

Part of gt-wl86.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Pre-flight auth detection in `cenv run`

Hard error when `auth.Detect` returns "no auth found", pointing the user at `cenv login`.

**Files:**
- Modify: `cmd/cenv/run.go:26-30`

- [ ] **Step 1: Add a failing test for the pre-flight error**

Append to a new file `cmd/cenv/run_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCmd_NoAuth(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)

	// Create an env with settings.json but no auth configured.
	envDir := filepath.Join(base, "bare-env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	err := runCmd.RunE(runCmd, []string{"bare-env"})
	if err == nil {
		t.Fatal("expected auth pre-flight error, got nil")
	}
	if !strings.Contains(err.Error(), "cenv login") {
		t.Errorf("error = %q, want it to mention 'cenv login'", err.Error())
	}
}
```

Run: `go test ./cmd/cenv/ -run TestRunCmd_NoAuth -v`

Expected: FAIL — the current `cenv run` exec's claude regardless of auth state. In `go test`, this either panics or tries to exec, so the test fails.

- [ ] **Step 2: Add the pre-flight check to `cenv run`**

In `cmd/cenv/run.go`, add `"github.com/technicalpickles/cenv/internal/auth"` to the imports. Then, between the `settings.Load` call and the `claudeArgs` block (insert after line 30), add:

```go
	if _, err := auth.Detect(envDir); err != nil {
		return fmt.Errorf("env %q has no auth configured; run 'cenv login %s' first", name, name)
	}
```

The full modified function body (lines 20-52 area) should now read:

```go
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !env.Exists(name) {
			return fmt.Errorf("environment %q does not exist", name)
		}

		envDir := env.Path(name)
		settingsPath := filepath.Join(envDir, "settings.json")
		if _, err := settings.Load(settingsPath); err != nil {
			return fmt.Errorf("preflight failed: %w", err)
		}

		if _, err := auth.Detect(envDir); err != nil {
			return fmt.Errorf("env %q has no auth configured; run 'cenv login %s' first", name, name)
		}

		var claudeArgs []string
		if len(args) > 1 {
			if args[1] != "--" {
				return fmt.Errorf("unexpected argument %q (use -- before claude arguments)", args[1])
			}
			claudeArgs = args[2:]
		}

		logf("[cenv] Using %q (%s)\n", name, envDir)

		claudePath, err := exec.LookPath("claude")
		if err != nil {
			return fmt.Errorf("claude not found in PATH")
		}

		environ := os.Environ()
		environ = append(environ, fmt.Sprintf("CLAUDE_CONFIG_DIR=%s", envDir))

		execArgs := append([]string{"claude"}, claudeArgs...)
		return syscall.Exec(claudePath, execArgs, environ)
	},
```

- [ ] **Step 3: Run test to verify it passes**

Run: `go test ./cmd/cenv/ -run TestRunCmd_NoAuth -v`

Expected: PASS.

- [ ] **Step 4: Verify auth.Detect already covers Bedrock (sanity check)**

No additional test needed here. `internal/auth/auth_test.go` already has `TestDetect_AWSBedrock` and `TestDetect_AWSBedrock_NoRegion` that cover the passing case. The pre-flight in `cenv run` is a simple `auth.Detect(envDir)` call, so if Detect returns no error, the pre-flight is satisfied by construction.

Run: `go test ./internal/auth/ -run TestDetect_AWSBedrock -v`

Expected: both tests PASS (unchanged from pre-plan behavior).

- [ ] **Step 5: Commit**

```bash
git add cmd/cenv/run.go cmd/cenv/run_test.go
git commit -m "feat(cmd): pre-flight auth detection in cenv run

If no auth is detectable (no awsAuthRefresh in settings, no non-empty
oauthAccount in .claude.json), fail fast with a pointer at
'cenv login'. Agents and scripts get a clean, fast error instead of
Claude Code's interactive 'Please run /login' prompt.

Part of gt-wl86.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: `cenv auth create` refuses OAuth

OAuth auth envs carry no tokens, so `auth create` is unhelpful for OAuth users. Refuse with a concrete alternative.

**Files:**
- Modify: `cmd/cenv/auth.go:26-36`

- [ ] **Step 1: Add a failing test**

Create `cmd/cenv/auth_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthCreateCmd_RefusesOAuth(t *testing.T) {
	// Redirect HOME so Detect reads our fake .claude dir.
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("CENV_BASE", t.TempDir())

	claudeDir := filepath.Join(fakeHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("creating claude dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}
	oauth := `{"oauthAccount": {"emailAddress": "user@example.com"}}`
	if err := os.WriteFile(filepath.Join(claudeDir, ".claude.json"), []byte(oauth), 0644); err != nil {
		t.Fatalf("writing .claude.json: %v", err)
	}

	err := authCreateCmd.RunE(authCreateCmd, []string{})
	if err == nil {
		t.Fatal("expected refusal for OAuth user, got nil")
	}
	if !strings.Contains(err.Error(), "cenv login") {
		t.Errorf("error = %q, want it to mention 'cenv login'", err.Error())
	}
}

func TestAuthCreateCmd_AcceptsBedrock(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("CENV_BASE", t.TempDir())

	claudeDir := filepath.Join(fakeHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("creating claude dir: %v", err)
	}
	bedrock := `{"awsAuthRefresh": {"region": "us-west-2"}}`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(bedrock), 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	err := authCreateCmd.RunE(authCreateCmd, []string{})
	if err != nil {
		t.Fatalf("Bedrock auth create unexpectedly failed: %v", err)
	}
}
```

Run: `go test ./cmd/cenv/ -run TestAuthCreateCmd -v`

Expected: `TestAuthCreateCmd_RefusesOAuth` FAILS (currently succeeds and creates `auth-anthropic` env). `TestAuthCreateCmd_AcceptsBedrock` should PASS already.

- [ ] **Step 2: Add the OAuth refusal**

In `cmd/cenv/auth.go`, modify the `authCreateCmd.RunE` function. After the `Detect` call (line 32-35), insert an OAuth-type rejection before the `logf` line:

```go
		detected, err := auth.Detect(claudeDir)
		if err != nil {
			return fmt.Errorf("detecting auth: %w", err)
		}
		if detected.Type == "anthropic" {
			return fmt.Errorf("OAuth users don't need auth envs; each env requires its own 'cenv login <name>'. Try 'cenv create <name>' then 'cenv login <name>' instead")
		}
		logf("[cenv] Detected auth type: %s\n", detected.Type)
```

- [ ] **Step 3: Run tests to verify**

Run: `go test ./cmd/cenv/ -run TestAuthCreateCmd -v`

Expected: both tests PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/cenv/auth.go cmd/cenv/auth_test.go
git commit -m "feat(cmd): refuse 'auth create' for OAuth users

OAuth auth envs carry no tokens between CLAUDE_CONFIG_DIRs, so an
auth-anthropic template env serves no purpose. Refuse with a
message pointing at 'cenv create' + 'cenv login'. Bedrock
behavior is unchanged.

Part of gt-wl86.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: Document the OAuth flow in README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Read current README**

Run: `head -60 README.md` to find the authentication section (or the closest place to describe the flow).

- [ ] **Step 2: Add an OAuth flow section**

Add a subsection under the existing auth docs. If no auth docs exist, create one after the "Getting started" or "Usage" section:

```markdown
### Anthropic OAuth users

OAuth login tokens are stored per-CLAUDE_CONFIG_DIR in the macOS
Keychain, so they don't transfer between cenv envs. Each env needs
its own login:

```sh
cenv create my-env           # creates the env; prints a hint
cenv login my-env            # opens Claude; type /login inside the REPL
cenv run my-env -- -p 'hi'   # env is now authenticated
```

`cenv login` requires a terminal. For scripts and agents, `cenv run`
will fail fast with a message pointing at `cenv login` if the target
env has never been authenticated.

`cenv auth create` is not available for OAuth users — the auth env
pattern only carries tokens for Bedrock.
```

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: document OAuth auth flow

cenv create -> cenv login -> cenv run, and note that cenv auth create
is Bedrock-only.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: Full test suite + manual smoke test

Make sure nothing regressed before closing the beans.

- [ ] **Step 1: Run full test suite**

Run: `go test ./...`

Expected: all tests pass.

- [ ] **Step 2: Manual smoke: OAuth env lifecycle**

Only run this if you have OAuth configured locally (check `~/.claude.json` for `oauthAccount`).

```sh
# Build
go build -o /tmp/cenv ./cmd/cenv

# Fresh env
/tmp/cenv create smoke-oauth
# Expect: "[cenv] Note: Anthropic OAuth detected. ... run 'cenv login smoke-oauth'"

# Pre-flight should trip
/tmp/cenv run smoke-oauth -- -p 'hi' --model sonnet 2>&1 | head -5
# Expect: "env 'smoke-oauth' has no auth configured; run 'cenv login smoke-oauth' first"

# Interactive login (requires TTY)
/tmp/cenv login smoke-oauth
# Inside REPL: type /login, complete OAuth, /exit

# Now run should work
/tmp/cenv run smoke-oauth -- -p 'hi' --model sonnet

# has_auth should be true
/tmp/cenv list --json | grep -A4 smoke-oauth

# Cleanup
/tmp/cenv remove smoke-oauth
```

- [ ] **Step 3: Manual smoke: non-TTY login rejection**

```sh
echo '' | /tmp/cenv login smoke-oauth
# Expect: "cenv login requires an interactive terminal"
```

- [ ] **Step 4: Update bean statuses**

```sh
pt beans update gt-5zn7 --status completed
pt beans update gt-wl86 --status completed
```

- [ ] **Step 5: Final commit if smoke test revealed anything**

If smoke tests surfaced issues, commit fixes. Otherwise no commit needed.

---

## Out of scope / follow-ups

- Keychain hash reverse-engineering to fully automate OAuth (would let `auth create` and cross-env token copy work). Reserve for later.
- Claude Code CLI flag for non-interactive `/login` (upstream change).
- Detecting stale/deleted keychain entries via env-specific probe.
