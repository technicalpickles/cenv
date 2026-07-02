# cenv CLI color adoption

**Date:** 2026-07-02
**Status:** Approved

## Background

The original `/designing-clis` audit of cenv (`docs/superpowers/specs/2026-07-02-cli-ux-audit-fixes-design.md`) explicitly scoped out color: "No color/symbols added (no color library in use today; not part of this round)." This spec picks that thread back up.

cenv's output surface is small: six commands print confirmations, one table, and machine-parseable values; `run` and `login` hand the terminal off to `claude` via `syscall.Exec` and print nothing themselves after that. Concretely, today's output is:

- Single-line `[cenv] ...` confirmations via `logf` (stderr, suppressible with `--quiet`/`CENV_QUIET`): `create`, `remove`, `trust`, `settings merge`, `login`, `run`, plus one warning line in `create` (keychain rollback failure).
- `remove`'s `y/N` confirmation prompt and its "Aborted." message.
- `list`'s two-column `NAME`/`AUTH` table (`text/tabwriter`), where AUTH is plain `yes`/`no`.
- Machine-parseable output: `path` (a bare path, meant for `$(cenv path foo)`), `settings get` (a JSON blob or scalar), `list --json`.
- Errors, returned from `RunE` and printed by cobra's default `Error: <err>` to stderr.

No command does slow or multi-step work — `create`'s I/O is local disk writes plus one `security` CLI shellout, sub-second in practice — so there's no case for progress bars or spinners here. This spec only covers the Color technique from the visual-techniques guide (plus the Symbols technique, since the guide's own top anti-pattern is color-only meaning).

## Goals

- Add a semantic color palette (success/error/warning/info/secondary) to cenv's existing confirmation messages, errors, and `list`'s AUTH column.
- Never rely on color alone: every colored message keeps a symbol prefix so meaning survives with color disabled.
- Respect standard color opt-outs: a `--no-color` flag and the `NO_COLOR` env var, on top of automatic TTY detection.
- Leave every machine-parseable output path (`path`, `settings get`, `list --json`) completely untouched — no color, ever, regardless of TTY state.

## Non-goals

- Progress bars, spinners, or other Structured Feedback patterns — no operation in cenv is slow enough to warrant them.
- Layout changes (panels, borders, multi-column restructuring) — `list`'s existing tabwriter table is sufficient structure for two columns.
- Changing `logf`'s `--quiet`/`CENV_QUIET` suppression behavior — color is orthogonal to whether a message prints at all.

## Design

### New package: `internal/style`

Wraps `github.com/fatih/color` with semantic helpers that return plain strings (no I/O of their own), so they compose into cenv's existing `logf`/`fmt.Fprintf` call sites without changing how those functions write or suppress output:

```go
package style

func Success(format string, args ...any) string  // green, "✓ " prefix
func Error(format string, args ...any) string     // red,   "✗ " prefix
func Warning(format string, args ...any) string   // yellow,"⚠ " prefix
func Info(format string, args ...any) string       // blue,  "→ " prefix
func Secondary(text string) string                  // gray, no prefix
```

`Secondary` has no symbol prefix (matching the guide's "Gray = timestamps/metadata" usage, which is de-emphasis, not a status) — used for `remove`'s "Aborted." and `list`'s "no" AUTH value.

### Enable/disable logic

A single package-level toggle, computed once at startup (in `cmd/cenv/main.go`'s `init` or `main`, before any command runs):

```
enabled = !noColorFlag && os.Getenv("NO_COLOR") == "" && isatty.IsTerminal(os.Stdout.Fd())
```

This sets `color.NoColor = !enabled` (fatih/color's own global switch, which every `style.*` call respects automatically).

Deliberately checking `os.Stdout` only, not stdout+stderr separately: cenv's colored output spans both streams (`logf` writes to stderr, `list`'s table writes to stdout), and per-stream detection would add real complexity for an edge case (stdout piped while stderr stays an interactive terminal, or vice versa) that doesn't come up in cenv's actual usage patterns. This matches fatih/color's own default behavior and is the common convention for small CLIs.

`--no-color` is a persistent flag on `rootCmd`, alongside the existing `--quiet`. `NO_COLOR` follows the [no-color.org](https://no-color.org) convention (presence of the env var disables color, regardless of value).

### Call site changes

| File:line (current) | Message | New treatment |
|---|---|---|
| `create.go` (created line) | `[cenv] Created environment %q` | `style.Success` |
| `create.go` (OAuth copy line) | `[cenv] Copied OAuth login from %s` | `style.Success` |
| `create.go` (keychain rollback) | `[cenv] Warning: failed to roll back keychain entry %q: %v` | `style.Warning` |
| `remove.go` (removed line) | `[cenv] Removed environment %q` | `style.Success` |
| `remove.go` (aborted) | `Aborted.` | `style.Secondary` |
| `trust.go` (trusted line) | `[cenv] Trusted %q in %q` | `style.Success` |
| `settings.go` (merged line) | `[cenv] Merged settings into %q` | `style.Success` |
| `login.go` (opening line) | `[cenv] Opening Claude in %q; run /login inside the REPL.` | `style.Info` |
| `run.go` (using line) | `[cenv] Using %q (%s)` | `style.Info` |
| `list.go` (AUTH column) | `yes` / `no` | `style.Success("yes")` / `style.Secondary("no")` |

Untouched: `remove`'s `y/N` prompt text (not a status message), `list`'s "No environments yet." / "Create one: ..." hint (plain informational text, not worth a symbol), and all of `path`, `settings get`/`settings show`, `list --json`.

### Error handling

`rootCmd.SilenceErrors = true` in `main.go`. `main()` changes from:

```go
if err := rootCmd.Execute(); err != nil {
    os.Exit(1)
}
```

to:

```go
if err := rootCmd.Execute(); err != nil {
    fmt.Fprintln(os.Stderr, style.Error("%v", err))
    os.Exit(1)
}
```

This is the one central touchpoint for every returned error across all commands, replacing cobra's built-in `Error: <err>` with the styled equivalent (`✗ <err>`).

### Testing

- `internal/style`: table-driven tests that force `color.NoColor` true/false and assert exact byte output (symbol, color codes present/absent, message text) for each helper.
- `cmd/cenv/*_test.go`: existing tests capture stdout/stderr already; under `go test`, `isatty` reports false so color is off by default and existing assertions on message text keep passing unchanged. No new test infrastructure needed there — this is incidental, not a design requirement, since `go test`'s captured pipes are never a TTY.
- One new test for the `--no-color` flag and `NO_COLOR` env var actually flipping `style` output, exercised at the `main.go` toggle-computation level (extracted into a small testable function rather than left inline in `main()`).

## Dependencies

Adds `github.com/fatih/color` (MIT license) and its transitive dependencies `github.com/mattn/go-isatty` and `github.com/mattn/go-colorable` (Windows console color support). `CLAUDE.md`'s "no other runtime dependencies" line reflected the state of the codebase at the time it was written, not a deliberate constraint — it will be updated to describe the actual dependency set after this change lands.
