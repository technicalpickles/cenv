# cenv

Manage isolated Claude Code configuration directories. Each env gets its own `settings.json`, plugins, hooks, and session history, independent of `~/.claude/`. Think `virtualenv` for Claude Code.

See `projects/cenv/2026-04-15-cenv-design.md` in the pickleton repo for the full design.

## Install

```
go install github.com/technicalpickles/cenv@latest
```

## Anthropic OAuth users

OAuth login tokens are stored per-`CLAUDE_CONFIG_DIR` in the macOS Keychain, so they don't transfer between cenv envs. Each env needs its own login:

```sh
cenv create my-env           # creates the env; prints a hint
cenv login my-env            # opens Claude; type /login inside the REPL
cenv run my-env -- -p 'hi'   # env is now authenticated
```

`cenv login` requires a terminal. For scripts and agents, `cenv run` fails fast with a message pointing at `cenv login` if the target env has never been authenticated.

`cenv auth create` is not available for OAuth users. The auth env pattern only carries tokens for Bedrock.

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
