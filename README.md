# cenv

Manage isolated Claude Code configuration directories. Each env gets its own `settings.json`, plugins, hooks, and session history, independent of `~/.claude/`. Think `virtualenv` for Claude Code.

See `projects/cenv/2026-04-15-cenv-design.md` in the pickleton repo for the full design.

## Install

```
go install github.com/technicalpickles/cenv@latest
```

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
