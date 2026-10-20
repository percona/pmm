#!/bin/bash
# Release notes waiting in documentation/docs/release-notes/unreleased/; its README.md has the format.
#
#   release-notes.sh check BASE_BRANCH HEAD_BRANCH [FILE...]
#       Checks the notes a PR adds or changes. On the release-notes-X.Y.Z PR it also fails
#       while an X.Y.Z note is missing from X.Y.Z.md or still in unreleased/.
#   release-notes.sh build VERSION
#       Adds the VERSION notes to VERSION.md, creating it from the template,
#       and deletes from unreleased/ the notes it now holds.
#
# The VERSION notes come from unreleased/ on origin/pmm-VERSION; NOTES_DIR replaces them in tests.
# DOC_DIR replaces the documentation/ directory.

set -o errexit -o nounset -o pipefail
shopt -s nullglob

doc=${DOC_DIR:-$(dirname "$0")/../..}
unreleased=$doc/docs/release-notes/unreleased
jira=https://perconadev.atlassian.net/browse
declare -A headings=(
  [improvement]="## 📈 Improvements"
  [fix]="## ✅ Fixed issues"
  [known-issue]="## 🔶 Known issues"
)

has_front_matter() {
  [ "$(head -n 1 "$1")" = "---" ] && [ "$(grep -c '^---$' "$1")" -ge 2 ]
}

field() {
  yq --front-matter=extract ".$2 // \"\"" "$1"
}

# body FILE prints the text after the front matter, without leading or trailing blank lines.
body() {
  awk '
    f >= 2 { line[++n] = $0 }
    /^---$/ && f < 2 { f++ }
    END {
      s = 1; while (s <= n && line[s] ~ /^[[:space:]]*$/) s++
      e = n; while (e >= s && line[e] ~ /^[[:space:]]*$/) e--
      for (i = s; i <= e; i++) print line[i]
    }' "$1"
}

# on_page PAGE KEY: the closing parenthesis keeps PMM-12 from matching PMM-123.
on_page() {
  grep -qF "browse/$2)" "$1"
}

# notes_for VERSION DIR prints the notes in DIR that target VERSION, in ticket order.
notes_for() {
  local file
  for file in $(printf '%s\n' "$2"/PMM-*.md | sort -V); do
    if has_front_matter "$file" && [ "$(field "$file" target)" = "$1" ]; then
      echo "$file"
    fi
  done
}

release_notes() {
  local dir=${NOTES_DIR:-}
  if [ -z "$dir" ]; then
    dir=$(mktemp -d)
    if git cat-file -e "origin/pmm-$1:documentation/docs/release-notes/unreleased" 2> /dev/null; then
      git archive "origin/pmm-$1" documentation/docs/release-notes/unreleased | tar -x -C "$dir" --strip-components=4
    fi
  fi
  notes_for "$1" "$dir"
}

# Versions with a pmm-X.Y.Z branch. RN_RELEASE_BRANCHES replaces the lookup in tests.
release_versions() {
  if [ -n "${RN_RELEASE_BRANCHES+x}" ]; then
    tr ' ' '\n' <<< "$RN_RELEASE_BRANCHES"
  else
    git ls-remote --heads origin 'pmm-*' | sed -n 's#.*refs/heads/pmm-\([0-9]*\.[0-9]*\.[0-9]*\)$#\1#p'
  fi
}

error() {
  echo "::error::$*"
  errors=$((errors + 1))
}

check_note() {
  local file=$1 name target type
  name=$(basename "$file")
  if ! [[ $name =~ ^PMM-[0-9]+\.md$ ]]; then
    error "$name: name the file after its ticket, for example PMM-12345.md"
    return
  fi
  if ! has_front_matter "$file"; then
    error "$name: start the file with front matter: target and type between two --- lines"
    return
  fi

  target=$(field "$file" target)
  type=$(field "$file" type)
  if [ -z "$target" ]; then
    error "$name: target is required: the version this change ships in, for example target: 3.11.0"
  elif ! [[ $target =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    error "$name: target must be a version like 3.11.0, not '$target'"
  elif [[ $base =~ ^pmm-([0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
    [ "$target" = "${BASH_REMATCH[1]}" ] || error "$name: this PR goes to $base, so target must be ${BASH_REMATCH[1]}"
  elif grep -qxF "$target" <<< "$releases"; then
    error "$name: pmm-$target is already branched. Open this PR against pmm-$target, or target a later version"
  elif [ -n "$latest" ] && [ "$(printf '%s\n%s\n' "$target" "$latest" | sort -V | tail -n 1)" = "$latest" ]; then
    error "$name: $target is older than the newest release branch, pmm-$latest. Target a later version"
  fi

  [ -n "${headings[$type]:-}" ] || error "$name: type must be one of: improvement fix known-issue"
  if [ "$type" = known-issue ] && [ -z "$(field "$file" title)" ]; then
    error "$name: a known-issue needs a title"
  fi
  [ -n "$(body "$file")" ] || error "$name: write the release note below the front matter"
}

# check_page VERSION: every VERSION note on the release branch is on the page, and none is left in unreleased/.
check_page() {
  local version=$1 page=$doc/docs/release-notes/$1.md file key
  if [ ! -f "$page" ]; then
    error "release-notes/$version.md does not exist. Run the Release notes workflow for $version."
    return
  fi
  for file in $(release_notes "$version"); do
    key=$(basename "$file" .md)
    on_page "$page" "$key" || error "$key is on pmm-$version but not on the page. Run the Release notes workflow for $version."
  done
  for file in $(notes_for "$version" "$unreleased"); do
    key=$(basename "$file" .md)
    if on_page "$page" "$key"; then
      error "$key is on the page but still in unreleased/. Delete unreleased/$key.md."
    else
      error "$key targets $version but is not on pmm-$version. Cherry-pick its change to pmm-$version, or change its target."
    fi
  done
}

check() {
  local file
  base=$1
  head=$2
  shift 2
  errors=0
  releases=$(release_versions | sort -V)
  latest=$(tail -n 1 <<< "$releases")

  for file in "$@"; do
    if [ "$(basename "$file")" != README.md ]; then
      check_note "$file"
    fi
  done
  if [[ $head =~ ^release-notes-([0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
    check_page "${BASH_REMATCH[1]}"
  fi

  [ "$errors" -eq 0 ] || exit 1
  echo "Release notes are fine."
}

render() {
  local file=$1 type=$2 link="[$3]($jira/$3)"
  if [ "$type" = known-issue ]; then
    printf '### %s (%s)\n\n' "$(field "$file" title)" "$link"
    body "$file"
  else
    body "$file" | awk -v lead="- $link: " 'NR == 1 { print lead $0; next } /^$/ { print; next } { print "    " $0 }'
  fi
}

# insert PAGE HEADING ENTRY_FILE puts the entry at the end of that section.
insert() {
  local tmp
  tmp=$(mktemp)
  awk -v h="$2" -v entry="$3" '
    function add(  line) {
      if (last != "") print ""
      while ((getline line < entry) > 0) print line
      print ""
      section = 0
    }
    section && /^## / { add() }
    $0 == h { section = 1 }
    { print; last = $0 }
    END { if (section) add() }' "$1" > "$tmp"
  cat "$tmp" > "$1"
  rm -f "$tmp"
}

build() {
  local version=$1 page=$doc/docs/release-notes/$1.md file key type heading entry

  if [ ! -f "$page" ]; then
    sed "s/{{VERSION}}/$version/g" "$doc/resources/release-notes-template.md" > "$page"
    sed -i "/^# Release notes$/a - [Percona Monitoring and Management $version]($version.md)" \
      "$doc/docs/release-notes/index.md"
    sed -i "/^ *- Release notes index: release-notes\/index.md$/a\\      - \"PMM $version\": release-notes/$version.md" \
      "$doc/mkdocs-base.yml"
    echo "Created $(basename "$page") from the template"
  fi

  for file in $(release_notes "$version"); do
    key=$(basename "$file" .md)
    if on_page "$page" "$key"; then
      continue
    fi
    type=$(field "$file" type)
    heading=${headings[$type]:-}
    if [ -z "$heading" ] || ! grep -qxF "$heading" "$page"; then
      echo "::error::$key: $(basename "$page") has no section for type '$type'"
      exit 1
    fi
    entry=$(mktemp)
    render "$file" "$type" "$key" > "$entry"
    insert "$page" "$heading" "$entry"
    rm -f "$entry"
    echo "$key: added as $type"
  done

  # A note left in unreleased/ that is not on the page stays, and fails the check.
  for file in $(notes_for "$version" "$unreleased"); do
    key=$(basename "$file" .md)
    if on_page "$page" "$key"; then
      rm "$file"
      echo "$key: removed from unreleased/"
    fi
  done
}

case ${1:-} in
  check | build)
    "$@"
    ;;
  *)
    echo "Usage: $0 check BASE_BRANCH HEAD_BRANCH [FILE...] | build VERSION" >&2
    exit 2
    ;;
esac
