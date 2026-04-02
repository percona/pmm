#!/usr/bin/env bash
# Sync the do/headless-main branch with a new upstream PMM release.
#
# Usage:
#   ./scripts/sync-upstream.sh                 # merge latest upstream tag
#   ./scripts/sync-upstream.sh v3.7.0          # merge a specific tag
#
# Prerequisites:
#   git remote add upstream https://github.com/percona/pmm.git
#
# This script:
#   1. Fetches upstream tags
#   2. Determines the target tag (latest or specified)
#   3. Creates a sync branch
#   4. Attempts the merge
#   5. Reports conflicts if any
#
# After resolving conflicts, commit and open a PR against do/headless-main.

set -euo pipefail

UPSTREAM_REMOTE="upstream"
TARGET_BRANCH="do/headless-main"

# Files we always own — use "ours" strategy on conflicts
KNOWN_OURS=(
    "managed/services/grafana/client.go"
    "managed/services/grafana/auth_server.go"
)

log()  { echo "[sync] $*"; }
warn() { echo "[sync] WARNING: $*" >&2; }
die()  { echo "[sync] ERROR: $*" >&2; exit 1; }

# Ensure upstream remote exists
if ! git remote get-url "${UPSTREAM_REMOTE}" >/dev/null 2>&1; then
    die "Remote '${UPSTREAM_REMOTE}' not found. Run: git remote add ${UPSTREAM_REMOTE} https://github.com/percona/pmm.git"
fi

log "Fetching upstream..."
git fetch "${UPSTREAM_REMOTE}" --tags

# Determine target tag
if [ -n "${1:-}" ]; then
    TARGET_TAG="$1"
    if ! git rev-parse "${TARGET_TAG}" >/dev/null 2>&1; then
        die "Tag ${TARGET_TAG} not found. Available tags:"
        git tag -l 'v3.*' --sort=-v:refname | head -10
    fi
else
    TARGET_TAG=$(git tag -l 'v3.*' --sort=-v:refname | head -1)
    if [ -z "${TARGET_TAG}" ]; then
        die "No v3.* tags found. Fetch upstream first."
    fi
    log "Latest upstream tag: ${TARGET_TAG}"
fi

# Read current base version
CURRENT_BASE=""
if [ -f UPSTREAM_VERSION ]; then
    CURRENT_BASE=$(cat UPSTREAM_VERSION)
fi

if [ "${CURRENT_BASE}" = "${TARGET_TAG}" ]; then
    log "Already at ${TARGET_TAG}. Nothing to do."
    exit 0
fi

log "Current base: ${CURRENT_BASE:-unknown}"
log "Target:       ${TARGET_TAG}"

# Ensure we're on the target branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "${CURRENT_BRANCH}" != "${TARGET_BRANCH}" ]; then
    die "Not on ${TARGET_BRANCH} (currently on ${CURRENT_BRANCH}). Switch first."
fi

SYNC_BRANCH="sync/upstream-${TARGET_TAG}"
log "Creating branch ${SYNC_BRANCH}..."
git checkout -b "${SYNC_BRANCH}"

log "Merging ${TARGET_TAG}..."
if git merge "${TARGET_TAG}" --no-edit; then
    log "Merge succeeded cleanly."
    echo "${TARGET_TAG}" > UPSTREAM_VERSION
    git add UPSTREAM_VERSION
    git commit --amend --no-edit
    log ""
    log "Next steps:"
    log "  1. Review changes:  git diff ${TARGET_BRANCH}..${SYNC_BRANCH}"
    log "  2. Push:            git push -u origin ${SYNC_BRANCH}"
    log "  3. Open PR:         gh pr create --base ${TARGET_BRANCH} --title 'Sync upstream ${TARGET_TAG}'"
else
    warn "Merge has conflicts."
    echo ""
    log "Conflicted files:"
    git diff --name-only --diff-filter=U
    echo ""
    log "Known files where we always take 'ours':"
    for f in "${KNOWN_OURS[@]}"; do
        if git diff --name-only --diff-filter=U | grep -q "${f}"; then
            log "  Resolving ${f} with ours..."
            git checkout --ours "${f}"
            git add "${f}"
        fi
    done
    echo ""
    REMAINING=$(git diff --name-only --diff-filter=U)
    if [ -z "${REMAINING}" ]; then
        log "All conflicts resolved. Completing merge..."
        echo "${TARGET_TAG}" > UPSTREAM_VERSION
        git add UPSTREAM_VERSION
        git commit --no-edit
        log "Merge complete. Push and open a PR."
    else
        log "Remaining conflicts to resolve manually:"
        echo "${REMAINING}"
        log ""
        log "After resolving:"
        log "  1. git add <files>"
        log "  2. echo '${TARGET_TAG}' > UPSTREAM_VERSION && git add UPSTREAM_VERSION"
        log "  3. git commit"
        log "  4. git push -u origin ${SYNC_BRANCH}"
        log "  5. gh pr create --base ${TARGET_BRANCH}"
    fi
fi
