#!/bin/bash
# SessionStart bootstrap: idempotent pnpm install + background Storybook on :6006.
set -euo pipefail

cd "${CLAUDE_PROJECT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

STORYBOOK_PORT=6006
STORYBOOK_LOG="${TMPDIR:-/tmp}/percona-ui-storybook.log"
# Stored inside node_modules so it is wiped together with it and never committed.
LOCK_HASH_FILE="node_modules/.claude-lockfile-hash"

lockfile_hash() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum pnpm-lock.yaml | awk '{print $1}'
  else
    shasum -a 256 pnpm-lock.yaml | awk '{print $1}'
  fi
}

port_in_use() {
  (exec 3<>"/dev/tcp/127.0.0.1/${STORYBOOK_PORT}") 2>/dev/null
}

current_hash="$(lockfile_hash)"
if [ -d node_modules ] && [ -f "$LOCK_HASH_FILE" ] && [ "$(cat "$LOCK_HASH_FILE")" = "$current_hash" ]; then
  echo "session-start: dependencies up to date (lockfile unchanged), skipping pnpm install"
else
  echo "session-start: running pnpm install..."
  # pnpm 10 prints an "ignored build scripts" warning for esbuild — known and non-blocking.
  pnpm install
  echo "$current_hash" > "$LOCK_HASH_FILE"
fi

if port_in_use; then
  echo "session-start: port ${STORYBOOK_PORT} already in use, not starting Storybook"
else
  echo "session-start: starting Storybook in background (log: ${STORYBOOK_LOG})"
  nohup pnpm storybook:dev > "$STORYBOOK_LOG" 2>&1 < /dev/null &
  echo "session-start: Storybook launching (pid $!) at http://localhost:${STORYBOOK_PORT}"
fi
