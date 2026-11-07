#!/usr/bin/env bash
# Captures MySQL EXPLAIN FORMAT=JSON plans for every <case>.sql here.
# Usage: capture.sh <image> <flavor> <version>   (flavor: mysql|ps)
# Needs Docker, so run it on a throwaway VM, never on the shared runner host.
# Writes <flavor>-<version>-<case>.json (and <flavor>-8.4-v2-<case>.json on 8.4)
# into the current directory, with "warnings" in pmm-agent's shape.
set -euo pipefail

if [ $# -ne 3 ]; then
  echo "usage: $0 <image> <flavor> <version>" >&2
  exit 2
fi
image=$1
flavor=$2
version=$3
dir=$(cd "$(dirname "$0")" && pwd)
out=${OUT_DIR:-$PWD}
name="explain-fixture-$flavor-$version-$$"
# Throwaway container, fixed password.
pw=fixture-root-pw

# Cases without WITH support on 5.7.
unsupported_57="cte cte-shadows-table"

cleanup() { docker rm -f "$name" >/dev/null 2>&1 || true; }
trap cleanup EXIT

docker run -d --name "$name" -e MYSQL_ROOT_PASSWORD="$pw" "$image" >/dev/null
for _ in $(seq 1 90); do
  if docker exec -e MYSQL_PWD="$pw" "$name" mysql -uroot -e 'SELECT 1' >/dev/null 2>&1; then
    ready=1
    # The entrypoint restarts the server once; require two consecutive successes.
    sleep 3
    docker exec -e MYSQL_PWD="$pw" "$name" mysql -uroot -e 'SELECT 1' >/dev/null 2>&1 && break
  fi
  ready=0
  sleep 2
done
[ "${ready:-0}" = 1 ] || { echo "server did not become ready" >&2; exit 1; }

mysql_run() { docker exec -i -e MYSQL_PWD="$pw" "$name" mysql -uroot -N -B -r "$@"; }

mysql_run < "$dir/schema.sql" >/dev/null
server_version=$(mysql_run -e 'SELECT VERSION()')
echo "server: $server_version"

# Writes one fixture: $1 case, $2 output file, $3 optional pre-statement.
capture_case() {
  local case=$1 target=$2 pre=${3:-} stmt raw
  stmt=$(sed -e 's/;[[:space:]]*$//' "$dir/$case.sql")
  raw=$(mktemp)
  {
    echo "USE shop;"
    [ -n "$pre" ] && echo "$pre;"
    echo "EXPLAIN FORMAT=JSON $stmt;"
    echo "SHOW WARNINGS;"
  } | mysql_run > "$raw"
  python3 - "$raw" "$target" <<'PY'
import json, sys
raw = open(sys.argv[1]).read()
# SHOW WARNINGS must follow EXPLAIN directly (any other statement clears
# them), so the plan and the warning rows share one stream: split after the plan.
m, end = json.JSONDecoder().raw_decode(raw)
warn = raw[end:]
warnings = []
# SHOW WARNINGS rows are Level<TAB>Code<TAB>Message; messages hold no tabs.
for line in warn.splitlines():
    parts = line.split("\t", 2)
    if len(parts) == 3:
        warnings.append({"Level": parts[0], "Code": int(parts[1]), "Message": parts[2]})
m["warnings"] = warnings
with open(sys.argv[2], "w") as f:
    json.dump(m, f, indent=2)
    f.write("\n")
PY
  rm -f "$raw"
  echo "captured $target"
}

skipped=""
for f in "$dir"/*.sql; do
  case=$(basename "$f" .sql)
  [ "$case" = schema ] && continue
  # CASES="a b" limits the run to those cases.
  [ -z "${CASES:-}" ] || [[ " $CASES " == *" $case "* ]] || continue
  if [ "$version" = 5.7 ] && [[ " $unsupported_57 " == *" $case "* ]]; then
    skipped="$skipped $case"
    continue
  fi
  capture_case "$case" "$out/$flavor-$version-$case.json"
  if [ "$version" = 8.4 ]; then
    capture_case "$case" "$out/$flavor-8.4-v2-$case.json" "SET explain_json_format_version=2"
  fi
done
[ -z "$skipped" ] || echo "skipped (no WITH support on 5.7):$skipped"
