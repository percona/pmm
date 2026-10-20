# shellcheck shell=bash disable=SC2034
# Shared helpers for the release-note scripts. Source it, don't run it.

RN_TYPES="highlight security improvement fix known-issue"
RN_JIRA="https://perconadev.atlassian.net/browse"

rn_has_front_matter() {
  [ "$(head -n 1 "$1")" = "---" ] && [ "$(grep -c '^---$' "$1")" -ge 2 ]
}

# rn_field FILE KEY prints the front-matter value, or nothing when the key is absent.
rn_field() {
  yq --front-matter=extract ".$2 // \"\"" "$1"
}

# rn_body FILE prints the text after the front matter, without leading or trailing blank lines.
rn_body() {
  awk '
    f >= 2 { line[++n] = $0 }
    /^---$/ && f < 2 { f++ }
    END {
      s = 1; while (s <= n && line[s] ~ /^[[:space:]]*$/) s++
      e = n; while (e >= s && line[e] ~ /^[[:space:]]*$/) e--
      for (i = s; i <= e; i++) print line[i]
    }' "$1"
}

# rn_target_of FILE prints its target, or nothing for a file that is not a valid note.
rn_target_of() {
  rn_has_front_matter "$1" && rn_field "$1" target
}

# Versions that have a pmm-X.Y.Z release branch. RN_RELEASE_BRANCHES overrides the lookup in tests.
rn_release_versions() {
  if [ -n "${RN_RELEASE_BRANCHES+x}" ]; then
    tr ' ' '\n' <<< "$RN_RELEASE_BRANCHES"
    return
  fi
  git ls-remote --heads origin 'pmm-*' | sed -n 's#.*refs/heads/pmm-\([0-9]*\.[0-9]*\.[0-9]*\)$#\1#p'
}

# rn_version_le A B succeeds when A <= B.
rn_version_le() {
  [ "$(printf '%s\n%s\n' "$1" "$2" | sort -V | tail -n 1)" = "$2" ]
}

# rn_on_page PAGE KEY succeeds when the page already has an entry linking the ticket.
rn_on_page() {
  grep -qF "browse/$2)" "$1"
}
