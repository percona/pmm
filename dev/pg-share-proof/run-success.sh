#!/usr/bin/env bash
# Turnkey driver for the SUCCESS path:
# 1. boot pmm-server
# 2. wait for its built-in PostgreSQL
# 3. apply the runtime edit (listen_addresses='*' + pg_hba host rule)
# 4. restart PG
# 5. run the client probe from a SEPARATE container on the bridge.
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
F="$DIR/pg-share-success.compose.yml"

echo "[1/4] Starting pmm-server (this pulls the image on first run)..."
docker compose -f "$F" up -d pmm-server

echo "[2/4] Waiting for built-in PostgreSQL to be ready..."
CID="$(docker compose -f "$F" ps -q pmm-server)"
until [ "$(docker inspect -f '{{.State.Health.Status}}' "$CID")" = "healthy" ]; do
  sleep 3
  echo "  ...still starting"
done
echo "  pmm-server is healthy."

echo "[3/4] Applying runtime edit + restarting PG (inside pmm-server)..."
docker compose -f "$F" exec -T pmm-server bash -s <<'REMOTE'
set -e
CONF=/srv/postgres14/postgresql.conf
HBA=/srv/postgres14/pg_hba.conf
grep -Eq "^listen_addresses = '\*'" "$CONF" || printf "\nlisten_addresses = '*'\n" >> "$CONF"
# scram (NOT trust) so the on-boot pg_hba migration leaves it intact.
# 0.0.0.0/0 is for the demo only — scope to the bridge subnet in real use.
grep -q "0.0.0.0/0" "$HBA" || printf "host    pmm-managed    pmm-managed    0.0.0.0/0    scram-sha-256\n" >> "$HBA"
supervisorctl restart postgresql
sleep 4
echo -n "  effective listen_addresses on server: "
/usr/bin/psql -U postgres -h /run/postgresql -d postgres -tAc "show listen_addresses;"
REMOTE

echo "[4/4] Running client probe from a separate container on the shared bridge..."
docker compose -f "$F" --profile probe run --rm client

echo
echo "Done. Cleanup with: docker compose -f \"$F\" --profile probe down -v"
