#!/usr/bin/env bats
# Tests for the release-note scripts. Run: bats documentation/resources/bin/tests

setup() {
  bin="$BATS_TEST_DIRNAME/.."
  doc="$BATS_TEST_TMPDIR/documentation"
  notes="$BATS_TEST_TMPDIR/notes"
  mkdir -p "$doc/docs/release-notes/unreleased" "$doc/resources" "$notes"
  cp "$bin/../release-notes-template.md" "$doc/resources/"
  cp "$bin/../../docs/release-notes/index.md" "$doc/docs/release-notes/"
  cp "$bin/../../mkdocs-base.yml" "$doc/"
  export RN_RELEASE_BRANCHES="3.9.1 3.10.0"
}

# note DIR KEY FRONT_MATTER [BODY]
note() {
  printf -- '---\n%b\n---\n%b\n' "$3" "${4-Fixed a thing.}" > "$1/$2.md"
}

@test "check: missing target fails" {
  note "$notes" PMM-1 'type: fix'
  run "$bin/rn-check.sh" main "$notes/PMM-1.md"
  [ "$status" -eq 1 ]
  [[ $output == *"target is required"* ]]
}

@test "check: target that is already branched fails on main" {
  note "$notes" PMM-1 'target: 3.10.0\ntype: fix'
  run "$bin/rn-check.sh" main "$notes/PMM-1.md"
  [ "$status" -eq 1 ]
  [[ $output == *"pmm-3.10.0 is already branched"* ]]
}

@test "check: target older than the newest release branch fails on main" {
  note "$notes" PMM-1 'target: 3.9.5\ntype: fix'
  run "$bin/rn-check.sh" main "$notes/PMM-1.md"
  [ "$status" -eq 1 ]
  [[ $output == *"older than the newest release branch, pmm-3.10.0"* ]]
}

@test "check: future target passes on main" {
  note "$notes" PMM-1 'target: 3.11.0\ntype: fix'
  note "$notes" PMM-2 'target: 3.10.1\ntype: improvement'
  run "$bin/rn-check.sh" main "$notes/PMM-1.md" "$notes/PMM-2.md"
  [ "$status" -eq 0 ]
  [[ $output == *"PMM-1.md: fix for 3.11.0"* ]]
}

@test "check: release branch needs its own version" {
  note "$notes" PMM-1 'target: 3.11.0\ntype: fix'
  run "$bin/rn-check.sh" pmm-3.10.0 "$notes/PMM-1.md"
  [ "$status" -eq 1 ]
  [[ $output == *"target must be 3.10.0"* ]]
}

@test "check: release branch accepts its own version" {
  note "$notes" PMM-1 'target: 3.10.0\ntype: fix'
  run "$bin/rn-check.sh" pmm-3.10.0 "$notes/PMM-1.md"
  [ "$status" -eq 0 ]
}

@test "check: bad fields fail" {
  note "$notes" PMM-1 'target: 3.11\ntype: fix'
  note "$notes" PMM-2 'target: 3.11.0\ntype: bugfix'
  note "$notes" PMM-3 'target: 3.11.0\ntype: known-issue'
  note "$notes" PMM-4 'target: 3.11.0\ntype: fix' ''
  run "$bin/rn-check.sh" main "$notes"/PMM-{1,2,3,4}.md
  [ "$status" -eq 1 ]
  [[ $output == *"PMM-1.md: target must be a version like 3.11.0"* ]]
  [[ $output == *"PMM-2.md: type must be one of"* ]]
  [[ $output == *"PMM-3.md: a known-issue needs a title"* ]]
  [[ $output == *"PMM-4.md: write the release note"* ]]
}

@test "check: file name and front matter are required, README is skipped" {
  note "$notes" fix-login 'target: 3.11.0\ntype: fix'
  echo "Fixed a thing." > "$notes/PMM-5.md"
  echo "# Notes" > "$notes/README.md"
  run "$bin/rn-check.sh" main "$notes/fix-login.md" "$notes/PMM-5.md" "$notes/README.md"
  [ "$status" -eq 1 ]
  [[ $output == *"name the file after its ticket"* ]]
  [[ $output == *"PMM-5.md: start the file with front matter"* ]]
  [[ $output != *"README"* ]]
}

@test "compile: creates the page and lists it in the index and nav" {
  note "$notes" PMM-15310 'target: 3.11.0\ntype: fix' 'Fixed the Disconnected status.'
  run "$bin/rn-compile.sh" 3.11.0 "$notes" "$doc"
  [ "$status" -eq 0 ]
  page="$doc/docs/release-notes/3.11.0.md"
  grep -qx '# Percona Monitoring and Management 3.11.0' "$page"
  grep -qx -- '- \[PMM-15310\](https://perconadev.atlassian.net/browse/PMM-15310): Fixed the Disconnected status.' "$page"
  ! sed -n '/^## ✅ Fixed issues/,/^## 🔶/p' "$page" | grep -q 'to be added'
  sed -n '/^## 📈 Improvements/,/^## ✅/p' "$page" | grep -q 'to be added'
  [ "$(sed -n '2p' "$doc/docs/release-notes/index.md")" = '- [Percona Monitoring and Management 3.11.0](3.11.0.md)' ]
  grep -A1 'Release notes index:' "$doc/mkdocs-base.yml" | grep -qx '      - "PMM 3.11.0": release-notes/3.11.0.md'
}

@test "compile: a re-run adds only new notes and keeps edits" {
  note "$notes" PMM-1 'target: 3.11.0\ntype: fix' 'First fix.'
  "$bin/rn-compile.sh" 3.11.0 "$notes" "$doc"
  page="$doc/docs/release-notes/3.11.0.md"
  sed -i 's/First fix\./First fix, edited./' "$page"
  note "$notes" PMM-2 'target: 3.11.0\ntype: fix' 'Second fix.'
  note "$notes" PMM-3 'target: 3.12.0\ntype: fix' 'Later fix.'
  run "$bin/rn-compile.sh" 3.11.0 "$notes" "$doc"
  [ "$status" -eq 0 ]
  [ "$(grep -c 'browse/PMM-1)' "$page")" -eq 1 ]
  grep -q 'First fix, edited.' "$page"
  grep -q 'browse/PMM-2): Second fix.' "$page"
  ! grep -q 'PMM-3' "$page"
  [ "$(grep -c 'Percona Monitoring and Management 3.11.0' "$doc/docs/release-notes/index.md")" -eq 1 ]
}

@test "compile: each type lands in its section and format" {
  note "$notes" PMM-1 'target: 3.11.0\ntype: known-issue\ntitle: QAN is empty after failover' 'QAN shows no data.\n\n**Workaround**: reload.'
  note "$notes" PMM-2 'target: 3.11.0\ntype: highlight\ntitle: Fleet view' 'See every cluster at once.'
  note "$notes" PMM-3 'target: 3.11.0\ntype: improvement' 'Faster.\n\nSecond paragraph.'
  run "$bin/rn-compile.sh" 3.11.0 "$notes" "$doc"
  [ "$status" -eq 0 ]
  page="$doc/docs/release-notes/3.11.0.md"
  sed -n '/^## 🔶 Known issues/,/^## 🚀/p' "$page" | grep -qxF '### QAN is empty after failover ([PMM-1](https://perconadev.atlassian.net/browse/PMM-1))'
  sed -n '/^## ✨ Release highlights/,/^## 🔒/p' "$page" | grep -qx '### Fleet view'
  sed -n '/^## 📈 Improvements/,/^## ✅/p' "$page" | grep -qx '    Second paragraph.'
}

@test "compile: a removed section comes back before the upgrade links" {
  "$bin/rn-compile.sh" 3.11.0 "$notes" "$doc"
  page="$doc/docs/release-notes/3.11.0.md"
  sed -i '/^## 🔶 Known issues/,/^## 🚀/{/^## 🚀/!d}' "$page"
  note "$notes" PMM-1 'target: 3.11.0\ntype: known-issue\ntitle: Slow' 'It is slow.'
  run "$bin/rn-compile.sh" 3.11.0 "$notes" "$doc"
  [ "$status" -eq 0 ]
  [ "$(grep -n '^## 🔶 Known issues' "$page" | cut -d: -f1)" -lt "$(grep -n '^## 🚀' "$page" | cut -d: -f1)" ]
}

@test "gate: passes when every note is on the page" {
  note "$notes" PMM-1 'target: 3.11.0\ntype: fix'
  note "$doc/docs/release-notes/unreleased" PMM-9 'target: 3.12.0\ntype: fix'
  "$bin/rn-compile.sh" 3.11.0 "$notes" "$doc"
  run "$bin/rn-gate.sh" 3.11.0 "$notes" "$doc"
  [ "$status" -eq 0 ]
}

@test "gate: fails for a release-branch note missing from the page" {
  "$bin/rn-compile.sh" 3.11.0 "$notes" "$doc"
  note "$notes" PMM-1 'target: 3.11.0\ntype: fix'
  run "$bin/rn-gate.sh" 3.11.0 "$notes" "$doc"
  [ "$status" -eq 1 ]
  [[ $output == *"PMM-1 is on pmm-3.11.0 but not on the page"* ]]
}

@test "gate: fails for a note left on main" {
  "$bin/rn-compile.sh" 3.11.0 "$notes" "$doc"
  note "$doc/docs/release-notes/unreleased" PMM-2 'target: 3.11.0\ntype: fix'
  run "$bin/rn-gate.sh" 3.11.0 "$notes" "$doc"
  [ "$status" -eq 1 ]
  [[ $output == *"PMM-2 targets 3.11.0 but is not on pmm-3.11.0"* ]]
}
