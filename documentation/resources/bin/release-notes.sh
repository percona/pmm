#!/bin/bash
# Release notes waiting in documentation/docs/release-notes/unreleased/, one file per entry named
# PMM-<ticket>-<type>.md. A note ships in the release cut from the branch it is on.
#
#   release-notes.sh check [FILE...]
#       Checks the notes a PR adds or changes.
#   release-notes.sh build VERSION
#       Adds the pmm-VERSION notes to VERSION.md, creating it from the template, and deletes each
#       note it adds from NOTES_DIR and from unreleased/. A note is added once: an entry or a
#       section the writer deletes stays deleted.
#
# NOTES_DIR is unreleased/ in a checkout of pmm-VERSION.
# DOC_DIR replaces the documentation/ directory.

set -o errexit -o nounset -o pipefail
shopt -s nullglob

doc=${DOC_DIR:-$(dirname "$0")/../..}
pages=$doc/docs/release-notes
unreleased=$pages/unreleased
jira=https://perconadev.atlassian.net/browse
name='^(PMM-[0-9]+)-(improvement|fix|known-issue)\.md$'
declare -A headings=(
  [improvement]="## 📈 Improvements"
  [fix]="## ✅ Fixed issues"
  [known-issue]="## 🔶 Known issues"
)

error() {
  echo "::error::$*"
  errors=$((errors + 1))
}

# notes DIR prints the notes in DIR, in ticket order.
notes() {
  local file
  for file in $(printf '%s\n' "$1"/PMM-*.md | sort -V); do
    if [[ $(basename "$file") =~ $name ]]; then
      echo "$file"
    fi
  done
}

# released NOTE prints the release page that has the note's ticket in the section of its type.
# Elsewhere on a page, in the summary for example, does not count.
released() {
  local page
  [[ $(basename "$1") =~ $name ]] || return 1
  for page in "$pages"/[0-9]*.md; do
    if awk -v h="${headings[${BASH_REMATCH[2]}]}" '$0 == h { s = 1; next } s && /^## / { exit } s' "$page" \
      | grep -qF "browse/${BASH_REMATCH[1]})"; then
      echo "$page"
      return
    fi
  done
  return 1
}

check() {
  local file base page
  errors=0

  for file in "$@"; do
    base=$(basename "$file")
    if ! [[ $base =~ $name ]]; then
      error "$base: name it PMM-<ticket>-<type>.md, where type is improvement, fix or known-issue"
      continue
    fi
    grep -q '[^[:space:]]' "$file" || error "$base: write the release note"
    if page=$(released "$file"); then
      error "$base: already published in $(basename "$page"). Delete this note."
    fi
  done

  [ "$errors" -eq 0 ] || exit 1
  echo "Release notes are fine."
}

# render NOTE TYPE KEY: improvements and fixes become a bullet; a known issue's first line is its title.
render() {
  local link="[$3]($jira/$3)"
  if [ "$2" = known-issue ]; then
    printf '### %s (%s)\n' "$(head -n 1 "$1")" "$link"
    tail -n +2 "$1"
  else
    awk -v lead="- $link: " 'NR == 1 { print lead $0; next } /^$/ { print; next } { print "    " $0 }' "$1"
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
  local version=$1 page=$pages/$1.md file heading entry

  if [ ! -f "$page" ]; then
    sed "s/{{VERSION}}/$version/g" "$doc/resources/release-notes-template.md" > "$page"
    sed -i "/^# Release notes$/a - [Percona Monitoring and Management $version]($version.md)" "$pages/index.md"
    sed -i "/^ *- Release notes index: release-notes\/index.md$/a\\      - \"PMM $version\": release-notes/$version.md" \
      "$doc/mkdocs-base.yml"
    echo "Created $(basename "$page") from the template"
  fi

  for file in $(notes "${NOTES_DIR:?set NOTES_DIR to unreleased/ in a checkout of pmm-$version}"); do
    [[ $(basename "$file") =~ $name ]]
    heading=${headings[${BASH_REMATCH[2]}]}
    if grep -qxF "$heading" "$page"; then
      entry=$(mktemp)
      render "$file" "${BASH_REMATCH[2]}" "${BASH_REMATCH[1]}" > "$entry"
      insert "$page" "$heading" "$entry"
      rm -f "$entry"
      echo "$(basename "$file"): added"
    else
      echo "::warning::$(basename "$file"): $(basename "$page") has no '$heading' section, so the note was deleted without being added"
    fi
    rm "$file"
  done

  for file in $(notes "$unreleased"); do
    if released "$file" > /dev/null; then
      rm "$file"
      echo "$(basename "$file"): removed from unreleased/"
    fi
  done
}

case ${1:-} in
  check | build)
    "$@"
    ;;
  *)
    echo "Usage: $0 check [FILE...] | build VERSION" >&2
    exit 2
    ;;
esac
