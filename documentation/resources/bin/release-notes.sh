#!/bin/bash
# Release notes waiting in documentation/docs/release-notes/unreleased/; its README.md has the format.
#
#   release-notes.sh check BASE_BRANCH FILE...  checks the notes a PR adds or changes
#   release-notes.sh build VERSION [NOTES_DIR]  adds the VERSION notes to VERSION.md, creating it from the
#                                               template, and deletes from unreleased/ the notes it now holds
#   release-notes.sh gate VERSION [NOTES_DIR]   fails while a VERSION note is missing from VERSION.md
#                                               or still in unreleased/
#
# NOTES_DIR defaults to unreleased/ on origin/pmm-VERSION. DOC_DIR overrides the documentation/ directory.

set -o errexit -o nounset -o pipefail
shopt -s nullglob

doc=${DOC_DIR:-$(dirname "$0")/../..}
unreleased=$doc/docs/release-notes/unreleased
types="highlight security improvement fix known-issue"
jira=https://perconadev.atlassian.net/browse
declare -A headings=(
  [highlight]="## ✨ Release highlights"
  [security]="## 🔒 Security updates"
  [improvement]="## 📈 Improvements"
  [fix]="## ✅ Fixed issues"
  [known-issue]="## 🔶 Known issues"
)

die() {
  echo "::error::$*"
  exit 1
}

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

branch_notes() {
  local dir
  dir=$(mktemp -d)
  if git cat-file -e "origin/pmm-$1:documentation/docs/release-notes/unreleased" 2> /dev/null; then
    git archive "origin/pmm-$1" documentation/docs/release-notes/unreleased | tar -x -C "$dir" --strip-components=4
  fi
  echo "$dir"
}

# Versions with a pmm-X.Y.Z branch. RN_RELEASE_BRANCHES replaces the lookup in tests.
release_versions() {
  if [ -n "${RN_RELEASE_BRANCHES+x}" ]; then
    tr ' ' '\n' <<< "$RN_RELEASE_BRANCHES"
  else
    git ls-remote --heads origin 'pmm-*' | sed -n 's#.*refs/heads/pmm-\([0-9]*\.[0-9]*\.[0-9]*\)$#\1#p'
  fi
}

check() {
  local base=$1 releases latest errors=0 file name target type before
  shift
  releases=$(release_versions | sort -V)
  latest=$(tail -n 1 <<< "$releases")

  fail() {
    echo "::error file=$file::$(basename "$file"): $*"
    errors=$((errors + 1))
  }

  for file in "$@"; do
    name=$(basename "$file")
    if [ "$name" = README.md ]; then
      continue
    fi
    if ! [[ $name =~ ^PMM-[0-9]+\.md$ ]]; then
      fail "name the file after its ticket, for example PMM-12345.md"
      continue
    fi
    if ! has_front_matter "$file"; then
      fail "start the file with front matter: target and type between two --- lines"
      continue
    fi

    target=$(field "$file" target)
    type=$(field "$file" type)
    before=$errors

    if [ -z "$target" ]; then
      fail "target is required: the version this change ships in, for example target: 3.11.0"
    elif ! [[ $target =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      fail "target must be a version like 3.11.0, not '$target'"
    elif [[ $base =~ ^pmm-([0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
      [ "$target" = "${BASH_REMATCH[1]}" ] || fail "this PR goes to $base, so target must be ${BASH_REMATCH[1]}"
    elif grep -qxF "$target" <<< "$releases"; then
      fail "pmm-$target is already branched. Open this PR against pmm-$target, or target a later version"
    elif [ -n "$latest" ] && [ "$(printf '%s\n%s\n' "$target" "$latest" | sort -V | tail -n 1)" = "$latest" ]; then
      fail "$target is older than the newest release branch, pmm-$latest. Target a later version"
    fi

    case " $types " in
      *" $type "*) ;;
      *) fail "type must be one of: $types" ;;
    esac
    case $type in
      highlight | security | known-issue)
        [ -n "$(field "$file" title)" ] || fail "a $type needs a title" ;;
    esac
    [ -n "$(body "$file")" ] || fail "write the release note below the front matter"

    if [ "$errors" -eq "$before" ]; then
      echo "$name: $type for $target"
    fi
  done

  [ "$errors" -eq 0 ]
}

render() {
  local file=$1 type=$2 link="[$3]($jira/$3)"
  case $type in
    improvement | fix)
      body "$file" | awk -v lead="- $link: " 'NR == 1 { print lead $0; next } /^$/ { print; next } { print "    " $0 }'
      ;;
    known-issue)
      printf '### %s (%s)\n\n' "$(field "$file" title)" "$link"
      body "$file"
      ;;
    *)
      printf '### %s\n\n' "$(field "$file" title)"
      body "$file"
      ;;
  esac
}

# insert PAGE HEADING ENTRY_FILE appends the entry to the end of the section
# and drops the section's "to be added" placeholder.
insert() {
  local tmp
  tmp=$(mktemp)
  awk -v h="$2" -v entry="$3" '
    function emit(  line) {
      while (n > 0 && buf[n] ~ /^[[:space:]]*$/) n--
      for (i = 1; i <= n; i++) print buf[i]
      print ""
      while ((getline line < entry) > 0) print line
      print ""
      n = 0
    }
    $0 == h { section = 1; print; next }
    section && /^## / { emit(); section = 0; print; next }
    section && /^!!! note/ { placeholder = 1; next }
    section && placeholder && (/^    / || /^[[:space:]]*$/) { next }
    section { placeholder = 0; buf[++n] = $0; next }
    { print }
    END { if (section) emit() }' "$1" > "$tmp"
  cat "$tmp" > "$1"
  rm -f "$tmp"
}

build() {
  local version=$1 notes page file key type entry
  notes=${2:-$(branch_notes "$version")}
  page=$doc/docs/release-notes/$version.md

  if [ ! -f "$page" ]; then
    sed "s/{{VERSION}}/$version/g" "$doc/resources/release-notes-template.md" > "$page"
    sed -i "/^# Release notes$/a - [Percona Monitoring and Management $version]($version.md)" \
      "$doc/docs/release-notes/index.md"
    sed -i "/^ *- Release notes index: release-notes\/index.md$/a\\      - \"PMM $version\": release-notes/$version.md" \
      "$doc/mkdocs-base.yml"
    echo "Created $(basename "$page") from the template"
  fi

  for file in $(notes_for "$version" "$notes"); do
    key=$(basename "$file" .md)
    if on_page "$page" "$key"; then
      continue
    fi
    type=$(field "$file" type)
    [ -n "${headings[$type]:-}" ] || die "$key: unknown type '$type'"
    grep -qxF "${headings[$type]}" "$page" ||
      die "$key: $(basename "$page") has no '${headings[$type]}' section. Add it back, then run this again."
    entry=$(mktemp)
    render "$file" "$type" "$key" > "$entry"
    insert "$page" "${headings[$type]}" "$entry"
    rm -f "$entry"
    echo "$key: added as $type"
  done

  # A note left in unreleased/ that is not on the page stays, and fails the gate.
  for file in $(notes_for "$version" "$unreleased"); do
    key=$(basename "$file" .md)
    if on_page "$page" "$key"; then
      rm "$file"
      echo "$key: removed from unreleased/"
    fi
  done
}

gate() {
  local version=$1 notes page file key errors=0
  notes=${2:-$(branch_notes "$version")}
  page=$doc/docs/release-notes/$version.md
  [ -f "$page" ] || die "release-notes/$version.md does not exist. Run the Release notes workflow for $version."

  for file in $(notes_for "$version" "$notes"); do
    key=$(basename "$file" .md)
    if ! on_page "$page" "$key"; then
      echo "::error::$key is on pmm-$version but not on the page. Run the Release notes workflow for $version."
      errors=$((errors + 1))
    fi
  done

  for file in $(notes_for "$version" "$unreleased"); do
    key=$(basename "$file" .md)
    if on_page "$page" "$key"; then
      echo "::error::$key is on the page but still in unreleased/. Delete unreleased/$key.md."
    else
      echo "::error::$key targets $version but is not on pmm-$version. Cherry-pick its change to pmm-$version, or change its target."
    fi
    errors=$((errors + 1))
  done

  [ "$errors" -eq 0 ] || exit 1
  echo "Every $version release note is on the page."
}

case ${1:-} in
  check | build | gate)
    "$@"
    ;;
  *)
    echo "Usage: $0 check BASE_BRANCH FILE... | build VERSION [NOTES_DIR] | gate VERSION [NOTES_DIR]" >&2
    exit 2
    ;;
esac
