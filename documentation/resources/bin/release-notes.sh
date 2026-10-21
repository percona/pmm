#!/bin/bash
# Release notes for the next release, whatever its number, go to documentation/docs/release-notes/next.md on main.
#
#   release-notes.sh rc VERSION
#       Turns next.md into VERSION.md, lists it in the index and nav, and empties next.md. The RC job runs
#       it on main right before it creates pmm-VERSION; it does nothing when VERSION.md already exists.
#   release-notes.sh publish VERSION
#       Fails unless VERSION.md is the same here and on origin/main. The site build runs it on pmm-VERSION.
#
# DOC_DIR replaces the documentation/ directory.

set -o errexit -o nounset -o pipefail

doc=${DOC_DIR:-$(dirname "$0")/../..}
pages=$doc/docs/release-notes

rc() {
  local version=$1 page=$pages/$1.md
  if [ -f "$page" ]; then
    echo "$(basename "$page") already exists"
    return
  fi
  sed "s/{{VERSION}}/$version/g" "$pages/next.md" > "$page"
  cp "$doc/resources/release-notes-template.md" "$pages/next.md"
  sed -i "/^# Release notes$/a - [Percona Monitoring and Management $version]($version.md)" "$pages/index.md"
  sed -i "/^ *- Release notes index: release-notes\/index.md$/a\\      - \"PMM $version\": release-notes/$version.md" \
    "$doc/mkdocs-base.yml"
  echo "Moved next.md to $(basename "$page")"
}

publish() {
  local version=$1
  if [ ! -f "$pages/$version.md" ]; then
    echo "::error::$version.md is missing on pmm-$version."
    exit 1
  fi
  if ! git -C "$pages" diff --exit-code origin/main HEAD -- "$version.md"; then
    echo "::error::$version.md differs between main and pmm-$version. Cherry-pick what is missing on either side, then publish again."
    exit 1
  fi
}

case ${1:-} in
  rc | publish)
    "$@"
    ;;
  *)
    echo "Usage: $0 rc VERSION | publish VERSION" >&2
    exit 2
    ;;
esac
