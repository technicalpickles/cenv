---
name: run-cenv
description: Build, run, and smoke-test the cenv CLI (manages isolated Claude Code config directories, like virtualenv for Claude Code). Use when asked to run cenv, build it, test it, or verify its subcommands (create/list/path/remove/settings/trust/run/login) work.
---

cenv is a Go CLI, not a GUI/server — there's no window or port to drive.
Build the binary and exercise it directly, or run the committed smoke
script `.claude/skills/run-cenv/smoke.sh`, which drives every subcommand
end-to-end in an isolated sandbox (via `CENV_BASE`) and checks exit
codes + output.

All paths below are relative to the repo root.

## Prerequisites

Go 1.26.1 (pinned in `mise.toml`). With mise installed:

```bash
mise install       # installs go@1.26.1 into ~/.local/share/mise
```

Without mise, any Go toolchain new enough to satisfy `go.mod` (`go 1.26.1`) works — just skip the `mise exec --` prefix below.

## Build

```bash
mise exec -- go build -o cenv ./cmd/cenv
# or, without mise:
go build -o cenv ./cmd/cenv
```

Confirm it built:

```bash
./cenv --help
# → lists: create, list, path, remove, run, settings, trust, login, completion
```

## Run (agent path): smoke script

```bash
bash .claude/skills/run-cenv/smoke.sh ./cenv
# → "== N passed, 0 failed =="
```

What it does: sets `CENV_BASE` to a throwaway temp dir (never touches
`~/.local/share/cenv` or `~/.claude`), then drives:

- `list` / `list --json` on an empty base, and after creating an env
- `create --bare`, `create --from <env>`, duplicate/invalid name rejection
- `path`, including the not-found error path
- `settings merge` / `settings show` / `settings get <dotpath>`
- `trust <env> <path>` and confirms it lands in `<env>/.claude.json`
- `remove`, including double-remove rejection
- `run` / `login` pre-flight checks: missing env, and missing-auth
  rejection (it does NOT launch the nested Claude REPL — see Gotchas)

Every check is `create --bare`, so no OAuth token is ever written to the
real macOS keychain. Cleans up its temp dir on exit via `trap`.

## Direct invocation

Most of what's interesting in this repo is the subcommand logic under
`cmd/cenv/*.go`, backed by pure-ish packages in `internal/` (`env`,
`settings`, `claudeconfig`, `auth`, `keychain`, `bootstrap`). None of it
needs the built binary — `go test ./...` exercises it directly and is
the fastest feedback loop for internal changes:

```bash
mise exec -- go test ./...
mise exec -- go vet ./...
# or: mise run check   (fmt + vet + test)
```

To probe a single subcommand manually without the smoke script, isolate
your shell first so you don't touch real cenv envs:

```bash
export CENV_BASE=$(mktemp -d)
./cenv create demo --bare
./cenv settings merge demo '{"permissions":{"allow":["Bash(ls:*)"]}}'
./cenv settings show demo
rm -rf "$CENV_BASE"
```

## Run (human path)

`cenv run <name> -- <claude-args>` and `cenv login <name>` both
`syscall.Exec` into the real `claude` binary (found via `PATH`) with
`CLAUDE_CONFIG_DIR` pointed at the env dir — they replace the current
process, so they only make sense in an interactive terminal against a
real, authenticated env:

```bash
cenv create myenv                  # auto-copies OAuth from ~/.claude if logged in
cenv run myenv -- -p 'hi'          # or: cenv login myenv, then /login inside Claude
```

Not scripted here: it hands off to Claude Code's own REPL, which is a
different project's UI, not cenv's.

## Test

```bash
mise exec -- go test ./...
```

All 7 packages pass (`cmd/cenv`, `internal/{auth,bootstrap,claudeconfig,env,keychain,settings}`).

## Gotchas

- **`cenv run --help` doesn't work as you'd expect.** `run` uses
  `cobra.MinimumNArgs(1)` + `DisableFlagParsing: true` so it can pass
  `-- <claude-args>` through untouched. That means `--help` is parsed
  as the environment name, so `cenv run --help` fails with `environment
  "--help" does not exist` instead of printing usage. Use `cenv help
  run` instead.
- **`create` (without `--bare`/`--from`) writes to the real macOS
  keychain** if `~/.claude` has an OAuth login — it copies the token so
  the new env is pre-authenticated. Fine when that's genuinely what you
  want to test, but the smoke script deliberately avoids it (always
  uses `--bare`) since `cenv remove` does *not* clean up the keychain
  entry it leaves behind — only `create`'s own failure-path rollback
  does.
- **`trust` doesn't touch `settings.json`.** It writes trusted-path
  state into `<env>/.claude.json` under `projects.<path>.hasTrustDialogAccepted`,
  a separate file from `settings.json`. Don't expect `settings show` to
  reflect it.
