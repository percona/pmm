#!/bin/bash
# Checks release-note fragments in a PR.
# Usage: rn-check.sh BASE_BRANCH FILE...

# shellcheck source-path=SCRIPTDIR
set -o errexit -o nounset -o pipefail

. "$(dirname "$0")/rn-common.sh"

base=$1
shift

releases=$(rn_release_versions | sort -V)
latest=$(tail -n 1 <<< "$releases")
errors=0

fail() {
  echo "::error file=$file::$(basename "$file"): $*"
  errors=$((errors + 1))
}

for file in "$@"; do
  name=$(basename "$file")
  [ "$name" = README.md ] && continue

  if ! [[ $name =~ ^PMM-[0-9]+\.md$ ]]; then
    fail "name the file after its ticket, for example PMM-12345.md"
    continue
  fi
  if ! rn_has_front_matter "$file"; then
    fail "start the file with front matter: target and type between two --- lines"
    continue
  fi

  target=$(rn_field "$file" target)
  type=$(rn_field "$file" type)
  before=$errors

  if [ -z "$target" ]; then
    fail "target is required: the version this change ships in, for example target: 3.11.0"
  elif ! [[ $target =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    fail "target must be a version like 3.11.0, not '$target'"
  elif [[ $base =~ ^pmm-([0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
    [ "$target" = "${BASH_REMATCH[1]}" ] || fail "this PR goes to $base, so target must be ${BASH_REMATCH[1]}"
  elif grep -qxF "$target" <<< "$releases"; then
    fail "pmm-$target is already branched. Open this PR against pmm-$target, or target a later version"
  elif [ -n "$latest" ] && rn_version_le "$target" "$latest"; then
    fail "$target is older than the newest release branch, pmm-$latest. Target a later version"
  fi

  case " $RN_TYPES " in
    *" $type "*) ;;
    *) fail "type must be one of: $RN_TYPES" ;;
  esac
  case $type in
    highlight | security | known-issue)
      [ -n "$(rn_field "$file" title)" ] || fail "a $type needs a title" ;;
  esac
  [ -n "$(rn_body "$file")" ] || fail "write the release note below the front matter"

  if [ "$errors" -eq "$before" ]; then
    echo "$name: $type for $target"
  fi
done

[ "$errors" -eq 0 ]
