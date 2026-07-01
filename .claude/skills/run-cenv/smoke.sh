#!/usr/bin/env bash
# Smoke-drives the cenv CLI end-to-end in an isolated sandbox.
# Exercises env lifecycle (create/list/path/remove), settings (get/merge/show),
# trust, and run/login's pre-flight error paths (missing env, missing auth).
#
# Does NOT touch ~/.claude or the real macOS keychain: CENV_BASE points at a
# throwaway directory, and every env here is created --bare so no OAuth token
# is ever written to the system keychain.
#
# Usage: .claude/skills/run-cenv/smoke.sh [path-to-cenv-binary]
set -euo pipefail

BIN="${1:-$PWD/cenv}"
if [ ! -x "$BIN" ]; then
  echo "cenv binary not found or not executable at $BIN" >&2
  echo "Build it first: go build -o cenv ./cmd/cenv" >&2
  exit 1
fi

export CENV_BASE="${TMPDIR:-/tmp}/cenv-smoke-$$"
cleanup() { rm -rf "$CENV_BASE"; }
trap cleanup EXIT
rm -rf "$CENV_BASE"

pass=0
fail=0

# check_exit CMD... -- expects the command to exit 0
check() {
  local desc="$1"; shift
  if "$@" >/tmp/cenv-smoke-out.$$ 2>&1; then
    echo "ok   - $desc"
    pass=$((pass+1))
  else
    echo "FAIL - $desc"
    sed 's/^/       /' /tmp/cenv-smoke-out.$$
    fail=$((fail+1))
  fi
  rm -f /tmp/cenv-smoke-out.$$
}

# check_fail CMD... -- expects the command to exit non-zero
check_fail() {
  local desc="$1"; shift
  if ! "$@" >/tmp/cenv-smoke-out.$$ 2>&1; then
    echo "ok   - $desc (failed as expected)"
    pass=$((pass+1))
  else
    echo "FAIL - $desc (expected non-zero exit, got 0)"
    sed 's/^/       /' /tmp/cenv-smoke-out.$$
    fail=$((fail+1))
  fi
  rm -f /tmp/cenv-smoke-out.$$
}

check_contains() {
  local desc="$1" needle="$2"; shift 2
  local out
  if out=$("$@" 2>&1) && grep -qF "$needle" <<<"$out"; then
    echo "ok   - $desc"
    pass=$((pass+1))
  else
    echo "FAIL - $desc"
    echo "       expected to contain: $needle"
    echo "       got: $out"
    fail=$((fail+1))
  fi
}

echo "== env lifecycle =="
check_contains "list on empty base" "No environments yet" "$BIN" list
check "create --bare demo" "$BIN" create demo --bare
check_fail "create existing name fails" "$BIN" create demo --bare
check_fail "create invalid name fails" "$BIN" create "bad name"
check_contains "list shows demo" "demo" "$BIN" list
check_contains "list --json shows demo" '"name": "demo"' "$BIN" list --json
check_contains "path prints env dir" "$(cd "$CENV_BASE" && pwd)/demo" "$BIN" path demo
check_fail "path on missing env fails" "$BIN" path ghost

echo "== settings =="
check "settings merge writes JSON" "$BIN" settings merge demo '{"permissions":{"allow":["Bash(ls:*)"]}}'
check_contains "settings show reflects merge" '"Bash(ls:*)"' "$BIN" settings show demo
check_contains "settings get dot-path" "Bash(ls:*)" "$BIN" settings get demo permissions.allow

echo "== trust =="
check "trust writes .claude.json" "$BIN" trust demo /tmp/some/workspace
check_contains ".claude.json has trusted path" "/tmp/some/workspace" cat "$CENV_BASE/demo/.claude.json"

echo "== create --from =="
check "create --from clones settings" "$BIN" create demo-clone --from demo
check_contains "clone has source settings" '"Bash(ls:*)"' "$BIN" settings show demo-clone
check "remove demo-clone" "$BIN" remove demo-clone
check_fail "remove already-removed env fails" "$BIN" remove demo-clone

echo "== run / login pre-flight (no nested Claude REPL launch) =="
check_fail "run on missing env fails" "$BIN" run ghost -- -p hi
check_fail "run on unauthenticated env fails" "$BIN" run demo -- -p hi
check_fail "login on missing env fails" "$BIN" login ghost

echo
echo "== $pass passed, $fail failed =="
[ "$fail" -eq 0 ]
