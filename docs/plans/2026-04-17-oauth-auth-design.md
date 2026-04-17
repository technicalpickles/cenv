# OAuth Auth Design for cenv

Date: 2026-04-17
Related beans: gt-wl86 (this work), gt-5zn7 (prerequisite bug), gt-8tv9 (downstream blocker)

## Problem

Anthropic OAuth users can't authenticate new cenv environments. OAuth state lives in two places cenv doesn't handle:

1. `~/.claude.json` (home root, not `~/.claude/.claude.json`) contains `oauthAccount`, `userID`, `claudeCodeFirstTokenDate`.
2. macOS Keychain entries named `Claude Code-credentials-<hash>`, where `<hash>` is derived from `CLAUDE_CONFIG_DIR`. Default `~/.claude/` uses the unhashed `Claude Code-credentials`. New envs produce different hashes.

Because Keychain entries are keyed by a hash of `CLAUDE_CONFIG_DIR`, **a logged-in keychain entry cannot be carried between envs**. Each env needs its own login.

A separate bug (gt-5zn7) means `oauthAccount` is a JSON object, not a string, so current detection silently fails for OAuth users.

## Goal

Make it possible for OAuth users to authenticate cenv envs via a deliberate, interactive step. Accept that each env requires one `claude /login` invocation on first use. Do not attempt to reverse-engineer the keychain hash scheme or carry tokens between envs.

## Non-goals

- Reverse-engineering Claude Code's Keychain naming scheme.
- Copying OAuth tokens between envs.
- Fully automatic auth during `cenv create` or `cenv run`.
- Changing Bedrock auth behavior.

## Behavior

### Caller model

Two distinct flows based on how cenv is invoked:

| Caller | Expectation |
|---|---|
| Agent / script (non-TTY) | Never prompts. If no auth is configured, fail fast with a clear message pointing at the fix. |
| Interactive user | `cenv login <env>` is the deliberate escape hatch. Nothing auto-launches during `create` or `run`. |

### Detection capabilities

What we can reliably detect:
- Bedrock auth configured: `<env>/settings.json` has `awsAuthRefresh` object.
- OAuth auth configured: `<env>/.claude.json` has non-empty `oauthAccount` (object or string).
- No auth configured: neither of the above.

What we can't detect (one-sided):
- OAuth was logged in but the keychain token was deleted or expired. Without the keychain hash, we can't probe `Claude Code-credentials-<hash>` directly. "Has `oauthAccount`" is therefore a proxy for "has ever been logged in," not a guarantee.

### Command changes

**`cenv create <name>` (default auto-detect path)**
- Detect OAuth in `~/.claude.json` via the fixed type check (gt-5zn7).
- When OAuth is detected, replace the current "see gt-wl86" warning with a concrete next-step: `Run 'cenv login <name>' to authenticate this env.`
- Settings copy behavior stays the same.

**`cenv create <name> --auth auth-anthropic`**
- Continues to copy settings.json from the named auth env. No change in behavior. OAuth users will find this path unhelpful because tokens don't transfer, but Bedrock users rely on it.

**`cenv auth create` (when OAuth-only)**
- Refuse with a clear message: `OAuth users don't need auth envs. Run 'cenv create <name>' then 'cenv login <name>' instead.`
- If Bedrock is also configured, continue to create `auth-aws-bedrock` normally.

**`cenv login <name>` (new command)**
- Wraps `CLAUDE_CONFIG_DIR=<env-path> claude`, dropping the user into the Claude Code REPL so they can type `/login`.
- Requires a TTY on stdin. If `isatty(stdin) == false`, error: `cenv login requires an interactive terminal`.
- Works for any env, not just OAuth. Useful for re-login scenarios.

**`cenv run <name> -- ...`**
- Pre-flight: call `auth.Detect(<env>)`.
- If it returns `no auth found`, error: `Env '<name>' has no auth configured. Run 'cenv login <name>' first.`
- Otherwise proceed. Claude Code's own "Please run /login" handles the "was authed, token missing" case.

**`cenv list --json`**
- `has_auth` uses the fixed `auth.Detect` logic. OAuth envs report `has_auth: true` once `oauthAccount` is present in `<env>/.claude.json`.

### Auth detection fix (prerequisite: gt-5zn7)

`internal/auth/auth.go:50-57` currently does `val.(string)`. Replace with:

```go
if val, ok := claudeData["oauthAccount"]; ok {
    switch v := val.(type) {
    case string:
        if v != "" {
            return &DetectResult{Type: "anthropic", EnvName: "auth-anthropic", Detail: v}, nil
        }
    case map[string]any:
        email, _ := v["emailAddress"].(string)
        return &DetectResult{Type: "anthropic", EnvName: "auth-anthropic", Detail: email}, nil
    }
}
```

Matching fix in `cmd/cenv/create.go:130-141` (`hasOAuth`): same object-aware check.

## Components

| File | Change |
|---|---|
| `internal/auth/auth.go` | Fix `oauthAccount` type handling (gt-5zn7). |
| `internal/auth/auth_test.go` | Add test cases for object-shaped `oauthAccount`. |
| `cmd/cenv/create.go` | Fix `hasOAuth` type handling. Replace "see gt-wl86" message with `cenv login` next-step. |
| `cmd/cenv/login.go` (new) | New `cenv login <name>` command. TTY check. Exec `claude` with `CLAUDE_CONFIG_DIR` set. |
| `cmd/cenv/run.go` | Pre-flight `auth.Detect` call. Hard error when no auth configured. |
| `cmd/cenv/auth.go` | `cenv auth create` refuses OAuth type explicitly. |
| `README.md` | Document the OAuth flow: `cenv create` then `cenv login`. |

## Testing

- Unit tests for `auth.Detect` covering object-shaped `oauthAccount`, string-shaped, empty, missing.
- Unit tests for `hasOAuth` same matrix.
- Integration-style test for `cenv run` pre-flight: fresh env with no settings should produce the `cenv login` error.
- Manual test matrix:
  1. OAuth user creates env, gets `cenv login` hint.
  2. `cenv login <env>` opens REPL; after `/login`, keychain entry exists, `cenv list --json` shows `has_auth: true`.
  3. `cenv run <env>` on authed env works.
  4. `cenv run <env>` on fresh env fails with pre-flight error.
  5. `cenv auth create` on OAuth-only system refuses with clear message.

## Out of scope / follow-ups

- Keychain hash reverse-engineering (would unblock fully automatic OAuth). Leave as open possibility, don't pursue now.
- `claude /login` invocation that bypasses the REPL (would need CLI flag support in Claude Code itself).
- Detection of stale/deleted keychain entries.
