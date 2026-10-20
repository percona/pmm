#!/bin/bash
# Adds the release notes of one version to its page, creating the page from the template first.
# Notes already on the page are skipped, so a re-run only adds new ones and keeps the doc team's edits.
# Usage: rn-compile.sh VERSION NOTES_DIR [DOCUMENTATION_DIR]

# shellcheck source-path=SCRIPTDIR
set -o errexit -o nounset -o pipefail
shopt -s nullglob

here=$(dirname "$0")
. "$here/rn-common.sh"

version=$1
notes=$2
doc=${3:-$here/../..}
page=$doc/docs/release-notes/$version.md

declare -A headings=(
  [highlight]="## ✨ Release highlights"
  [security]="## 🔒 Security updates"
  [improvement]="## 📈 Improvements"
  [fix]="## ✅ Fixed issues"
  [known-issue]="## 🔶 Known issues"
)

scaffold() {
  sed "s/{{VERSION}}/$version/g" "$doc/resources/release-notes-template.md" > "$page"
  sed -i "/^# Release notes$/a - [Percona Monitoring and Management $version]($version.md)" \
    "$doc/docs/release-notes/index.md"
  sed -i "/^ *- Release notes index: release-notes\/index.md$/a\\      - \"PMM $version\": release-notes/$version.md" \
    "$doc/mkdocs-base.yml"
  echo "Created $(basename "$page") from the template"
}

render() {
  local file=$1 type=$2 link="[$3]($RN_JIRA/$3)"
  case $type in
    improvement | fix)
      rn_body "$file" | awk -v lead="- $link: " 'NR == 1 { print lead $0; next } /^$/ { print; next } { print "    " $0 }'
      ;;
    known-issue)
      printf '### %s (%s)\n\n' "$(rn_field "$file" title)" "$link"
      rn_body "$file"
      ;;
    *)
      printf '### %s\n\n' "$(rn_field "$file" title)"
      rn_body "$file"
      ;;
  esac
}

# insert HEADING ENTRY_FILE appends the entry to the end of that section and drops the section's
# "to be added" placeholder. A section the doc team removed comes back before the upgrade links.
insert() {
  local heading=$1 entry=$2 tmp
  tmp=$(mktemp)
  if ! grep -qxF "$heading" "$page"; then
    awk -v h="$heading" '
      /^## .*Ready to upgrade/ && !done { print h; print ""; done = 1 }
      { print }
      END { if (!done) { print ""; print h } }' "$page" > "$tmp"
    mv "$tmp" "$page"
  fi
  awk -v h="$heading" -v entry="$entry" '
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
    END { if (section) emit() }' "$page" > "$tmp"
  mv "$tmp" "$page"
}

[ -f "$page" ] || scaffold

for file in $(printf '%s\n' "$notes"/PMM-*.md | sort -V); do
  [ "$(rn_target_of "$file")" = "$version" ] || continue
  key=$(basename "$file" .md)
  if rn_on_page "$page" "$key"; then
    continue
  fi
  type=$(rn_field "$file" type)
  if [ -z "${headings[$type]:-}" ]; then
    echo "::error::$key: unknown type '$type'"
    exit 1
  fi
  entry=$(mktemp)
  render "$file" "$type" "$key" > "$entry"
  insert "${headings[$type]}" "$entry"
  rm -f "$entry"
  echo "$key: added as $type"
done
