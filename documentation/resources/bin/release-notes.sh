#!/bin/bash
# Release notes pages, documentation/docs/release-notes/X.Y.Z.md. Entries are written on main.
#
#   release-notes.sh new VERSION
#       Creates VERSION.md from the template and lists it in the index and nav, unless it exists.
#       The RC job runs it on main for the next patch version when it creates pmm-X.Y.Z.
#   release-notes.sh check
#       For each version in RC (pmm-X.Y.Z exists, vX.Y.Z doesn't), fails on every ticket in
#       X.Y.Z.md here that X.Y.Z.md on origin/pmm-X.Y.Z doesn't have.
#
# DOC_DIR replaces the documentation/ directory.

set -o errexit -o nounset -o pipefail
shopt -s nullglob

doc=${DOC_DIR:-$(dirname "$0")/../..}
pages=$doc/docs/release-notes

new() {
  local version=$1 page=$pages/$1.md
  if [ -f "$page" ]; then
    echo "$(basename "$page") already exists"
    return
  fi
  sed "s/{{VERSION}}/$version/g" "$doc/resources/release-notes-template.md" > "$page"
  sed -i "/^# Release notes$/a - [Percona Monitoring and Management $version]($version.md)" "$pages/index.md"
  sed -i "/^ *- Release notes index: release-notes\/index.md$/a\\      - \"PMM $version\": release-notes/$version.md" \
    "$doc/mkdocs-base.yml"
  echo "Created $(basename "$page") from the template"
}

tickets() {
  grep -o 'browse/PMM-[0-9]*' | sed 's#browse/##' | sort -u
}

check() {
  local page version path ticket errors=0
  for page in "$pages"/[0-9]*.md; do
    version=$(basename "$page" .md)
    git -C "$pages" rev-parse -q --verify "origin/pmm-$version" > /dev/null || continue
    ! git -C "$pages" rev-parse -q --verify "refs/tags/v$version" > /dev/null || continue
    path=$(realpath --relative-to="$(git -C "$pages" rev-parse --show-toplevel)" "$page")
    for ticket in $(comm -23 <(tickets < "$page") <(git -C "$pages" show "origin/pmm-$version:$path" 2> /dev/null | tickets)); do
      echo "::error::$ticket is in $version.md on main but not on pmm-$version. Cherry-pick its fix to pmm-$version, or move the entry to the next release."
      errors=$((errors + 1))
    done
  done
  [ "$errors" -eq 0 ] || exit 1
  echo "Every release in RC has its entries on its release branch."
}

case ${1:-} in
  new | check)
    "$@"
    ;;
  *)
    echo "Usage: $0 new VERSION | check" >&2
    exit 2
    ;;
esac
