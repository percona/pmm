#!/usr/bin/env bash
#
# One-command PMM + HolmesGPT (ADRE) install. Runs openssl + the ADRE provisioning API from the host
# (no throwaway init containers), and uses docker compose only for the two real services.
#
#   cp .env.example .env      # set PMM_IMAGE, HOLMES_IMAGE, MODEL_NAME, LITELLM_MODEL, LLM_API_KEY
#   ./install.sh
#
# Re-runnable: skips cert generation if present, and the provisioning calls are idempotent.
set -euo pipefail
cd "$(dirname "$0")"

[ -f .env ] || { echo "ERROR: no .env — copy .env.example to .env and fill it in first."; exit 1; }
set -a; . ./.env; set +a

: "${PMM_IMAGE:?set PMM_IMAGE to an ADRE-capable PMM Server image (tibi-holmes build)}"
: "${HOLMES_IMAGE:?set HOLMES_IMAGE (Holmes >= 0.36.0 for native HTTPS)}"
: "${MODEL_NAME:?set MODEL_NAME, e.g. gpt-4.1 (no slashes)}"
: "${LITELLM_MODEL:?set LITELLM_MODEL, e.g. openai/gpt-4.1}"
: "${LLM_API_KEY:?set LLM_API_KEY (your provider key)}"

CONFIG_DIR="${HOLMES_CONFIG_DIR:-./holmes-config}"
PMM_ADMIN_USER="${PMM_ADMIN_USER:-admin}"
PMM_ADMIN_PASSWORD="${PMM_ADMIN_PASSWORD:-admin}"
HOLMES_URL="${HOLMES_URL:-https://holmesgpt:5050}"          # how PMM reaches Holmes (docker network name)
PMM_CALLBACK_URL="${PMM_CALLBACK_URL:-https://pmm-server:8443}"  # how Holmes reaches PMM (docker network name)
PMM_LOCAL="${PMM_LOCAL:-https://localhost}"                 # how THIS script reaches PMM (published :443)

# Use sudo for docker only if the current user can't talk to the daemon (override with DOCKER=...).
if [ -z "${DOCKER:-}" ]; then
  if docker info >/dev/null 2>&1; then DOCKER="docker"; else DOCKER="sudo docker"; fi
fi

echo "==> 1/4  TLS cert  ($CONFIG_DIR/certs/tls.{crt,key})"
mkdir -p "$CONFIG_DIR/certs"
# PMM renders .env/config.yaml/model_list.yaml here as uid 1000 (pmm) — make the dir writable no
# matter who ran install.sh. The rendered secret files are still written 0600 by PMM.
chmod 0777 "$CONFIG_DIR"
if [ -f "$CONFIG_DIR/certs/tls.crt" ]; then
  echo "         exists, keeping it"
else
  openssl req -x509 -newkey rsa:2048 -nodes -days 825 \
    -keyout "$CONFIG_DIR/certs/tls.key" -out "$CONFIG_DIR/certs/tls.crt" \
    -subj "/CN=holmesgpt" -addext "subjectAltName=DNS:holmesgpt,DNS:localhost"
  # Readable by the Holmes container user (self-signed dev cert).
  chmod 0644 "$CONFIG_DIR/certs/tls.crt" "$CONFIG_DIR/certs/tls.key"
  echo "         generated self-signed cert"
fi

echo "==> 2/4  start PMM"
$DOCKER compose up -d pmm-server

echo "==> 3/4  provision ADRE via the PMM API"
AUTH=(-u "$PMM_ADMIN_USER:$PMM_ADMIN_PASSWORD" -fksS)
JSON=(-H "Content-Type: application/json")
# A cold PMM FB start (fresh /srv init) can take several minutes — wait up to ~10m.
printf '         waiting for PMM ADRE API (can take a few minutes on first start)'
for i in $(seq 1 120); do
  if curl "${AUTH[@]}" "$PMM_LOCAL/v1/adre/settings" >/dev/null 2>&1; then echo " — up"; break; fi
  if [ "$i" = 120 ]; then
    echo; echo "ERROR: PMM ADRE API never returned 2xx — check PMM_ADMIN_PASSWORD, or PMM_IMAGE is not an ADRE build."; exit 1
  fi
  printf '.'; sleep 5
done

curl "${AUTH[@]}" "${JSON[@]}" -X PUT "$PMM_LOCAL/v1/adre/deployment/models" \
  -d "{\"models\":[{\"name\":\"$MODEL_NAME\",\"litellm_model\":\"$LITELLM_MODEL\",\"api_key\":\"$LLM_API_KEY\"}]}" >/dev/null
echo "         + model '$MODEL_NAME' ($LITELLM_MODEL)"

curl "${AUTH[@]}" "${JSON[@]}" -X POST "$PMM_LOCAL/v1/adre/settings" \
  -d "{\"enabled\":true,\"url\":\"$HOLMES_URL\",\"tls_skip_verify\":true,\"chat_model\":\"$MODEL_NAME\",\"investigation_model\":\"$MODEL_NAME\"}" >/dev/null
echo "         + ADRE enabled, url=$HOLMES_URL (skip-verify for self-signed cert)"

curl "${AUTH[@]}" "${JSON[@]}" -X PUT "$PMM_LOCAL/v1/adre/deployment/provisioning" \
  -d "{\"pmm_url\":\"$PMM_CALLBACK_URL\"}" >/dev/null
echo "         + callback url=$PMM_CALLBACK_URL"

curl "${AUTH[@]}" -X POST "$PMM_LOCAL/v1/adre/deployment/provision" -d '{}' >/dev/null
echo "         + secrets provisioned (PMM_API_TOKEN + HOLMES_API_KEY, .env rendered)"

curl "${AUTH[@]}" -X POST "$PMM_LOCAL/v1/adre/deployment/apply" -d '{}' >/dev/null
echo "         + config applied (config.yaml / model_list.yaml / skills rendered)"

echo "==> 4/4  start HolmesGPT (HTTPS)"
$DOCKER compose up -d holmesgpt

echo "==> done."
echo "    PMM UI:        https://<this-host>"
printf '    Holmes healthz: '
curl -sk --max-time 8 https://127.0.0.1:5050/healthz || echo "(still starting — check: $DOCKER compose logs -f holmesgpt)"
echo
