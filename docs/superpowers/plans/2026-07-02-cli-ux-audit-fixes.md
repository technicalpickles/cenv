# cenv CLI UX Audit Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Task headers use stable kebab-case slugs, not integer ordinals — cite the slug (e.g. `wording-standardization`) in commit messages and cross-references, not a task number.

**Goal:** Fix six small, independent CLI UX issues found in a `/designing-clis` audit of `cenv`: inconsistent "not found" wording, missing `--version`, no help examples, a silently-dropped `--bare`/`--from` flag conflict, no confirmation on `cenv remove`, and no auth-status visibility in `cenv list`.

**Architecture:** Each task touches one command file (plus its test file) and is independently committable. No new packages; everything lives in `cmd/cenv/` except one wording fix in `internal/env/env.go`.

**Tech Stack:** Go 1.26.1, cobra (CLI framework), `text/tabwriter` (new — for `cenv list` table output). No new dependencies.

## Task Ordering

These six tasks are not fully independent: `help-examples` touches `login.go`, `run.go`, and `create.go`, which are also modified by `wording-standardization` and `bare-from-conflict`; and both `help-examples` and `remove-confirmation` touch `remove.go`. Run the tasks **in the order listed below** (sequentially, not in parallel) to avoid two tasks editing the same file at the same time. `version-flag` and `list-auth-status` touch files no other task touches and are safe to run in parallel with anything once their file-overlapping neighbors are done, but there's no wall-clock benefit worth the coordination cost here — just run all six in order.

**Update (2026-07-02):** `version-flag` was found already implemented upstream (commit `ccff1ae`, PR #8, merged into `origin/main` before this plan's implementation branch was created) — skipped, no changes made. The remaining five tasks proceed as written.

## Global Constraints

- Go toolchain: 1.26.1 (pinned in `mise.toml`) — do not require a newer version.
- `mise run check` (fmt + vet + test) must pass before any task is considered done.
- Canonical "not found" wording across the CLI: `environment %q not found` (verbatim, with the `%q`-quoted name).
- No color or symbols added in this round (spec explicitly scopes this out).
- Work happens on a branch; `main` requires a passing CI run and a PR (branch protection, per repo `CLAUDE.md`) — do not push directly to `main`.
- Spec: `docs/superpowers/specs/2026-07-02-cli-ux-audit-fixes-design.md`

---

### wording-standardization: Standardize "not found" error wording

**Files:**
- Modify: `cmd/cenv/login.go:26`
- Modify: `cmd/cenv/login_test.go:16-17`
- Modify: `cmd/cenv/run.go:24`
- Modify: `cmd/cenv/run_test.go` (add a new test function)
- Modify: `internal/env/env.go:75` (inside `Remove`)
- Modify: `internal/env/env_test.go` (inside `TestRemove`, subtest `"errors on non-existent env"`)

**Interfaces:**
- No signature changes. Only the string passed to `fmt.Errorf` changes in three call sites.

- [ ] **Step 1: Update the existing login test to expect the new wording**

In `cmd/cenv/login_test.go`, change:

```go
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error = %q, want it to mention 'does not exist'", err.Error())
	}
```

to:

```go
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found'", err.Error())
	}
```

- [ ] **Step 2: Run the login test to verify it now fails**

Run: `go test ./cmd/cenv/... -run TestLoginCmd_NonexistentEnv -v`
Expected: FAIL — current code still returns "does not exist".

- [ ] **Step 3: Fix the wording in login.go**

In `cmd/cenv/login.go`, change line 26 from:

```go
			return fmt.Errorf("environment %q does not exist", name)
```

to:

```go
			return fmt.Errorf("environment %q not found", name)
```

- [ ] **Step 4: Run the login test again to verify it passes**

Run: `go test ./cmd/cenv/... -run TestLoginCmd_NonexistentEnv -v`
Expected: PASS

- [ ] **Step 5: Add a new test asserting run.go's wording**

Add to `cmd/cenv/run_test.go`:

```go
func TestRunCmd_NonexistentEnv(t *testing.T) {
	t.Setenv("CENV_BASE", t.TempDir())

	err := runCmd.RunE(runCmd, []string{"does-not-exist"})
	if err == nil {
		t.Fatal("expected error for nonexistent env, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention 'not found'", err.Error())
	}
}
```

- [ ] **Step 6: Run it to verify it fails**

Run: `go test ./cmd/cenv/... -run TestRunCmd_NonexistentEnv -v`
Expected: FAIL — current code returns "does not exist".

- [ ] **Step 7: Fix the wording in run.go**

In `cmd/cenv/run.go`, change line 24 from:

```go
			return fmt.Errorf("environment %q does not exist", name)
```

to:

```go
			return fmt.Errorf("environment %q not found", name)
```

- [ ] **Step 8: Run it again to verify it passes**

Run: `go test ./cmd/cenv/... -run TestRunCmd_NonexistentEnv -v`
Expected: PASS

- [ ] **Step 9: Update env_test.go to assert the new wording**

In `internal/env/env_test.go`, inside `TestRemove`, change the `"errors on non-existent env"` subtest from:

```go
	t.Run("errors on non-existent env", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("CENV_BASE", tmp)

		if err := env.Remove("nonexistent"); err == nil {
			t.Error("Remove(\"nonexistent\") expected error, got nil")
		}
	})
```

to:

```go
	t.Run("errors on non-existent env", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("CENV_BASE", tmp)

		err := env.Remove("nonexistent")
		if err == nil {
			t.Fatal("Remove(\"nonexistent\") expected error, got nil")
		}
		if !strings.Contains(err.Error(), "environment \"nonexistent\" not found") {
			t.Errorf("Remove() error = %q, want it to contain %q", err.Error(), "environment \"nonexistent\" not found")
		}
	})
```

This requires adding `"strings"` to the import block at the top of `internal/env/env_test.go` if not already present (it currently imports `os`, `path/filepath`, `testing` — add `strings`).

- [ ] **Step 10: Run it to verify it fails**

Run: `go test ./internal/env/... -run TestRemove -v`
Expected: FAIL — current code returns `env %q not found` (missing "environment", wrong noun).

- [ ] **Step 11: Fix the wording in internal/env/env.go**

In `internal/env/env.go`, change line 75 from:

```go
		return fmt.Errorf("env %q not found", name)
```

to:

```go
		return fmt.Errorf("environment %q not found", name)
```

- [ ] **Step 12: Run it again to verify it passes**

Run: `go test ./internal/env/... -run TestRemove -v`
Expected: PASS

- [ ] **Step 13: Run the full test suite and commit**

Run: `mise run check`
Expected: all pass.

```bash
git add cmd/cenv/login.go cmd/cenv/login_test.go cmd/cenv/run.go cmd/cenv/run_test.go internal/env/env.go internal/env/env_test.go
git commit -m "fix: standardize 'environment %q not found' wording

login.go and run.go said 'does not exist'; env.Remove said 'env %q not
found' (dropping the word 'environment'). All three now match the
wording already used by path.go, settings.go, and trust.go."
```

---

### version-flag: Add `--version` flag

**SKIPPED — already implemented upstream.** See "Update (2026-07-02)" note above. Commit `ccff1ae` ("Add --version flag", PR #8) already added `var version = "dev"` + `rootCmd.Version` in `main.go` and the `-ldflags` build wiring in `mise.toml`, plus a `release.yml` tweak this plan didn't originally account for. No `main_test.go` was added by that PR, but per explicit decision this plan does not backfill tests for already-shipped code from a separate PR. Do not redo this task.

---

### bare-from-conflict: Reject conflicting `--bare` and `--from`

**Files:**
- Modify: `cmd/cenv/create.go`
- Create: `cmd/cenv/create_test.go`

**Interfaces:**
- No new exported symbols. Uses existing package-level `createBare bool` and `createFrom string` vars from `create.go:16-19`.

- [ ] **Step 1: Write the failing test**

Create `cmd/cenv/create_test.go`:

```go
package main

import (
	"strings"
	"testing"
)

func TestCreateCmd_BareAndFromMutuallyExclusive(t *testing.T) {
	t.Setenv("CENV_BASE", t.TempDir())

	createBare = true
	createFrom = "user"
	t.Cleanup(func() {
		createBare = false
		createFrom = ""
	})

	err := createCmd.RunE(createCmd, []string{"myenv"})
	if err == nil {
		t.Fatal("expected error for --bare + --from, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error = %q, want it to mention 'mutually exclusive'", err.Error())
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./cmd/cenv/... -run TestCreateCmd_BareAndFromMutuallyExclusive -v`
Expected: FAIL — today `--bare` silently wins and the env gets created with no error.

- [ ] **Step 3: Add the validation check**

In `cmd/cenv/create.go`, inside `RunE`, right after `name := args[0]` (line 26), add:

```go
		name := args[0]

		if createBare && createFrom != "" {
			return fmt.Errorf("--bare and --from are mutually exclusive")
		}

		if err := env.ValidateName(name); err != nil {
```

(This inserts the check before `env.ValidateName` so the conflict is caught before any filesystem side effects.)

- [ ] **Step 4: Run it again to verify it passes**

Run: `go test ./cmd/cenv/... -run TestCreateCmd_BareAndFromMutuallyExclusive -v`
Expected: PASS

- [ ] **Step 5: Run the full test suite and commit**

Run: `mise run check`
Expected: all pass.

```bash
git add cmd/cenv/create.go cmd/cenv/create_test.go
git commit -m "fix: error on conflicting --bare and --from flags

Previously --bare silently won and --from was dropped with no
warning. Now it's a hard error before any directory is created."
```

---

### help-examples: Add `Example:` text to every command

**Files:**
- Modify: `cmd/cenv/create.go`
- Modify: `cmd/cenv/run.go`
- Modify: `cmd/cenv/login.go`
- Modify: `cmd/cenv/remove.go`
- Modify: `cmd/cenv/path.go`
- Modify: `cmd/cenv/trust.go`
- Modify: `cmd/cenv/settings.go` (three subcommands: show, get, merge)
- Create: `cmd/cenv/help_examples_test.go`

**Interfaces:**
- No new exported symbols. Sets the `Example` field (a plain `string`, cobra's built-in field) on each existing `*cobra.Command` var: `createCmd`, `runCmd`, `loginCmd`, `removeCmd`, `pathCmd`, `trustCmd`, `settingsShowCmd`, `settingsGetCmd`, `settingsMergeCmd`.

- [ ] **Step 1: Write the failing test**

Create `cmd/cenv/help_examples_test.go`:

```go
package main

import "testing"

func TestHelpExamples_Present(t *testing.T) {
	cases := map[string]string{
		"create":         createCmd.Example,
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

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./cmd/cenv/... -run TestHelpExamples_Present -v`
Expected: FAIL — all nine `Example` fields are currently empty, so all nine subtests-as-errors fire (the test reports every missing one at once, not just the first).

- [ ] **Step 3: Add Example to createCmd**

In `cmd/cenv/create.go`, add `Example` to the `createCmd` struct literal (after `Short`):

```go
	Short: "Create a new environment",
	Example: `  cenv create myenv
  cenv create myenv --from user
  cenv create myenv --bare`,
```

- [ ] **Step 4: Add Example to runCmd**

In `cmd/cenv/run.go`, add to `runCmd` (after `Short`):

```go
	Short: "Launch Claude in an environment",
	Example: `  cenv run myenv
  cenv run myenv -- --model opus`,
```

- [ ] **Step 5: Add Example to loginCmd**

In `cmd/cenv/login.go`, add to `loginCmd` (after `Short`, before `Long`):

```go
	Short: "Open Claude in an environment so you can run /login",
	Example: `  cenv login myenv`,
	Long: `Opens the Claude Code REPL with CLAUDE_CONFIG_DIR pointed at the named
```

(i.e. insert the `Example:` line between the existing `Short:` and `Long:` lines — the rest of `Long` stays as-is.)

- [ ] **Step 6: Add Example to removeCmd**

In `cmd/cenv/remove.go`, add to `removeCmd` (after `Short`):

```go
	Short: "Remove an environment",
	Example: `  cenv remove myenv
  cenv remove myenv --force`,
```

(Note: `--force` doesn't exist yet at this point in the plan — it's added by the `remove-confirmation` task. This example documents the flag ahead of that task landing; if executing tasks out of order, swap this example to just `  cenv remove myenv` until `remove-confirmation` is done.)

- [ ] **Step 7: Add Example to pathCmd**

In `cmd/cenv/path.go`, add to `pathCmd` (after `Short`):

```go
	Short: "Print the directory path of an environment",
	Example: `  cenv path myenv`,
```

- [ ] **Step 8: Add Example to trustCmd**

In `cmd/cenv/trust.go`, add to `trustCmd` (after `Short`, before `Long`):

```go
	Short: "Mark workspace path(s) as trusted in an environment",
	Example: `  cenv trust myenv ~/projects/foo
  cenv trust myenv ~/projects/foo ~/projects/bar`,
	Long: `Mark one or more workspace paths as trusted in an environment's Claude
```

- [ ] **Step 9: Add Example to the three settings subcommands**

In `cmd/cenv/settings.go`, add to each of the three subcommands (after their respective `Short`):

```go
var settingsShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show settings for an environment as JSON",
	Example: `  cenv settings show myenv`,
```

```go
var settingsGetCmd = &cobra.Command{
	Use:   "get <name> <key>",
	Short: "Get a value from settings by dot-path key",
	Example: `  cenv settings get myenv statusLine.type`,
```

```go
var settingsMergeCmd = &cobra.Command{
	Use:   "merge <name> <json|file>",
	Short: "Deep merge JSON or a JSON file into environment settings",
	Example: `  cenv settings merge myenv '{"statusLine":{"type":"command"}}'`,
```

- [ ] **Step 10: Run it again to verify it passes**

Run: `go test ./cmd/cenv/... -run TestHelpExamples_Present -v`
Expected: PASS

- [ ] **Step 11: Spot-check the rendered help text**

Run: `mise run build && ./cenv create --help && ./cenv settings get --help`
Expected: both show an "Examples:" section with the text you added, formatted below the Usage line.

- [ ] **Step 12: Run the full test suite and commit**

Run: `mise run check`
Expected: all pass.

```bash
git add cmd/cenv/create.go cmd/cenv/run.go cmd/cenv/login.go cmd/cenv/remove.go cmd/cenv/path.go cmd/cenv/trust.go cmd/cenv/settings.go cmd/cenv/help_examples_test.go
git commit -m "docs: add Example text to every command's help output

Nothing set cobra's Example field before this, so --help never showed
a realistic invocation for any command."
```

---

### remove-confirmation: Confirm before `cenv remove` deletes

**Files:**
- Modify: `cmd/cenv/remove.go`
- Create: `cmd/cenv/remove_test.go`

**Interfaces:**
- Produces: `parseConfirmation(input string) bool` (pure function, package-private) and `confirmRemoval(name string) bool` (reads `os.Stdin`, package-private) in `remove.go`. No other task consumes these.
- Produces: `--force` bool flag (`removeForce` package var) on `removeCmd`.

- [ ] **Step 1: Write the failing tests**

Create `cmd/cenv/remove_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfirmation(t *testing.T) {
	cases := map[string]bool{
		"y\n":   true,
		"y":     true,
		"Y\n":   true,
		"yes\n": true,
		"YES":   true,
		" y \n": true,
		"n\n":   false,
		"no\n":  false,
		"\n":    false,
		"":      false,
		"maybe": false,
	}
	for input, want := range cases {
		if got := parseConfirmation(input); got != want {
			t.Errorf("parseConfirmation(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestRemoveCmd_ForceFlag_SkipsPrompt(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)
	envDir := filepath.Join(base, "myenv")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}

	removeForce = true
	t.Cleanup(func() { removeForce = false })

	if err := removeCmd.RunE(removeCmd, []string{"myenv"}); err != nil {
		t.Fatalf("remove err: %v", err)
	}
	if _, err := os.Stat(envDir); !os.IsNotExist(err) {
		t.Error("env dir still exists after forced remove")
	}
}

func TestRemoveCmd_NonTTY_ProceedsWithoutPrompt(t *testing.T) {
	base := t.TempDir()
	t.Setenv("CENV_BASE", base)
	envDir := filepath.Join(base, "myenv")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}

	// Redirect stdin to a pipe so isTerminal returns false, simulating a
	// script/CI invocation with no interactive terminal attached.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	t.Cleanup(func() { r.Close(); w.Close() })
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig })

	removeForce = false
	if err := removeCmd.RunE(removeCmd, []string{"myenv"}); err != nil {
		t.Fatalf("remove err: %v", err)
	}
	if _, err := os.Stat(envDir); !os.IsNotExist(err) {
		t.Error("env dir still exists after non-TTY remove")
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./cmd/cenv/... -run 'TestParseConfirmation|TestRemoveCmd' -v`
Expected: FAIL to compile — `parseConfirmation`, `confirmRemoval`, and `removeForce` don't exist yet.

- [ ] **Step 3: Implement the confirmation logic**

Replace the full contents of `cmd/cenv/remove.go` with:

```go
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/env"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an environment",
	Example: `  cenv remove myenv
  cenv remove myenv --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if !removeForce && !confirmRemoval(name) {
			fmt.Println("Aborted.")
			return nil
		}

		if err := env.Remove(name); err != nil {
			return err
		}
		logf("[cenv] Removed environment %q\n", name)
		return nil
	},
}

// confirmRemoval prompts for confirmation when stdin is a terminal.
// In non-interactive contexts (scripts, CI, piped input) it returns true
// immediately rather than blocking on input that will never arrive.
func confirmRemoval(name string) bool {
	if !isTerminal(os.Stdin) {
		return true
	}
	fmt.Fprintf(os.Stderr, "Remove environment %q? [y/N] ", name)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return parseConfirmation(line)
}

// parseConfirmation reports whether input is an affirmative response
// ("y" or "yes", case-insensitive, surrounding whitespace ignored).
func parseConfirmation(input string) bool {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

func init() {
	removeCmd.Flags().BoolVar(&removeForce, "force", false, "Skip confirmation prompt")
	rootCmd.AddCommand(removeCmd)
}
```

(This subsumes the `Example` field already added by the `help-examples` task — if that task ran first, just confirm the `Example` text matches; there's nothing left to change there.)

- [ ] **Step 4: Run it again to verify it passes**

Run: `go test ./cmd/cenv/... -run 'TestParseConfirmation|TestRemoveCmd' -v`
Expected: PASS

- [ ] **Step 5: Manually verify the interactive prompt**

Run: `mise run build && ./cenv create demo-remove-test && ./cenv remove demo-remove-test`
Expected: prompts `Remove environment "demo-remove-test"? [y/N]`. Type `n` (or just Enter) — expect `Aborted.` and the env still present (`./cenv list` shows it). Run `./cenv remove demo-remove-test` again and type `y` — expect `[cenv] Removed environment "demo-remove-test"` and it's gone from `./cenv list`. Clean up with `./cenv remove demo-remove-test --force` if the first attempt left it behind.

- [ ] **Step 6: Run the full test suite and commit**

Run: `mise run check`
Expected: all pass.

```bash
git add cmd/cenv/remove.go cmd/cenv/remove_test.go
git commit -m "feat: confirm before cenv remove deletes an environment

Prompts y/N when stdin is a terminal; --force or a non-TTY context
(scripts, CI) skips the prompt and proceeds immediately."
```

---

### list-auth-status: Show auth status in `cenv list`

**Files:**
- Modify: `cmd/cenv/list.go`
- Create: `cmd/cenv/list_test.go`

**Interfaces:**
- Consumes: `auth.Detect(configDir string) error` from `internal/auth` (nil = authenticated) — already used elsewhere via `env.Inspect`, but this task calls it directly per-env instead of going through `env.Inspect` (which also walks the whole directory tree for size/mtime, unneeded here).
- No new exported symbols; `--json` output (`env.Info`, unchanged) is untouched.

- [ ] **Step 1: Write the failing test**

Create `cmd/cenv/list_test.go`:

```go
package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./cmd/cenv/... -run TestListCmd_PlainOutputShowsAuthStatus -v`
Expected: FAIL — current output is bare names (`authed\nbare\n`), no `NAME`/`AUTH` header, no `yes`/`no` column.

- [ ] **Step 3: Implement the table output**

Replace the full contents of `cmd/cenv/list.go` with:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/technicalpickles/cenv/internal/auth"
	"github.com/technicalpickles/cenv/internal/env"
)

var listJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all environments",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := env.List()
		if err != nil {
			return err
		}

		if listJSON {
			infos := make([]*env.Info, 0, len(names))
			for _, name := range names {
				info, err := env.Inspect(name)
				if err != nil {
					return fmt.Errorf("inspecting %q: %w", name, err)
				}
				infos = append(infos, info)
			}
			out, err := json.MarshalIndent(infos, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling JSON: %w", err)
			}
			fmt.Println(string(out))
			return nil
		}

		if len(names) == 0 {
			fmt.Println("No environments yet.")
			fmt.Println("Create one: cenv create <name>")
			return nil
		}

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
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Emit environments as JSON with metadata")
	rootCmd.AddCommand(listCmd)
}
```

- [ ] **Step 4: Run it again to verify it passes**

Run: `go test ./cmd/cenv/... -run TestListCmd_PlainOutputShowsAuthStatus -v`
Expected: PASS

- [ ] **Step 5: Run the full list test file plus the JSON path to make sure nothing else broke**

Run: `go test ./cmd/cenv/... -run TestListCmd -v`
Expected: PASS (only one test exists so far, but confirms no regressions if more get added later).

- [ ] **Step 6: Manually verify both output modes**

Run: `mise run build && ./cenv list && ./cenv list --json`
Expected: plain output shows an aligned `NAME`/`AUTH` table; `--json` output is unchanged (still has `has_auth`, `size`, `mtime`, etc. per `env.Info`).

- [ ] **Step 7: Run the full test suite and commit**

Run: `mise run check`
Expected: all pass.

```bash
git add cmd/cenv/list.go cmd/cenv/list_test.go
git commit -m "feat: show auth status in cenv list's plain output

Previously only --json exposed has_auth (via env.Inspect, which also
walks the whole env directory for size/mtime). Plain list now checks
auth.Detect directly per env and renders an aligned NAME/AUTH table."
```

---

## After all tasks

Open a PR (branch protection on `main` requires one): push the branch, then `gh pr create` summarizing all six fixes with a link back to the spec at `docs/superpowers/specs/2026-07-02-cli-ux-audit-fixes-design.md`.
