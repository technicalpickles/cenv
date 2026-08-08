# cenv claude and exec commands

**Date:** 2026-08-08
**Status:** Approved

## Background

`cenv run <name> [-- claude-args...]` (`cmd/cenv/run.go`) launches the `claude` CLI with `CLAUDE_CONFIG_DIR` pointed at a cenv environment, via `syscall.Exec` after an auth preflight (`internal/auth.Detect`). It's hardcoded to the `claude` binary — there's no way to run a different program (e.g. `a2acode`, or anything else built on the Claude Agent SDK) under a cenv-managed config dir without manually shelling out `CLAUDE_CONFIG_DIR=$(cenv path myenv) some-command`.

Two problems with `run` as the home for this:

1. Its name is opinionated toward launching Claude specifically. Generalizing it to accept an arbitrary command has no clean way to keep today's `-- claude-args...` convention (args appended to an implicit `claude`) working alongside a "here's a totally different program" case — the two conventions collide on what the first token after `--` means.
2. `cenv login` (`cmd/cenv/login.go`) already hardcodes launching `claude` for a second, narrower purpose (open the REPL for `/login`, no auth preflight). A `run`/`exec` split that's really just two different default targets for the same mechanism doesn't read as clearly as naming the opinionated command after what it launches.

Renaming `run` to `claude` removes that ambiguity: `cenv claude` obviously launches Claude; `cenv exec` obviously doesn't assume anything about what it launches. This also lines up with `login`, which is unambiguously claude-specific already.

## Goals

- Add `cenv exec <name> -- <command> [args...]`: run an arbitrary command with `CLAUDE_CONFIG_DIR` set to the named environment, no assumptions about what the command is.
- Rename `cenv run` to `cenv claude`, keeping identical behavior (same flags, same preflight, same exec mechanism).
- Keep `cenv run` working as a deprecated alias of `cenv claude` — same behavior, but marked deprecated so users are nudged toward the new name without anything breaking.

## Non-goals

- Changing `cenv login`'s behavior or name — it stays as-is, a distinct claude-specific flow with no auth preflight (by design, since its whole purpose is to establish auth).
- Subprocess mode (spawn-and-wait instead of `syscall.Exec`) for either command — both replace the current process, matching `run`'s and `login`'s existing rationale (signals/TTY/exit codes pass straight through).
- Making the auth preflight conditional on what command is being run. `cenv exec` always runs the same `auth.Detect` check `cenv claude` does — the point of a cenv environment is a working, authed Claude setup, and that guarantee shouldn't depend on which program you're launching under it.

## Design

### `cenv claude` (renamed from `run`)

```
cenv claude <name> [-- claude-args...]
```

Identical to today's `run.go`: validate env exists, preflight-load `settings.json`, auth preflight via `auth.Detect(envDir)`, resolve `claude` via `exec.LookPath`, set `CLAUDE_CONFIG_DIR=<envdir>` in the environment, `syscall.Exec` into it. Only the command name and help text change.

### `cenv run` (deprecated alias)

A second `cobra.Command` registered alongside `claudeCmd`, sharing the same `RunE` function (factor the current `runCmd.RunE` body out into a standalone function both commands call), with cobra's built-in `Deprecated` field set:

```go
runCmd.Deprecated = "use 'cenv claude' instead"
```

Cobra prints that message to stderr and still executes the command normally — no behavior change, just a nudge. Cobra also drops deprecated commands out of the default `--help` command listing.

### `cenv exec` (new)

```
cenv exec <name> -- <command> [args...]
```

New file `cmd/cenv/exec.go`. `DisableFlagParsing: true` (matching `run`/`claude`), since everything after `<name>` is opaque to cenv. `--` is required; the first token after it is the program to run, the rest are its args — this is a full command line, not an implicit-`claude` arg list, so there's no ambiguity about what's being launched.

Flow, matching `claude`'s:

1. Validate env exists (`env.Exists`) → `"environment %q not found"`.
2. Preflight-load `settings.json` (`settings.Load`) → surfaces malformed settings before exec.
3. Auth preflight via `auth.Detect(envDir)` → `"env %q has no auth configured; run 'cenv login %s' first"`.
4. Require `--` and at least one token after it → new error, e.g. `"missing command (use: cenv exec <name> -- <command> [args...])"`.
5. Resolve the command via `exec.LookPath(command[0])` → `"%s not found in PATH"`.
6. Build `environ` = `os.Environ()` + `CLAUDE_CONFIG_DIR=<envdir>`.
7. `syscall.Exec` into the resolved binary with the given args.

### Shared preflight logic

Steps 1-3 (env-exists, settings preflight, auth preflight) are identical across `claude`/`run` and `exec`. Factor them into a small shared helper (e.g. `preflightEnv(name string) (envDir string, err error)` in `run.go` or a new small file) that both commands' `RunE` call, rather than duplicating the three checks in `exec.go`. The exact extraction point is left to the implementation plan.

### Testing

Mirror `run_test.go`'s existing pattern — these are error-path-only tests, since `syscall.Exec` replaces the test process before anything else could be asserted:

- Retarget `run_test.go`'s two existing tests (`TestRunCmd_NonexistentEnv`, `TestRunCmd_NoAuth`) at `claudeCmd`, since that's now the primary command under test.
- Add a light smoke test confirming `cenv run` still works and is marked deprecated (e.g. asserting `runCmd.Deprecated != ""` and that `runCmd.RunE` behaves the same as `claudeCmd.RunE` on at least one error path).
- New `exec_test.go` covering `cenv exec`'s error paths: nonexistent env, no auth configured, missing `--`, missing command after `--`.

## Dependencies

None — reuses `internal/env`, `internal/settings`, `internal/auth`, `internal/style`, `os/exec`, and `syscall`, all already used by `run.go`/`login.go`.
