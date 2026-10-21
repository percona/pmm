#!/usr/bin/env bats
# Tests for release-notes.sh. Run: bats documentation/resources/bin/tests

setup() {
  bin="$BATS_TEST_DIRNAME/.."
  doc="$BATS_TEST_TMPDIR/documentation"
  notes="$BATS_TEST_TMPDIR/notes"
  unreleased="$doc/docs/release-notes/unreleased"
  mkdir -p "$unreleased" "$doc/resources" "$notes"
  cp "$bin/../release-notes-template.md" "$doc/resources/"
  cp "$bin/../../docs/release-notes/index.md" "$doc/docs/release-notes/"
  cp "$bin/../../mkdocs-base.yml" "$doc/"
  export DOC_DIR="$doc" NOTES_DIR="$notes"
  rn="$bin/release-notes.sh"
  page="$doc/docs/release-notes/3.11.0.md"
}

# note DIR FILE [TEXT]
note() {
  printf '%b\n' "${3-Fixed a thing.}" > "$1/$2"
}

@test "check: a well-named note passes" {
  note "$notes" PMM-1-fix.md
  note "$notes" PMM-2-improvement.md
  note "$notes" PMM-3-known-issue.md 'Slow\n\nIt is slow.'
  run "$rn" check "$notes"/PMM-*.md
  [ "$status" -eq 0 ]
}

@test "check: a bad name or an empty note fails" {
  note "$notes" PMM-1.md
  note "$notes" PMM-2-bugfix.md
  note "$notes" pmm-3-fix.md
  note "$notes" PMM-4-fix.md ''
  run "$rn" check "$notes"/*.md
  [ "$status" -eq 1 ]
  [[ $output == *"PMM-1.md: name it PMM-<ticket>-<type>.md"* ]]
  [[ $output == *"PMM-2-bugfix.md: name it"* ]]
  [[ $output == *"pmm-3-fix.md: name it"* ]]
  [[ $output == *"PMM-4-fix.md: write the release note"* ]]
}

@test "check: a note already published in its section fails" {
  note "$notes" PMM-1-fix.md
  "$rn" build 3.11.0
  note "$unreleased" PMM-1-fix.md
  note "$unreleased" PMM-1-known-issue.md 'Still slow\n\nIt is slow.'
  run "$rn" check "$unreleased/PMM-1-fix.md" "$unreleased/PMM-1-known-issue.md"
  [ "$status" -eq 1 ]
  [[ $output == *"PMM-1-fix.md: already published in 3.11.0.md"* ]]
  [[ $output != *"PMM-1-known-issue.md"* ]]
}

@test "build: creates the page and lists it in the index and nav" {
  note "$notes" PMM-15310-fix.md 'Fixed the Disconnected status.'
  run "$rn" build 3.11.0
  [ "$status" -eq 0 ]
  grep -qx '# Percona Monitoring and Management 3.11.0' "$page"
  sed -n '/^## ✅ Fixed issues/,/^## 🔶/p' "$page" \
    | grep -qx -- '- \[PMM-15310\](https://perconadev.atlassian.net/browse/PMM-15310): Fixed the Disconnected status.'
  [ "$(sed -n '2p' "$doc/docs/release-notes/index.md")" = '- [Percona Monitoring and Management 3.11.0](3.11.0.md)' ]
  grep -A1 'Release notes index:' "$doc/mkdocs-base.yml" | grep -qx '      - "PMM 3.11.0": release-notes/3.11.0.md'
}

@test "build: a re-run adds only new notes and keeps edits" {
  note "$notes" PMM-1-fix.md 'First fix.'
  "$rn" build 3.11.0
  sed -i 's/First fix\./First fix, edited./' "$page"
  note "$notes" PMM-2-fix.md 'Second fix.'
  run "$rn" build 3.11.0
  [ "$status" -eq 0 ]
  [ "$(grep -c 'browse/PMM-1)' "$page")" -eq 1 ]
  grep -q 'First fix, edited.' "$page"
  grep -q 'browse/PMM-2): Second fix.' "$page"
  [ "$(grep -c 'Percona Monitoring and Management 3.11.0' "$doc/docs/release-notes/index.md")" -eq 1 ]
}

@test "build: improvements and known issues get their format" {
  note "$notes" PMM-1-known-issue.md 'QAN is empty after failover\n\nQAN shows no data.'
  note "$notes" PMM-2-improvement.md 'Faster.\n\nSecond paragraph.'
  run "$rn" build 3.11.0
  [ "$status" -eq 0 ]
  sed -n '/^## 🔶 Known issues/,/^## 🚀/p' "$page" \
    | grep -qxF '### QAN is empty after failover ([PMM-1](https://perconadev.atlassian.net/browse/PMM-1))'
  sed -n '/^## 📈 Improvements/,/^## ✅/p' "$page" | grep -qx '    Second paragraph.'
}

@test "build: a ticket mentioned elsewhere on the page is still added" {
  "$rn" build 3.11.0
  sed -i 's|^## 📋 Release summary$|&\n\nSee [PMM-1](https://perconadev.atlassian.net/browse/PMM-1).|' "$page"
  note "$notes" PMM-1-fix.md 'Fixed it.'
  run "$rn" build 3.11.0
  [ "$status" -eq 0 ]
  sed -n '/^## ✅ Fixed issues/,/^## 🔶/p' "$page" | grep -q 'browse/PMM-1): Fixed it.'
}

@test "build: the same ticket can be a fix and a known issue" {
  note "$notes" PMM-1-fix.md 'Fixed it.'
  note "$notes" PMM-1-known-issue.md 'Still slow\n\nIt is slow.'
  run "$rn" build 3.11.0
  [ "$status" -eq 0 ]
  [ "$(grep -c 'browse/PMM-1)' "$page")" -eq 2 ]
}

@test "build: an entry the writer deleted is not added back" {
  note "$notes" PMM-1-fix.md 'Fixed it.'
  "$rn" build 3.11.0
  sed -i '/browse\/PMM-1)/d' "$page"
  run "$rn" build 3.11.0
  [ "$status" -eq 0 ]
  ! grep -q 'browse/PMM-1)' "$page"
}

@test "build: a section the writer deleted is not added back, its note is deleted with a warning" {
  "$rn" build 3.11.0
  sed -i '/^## 📈 Improvements/d' "$page"
  note "$notes" PMM-1-improvement.md 'Faster.'
  note "$notes" PMM-2-fix.md 'Fixed it.'
  run "$rn" build 3.11.0
  [ "$status" -eq 0 ]
  [[ $output == *"PMM-1-improvement.md: 3.11.0.md has no '## 📈 Improvements' section"* ]]
  ! grep -q '^## 📈 Improvements' "$page"
  grep -q 'browse/PMM-2): Fixed it.' "$page"
  [ ! -e "$notes/PMM-1-improvement.md" ]
}

@test "build: deletes each note it adds, from the release branch and from unreleased/" {
  note "$notes" PMM-1-fix.md
  note "$unreleased" PMM-1-fix.md
  note "$unreleased" PMM-1-improvement.md
  note "$unreleased" PMM-2-fix.md
  run "$rn" build 3.11.0
  [ "$status" -eq 0 ]
  [ ! -e "$notes/PMM-1-fix.md" ]
  [ ! -e "$unreleased/PMM-1-fix.md" ]
  [ -e "$unreleased/PMM-1-improvement.md" ]
  [ -e "$unreleased/PMM-2-fix.md" ]
}

@test "build: deletes from unreleased/ a note already in an earlier release" {
  note "$notes" PMM-1-fix.md 'Fixed it.'
  "$rn" build 3.10.0
  note "$unreleased" PMM-1-fix.md 'Fixed it.'
  run "$rn" build 3.11.0
  [ "$status" -eq 0 ]
  [ ! -e "$unreleased/PMM-1-fix.md" ]
}
