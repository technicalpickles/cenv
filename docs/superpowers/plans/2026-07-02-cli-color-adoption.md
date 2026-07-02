# cenv CLI Color Adoption Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Task headers use stable kebab-case slugs, not integer ordinals — cite the slug (e.g. `style-package`) in commit messages and cross-references, not a task number.

**Goal:** Add a semantic color palette (success/error/warning/info/secondary) to cenv's existing confirmation messages, errors, and `cenv list`'s AUTH column, using `github.com/fatih/color`, with `--no-color`/`NO_COLOR` opt-outs and automatic TTY detection.

**Architecture:** A new `internal/style` package wraps `fatih/color` with semantic helper functions that return plain strings (no I/O). `cmd/cenv/main.go` computes one enable/disable toggle at startup (after flag parsing, via `PersistentPreRun`) and sets `fatih/color`'s global `color.NoColor` switch, which every `style.*` call then respects automatically. Command files compose `style.*` calls into their existing `logf`/`fmt.Fprintf` call sites — no change to *when* or *whether* a message prints, only to its formatting.

**Tech Stack:** Go 1.26.1, cobra (CLI framework, existing), `github.com/fatih/color` (new dependency).

## Task Ordering

Run these tasks **in the order listed** (sequentially, not in parallel). `style-package` must land first (everything else imports it). `create-and-remove-messages` must land before `trust-and-settings-messages`, because it introduces a shared test helper (`captureStderr`) that task consumes. `login-and-run-messages` and `list-auth-coloring` have no file overlap with any other task and could technically run any time after `style-package`, but there's no wall-clock benefit worth the coordination cost — just run all six in order.

## Global Constraints

- Go toolchain: 1.26.1 (pinned in `mise.toml`) — do not require a newer version.
- `mise run check` (fmt + vet + test) must pass before any task is considered done.
- Never color machine-parseable output: `cenv path`, `cenv settings get`/`show`, `cenv list --json` are untouched by this plan, in every task.
- Never rely on color alone: every colored *status message* (success/error/warning/info) keeps its symbol prefix (✓/✗/⚠/→) so meaning survives with color disabled. Table cells (`list`'s AUTH column) are the one exception — see `list-auth-coloring` for why.
- Work happens on a branch in a fresh worktree off `main` (current tip: `e08d28b`); `main` requires a passing CI run and a PR (branch protection, per repo `CLAUDE.md`) — do not push directly to `main`.
- Spec: `docs/superpowers/specs/2026-07-02-cli-color-adoption-design.md`

## Sandbox note for `go get`

This environment's network sandbox may not allow `proxy.golang.org` (Go's default module proxy) even when `github.com` itself is reachable. If `go get github.com/fatih/color` fails with a network/connection error in `style-package` Step 3, retry with the proxy bypassed:

```bash
GOPROXY=direct GOSUMDB=off go get github.com/fatih/color
```

`GOPROXY=direct` fetches straight from the module's VCS host (github.com) instead of the proxy; `GOSUMDB=off` skips the checksum database lookup (also normally at `sum.golang.org`), which isn't needed when fetching direct from a trusted source you can inspect.

---

### style-package: Add `internal/style` package and the `fatih/color` dependency

**Files:**
- Create: `internal/style/style.go`
- Create: `internal/style/style_test.go`
- Modify: `go.mod`, `go.sum` (via `go get`/`go mod tidy`)
- Modify: `CLAUDE.md:7`

**Interfaces:**
- Produces (consumed by every later task): `style.Success(format string, args ...any) string`, `style.Error(format string, args ...any) string`, `style.Warning(format string, args ...any) string`, `style.Info(format string, args ...any) string` — each colors the whole formatted string and prepends a symbol (`✓ `, `✗ `, `⚠ `, `→ ` respectively).
- Produces: `style.Secondary(text string) string` (gray, no symbol — de-emphasis, not a status) and `style.Green(text string) string` (green, no symbol — for table cells where a repeated status symbol per row would be redundant; the design doc's palette table shows the *table row* getting "green yes / gray no" with no symbol character, unlike every other row in that table which explicitly shows one).
- All six respect `fatih/color`'s package-level `color.NoColor` switch automatically (that's `fatih/color`'s own mechanism — nothing in `internal/style` needs to check it explicitly).

- [ ] **Step 1: Write the failing test**

Create `internal/style/style_test.go`:

```go
package style

import (
	"strings"
	"testing"

	"github.com/fatih/color"
)

func withColor(t *testing.T, enabled bool, fn func()) {
	t.Helper()
	orig := color.NoColor
	color.NoColor = !enabled
	t.Cleanup(func() { color.NoColor = orig })
	fn()
}

func TestSymbolHelpers(t *testing.T) {
	cases := []struct {
		name string
		fn   func() string
		want string // plain text, including symbol
		code string // ANSI SGR code fatih/color should use
	}{
		{"Success", func() string { return Success("Created %q", "foo") }, `✓ Created "foo"`, "32"},
		{"Error", func() string { return Error("environment %q not found", "foo") }, `✗ environment "foo" not found`, "31"},
		{"Warning", func() string { return Warning("disk usage high") }, `⚠ disk usage high`, "33"},
		{"Info", func() string { return Info("Using %q (%s)", "foo", "/tmp/foo") }, `→ Using "foo" (/tmp/foo)`, "34"},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/color disabled", func(t *testing.T) {
			withColor(t, false, func() {
				if got := tc.fn(); got != tc.want {
					t.Errorf("%s() = %q, want %q", tc.name, got, tc.want)
				}
			})
		})
		t.Run(tc.name+"/color enabled", func(t *testing.T) {
			withColor(t, true, func() {
				got := tc.fn()
				if !strings.Contains(got, tc.want) {
					t.Errorf("%s() = %q, want it to contain plain text %q", tc.name, got, tc.want)
				}
				if !strings.Contains(got, "\x1b["+tc.code) {
					t.Errorf("%s() = %q, want it to contain ANSI code %q", tc.name, got, tc.code)
				}
			})
		})
	}
}

func TestPlainColorHelpers(t *testing.T) {
	cases := []struct {
		name string
		fn   func(string) string
		code string
	}{
		{"Secondary", Secondary, "90"},
		{"Green", Green, "32"},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/color disabled", func(t *testing.T) {
			withColor(t, false, func() {
				if got := tc.fn("no"); got != "no" {
					t.Errorf("%s(%q) = %q, want %q", tc.name, "no", got, "no")
				}
			})
		})
		t.Run(tc.name+"/color enabled", func(t *testing.T) {
			withColor(t, true, func() {
				got := tc.fn("no")
				if !strings.Contains(got, "no") {
					t.Errorf("%s(%q) = %q, want it to contain %q", tc.name, "no", got, "no")
				}
				if !strings.Contains(got, "\x1b["+tc.code) {
					t.Errorf("%s(%q) = %q, want it to contain ANSI code %q", tc.name, "no", got, tc.code)
				}
			})
		})
	}
}
```

- [ ] **Step 2: Run it to verify it fails to compile**

Run: `go test ./internal/style/... -v`
Expected: FAIL to compile — `internal/style` package (and `Success`/`Error`/`Warning`/`Info`/`Secondary`/`Green`) don't exist yet, and `github.com/fatih/color` isn't a dependency yet.

- [ ] **Step 3: Add the `fatih/color` dependency**

Run: `go get github.com/fatih/color`

If this fails with a network error (see "Sandbox note for `go get`" above), retry with:

```bash
GOPROXY=direct GOSUMDB=off go get github.com/fatih/color
```

Expected: `go.mod` gains a `require github.com/fatih/color vX.Y.Z` line; `go.sum` gains matching entries. Two transitive dependencies (`github.com/mattn/go-isatty`, `github.com/mattn/go-colorable`) come along with it.

- [ ] **Step 4: Implement the style package**

Create `internal/style/style.go`:

```go
// Package style provides semantic color/symbol formatting for cenv's CLI
// output. Every colored status message pairs its color with a symbol
// (✓/✗/⚠/→) so meaning survives when color is disabled — piped output,
// --no-color, NO_COLOR, or a non-color terminal. Whether color is actually
// emitted is controlled by fatih/color's package-level color.NoColor
// switch, set once at startup by cmd/cenv/main.go; this package never
// checks TTY/env state itself.
package style

import "github.com/fatih/color"

// Success formats a success message: green text, "✓ " prefix.
func Success(format string, args ...any) string {
	return color.GreenString("✓ "+format, args...)
}

// Error formats an error message: red text, "✗ " prefix.
func Error(format string, args ...any) string {
	return color.RedString("✗ "+format, args...)
}

// Warning formats a warning message: yellow text, "⚠ " prefix.
func Warning(format string, args ...any) string {
	return color.YellowString("⚠ "+format, args...)
}

// Info formats an informational message: blue text, "→ " prefix.
func Info(format string, args ...any) string {
	return color.BlueString("→ "+format, args...)
}

// Secondary de-emphasizes text: gray, no symbol prefix. For metadata and
// de-emphasized values (e.g. "no" in an auth-status column), not statuses.
func Secondary(text string) string {
	return color.HiBlackString("%s", text)
}

// Green highlights text as affirmative without a status-line symbol
// prefix: for table cells where a symbol on every row would be redundant
// with the column header (e.g. "yes" in an auth-status column).
func Green(text string) string {
	return color.GreenString("%s", text)
}
```

- [ ] **Step 5: Run the test again to verify it passes**

Run: `go test ./internal/style/... -v`
Expected: PASS

- [ ] **Step 6: Update CLAUDE.md's dependency description**

In `CLAUDE.md`, change line 7 from:

```
`cenv` manages isolated Claude Code configuration directories — each env gets its own `settings.json`, `.claude.json`, plugins, hooks, and session history, independent of `~/.claude/`. Think `virtualenv` for Claude Code. A Go CLI built with cobra; no other runtime dependencies.
```

to:

```
`cenv` manages isolated Claude Code configuration directories — each env gets its own `settings.json`, `.claude.json`, plugins, hooks, and session history, independent of `~/.claude/`. Think `virtualenv` for Claude Code. A Go CLI built with cobra, with `fatih/color` for terminal output styling.
```

- [ ] **Step 7: Tidy modules and run the full test suite**

Run: `go mod tidy && mise run check`
Expected: all pass; `go.mod`/`go.sum` unchanged by `tidy` beyond what Step 3 already added (or only whitespace/ordering normalization).

- [ ] **Step 8: Commit**

```bash
git add internal/style/style.go internal/style/style_test.go go.mod go.sum CLAUDE.md
git commit -m "feat: add internal/style package with fatih/color

Semantic color+symbol helpers (Success/Error/Warning/Info/Secondary/
Green) for the CLI's confirmation, error, and table output. Nothing
calls these yet — that's the rest of this branch."
```

---

### color-toggle-and-errors: Wire `--no-color`/`NO_COLOR` and colorize errors

**Files:**
- Modify: `cmd/cenv/main.go`
- Create: `cmd/cenv/main_test.go`

**Interfaces:**
- Consumes: `style.Error(format string, args ...any) string` from `style-package`.
- Produces: `colorEnabled(noColorFlag bool, noColorEnvSet bool, stdoutIsTTY bool) bool` (pure function, package-private) — not consumed by any other task, but documents the exact opt-out precedence for future reference.
- Produces: `--no-color` bool flag (`noColor` package var) on `rootCmd`, alongside the existing `quiet`/`--quiet`.

- [ ] **Step 1: Write the failing test**

Create `cmd/cenv/main_test.go`:

```go
package main

import "testing"

func TestColorEnabled(t *testing.T) {
	cases := []struct {
		name          string
		noColorFlag   bool
		noColorEnvSet bool
		stdoutIsTTY   bool
		want          bool
	}{
		{"tty, nothing disabled", false, false, true, true},
		{"--no-color set", true, false, true, false},
		{"NO_COLOR set", false, true, true, false},
		{"stdout not a tty", false, false, false, false},
		{"everything disabled", true, true, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := colorEnabled(tc.noColorFlag, tc.noColorEnvSet, tc.stdoutIsTTY)
			if got != tc.want {
				t.Errorf("colorEnabled(%v, %v, %v) = %v, want %v",
					tc.noColorFlag, tc.noColorEnvSet, tc.stdoutIsTTY, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run it to verify it fails to compile**

Run: `go test ./cmd/cenv/... -run TestColorEnabled -v`
Expected: FAIL to compile — `colorEnabled` doesn't exist yet.

- [ ] **Step 3: Implement the toggle, flag, and colorized error printing**

Replace the full contents of `cmd/cenv/main.go` with:

```go
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/style"
)

var quiet bool
var noColor bool

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "cenv",
	Short: "Manage isolated Claude Code configuration directories",
	Long: `cenv manages isolated Claude Code configuration directories.
Each one gets its own settings, permissions, hooks, plugins, and session
history, completely independent of ~/.claude/. Think virtualenv for Claude Code.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", envBool("CENV_QUIET"), "Suppress [cenv] informational output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		applyColorSetting()
	}
}

// logf writes informational output to stderr unless quiet mode is enabled.
func logf(format string, args ...any) {
	if quiet {
		return
	}
	fmt.Fprintf(os.Stderr, format, args...)
}

// envBool reads a boolean env var. Empty or unparseable values return false.
func envBool(key string) bool {
	v := os.Getenv(key)
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}

// colorEnabled decides whether cenv should emit ANSI color codes. Color is
// enabled only when nothing disables it: no --no-color flag, no NO_COLOR
// env var (per https://no-color.org, presence disables regardless of
// value), and stdout is an actual terminal (not piped/redirected).
func colorEnabled(noColorFlag bool, noColorEnvSet bool, stdoutIsTTY bool) bool {
	return !noColorFlag && !noColorEnvSet && stdoutIsTTY
}

// applyColorSetting sets fatih/color's global switch from the current
// flag/env/TTY state. Called from PersistentPreRun (after flags are
// parsed, before any command's RunE) and again from main's error path
// (in case Execute failed before reaching PersistentPreRun, e.g. on a
// flag-parse error).
func applyColorSetting() {
	color.NoColor = !colorEnabled(noColor, os.Getenv("NO_COLOR") != "", isTerminal(os.Stdout))
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		applyColorSetting()
		fmt.Fprintln(os.Stderr, style.Error("%v", err))
		os.Exit(1)
	}
}
```

(`SilenceErrors: true` is the key change alongside the existing `SilenceUsage: true` — without it, cobra still prints its own plain `Error: <err>` in addition to whatever `main` prints.)

- [ ] **Step 4: Run the test again to verify it passes**

Run: `go test ./cmd/cenv/... -run TestColorEnabled -v`
Expected: PASS

- [ ] **Step 5: Run the full test suite**

Run: `mise run check`
Expected: all pass — no existing test asserts on cobra's own `Error: ...` prefix (confirm with `grep -rn '"Error: '  cmd/cenv/*_test.go` if unsure; it should find nothing).

- [ ] **Step 6: Manually verify colorized errors and the opt-outs**

Run: `mise run build`

```bash
./cenv run does-not-exist        # expect a red "✗ environment ... not found" on a color terminal
./cenv run does-not-exist --no-color   # expect plain "✗ environment ... not found", no ANSI codes
NO_COLOR=1 ./cenv run does-not-exist   # expect plain, same as --no-color
./cenv run does-not-exist | cat        # piping: expect plain (stdout no longer a TTY)
```

Expected: first case shows a colored line if run in a real terminal; the other three show identical plain text with no escape codes (verify by piping through `cat -v` if unsure — `cat -v` renders `\x1b[` as `^[[`, so its absence confirms no color codes are present).

- [ ] **Step 7: Commit**

```bash
git add cmd/cenv/main.go cmd/cenv/main_test.go
git commit -m "feat: add --no-color flag and colorize CLI errors

SilenceErrors + a single styled error printer in main() replaces
cobra's plain 'Error: ...' output. Color is enabled only when stdout
is a real terminal and neither --no-color nor NO_COLOR disable it."
```

---

### create-and-remove-messages: Colorize `create` and `remove` output

**Files:**
- Modify: `cmd/cenv/create.go`
- Modify: `cmd/cenv/remove.go`
- Create: `cmd/cenv/testhelpers_test.go`
- Modify: `cmd/cenv/create_test.go`
- Modify: `cmd/cenv/remove_test.go`

**Interfaces:**
- Consumes: `style.Success`, `style.Warning`, `style.Secondary` from `style-package`.
- Produces (consumed by `trust-and-settings-messages` and `login-and-run-messages`): `captureStderr(t *testing.T, fn func()) string` in `cmd/cenv/testhelpers_test.go` — redirects `os.Stderr` to a pipe for the duration of `fn`, returns everything written to it.

- [ ] **Step 1: Add the shared stderr-capture test helper**

Create `cmd/cenv/testhelpers_test.go`:

```go
package main

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// captureStderr runs fn with os.Stderr redirected to a pipe and returns
// everything written to it during the call.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = orig })

	fn()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
```

- [ ] **Step 2: Write the failing tests for create.go and remove.go's messages**

Add to `cmd/cenv/create_test.go` (append to the existing file, keep its current `TestCreateCmd_BareAndFromMutuallyExclusive`):

```go
func TestCreateCmd_SuccessMessageHasSymbol(t *testing.T) {
	t.Setenv("CENV_BASE", t.TempDir())
	createBare = true
	t.Cleanup(func() { createBare = false })

	out := captureStderr(t, func() {
		if err := createCmd.RunE(createCmd, []string{"myenv"}); err != nil {
			t.Fatalf("create err: %v", err)
		}
	})

	if !strings.Contains(out, `✓ Created environment "myenv"`) {
		t.Errorf("output = %q, want it to contain the ✓-prefixed success message", out)
	}
}
```

This requires adding `"strings"` to the import block at the top of `cmd/cenv/create_test.go` if not already present (it currently imports `strings` and `testing` only — confirm before adding a duplicate).

Add to `cmd/cenv/remove_test.go` (append to the existing file):

```go
func TestRemoveCmd_SuccessMessageHasSymbol(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)
	envDir := filepath.Join(base, "myenv")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	removeForce = true
	t.Cleanup(func() { removeForce = false })

	out := captureStderr(t, func() {
		if err := removeCmd.RunE(removeCmd, []string{"myenv"}); err != nil {
			t.Fatalf("remove err: %v", err)
		}
	})

	if !strings.Contains(out, `✓ Removed environment "myenv"`) {
		t.Errorf("output = %q, want it to contain the ✓-prefixed success message", out)
	}
}
```

- [ ] **Step 3: Run both to verify they fail**

Run: `go test ./cmd/cenv/... -run 'TestCreateCmd_SuccessMessageHasSymbol|TestRemoveCmd_SuccessMessageHasSymbol' -v`
Expected: FAIL — current messages are plain `[cenv] Created environment "myenv"` / `[cenv] Removed environment "myenv"`, no `✓` symbol.

- [ ] **Step 4: Colorize create.go's messages**

In `cmd/cenv/create.go`, add the style import to the import block:

```go
import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/bootstrap"
	"github.com/technicalpickles/cenv/internal/claudeconfig"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/keychain"
	"github.com/technicalpickles/cenv/internal/settings"
	"github.com/technicalpickles/cenv/internal/style"
)
```

Change:

```go
			if copied {
				logf("[cenv] Copied OAuth login from %s\n", displaySourceName(sourceDir))
			}
```

to:

```go
			if copied {
				logf("%s\n", style.Success("Copied OAuth login from %s", displaySourceName(sourceDir)))
			}
```

Change:

```go
		cleanupNeeded = false
		logf("[cenv] Created environment %q\n", name)
```

to:

```go
		cleanupNeeded = false
		logf("%s\n", style.Success("Created environment %q", name))
```

Change:

```go
		if delErr := kc.Delete(dstSvc); delErr != nil {
			logf("[cenv] Warning: failed to roll back keychain entry %q: %v\n", dstSvc, delErr)
		}
```

to:

```go
		if delErr := kc.Delete(dstSvc); delErr != nil {
			logf("%s\n", style.Warning("failed to roll back keychain entry %q: %v", dstSvc, delErr))
		}
```

(The `[cenv]` prefix is dropped from these three messages — the symbol now plays that role, signaling "this is cenv talking to you" the way the prefix used to. Untouched everywhere else: `logf` itself, `--quiet`/`CENV_QUIET` suppression, every returned `fmt.Errorf`.)

- [ ] **Step 5: Colorize remove.go's messages**

In `cmd/cenv/remove.go`, add the style import:

```go
import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/style"
)
```

Change:

```go
		if !removeForce && !confirmRemoval(name) {
			fmt.Println("Aborted.")
			return nil
		}

		if err := env.Remove(name); err != nil {
			return err
		}
		logf("[cenv] Removed environment %q\n", name)
```

to:

```go
		if !removeForce && !confirmRemoval(name) {
			fmt.Println(style.Secondary("Aborted."))
			return nil
		}

		if err := env.Remove(name); err != nil {
			return err
		}
		logf("%s\n", style.Success("Removed environment %q", name))
```

- [ ] **Step 6: Run the tests again to verify they pass**

Run: `go test ./cmd/cenv/... -run 'TestCreateCmd_SuccessMessageHasSymbol|TestRemoveCmd_SuccessMessageHasSymbol' -v`
Expected: PASS

- [ ] **Step 7: Run the full test suite**

Run: `mise run check`
Expected: all pass, including the pre-existing `TestCreateCmd_BareAndFromMutuallyExclusive`, `TestParseConfirmation`, `TestRemoveCmd_ForceFlag_SkipsPrompt`, `TestRemoveCmd_NonTTY_ProceedsWithoutPrompt` (none of these assert on the success/aborted message text, only on error text or side effects, so none should break).

- [ ] **Step 8: Manually verify**

Run: `mise run build && ./cenv create demo-color-test --bare && ./cenv remove demo-color-test`
Expected: `✓ Created environment "demo-color-test"` (colored if in a terminal), then a `y/N` prompt; typing `y` shows `✓ Removed environment "demo-color-test"`; typing `n` shows a gray `Aborted.`.

- [ ] **Step 9: Commit**

```bash
git add cmd/cenv/create.go cmd/cenv/remove.go cmd/cenv/testhelpers_test.go cmd/cenv/create_test.go cmd/cenv/remove_test.go
git commit -m "feat: colorize create and remove confirmation messages

Success lines get a green checkmark, the keychain-rollback warning
gets a yellow warning symbol, and remove's Aborted. is de-emphasized
in gray. Drops the old [cenv] text prefix in favor of the symbol."
```

---

### trust-and-settings-messages: Colorize `trust` and `settings merge` output

**Files:**
- Modify: `cmd/cenv/trust.go`
- Modify: `cmd/cenv/settings.go`

**Interfaces:**
- Consumes: `style.Success` from `style-package`; `captureStderr` from `create-and-remove-messages`.

- [ ] **Step 1: Write the failing tests**

Create `cmd/cenv/trust_success_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrustCmd_SuccessMessageHasSymbol(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)
	envDir := filepath.Join(base, "myenv")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, ".claude.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("writing .claude.json: %v", err)
	}

	out := captureStderr(t, func() {
		if err := trustCmd.RunE(trustCmd, []string{"myenv", "/tmp/some/path"}); err != nil {
			t.Fatalf("trust err: %v", err)
		}
	})

	if !strings.Contains(out, "✓ Trusted") {
		t.Errorf("output = %q, want it to contain the ✓-prefixed success message", out)
	}
}
```

Add to `cmd/cenv/settings.go`'s test file — check first whether `cmd/cenv/settings_test.go` already exists; if not, create it:

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsMergeCmd_SuccessMessageHasSymbol(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)
	envDir := filepath.Join(base, "myenv")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "settings.json"), []byte(`{}`), 0644); err != nil {
		t.Fatalf("writing settings.json: %v", err)
	}

	out := captureStderr(t, func() {
		if err := settingsMergeCmd.RunE(settingsMergeCmd, []string{"myenv", `{"foo":"bar"}`}); err != nil {
			t.Fatalf("settings merge err: %v", err)
		}
	})

	if !strings.Contains(out, `✓ Merged settings into "myenv"`) {
		t.Errorf("output = %q, want it to contain the ✓-prefixed success message", out)
	}
}
```

- [ ] **Step 2: Run both to verify they fail**

Run: `go test ./cmd/cenv/... -run 'TestTrustCmd_SuccessMessageHasSymbol|TestSettingsMergeCmd_SuccessMessageHasSymbol' -v`
Expected: FAIL — current messages are plain `[cenv] Trusted ...` / `[cenv] Merged settings into ...`.

- [ ] **Step 3: Colorize trust.go's message**

In `cmd/cenv/trust.go`, add the style import:

```go
import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/claudeconfig"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/style"
)
```

Change:

```go
		for _, p := range absPaths {
			logf("[cenv] Trusted %q in %q\n", p, name)
		}
```

to:

```go
		for _, p := range absPaths {
			logf("%s\n", style.Success("Trusted %q in %q", p, name))
		}
```

- [ ] **Step 4: Colorize settings.go's message**

In `cmd/cenv/settings.go`, add the style import:

```go
import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/settings"
	"github.com/technicalpickles/cenv/internal/style"
)
```

Change:

```go
		logf("[cenv] Merged settings into %q\n", name)
```

to:

```go
		logf("%s\n", style.Success("Merged settings into %q", name))
```

- [ ] **Step 5: Run the tests again to verify they pass**

Run: `go test ./cmd/cenv/... -run 'TestTrustCmd_SuccessMessageHasSymbol|TestSettingsMergeCmd_SuccessMessageHasSymbol' -v`
Expected: PASS

- [ ] **Step 6: Run the full test suite**

Run: `mise run check`
Expected: all pass.

- [ ] **Step 7: Manually verify**

Run: `mise run build && ./cenv create demo-trust-test --bare && ./cenv trust demo-trust-test /tmp && ./cenv settings merge demo-trust-test '{"foo":"bar"}' && ./cenv remove demo-trust-test --force`
Expected: `✓ Trusted "/tmp" in "demo-trust-test"` and `✓ Merged settings into "demo-trust-test"`, both colored in a terminal.

- [ ] **Step 8: Commit**

```bash
git add cmd/cenv/trust.go cmd/cenv/trust_success_test.go cmd/cenv/settings.go cmd/cenv/settings_test.go
git commit -m "feat: colorize trust and settings merge confirmation messages"
```

---

### login-and-run-messages: Colorize `login` and `run` output

**Files:**
- Modify: `cmd/cenv/login.go`
- Modify: `cmd/cenv/run.go`

**Interfaces:**
- Consumes: `style.Info` from `style-package`.

**Note:** `login` and `run` both `syscall.Exec` into `claude` after printing their info message, replacing the current process — there is no return from a successful run, and no `claude` binary is guaranteed to be installed in CI. So this task doesn't add new tests: `style.Info`'s formatting is already covered by `style-package`'s own tests, and the only thing changing here is which string `logf` is given at each call site — there's no new behavior to assert beyond what manual verification (Step 4) checks. This is a case where TDD's red step doesn't apply: there is no new externally-observable contract to test, only an internal call-site edit.

- [ ] **Step 1: Colorize login.go's message**

In `cmd/cenv/login.go`, add the style import:

```go
import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/style"
)
```

Change:

```go
		logf("[cenv] Opening Claude in %q; run /login inside the REPL.\n", name)
```

to:

```go
		logf("%s\n", style.Info("Opening Claude in %q; run /login inside the REPL.", name))
```

- [ ] **Step 2: Colorize run.go's message**

In `cmd/cenv/run.go`, add the style import:

```go
import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/auth"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/settings"
	"github.com/technicalpickles/cenv/internal/style"
)
```

Change:

```go
		logf("[cenv] Using %q (%s)\n", name, envDir)
```

to:

```go
		logf("%s\n", style.Info("Using %q (%s)", name, envDir))
```

- [ ] **Step 3: Run the full test suite to confirm nothing broke**

Run: `mise run check`
Expected: all pass. `login_test.go`/`run_test.go`'s existing tests (`TestLoginCmd_NonexistentEnv`, `TestRunCmd_NonexistentEnv`, plus whatever else exists) only assert on returned error text, which is untouched by this task — these commands still `return fmt.Errorf(...)` exactly as before; only the one `logf` info line changed.

- [ ] **Step 4: Manually verify (requires `claude` in PATH, or expect the "not found" error after seeing the colored line)**

Run: `mise run build && ./cenv create demo-login-test --bare`

```bash
./cenv login demo-login-test
```

Expected: prints `→ Opening Claude in "demo-login-test"; run /login inside the REPL.` (colored in a terminal) before attempting to exec `claude`. If `claude` isn't installed, it'll then fail with `claude not found in PATH` — that's expected and fine, the info line printed first is what this task verifies. Clean up: `./cenv remove demo-login-test --force`.

- [ ] **Step 5: Commit**

```bash
git add cmd/cenv/login.go cmd/cenv/run.go
git commit -m "feat: colorize login and run info messages"
```

---

### list-auth-coloring: Colorize `cenv list`'s AUTH column

**Files:**
- Modify: `cmd/cenv/list.go`
- Modify: `cmd/cenv/list_test.go`

**Interfaces:**
- Consumes: `style.Green`, `style.Secondary` from `style-package`.
- No new exported symbols; `--json` output (`env.Info`, unchanged) stays untouched.

**Why the AUTH column doesn't get a symbol, unlike every other colored message:** `text/tabwriter` computes column *padding* from cell byte length, but the AUTH column here is the last cell on each line (`fmt.Fprintf(w, "%s\t%s\n", name, status)` — no trailing tab after `status`). Per `text/tabwriter`'s semantics, a cell terminated by `\n` rather than `\t` is not part of an aligned column and is written out as-is, unpadded. That means embedded ANSI codes in the AUTH cell can't cause misalignment — but a `✓`/symbol character in a table cell would still visually clutter a table whose header (`AUTH`) already conveys the meaning perfectly well without it. Hence `style.Green`/`style.Secondary` (no symbol), not `style.Success`.

- [ ] **Step 1: Write the failing assertion for colored output**

`cmd/cenv/list_test.go` already has `TestListCmd_PlainOutputShowsAuthStatus`, which checks `strings.Contains(authedLine, "yes")` and `strings.Contains(bareLine, "no")` — those keep passing unchanged (color is off by default under `go test`, since stdout isn't a TTY in a test binary, so `style.Green("yes")`/`style.Secondary("no")` render as plain `"yes"`/`"no"`). Add a new test that forces color on to prove the column actually gets colored when enabled. Append to `cmd/cenv/list_test.go`:

```go
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
```

This requires adding `"github.com/fatih/color"` to the import block at the top of `cmd/cenv/list_test.go` (it currently imports `bytes`, `io`, `os`, `path/filepath`, `strings`, `testing`; add `color` alongside them).

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./cmd/cenv/... -run TestListCmd_ColoredOutputHasNoSymbolClutter -v`
Expected: FAIL — current AUTH column is plain, no ANSI codes present.

- [ ] **Step 3: Colorize the AUTH column**

In `cmd/cenv/list.go`, add the style import:

```go
import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/auth"
	"github.com/technicalpickles/cenv/internal/env"
	"github.com/technicalpickles/cenv/internal/style"
)
```

Change:

```go
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tAUTH")
		for _, name := range names {
			status := "no"
			if auth.Detect(env.Path(name)) == nil {
				status = "yes"
			}
			fmt.Fprintf(w, "%s\t%s\n", name, status)
		}
		return w.Flush()
```

to:

```go
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tAUTH")
		for _, name := range names {
			status := style.Secondary("no")
			if auth.Detect(env.Path(name)) == nil {
				status = style.Green("yes")
			}
			fmt.Fprintf(w, "%s\t%s\n", name, status)
		}
		return w.Flush()
```

- [ ] **Step 4: Run the test again to verify it passes**

Run: `go test ./cmd/cenv/... -run TestListCmd_ColoredOutputHasNoSymbolClutter -v`
Expected: PASS

- [ ] **Step 5: Run the full test suite**

Run: `mise run check`
Expected: all pass, including the pre-existing `TestListCmd_PlainOutputShowsAuthStatus` (unaffected — it runs with color off, same as always under `go test`).

- [ ] **Step 6: Manually verify both output modes**

Run: `mise run build && ./cenv create demo-list-test --bare && ./cenv list && ./cenv list --json && ./cenv remove demo-list-test --force`
Expected: plain-mode `list` shows the AUTH column colored (green `yes` for anything authed, gray `no` otherwise) with columns still aligned; `--json` output is byte-for-byte unchanged (still has `has_auth`, `size`, `mtime`, etc., no color codes — JSON output is never touched by `style`).

- [ ] **Step 7: Commit**

```bash
git add cmd/cenv/list.go cmd/cenv/list_test.go
git commit -m "feat: colorize cenv list's AUTH column

Green 'yes' / gray 'no', no status symbol (would clutter a table
whose AUTH header already conveys the meaning). --json is untouched."
```

---

## After all tasks

Open a PR (branch protection on `main` requires one): push the branch, then `gh pr create` summarizing the color adoption with a link back to the spec at `docs/superpowers/specs/2026-07-02-cli-color-adoption-design.md`.
