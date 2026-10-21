#!/usr/bin/env bats
# Tests for release-notes.sh. Run: bats documentation/resources/bin/tests

setup() {
  bin="$BATS_TEST_DIRNAME/.."
  repo="$BATS_TEST_TMPDIR/repo"
  doc="$repo/documentation"
  mkdir -p "$doc/docs/release-notes" "$doc/resources"
  cp "$bin/../release-notes-template.md" "$doc/resources/"
  cp "$bin/../../docs/release-notes/index.md" "$doc/docs/release-notes/"
  cp "$bin/../../mkdocs-base.yml" "$doc/"
  export DOC_DIR="$doc" GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@t GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@t
  rn="$bin/release-notes.sh"
  page="$doc/docs/release-notes/3.11.0.md"
  git -C "$repo" init -q
  "$rn" new 3.11.0 > /dev/null
}

# entry TICKET adds a fixed-issue entry to 3.11.0.md.
entry() {
  sed -i "/^## ✅ Fixed issues$/a - [$1](https://perconadev.atlassian.net/browse/$1): Fixed it." "$page"
}

# at REF commits the page and points REF (remotes/origin/pmm-3.11.0 or tags/v3.11.0) at it.
at() {
  git -C "$repo" add -A
  git -C "$repo" commit -qm "$1" --allow-empty
  git -C "$repo" update-ref "refs/$1" HEAD
}

@test "new: creates the page from the template and lists it in the index and nav" {
  grep -q '^# Percona Monitoring and Management 3.11.0$' "$page"
  grep -qF -- '- [Percona Monitoring and Management 3.11.0](3.11.0.md)' "$doc/docs/release-notes/index.md"
  grep -qF -- '- "PMM 3.11.0": release-notes/3.11.0.md' "$doc/mkdocs-base.yml"
}

@test "new: leaves an existing page, index and nav alone" {
  entry PMM-1
  run "$rn" new 3.11.0
  [ "$status" -eq 0 ]
  grep -q 'browse/PMM-1)' "$page"
  [ "$(grep -c '"PMM 3.11.0"' "$doc/mkdocs-base.yml")" -eq 1 ]
}

@test "check: passes when the release branch has every entry" {
  entry PMM-1
  at remotes/origin/pmm-3.11.0
  run "$rn" check
  [ "$status" -eq 0 ]
}

@test "check: fails on each entry the release branch doesn't have" {
  entry PMM-1
  at remotes/origin/pmm-3.11.0
  entry PMM-2
  entry PMM-3
  run "$rn" check
  [ "$status" -eq 1 ]
  [[ $output == *"PMM-2 is in 3.11.0.md on main but not on pmm-3.11.0"* ]]
  [[ $output == *"PMM-3 is in 3.11.0.md on main but not on pmm-3.11.0"* ]]
  [[ $output != *"PMM-1 "* ]]
}

@test "check: skips a version without a release branch" {
  entry PMM-1
  run "$rn" check
  [ "$status" -eq 0 ]
}

@test "check: skips a released version" {
  at remotes/origin/pmm-3.11.0
  entry PMM-1
  at tags/v3.11.0
  run "$rn" check
  [ "$status" -eq 0 ]
}
