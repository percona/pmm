#!/bin/bash
# Fails while a release note of VERSION is missing from its page or still waits in unreleased/.
# Usage: rn-gate.sh VERSION RELEASE_BRANCH_NOTES_DIR [DOCUMENTATION_DIR]
# RELEASE_BRANCH_NOTES_DIR is unreleased/ from pmm-VERSION; DOCUMENTATION_DIR is the tree being merged.

# shellcheck source-path=SCRIPTDIR
set -o errexit -o nounset -o pipefail
shopt -s nullglob

here=$(dirname "$0")
. "$here/rn-common.sh"

version=$1
notes=$2
doc=${3:-$here/../..}
page=$doc/docs/release-notes/$version.md
errors=0

if [ ! -f "$page" ]; then
  echo "::error::release-notes/$version.md does not exist. Run the Release notes workflow for $version."
  exit 1
fi

for file in "$notes"/PMM-*.md; do
  [ "$(rn_target_of "$file")" = "$version" ] || continue
  key=$(basename "$file" .md)
  if ! rn_on_page "$page" "$key"; then
    echo "::error::$key is on pmm-$version but not on the page. Run the Release notes workflow for $version."
    errors=$((errors + 1))
  fi
done

for file in "$doc/docs/release-notes/unreleased"/PMM-*.md; do
  [ "$(rn_target_of "$file")" = "$version" ] || continue
  key=$(basename "$file" .md)
  if rn_on_page "$page" "$key"; then
    echo "::error::$key is on the page but still in unreleased/. Delete unreleased/$key.md."
  else
    echo "::error::$key targets $version but is not on pmm-$version. Cherry-pick its change to pmm-$version, or change its target."
  fi
  errors=$((errors + 1))
done

if [ "$errors" -gt 0 ]; then
  exit 1
fi
echo "Every $version release note is on the page."
