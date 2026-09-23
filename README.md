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
cenv create my-env              # copies OAuth (keychain + oauthAccount) from ~/.claude
cenv claude my-env -- -p 'hi'   # env is authenticated
```

Cloning from another cenv env works the same way:

```sh
cenv create my-clone --from my-env   # my-clone is also authenticated
```

If you want to authenticate fresh (different account, or source has no OAuth), `cenv login <env>` drops you into Claude's REPL for `/login`. `cenv login` requires a terminal.

For scripts and agents, `cenv claude` fails fast with a message pointing at `cenv login` if the target env has never been authenticated. (`cenv run` still works as a deprecated alias for `cenv claude`.)

To run something other than `claude` itself under an env's config — e.g. a different tool built on the Claude Agent SDK — use `cenv exec`:

```sh
cenv exec my-env -- a2acode serve
```

`cenv exec` runs the same auth pre-flight as `cenv claude`, then execs whatever command you give it with `CLAUDE_CONFIG_DIR` pointed at the env.

### Checking that an env's login still works

Each env's OAuth token ages on its own, so an env you haven't touched in a while can quietly expire. `cenv auth status` runs `claude auth status` in the env and exits nonzero if nothing is stored:

```sh
cenv auth status my-env          # JSON, straight from claude
cenv auth status my-env --text   # human-readable
cenv auth status my-env --live   # also make one real request
```

`claude auth status` only reports what's stored. It never checks expiry or hits the network, so an expired login still says `"loggedIn": true`. `--live` makes one minimal real request (Haiku, no tools, `--safe-mode`, no session saved; about $0.002) and fails with a `cenv login` hint if it's rejected. Use it as a pre-flight before launching an env unattended.

### Keeping envs warm

`cenv auth refresh` runs that same live request across envs, so Claude Code gets a chance to refresh each token before it goes stale:

```sh
cenv auth refresh my-env other-env
cenv auth refresh --all
```

Claude Code refreshes during a request once the access token is expired or within 5 minutes of expiring, so a daily run is enough to refresh every env you're not actively using. Envs with nothing stored are skipped. If any env's login is already dead, it keeps going through the rest, then exits nonzero with the `cenv login` commands to fix them.

To run it daily with launchd, save this as `~/Library/LaunchAgents/com.github.technicalpickles.cenv-refresh.plist` (adjust the paths; launchd doesn't use your shell's `PATH`, so both `cenv` and `claude` need to be findable from the one set here):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.github.technicalpickles.cenv-refresh</string>
  <key>ProgramArguments</key>
  <array>
    <string>/Users/you/go/bin/cenv</string>
    <string>auth</string>
    <string>refresh</string>
    <string>--all</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>PATH</key>
    <string>/Users/you/.local/bin:/usr/bin:/bin</string>
  </dict>
  <key>StartCalendarInterval</key>
  <dict>
    <key>Hour</key>
    <integer>9</integer>
  </dict>
  <key>StandardOutPath</key>
  <string>/Users/you/Library/Logs/cenv-refresh.log</string>
  <key>StandardErrorPath</key>
  <string>/Users/you/Library/Logs/cenv-refresh.log</string>
</dict>
</plist>
```

Then load it with `launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.github.technicalpickles.cenv-refresh.plist`. The log shows which envs need a `cenv login`.

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
