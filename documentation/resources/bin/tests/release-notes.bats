#!/usr/bin/env bats
# Tests for release-notes.sh. Run: bats documentation/resources/bin/tests

setup() {
  bin="$BATS_TEST_DIRNAME/.."
  repo="$BATS_TEST_TMPDIR/repo"
  doc="$repo/documentation"
  pages="$doc/docs/release-notes"
  mkdir -p "$pages" "$doc/resources"
  cp "$bin/../release-notes-template.md" "$doc/resources/"
  cp "$bin/../release-notes-template.md" "$pages/next.md"
  cp "$bin/../../docs/release-notes/index.md" "$pages/"
  cp "$bin/../../mkdocs-base.yml" "$doc/"
  export DOC_DIR="$doc" GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@t GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@t
  rn="$bin/release-notes.sh"
  git -C "$repo" init -q
}

# entry FILE TICKET adds a fixed-issue entry to FILE.
entry() {
  sed -i "/^## ✅ Fixed issues$/a - [$2](https://perconadev.atlassian.net/browse/$2): Fixed it." "$pages/$1"
}

# commit REF commits everything and points REF at it.
commit() {
  git -C "$repo" add -A
  git -C "$repo" commit -qm "$1" --allow-empty
  git -C "$repo" update-ref "refs/$1" HEAD
}

@test "rc: moves next.md to the version, lists it in the index and nav, and empties next.md" {
  entry next.md PMM-1
  run "$rn" rc 3.11.0
  [ "$status" -eq 0 ]
  grep -q '^# Percona Monitoring and Management 3.11.0$' "$pages/3.11.0.md"
  grep -q 'browse/PMM-1)' "$pages/3.11.0.md"
  cmp -s "$pages/next.md" "$doc/resources/release-notes-template.md"
  grep -qF -- '- [Percona Monitoring and Management 3.11.0](3.11.0.md)' "$pages/index.md"
  grep -qF -- '- "PMM 3.11.0": release-notes/3.11.0.md' "$doc/mkdocs-base.yml"
}

@test "rc: a rerun leaves the version and next.md alone" {
  "$rn" rc 3.11.0 > /dev/null
  entry next.md PMM-2
  run "$rn" rc 3.11.0
  [ "$status" -eq 0 ]
  grep -q 'browse/PMM-2)' "$pages/next.md"
  ! grep -q 'browse/PMM-2)' "$pages/3.11.0.md"
  [ "$(grep -c '"PMM 3.11.0"' "$doc/mkdocs-base.yml")" -eq 1 ]
}

@test "publish: passes when the version matches main" {
  "$rn" rc 3.11.0 > /dev/null
  commit remotes/origin/main
  run "$rn" publish 3.11.0
  [ "$status" -eq 0 ]
}

@test "publish: fails when the version differs from main" {
  "$rn" rc 3.11.0 > /dev/null
  commit heads/release
  entry 3.11.0.md PMM-3
  commit remotes/origin/main
  git -C "$repo" checkout -q release
  run "$rn" publish 3.11.0
  [ "$status" -eq 1 ]
  [[ $output == *"+- [PMM-3]"* || $output == *"-- [PMM-3]"* ]]
  [[ $output == *"3.11.0.md differs between main and pmm-3.11.0"* ]]
}

@test "publish: fails when the version is missing" {
  commit remotes/origin/main
  run "$rn" publish 3.11.0
  [ "$status" -eq 1 ]
  [[ $output == *"3.11.0.md is missing on pmm-3.11.0"* ]]
}
