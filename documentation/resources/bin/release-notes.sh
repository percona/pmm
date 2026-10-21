#!/bin/bash
# Release notes pages, documentation/docs/release-notes/X.Y.Z.md. Entries are written on main.
# A version is released once vX.Y.Z is tagged, and in RC while pmm-X.Y.Z exists without the tag.
#
#   release-notes.sh new VERSION
#       Creates VERSION.md from the template and lists it in the index and nav, unless it exists.
#       The RC job runs it on main for the next patch version when it creates pmm-X.Y.Z.
#   release-notes.sh check
#       For each version in RC, fails on every ticket in X.Y.Z.md here that pmm-X.Y.Z doesn't have.
#   release-notes.sh publish VERSION
#       Fails on any page other than VERSION.md that is not released. The site build runs it on pmm-VERSION.
#
# DOC_DIR replaces the documentation/ directory.

set -o errexit -o nounset -o pipefail

doc=${DOC_DIR:-$(dirname "$0")/../..}
pages=$doc/docs/release-notes
errors=0

error() {
  echo "::error::$*"
  errors=$((errors + 1))
}

versions() {
  local page v
  for page in "$pages"/*.md; do
    v=$(basename "$page" .md)
    if [[ $v =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      echo "$v"
    fi
  done
}

released() {
  git -C "$pages" rev-parse -q --verify "refs/tags/v$1" > /dev/null
}

in_rc() {
  ! released "$1" && git -C "$pages" rev-parse -q --verify "origin/pmm-$1" > /dev/null
}

tickets() {
  grep -o 'browse/PMM-[0-9]*' | sed 's#browse/##' | sort -u
}

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

check() {
  local v path ticket
  for v in $(versions); do
    in_rc "$v" || continue
    path=$(realpath --relative-to="$(git -C "$pages" rev-parse --show-toplevel)" "$pages/$v.md")
    for ticket in $(comm -23 <(tickets < "$pages/$v.md") <(git -C "$pages" show "origin/pmm-$v:$path" 2> /dev/null | tickets)); do
      error "$ticket is in $v.md on main but not on pmm-$v. Cherry-pick its fix to pmm-$v, or move the entry to the next release."
    done
  done
}

publish() {
  local v
  for v in $(versions); do
    if [ "$v" != "$1" ] && ! released "$v"; then
      error "$v.md is not released, so pmm-$1 can't publish it. Remove it from pmm-$1."
    fi
  done
}

case ${1:-} in
  new)
    "$@"
    ;;
  check | publish)
    "$@"
    [ "$errors" -eq 0 ] || exit 1
    echo "Release notes pages are fine."
    ;;
  *)
    echo "Usage: $0 new VERSION | check | publish VERSION" >&2
    exit 2
    ;;
esac
