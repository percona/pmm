#!/usr/bin/env bash
# Captures pmm-agent's MongoDB explain output for every <case>.example.json here, a QAN query example.
# Usage: capture.sh <image> <version> <capture binary>
# Build the binary with Go anywhere: go build -o capture ./managed/services/mcp/testdata/explain/mongodb/capture
# Needs Docker, so run it on a throwaway VM, never on the shared runner host.
# Writes psmdb-<version>-<case>.json into the current directory.
set -euo pipefail

if [ $# -ne 3 ]; then
  echo "usage: $0 <image> <version> <capture binary>" >&2
  exit 2
fi
image=$1
version=$2
bin=$(realpath "$3")
dir=$(cd "$(dirname "$0")" && pwd)
out=${OUT_DIR:-$PWD}
name="explain-fixture-psmdb-$version-$$"

cleanup() { docker rm -f "$name" >/dev/null 2>&1 || true; }
trap cleanup EXIT

docker run -d --name "$name" -p 127.0.0.1::27017 "$image" >/dev/null
ready=0
for _ in $(seq 1 60); do
  if docker exec "$name" mongosh --quiet --eval 'db.runCommand({ping: 1}).ok' >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 2
done
[ "$ready" = 1 ] || { echo "server did not become ready" >&2; exit 1; }
port=$(docker port "$name" 27017/tcp | head -1 | sed 's/.*://')

docker exec -i "$name" mongosh --quiet < "$dir/setup.js" >/dev/null
echo "server: $(docker exec "$name" mongosh --quiet --eval 'db.version()')"

for f in "$dir"/*.example.json; do
  case=$(basename "$f" .example.json)
  target="$out/psmdb-$version-$case.json"
  "$bin" "mongodb://127.0.0.1:$port/shop" "$f" | python3 -c 'import json, sys; json.dump(json.load(sys.stdin), sys.stdout, indent=2); print()' > "$target"
  echo "captured $target"
done
