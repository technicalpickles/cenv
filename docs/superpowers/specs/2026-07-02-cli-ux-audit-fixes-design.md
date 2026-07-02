# cenv CLI UX audit fixes

Design for fixing six findings from a `/designing-clis` UX audit of the `cenv` CLI (see conversation history for the full audit). All six are small, independent, low-risk polish fixes bundled into one round of work.

## 1. Standardize "not found" wording

Same condition (env doesn't exist) is currently phrased four different ways across the codebase:

- `create.go:33` — `environment %q already exists` (different condition — leave as-is)
- `login.go:26`, `run.go:24` — `environment %q does not exist`
- `path.go:17`, `settings.go:25/52/89`, `trust.go:29` — `environment %q not found`
- `env.go:75` (`Remove`) — `env %q not found`

Canonical form: **`environment %q not found`**.

Changes:
- `login.go:26` — `does not exist` → `not found`
- `run.go:24` — `does not exist` → `not found`
- `env.go:75` (`Remove`) — `env %q not found` → `environment %q not found`

No changes needed in `path.go`, `settings.go`, `trust.go` — already canonical.

## 2. `--version` flag

- Add `var version = "dev"` in `main.go`.
- Set `rootCmd.Version = version` in the `rootCmd` definition (cobra wires up `--version`/`-v` automatically once `Version` is non-empty).
- Update the `mise run build` task to inject the real version via ldflags:
  ```
  go build -ldflags "-X main.version=$(git describe --tags --always --dirty)" -o cenv ./cmd/cenv
  ```
- Local dev builds outside a tag show something like `0.1.0-3-gabc1234-dirty`; release builds (once tagged by release-please) show the clean semver.
- No CI workflow changes needed beyond whatever already runs `mise run build` — `git describe` works against any checkout with tag history.

## 3. `--bare`/`--from` conflict

In `create.go`, before the settings-source `switch` (currently starting at line 56), add a validation check:

```go
if createBare && createFrom != "" {
    return fmt.Errorf("--bare and --from are mutually exclusive")
}
```

Currently `--bare --from user` silently drops `--from` with no warning. After this change it's a hard error.

## 4. Examples in help text

Add an `Example:` field to each command's cobra definition:

- `create` — e.g. `cenv create myenv`, `cenv create myenv --from user`
- `run` — e.g. `cenv run myenv`, `cenv run myenv -- --model opus`
- `login` — e.g. `cenv login myenv`
- `remove` — e.g. `cenv remove myenv`
- `path` — e.g. `cenv path myenv`
- `trust` — e.g. `cenv trust myenv ~/projects/foo`
- `settings show` / `settings get` / `settings merge` — one example each

No change to `SilenceUsage` (stays `true` on root) and no change to the existing terse arg-count error style (e.g. `Error: accepts 1 arg(s), received 0`) — that terseness is a consistent existing pattern across every command, not something this round redesigns.

## 5. `cenv remove` confirmation

`remove.go` currently deletes immediately with no confirmation, no undo, and no `--force` escape hatch.

New behavior:
- Add a `--force` bool flag to `removeCmd`.
- If `--force` is passed, skip confirmation and remove immediately (current behavior).
- Otherwise, check `isTerminal(os.Stdin)` (reusing the existing helper from `tty.go`, already used by `login.go`):
  - **TTY**: prompt `Remove environment "foo"? [y/N] ` on stderr, read a line from stdin. Proceed only on `y` or `yes` (case-insensitive); anything else (including empty/EOF) aborts with a non-error message like `Aborted.` and exit code 0 (user chose not to proceed, not a failure).
  - **Non-TTY** (scripts, CI, piped input): skip the prompt and proceed, matching how automation is expected to work elsewhere in the CLI (`login.go` already requires a TTY rather than erroring cleverly around it — this is the same "TTY changes behavior, non-TTY doesn't get stuck" pattern, just inverted since `remove` should still function unattended).

## 6. `cenv list` shows auth status

Plain `cenv list` currently prints bare names; `--json` has `HasAuth` (via `env.Inspect`) but `env.Inspect` also walks the whole env directory computing size/mtime, which is unnecessary work for this display.

Changes to `list.go`:
- For the non-JSON path, call `auth.Detect(env.Path(name))` directly per env (not `env.Inspect`) to avoid the directory walk.
- Render as an aligned table via `text/tabwriter`:
  ```
  NAME      AUTH
  default   yes
  scratch   no
  ```
- Empty-list message (`No environments yet. Create one: cenv create <name>`) is unchanged.
- `--json` output is unchanged (already has `HasAuth`).

## Out of scope

- No color/symbols added (no color library in use today; not part of this round).
- No changes to `settings`, `trust`, or `path` command behavior beyond the wording fix in #1.
- No retroactive versioning of already-released builds.

## Testing

- Existing unit tests in `internal/env`, `cmd/cenv` cover the touched functions; extend with:
  - `create_test.go`: case for `--bare --from user` returning the mutual-exclusivity error.
  - `remove_test.go` (new or extended): TTY-prompt accept/decline paths (inject a fake stdin), `--force` path, non-TTY path.
  - `list_test.go`: table output includes auth column; JSON output unchanged.
- Manual smoke test via `mise run build` + running each of the six changed commands, matching the audit's original test scenarios.
