# cenv

Manage isolated Claude Code configuration directories. Each env gets its own `settings.json`, plugins, hooks, and session history, independent of `~/.claude/`. Think `virtualenv` for Claude Code.

See `projects/cenv/2026-04-15-cenv-design.md` in the pickleton repo for the full design.

## Install

```
go install github.com/technicalpickles/cenv/cmd/cenv@latest
```

This drops the `cenv` binary in `$GOBIN` (or `$GOPATH/bin`, or `~/go/bin` if neither is set) — make sure that directory is on your `PATH`. Once a tagged release exists, `@latest` resolves to it directly; until then it builds from the latest commit on `main`.

A prebuilt macOS binary is also attached to each [GitHub Release](https://github.com/technicalpickles/cenv/releases), if you'd rather skip the Go toolchain entirely.

## Anthropic OAuth users

`cenv create` auto-copies your OAuth login from `~/.claude` into the new env, so new envs are already authenticated:

```sh
cenv create my-env           # copies OAuth (keychain + oauthAccount) from ~/.claude
cenv run my-env -- -p 'hi'   # env is authenticated
```

Cloning from another cenv env works the same way:

```sh
cenv create my-clone --from my-env   # my-clone is also authenticated
```

If you want to authenticate fresh (different account, or source has no OAuth), `cenv login <env>` drops you into Claude's REPL for `/login`. `cenv login` requires a terminal.

For scripts and agents, `cenv run` fails fast with a message pointing at `cenv login` if the target env has never been authenticated.

## Running under Claude Code's sandbox

Claude Code's sandbox blocks writes outside an allowlist. cenv stores envs at `~/.local/share/cenv/` (or `$CENV_BASE`), so fresh installs hit `operation not permitted` on first `cenv create`.

Add the env base to your sandbox `allowWrite` list in `.claude/settings.json`:

```json
{
  "sandbox": {
    "filesystem": {
      "allowWrite": [
        "~/.local/share/cenv"
      ]
    }
  }
}
```

If you set `CENV_BASE`, add that path instead.

## Development

Common tasks are defined in `mise.toml`. With [mise](https://mise.jdx.dev/) installed:

```sh
mise install       # install the Go toolchain pinned in mise.toml
mise run build     # build ./cenv
mise run install   # go install ./cmd/cenv
mise run test      # go test ./...
mise run check     # fmt + vet + test
```

`mise run install` (and the raw `go install ./cmd/cenv` below) puts the binary in `$GOBIN`/`$GOPATH/bin`/`~/go/bin` like the GitHub install above — make sure that directory is on your `PATH`, or `cenv` won't be found afterward.

Without mise, the raw commands work too:

```sh
go build -o cenv ./cmd/cenv
go install ./cmd/cenv
go test ./...
```

### Layout

- `cmd/cenv/` — CLI entry point and subcommands
- `internal/` — supporting packages
- `docs/plans/` — design docs
- `scratch/` — throwaway exploration (gitignored where appropriate)
