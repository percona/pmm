#!/usr/bin/env bash
#
# One-command PMM + HolmesGPT (ADRE) install on a single Docker host.
#
#   mkdir pmm-adre && cd pmm-adre
#   curl -fsSL --remote-name-all https://raw.githubusercontent.com/percona/pmm/tibi-holmes/dev/adre/deploy/{compose.yaml,install.sh}
#   bash install.sh
#
# Nothing is required up front. Optional settings come from the environment or ./.env (see .env.example);
# the environment wins. Set MODEL_NAME + LITELLM_MODEL + LLM_API_KEY to add a model now, or add it later in
# PMM → AI Assistant Deployment → Models & Keys. Re-runnable: the cert, admin password and provisioning are kept,
# and models are upserted, never removed.
set -euo pipefail
cd "$(dirname "$0")"

if [ -f .env ]; then
  while IFS='=' read -r k v; do
    [[ "$k" =~ ^[A-Z_][A-Z0-9_]*$ ]] || continue
    [ -n "${!k:-}" ] || export "$k=$v"
  done < .env
fi

export PMM_IMAGE="${PMM_IMAGE:-perconalab/pmm-server-fb:PR-4259-16e7b4f}"
export HOLMES_IMAGE="${HOLMES_IMAGE:-robustadev/holmes:0.45.0}"
export HOLMES_CONFIG_DIR="${HOLMES_CONFIG_DIR:-./holmes-config}"
PMM_ADMIN_PASSWORD="${PMM_ADMIN_PASSWORD:-$(openssl rand -base64 18 | tr -d '/+=')}"
export PMM_ADMIN_PASSWORD
HOLMES_URL="${HOLMES_URL:-https://holmesgpt:5050}"  # how PMM reaches Holmes (docker network name)
PMM_LOCAL="${PMM_LOCAL:-https://localhost}"         # how THIS script reaches PMM (published :443)

MODEL_NAME="${MODEL_NAME:-}" LITELLM_MODEL="${LITELLM_MODEL:-}" LLM_API_KEY="${LLM_API_KEY:-}"
if [ -n "$MODEL_NAME$LITELLM_MODEL$LLM_API_KEY" ] && { [ -z "$MODEL_NAME" ] || [ -z "$LITELLM_MODEL" ] || [ -z "$LLM_API_KEY" ]; }; then
  echo "ERROR: set all of MODEL_NAME, LITELLM_MODEL and LLM_API_KEY, or none (then add the model in the PMM UI)."; exit 1
fi
# Optional cheaper model for Fast chat; MODEL_NAME then serves Investigation mode only.
FAST_MODEL_NAME="${FAST_MODEL_NAME:-}" FAST_LITELLM_MODEL="${FAST_LITELLM_MODEL:-}"
FAST_LLM_API_KEY="${FAST_LLM_API_KEY:-$LLM_API_KEY}"
if [ -n "$FAST_MODEL_NAME$FAST_LITELLM_MODEL" ] && { [ -z "$MODEL_NAME" ] || [ -z "$FAST_MODEL_NAME" ] || [ -z "$FAST_LITELLM_MODEL" ]; }; then
  echo "ERROR: FAST_MODEL_NAME and FAST_LITELLM_MODEL go together, and need MODEL_NAME/LITELLM_MODEL/LLM_API_KEY."; exit 1
fi

for tool in curl openssl; do
  command -v "$tool" >/dev/null || { echo "ERROR: $tool is required."; exit 1; }
done
# Use sudo for docker only if the current user can't talk to the daemon (override with DOCKER=...).
if [ -z "${DOCKER:-}" ]; then
  if docker info >/dev/null 2>&1; then DOCKER="docker"; else DOCKER="sudo docker"; fi
fi
$DOCKER compose version >/dev/null 2>&1 || { echo "ERROR: the Docker Compose plugin ('docker compose') is required."; exit 1; }

# Persist what docker compose and later re-runs need; keep anything the user already put in .env.
umask 077
touch .env
keep() { grep -q "^$1=" .env || printf '%s=%s\n' "$1" "$2" >> .env; }
keep PMM_IMAGE "$PMM_IMAGE"
keep HOLMES_IMAGE "$HOLMES_IMAGE"
keep HOLMES_CONFIG_DIR "$HOLMES_CONFIG_DIR"
keep PMM_ADMIN_PASSWORD "$PMM_ADMIN_PASSWORD"
umask 022

json_str() { local s=${1//\\/\\\\}; printf '"%s"' "${s//\"/\\\"}"; }
# api METHOD PATH [JSON] — the body goes through stdin so secrets never show up in `ps`.
api() {
  local out body=${3-}
  [ -n "$body" ] || body='{}'
  if ! out=$(printf '%s' "$body" | curl -ksS --fail-with-body -u "admin:$PMM_ADMIN_PASSWORD" \
    -H "Content-Type: application/json" -X "$1" --data-binary @- "$PMM_LOCAL$2"); then
    echo "ERROR: $1 $2 failed: $out"; exit 1
  fi
  printf '%s' "$out"
}

echo "==> 1/5  TLS cert for HolmesGPT  ($HOLMES_CONFIG_DIR/certs)"
mkdir -p "$HOLMES_CONFIG_DIR/certs"
# PMM renders .env/config.yaml/model_list.yaml here as uid 1000 (pmm); the secret files themselves are 0600.
chmod 0777 "$HOLMES_CONFIG_DIR"
if [ -f "$HOLMES_CONFIG_DIR/certs/tls.crt" ]; then
  echo "         exists, keeping it"
else
  openssl req -x509 -newkey rsa:2048 -nodes -days 825 \
    -keyout "$HOLMES_CONFIG_DIR/certs/tls.key" -out "$HOLMES_CONFIG_DIR/certs/tls.crt" \
    -subj "/CN=holmesgpt" -addext "subjectAltName=DNS:holmesgpt,DNS:localhost" 2>/dev/null
  # Readable by the Holmes container user (self-signed dev cert).
  chmod 0644 "$HOLMES_CONFIG_DIR/certs/tls.crt" "$HOLMES_CONFIG_DIR/certs/tls.key"
  echo "         generated self-signed cert"
fi

echo "==> 2/5  start PMM Server ($PMM_IMAGE) and the image renderer"
$DOCKER compose up -d --quiet-pull pmm-server grafana-image-renderer
printf '         waiting for PMM (a first start can take a few minutes)'
for i in $(seq 1 120); do
  [ "$(curl -ks --max-time 5 "$PMM_LOCAL/v1/server/readyz")" = "{}" ] && { echo " — ready"; break; }
  [ "$i" = 120 ] && { echo; echo "ERROR: PMM never became ready — check: $DOCKER compose logs pmm-server"; exit 1; }
  printf '.'; sleep 5
done
# GF_SECURITY_ADMIN_PASSWORD only applies when the PMM data volume is first created; reset it on an existing one.
if ! curl -ksf -o /dev/null -u "admin:$PMM_ADMIN_PASSWORD" "$PMM_LOCAL/v1/server/version"; then
  $DOCKER exec pmm-server change-admin-password "$PMM_ADMIN_PASSWORD" >/dev/null
  echo "         admin password set"
fi
curl -ksf -o /dev/null -u "admin:$PMM_ADMIN_PASSWORD" "$PMM_LOCAL/v1/adre/settings" \
  || { echo "ERROR: /v1/adre/settings is unavailable — PMM_IMAGE is not an ADRE build."; exit 1; }

echo "==> 3/5  pick the PMM URL HolmesGPT calls back on"
# It must work from the Holmes container AND from browsers (answers embed panel image links), so prefer this
# host's IP over the Docker-only https://pmm-server:8443; fall back to that when the IP is not reachable.
if [ -z "${PMM_CALLBACK_URL:-}" ]; then
  $DOCKER pull -q "$HOLMES_IMAGE" >/dev/null
  host_ip=$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{for (i = 1; i < NF; i++) if ($i == "src") print $(i + 1)}')
  [ -n "$host_ip" ] || host_ip=$(hostname -I 2>/dev/null | awk '{print $1}')
  PMM_CALLBACK_URL=https://pmm-server:8443
  if [ -n "$host_ip" ] && [ "$($DOCKER run --rm --network pmm --entrypoint curl "$HOLMES_IMAGE" \
      -sk --max-time 10 -o /dev/null -w '%{http_code}' "https://$host_ip/v1/server/readyz")" = 200 ]; then
    PMM_CALLBACK_URL="https://$host_ip"
  else
    echo "         WARNING: https://${host_ip:-<host ip>} is not reachable from containers; panel image links in"
    echo "         answers will not open in a browser. Set PMM_CALLBACK_URL to fix that and re-run."
  fi
fi
keep PMM_CALLBACK_URL "$PMM_CALLBACK_URL"
echo "         $PMM_CALLBACK_URL"

echo "==> 4/5  provision ADRE"
settings="{\"enabled\":true,\"url\":$(json_str "$HOLMES_URL"),\"tls_skip_verify\":true"
[ -n "$MODEL_NAME" ] && settings+=",\"chat_model\":$(json_str "${FAST_MODEL_NAME:-$MODEL_NAME}"),\"investigation_model\":$(json_str "$MODEL_NAME")"
api POST /v1/adre/settings "$settings}" >/dev/null
echo "         + ADRE enabled, Holmes at $HOLMES_URL (self-signed cert, TLS verify skipped)"
if [ -n "$MODEL_NAME" ]; then
  mdls="{\"name\":$(json_str "$MODEL_NAME"),\"litellm_model\":$(json_str "$LITELLM_MODEL"),\"api_key\":$(json_str "$LLM_API_KEY")}"
  [ -n "$FAST_MODEL_NAME" ] && mdls+=",{\"name\":$(json_str "$FAST_MODEL_NAME"),\"litellm_model\":$(json_str "$FAST_LITELLM_MODEL"),\"api_key\":$(json_str "$FAST_LLM_API_KEY")}"
  api PUT /v1/adre/deployment/models "{\"models\":[$mdls]}" >/dev/null
  echo "         + model $MODEL_NAME ($LITELLM_MODEL)${FAST_MODEL_NAME:+, fast chat model $FAST_MODEL_NAME ($FAST_LITELLM_MODEL)}"
fi
api PUT /v1/adre/deployment/provisioning "{\"pmm_url\":$(json_str "$PMM_CALLBACK_URL")}" >/dev/null
api POST /v1/adre/deployment/provision >/dev/null
api POST /v1/adre/deployment/apply >/dev/null
echo "         + PMM_API_TOKEN, HOLMES_API_KEY, config.yaml, model_list.yaml and skills rendered"

echo "==> 5/5  start HolmesGPT ($HOLMES_IMAGE)"
# Recreate so Holmes reads the .env PMM just rendered.
$DOCKER compose up -d --quiet-pull --force-recreate holmesgpt
for i in $(seq 1 40); do
  curl -sk --max-time 3 https://127.0.0.1:5050/healthz >/dev/null 2>&1 && break
  [ "$i" = 40 ] && { echo "ERROR: HolmesGPT did not start — check: $DOCKER compose logs holmesgpt"; exit 1; }
  sleep 3
done
api POST /v1/adre/deployment/restarted >/dev/null
reload=$(api POST /v1/adre/deployment/apply)
case "$reload" in
  *'"reloaded":true'*) echo "         PMM → HolmesGPT OK (authenticated config reload)" ;;
  *) echo "ERROR: PMM could not reach HolmesGPT: $reload"; exit 1 ;;
esac

echo
echo "==> done.  PMM: ${PMM_CALLBACK_URL/pmm-server:8443/<this-host>}   login: admin / $PMM_ADMIN_PASSWORD  (saved in .env)"
if curl -ksf -u "admin:$PMM_ADMIN_PASSWORD" "$PMM_LOCAL/v1/adre/deployment" | grep -q '"models":\[\]'; then
  echo "    Next: add an LLM in PMM → AI Assistant Deployment → Models & Keys, then open AI Assistant."
fi
