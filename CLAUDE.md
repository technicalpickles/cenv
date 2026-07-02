# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`cenv` manages isolated Claude Code configuration directories — each env gets its own `settings.json`, `.claude.json`, plugins, hooks, and session history, independent of `~/.claude/`. Think `virtualenv` for Claude Code. A Go CLI built with cobra, with `fatih/color` for terminal output styling.

## Commands

```sh
mise install       # install the pinned Go toolchain (go 1.26.1, see mise.toml)
mise run build      # go build -o cenv ./cmd/cenv
mise run install    # go install ./cmd/cenv (drops binary in $GOBIN or $GOPATH/bin — make sure that's on PATH)
mise run test       # go test ./...
mise run vet        # go vet ./...
mise run fmt        # go fmt ./...
mise run check      # fmt + vet + test (what CI runs)
```

Without mise, the raw `go build`/`go test`/`go vet` commands work the same way.

Run a single test:

```sh
go test ./internal/settings/... -run TestDeepMerge
go test ./cmd/cenv/... -run TestRun_MissingEnv -v
```

Keychain tests (`cmd/cenv/create_keychain_test.go`) hit the real macOS keychain and are gated behind a build tag, excluded from `go test ./...` and from CI:

```sh
go test -tags keychain ./cmd/cenv/...
```

CI (`.github/workflows/ci.yml`) runs `mise run check` on `macos-latest` for every push to `main` and every PR. **Branch protection on `main` requires a passing CI run and a PR — this is enforced for admins too, so direct `git push` to `main` will be rejected.** Always work on a branch and open a PR.

## Releases

`release-please` (`.github/workflows/release-please.yml`) watches pushes to `main`, maintains a release PR that bumps the version and generates `CHANGELOG.md` from conventional commits, and cuts the git tag + GitHub Release when that PR merges. The published-release event then fires `.github/workflows/release.yml`, which cross-compiles a darwin arm64+amd64 binary, `lipo`s them into a universal binary, and attaches it to the release.

**Gotcha:** this cross-workflow trigger only fires if the release was created with a PAT/GitHub App token in the `RELEASE_PLEASE_TOKEN` secret — the default `GITHUB_TOKEN` is blocked by GitHub from triggering other workflows, so without that secret release PRs still open but merging them silently skips the binary build.

## Architecture

The CLI layer (`cmd/cenv/`) is thin glue: each subcommand file (`create.go`, `run.go`, `login.go`, etc.) registers a cobra command in its own `init()` and calls into `internal/` packages, which hold all the actual logic and are independently unit-tested. `main.go` just owns the root command and a shared `logf` helper (writes to stderr, silenced by `--quiet`/`CENV_QUIET`).

Packages, in the order data flows through `cenv create`:

- **`internal/env`** — env directory CRUD: `BasePath()` (`$CENV_BASE` or `~/.local/share/cenv`), `Path`, `Exists`, `List`, `Remove`, `Inspect` (size/mtime/auth-status metadata), `ValidateName`.
- **`internal/settings`** — generic JSON file operations used for `settings.json`: `Load`/`Save`, `DeepMerge` (objects merge recursively, scalars/arrays from the overlay win), `GetByDotPath` (dot-path traversal for `cenv settings get`), `IsJSON`/`ResolveOverlay` (decide whether a CLI arg is inline JSON or a file path).
- **`internal/bootstrap`** — writes the two files a fresh env needs: `.claude.json` with onboarding pre-completed (`WriteOnboarding`) and `settings.json` (`WriteSettings`). `ExtractAuth` pulls just the auth-relevant keys (`env`, `awsAuthRefresh`, `statusLine`) out of a settings map, used when auto-detecting from `~/.claude` (as opposed to `--from`, which clones everything).
- **`internal/claudeconfig`** — reads/writes specific fields of `.claude.json` without disturbing the rest: `ReadOAuth`/`MergeOAuth` for the OAuth account blob, `MergeTrust` for workspace-trust entries (`projects.<path>.hasTrustDialogAccepted`).
- **`internal/keychain`** — wraps the macOS `security` CLI to store/read/delete OAuth tokens, mirroring claude-code's own keychain scheme: service name is `"Claude Code-credentials"` for `~/.claude`, or `"Claude Code-credentials-<8 hex chars of sha256(configDir)>"` for any other config dir. Shells out via a `Runner` interface so tests can stub `security` without touching the real keychain.
- **`internal/auth`** — `Detect(configDir)` is the auth predicate used both for `env.Info.HasAuth` and as a preflight check in `cenv run`: nil error means either `settings.json` has a non-empty `awsAuthRefresh` (Bedrock) or `.claude.json` has a non-empty `oauthAccount` (Anthropic OAuth).

### The OAuth-copy path (`cmd/cenv/create.go: copyAuth`)

This is the trickiest bit of plumbing in the codebase, worth understanding before touching `create`. Copying an authenticated login from a source (either `~/.claude` or another cenv env) to a new env means moving state from *two* different places in lockstep:

1. The keychain token (via `internal/keychain`, keyed by a service name derived from the source's config dir).
2. The `oauthAccount` field in the source's `.claude.json` (via `internal/claudeconfig.ReadOAuth`/`MergeOAuth`).

The `.claude.json` path is asymmetric depending on the source: `~/.claude.json` lives at `$HOME` (outside `~/.claude/`), but for cenv env dirs it's `<envdir>/.claude.json` (inside the dir, since cenv sets `CLAUDE_CONFIG_DIR` for its envs). `copyAuth` writes the keychain entry before the config file, and rolls the keychain entry back if the config write fails. If the source is missing either half (keychain token or `oauthAccount`), that's read as "not fully authed" (common for Bedrock-only or fresh users) and `copyAuth` no-ops rather than writing partial state.

### Process replacement, not subprocess

`cenv run` and `cenv login` both use `syscall.Exec` (not `os/exec` + wait) to replace the current process with `claude`, after injecting `CLAUDE_CONFIG_DIR=<envdir>` into the environment. This means signals, TTY, and exit codes pass straight through to the real `claude` process — there's no cenv process left to relay them.
