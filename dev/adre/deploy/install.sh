#!/usr/bin/env bash
#
# One-command PMM + HolmesGPT (PMM Assistant) install on a single Docker host.
#
#   mkdir pmm-assistant && cd pmm-assistant
#   curl -fsSL --remote-name-all https://raw.githubusercontent.com/percona/pmm/tibi-holmes/dev/adre/deploy/{compose.yaml,install.sh}
#   bash install.sh                     # PMM Server, the image renderer and HolmesGPT
#   bash install.sh --existing-server   # only the renderer and HolmesGPT, next to a PMM Server that already runs
#
# Nothing is required up front for a new install. Optional settings come from the environment or ./.env (see
# .env.example); the environment wins. Set MODEL_NAME + LITELLM_MODEL + LLM_API_KEY to add a model now, or add it later in
# PMM → Configuration → Assistant → Deployment → Models & Keys. Re-runnable: the cert, admin password and provisioning are kept,
# and models are upserted, never removed.
set -euo pipefail
cd "$(dirname "$0")"

usage() {
  cat <<'EOF'
Usage: bash install.sh [--existing-server[=CONTAINER]]

  (no option)                    Start PMM Server, the image renderer and HolmesGPT with docker compose.
  --existing-server[=CONTAINER]  Use a PMM Server container that is already running (default: pmm-server).
                                 Checks it has what HolmesGPT and the renderer need, then starts only those two.
                                 Needs that server's PMM_ADMIN_PASSWORD (environment or ./.env); it is never changed.
EOF
}
for arg in "$@"; do
  case $arg in
    --existing-server) export PMM_EXISTING_SERVER=pmm-server ;;
    --existing-server=?*) export PMM_EXISTING_SERVER=${arg#*=} ;;
    -h | --help) usage; exit 0 ;;
    *) echo "ERROR: unknown option '$arg'."; usage; exit 1 ;;
  esac
done

if [ -f .env ]; then
  while IFS='=' read -r k v; do
    [[ "$k" =~ ^[A-Z_][A-Z0-9_]*$ ]] || continue
    [ -n "${!k:-}" ] || export "$k=$v"
  done < .env
fi

EXISTING="${PMM_EXISTING_SERVER:-}"
PMM_CONTAINER="${EXISTING:-pmm-server}"
if [ -n "$EXISTING" ] && [ -z "${PMM_ADMIN_PASSWORD:-}" ]; then
  echo "ERROR: --existing-server needs that server's admin password: set PMM_ADMIN_PASSWORD in ./.env or the environment."; exit 1
fi
export PMM_IMAGE="${PMM_IMAGE:-perconalab/pmm-server-fb:PR-4259-19150b4}"
export HOLMES_IMAGE="${HOLMES_IMAGE:-robustadev/holmes:0.45.0}"
export HOLMES_CONFIG_DIR="${HOLMES_CONFIG_DIR:-./holmes-config}"
PMM_ADMIN_PASSWORD="${PMM_ADMIN_PASSWORD:-$(openssl rand -base64 18 | tr -d '/+=')}"
export PMM_ADMIN_PASSWORD
HOLMES_URL="${HOLMES_URL:-https://holmesgpt:5050}"  # how PMM reaches Holmes (docker network name)
PMM_LOCAL="${PMM_LOCAL:-}"                          # how THIS script reaches PMM; default https://localhost on its published port
PMM_PORT=443
RENDERER_URL=http://grafana-image-renderer:8081/render

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

wait_ready() {
  printf '         waiting for PMM (a first start can take a few minutes)'
  for i in $(seq 1 120); do
    [ "$(curl -ks --max-time 5 "$PMM_LOCAL/v1/server/readyz")" = "{}" ] && { echo " — ready"; return; }
    [ "$i" = 120 ] && { echo; echo "ERROR: PMM never became ready — check: $1"; exit 1; }
    printf '.'; sleep 5
  done
}

require_assistant() {
  curl -ksf -o /dev/null -u "admin:$PMM_ADMIN_PASSWORD" "$PMM_LOCAL/v1/adre/settings" \
    || { echo "ERROR: /v1/adre/settings is unavailable — $1 is not a build with the PMM Assistant."; exit 1; }
}

# Succeeds, printing where it comes from, when container $1 exists but is not managed from this directory.
managed_elsewhere() {
  local wd
  wd=$($DOCKER inspect -f '{{index .Config.Labels "com.docker.compose.project.working_dir"}}' "$1" 2>/dev/null) || return 1
  [ "$wd" != "$(pwd)" ] || return 1
  printf '%s' "${wd:-outside docker compose}"
}

join_pmm_network() {
  $DOCKER inspect -f '{{range $k, $v := .NetworkSettings.Networks}}{{println $k}}{{end}}' "$1" | grep -x pmm >/dev/null \
    || { $DOCKER network connect pmm "$1"; echo "         + connected '$1' to the 'pmm' network"; }
}

# Docker fixes a container's env, mounts and networking when it is created, so an existing PMM Server is checked
# before anything is changed; what only a recreate can fix is listed with the docker run options that fix it.
check_existing_server() {
  local c=$EXISTING running env cfg_dir mount drop_mount='' callback ip port where image opts added user_env rerun
  local -a recreate=() flags=()
  running=$($DOCKER inspect -f '{{.State.Running}}' "$c" 2>/dev/null) || { echo "ERROR: there is no container named '$c'."; exit 1; }
  [ "$running" = true ] || { echo "ERROR: container '$c' is not running."; exit 1; }
  env=$($DOCKER inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$c")
  getenv() { sed -n "s/^$1=//p" <<<"$env" | tail -n1; }

  if [ "$($DOCKER inspect -f '{{.HostConfig.NetworkMode}}' "$c")" = host ]; then
    recreate+=("it uses host networking, so it cannot join the 'pmm' network HolmesGPT and the renderer are on")
    flags+=(--network pmm -p 443:8443)
  else
    port=$($DOCKER inspect -f '{{with index .NetworkSettings.Ports "8443/tcp"}}{{(index . 0).HostPort}}{{end}}' "$c")
    if [ -n "$port" ]; then
      PMM_PORT=$port
      PMM_LOCAL=${PMM_LOCAL:-https://localhost:$port}
    elif [ -z "$PMM_LOCAL" ]; then
      recreate+=("its HTTPS port 8443 is not published, so this script cannot reach PMM (or set PMM_LOCAL)")
      flags+=(-p 443:8443)
    fi
  fi

  if where=$(managed_elsewhere holmesgpt); then
    echo "ERROR: a holmesgpt container from $where exists; remove it (docker rm -f holmesgpt) or run install.sh"
    echo "       where it was installed."; exit 1
  fi

  if [ -n "$PMM_LOCAL" ]; then
    wait_ready "$DOCKER logs $c"
    case $(curl -ks -o /dev/null -w '%{http_code}' -u "admin:$PMM_ADMIN_PASSWORD" "$PMM_LOCAL/v1/server/version") in
      200) ;;
      401) echo "ERROR: PMM_ADMIN_PASSWORD is not the admin password of '$c' (install.sh never changes it on an existing server)."; exit 1 ;;
      *) echo "ERROR: $PMM_LOCAL does not answer like PMM Server; set PMM_LOCAL to its HTTPS URL."; exit 1 ;;
    esac
    require_assistant "the image of '$c'"
  fi

  cfg_dir=$(getenv ADRE_CONFIG_DIR)
  cfg_dir=${cfg_dir:-/holmes-config}
  mount=$($DOCKER inspect -f '{{range .Mounts}}{{.Type}}|{{.Source}}|{{.Destination}}{{println}}{{end}}' "$c" |
    awk -F'|' -v d="$cfg_dir" '$3 == d')
  if [ "${mount%%|*}" = bind ]; then
    HOLMES_CONFIG_DIR=$(cut -d'|' -f2 <<<"$mount")
  else
    recreate+=("PMM renders the HolmesGPT config into $cfg_dir, which is not a host directory HolmesGPT can mount too")
    flags+=(-v "$(realpath -m "$HOLMES_CONFIG_DIR"):$cfg_dir")
    drop_mount=$cfg_dir
    [ -n "$(getenv ADRE_CONFIG_DIR)" ] || flags+=(-e "ADRE_CONFIG_DIR=$cfg_dir")
  fi

  if [ -n "$(getenv GF_RENDERING_SERVER_URL)" ]; then
    RENDERER_URL=$(getenv GF_RENDERING_SERVER_URL)
  else
    recreate+=("GF_RENDERING_SERVER_URL is not set, so Grafana cannot render panel images for answers")
    flags+=(-e "GF_RENDERING_SERVER_URL=$RENDERER_URL")
  fi
  callback=$(getenv GF_RENDERING_CALLBACK_URL)
  if [ -z "$callback" ]; then
    recreate+=("GF_RENDERING_CALLBACK_URL is not set, so the renderer cannot load panels back from Grafana")
    flags+=(-e "GF_RENDERING_CALLBACK_URL=http://$c:8080/graph/")
  else
    # The renderer calls back over the network, so a port bound to 127.0.0.1 (Grafana's :3000) never answers it.
    ip=$($DOCKER inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}' "$c" | awk '{print $1}')
    if [ -n "$ip" ] && [ "$($DOCKER exec "$c" curl -sk -o /dev/null --max-time 5 -w '%{http_code}' \
        "$(sed -E "s#^([a-zA-Z]+://)[^/:]+#\1$ip#" <<<"$callback")")" = 000 ]; then
      recreate+=("GF_RENDERING_CALLBACK_URL ($callback) does not answer on the container's network address, so the renderer cannot load panels")
      flags+=(-e "GF_RENDERING_CALLBACK_URL=http://$c:8080/graph/")
    fi
  fi

  case $(getenv PMM_ENABLE_UPDATES) in
    false | 0) ;;
    *)
      echo "         WARNING: in-app updates are on; they can replace this build with a stock PMM release, which has no"
      echo "         Assistant. Add -e PMM_ENABLE_UPDATES=false when you next recreate '$c'."
      flags+=(-e PMM_ENABLE_UPDATES=false) ;;
  esac

  [ ${#recreate[@]} -eq 0 ] && return
  image=$($DOCKER inspect -f '{{.Config.Image}}' "$c")
  # One option per line so a config-dir mount being replaced can be dropped; values of -e options are not printed.
  opts=$($DOCKER inspect -f '{{with .HostConfig.RestartPolicy.Name}}{{if ne . "no"}}--restart {{.}}{{println}}{{end}}{{end}}
{{- with .HostConfig.NetworkMode}}{{if and (ne . "default") (ne . "bridge") (ne . "host")}}--network {{.}}{{println}}{{end}}{{end}}
{{- range $p, $b := .HostConfig.PortBindings}}{{range $b}}-p {{if .HostIp}}{{.HostIp}}:{{end}}{{.HostPort}}:{{$p}}{{println}}{{end}}{{end}}
{{- range .Mounts}}{{if eq .Type "volume"}}-v {{.Name}}:{{.Destination}}{{else}}-v {{.Source}}:{{.Destination}}{{end}}{{println}}{{end}}' "$c" |
    { grep -v -e '^$' ${drop_mount:+-e "^-v .*:$drop_mount\$"} || true; } | paste -sd' ')
  # The -e options it was started with (not from the image, not replaced by a flag above) go to an env file, so the
  # new container keeps them and no value is printed.
  added=$(printf '%s\n' "${flags[@]}" | sed -n 's/^\([A-Z_][A-Z0-9_]*\)=.*/\1/p')
  user_env=$(comm -23 <(sed 's/=.*//' <<<"$env" | grep . | sort -u) \
    <({ $DOCKER image inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$image" 2>/dev/null | sed 's/=.*//'; echo "$added"; } |
      grep . | sort -u))
  if [ -n "$user_env" ]; then
    (umask 077; for n in $user_env; do grep "^$n=" <<<"$env" | tail -n1; done > "$c.env")
    opts+=" --env-file $(pwd)/$c.env"
  fi
  rerun="bash install.sh --existing-server"
  [ "$c" = pmm-server ] || rerun+="=$c"
  echo "ERROR: '$c' has to be recreated; a restart is not enough, because Docker fixes a container's env, mounts and"
  echo "       networking when it is created. To fix:"
  printf '         - %s\n' "${recreate[@]}"
  echo "       Recreate it with the same image, data volume and ports, adding: ${flags[*]}"
  [ -z "$user_env" ] || echo "       Its other -e options ($(paste -sd' ' <<<"$user_env")) are saved in ./$c.env (mode 0600)."
  echo "       For example (data in the volume is kept):"
  echo "         $DOCKER stop $c && $DOCKER rm $c"
  echo "         $DOCKER run -d --name $c $opts ${flags[*]} $image"
  echo "       Then run: $rerun"
  echo "       Nothing else was changed."
  exit 2
}

if [ -n "$EXISTING" ]; then
  echo "==> check the running PMM Server '$EXISTING'"
  check_existing_server
  echo "         OK: reachable at $PMM_LOCAL, shares $HOLMES_CONFIG_DIR, renders through $RENDERER_URL"
else
  PMM_LOCAL=${PMM_LOCAL:-https://localhost}
  if where=$(managed_elsewhere pmm-server); then
    echo "ERROR: a pmm-server container from $where exists. To add HolmesGPT and the renderer next to it, run:"
    echo "       bash install.sh --existing-server"; exit 1
  fi
fi

# Persist what docker compose and later re-runs need; keep anything the user already put in .env.
umask 077
touch .env
keep() { grep -q "^$1=" .env || printf '%s=%s\n' "$1" "$2" >> .env; }
if [ -n "$EXISTING" ]; then keep PMM_EXISTING_SERVER "$EXISTING"; else keep PMM_IMAGE "$PMM_IMAGE"; fi
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

if [ -n "$EXISTING" ]; then
  echo "==> 2/5  start the image renderer next to '$EXISTING'"
  if [ "$(sed -E 's#^[a-zA-Z]+://([^:/]+).*#\1#' <<<"$RENDERER_URL")" != grafana-image-renderer ]; then
    echo "         '$EXISTING' uses its own renderer at $RENDERER_URL"
  elif [ "$($DOCKER inspect -f '{{.State.Running}}' grafana-image-renderer 2>/dev/null)" = true ]; then
    echo "         already running, keeping it"
  else
    $DOCKER compose up -d --quiet-pull grafana-image-renderer
  fi
  # PMM and HolmesGPT reach each other by container name on the 'pmm' network, which compose creates.
  $DOCKER network inspect pmm >/dev/null 2>&1 || $DOCKER compose create holmesgpt >/dev/null
  join_pmm_network "$EXISTING"
  [ "$($DOCKER exec "$EXISTING" curl -s -o /dev/null --max-time 5 -w '%{http_code}' "${RENDERER_URL%/render}/")" != 000 ] \
    || echo "         WARNING: '$EXISTING' cannot reach the renderer at $RENDERER_URL; answers will have no panel images."
else
  echo "==> 2/5  start PMM Server ($PMM_IMAGE) and the image renderer"
  $DOCKER compose up -d --quiet-pull pmm-server grafana-image-renderer
  wait_ready "$DOCKER compose logs pmm-server"
  # GF_SECURITY_ADMIN_PASSWORD only applies when the PMM data volume is first created; reset it on an existing one.
  if ! curl -ksf -o /dev/null -u "admin:$PMM_ADMIN_PASSWORD" "$PMM_LOCAL/v1/server/version"; then
    $DOCKER exec pmm-server change-admin-password "$PMM_ADMIN_PASSWORD" >/dev/null
    echo "         admin password set"
  fi
  require_assistant PMM_IMAGE
fi

echo "==> 3/5  pick the PMM URL HolmesGPT calls back on"
# It must work from the Holmes container AND from browsers (answers embed panel image links), so prefer this
# host's IP over the Docker-only https://pmm-server:8443; fall back to that when the IP is not reachable.
if [ -z "${PMM_CALLBACK_URL:-}" ]; then
  $DOCKER pull -q "$HOLMES_IMAGE" >/dev/null
  host_ip=$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{for (i = 1; i < NF; i++) if ($i == "src") print $(i + 1)}')
  [ -n "$host_ip" ] || host_ip=$(hostname -I 2>/dev/null | awk '{print $1}')
  port_sfx=''
  [ "$PMM_PORT" = 443 ] || port_sfx=":$PMM_PORT"
  PMM_CALLBACK_URL=https://$PMM_CONTAINER:8443
  if [ -n "$host_ip" ] && [ "$($DOCKER run --rm --network pmm --entrypoint curl "$HOLMES_IMAGE" \
      -sk --max-time 10 -o /dev/null -w '%{http_code}' "https://$host_ip$port_sfx/v1/server/readyz")" = 200 ]; then
    PMM_CALLBACK_URL="https://$host_ip$port_sfx"
  else
    echo "         WARNING: https://${host_ip:-<host ip>}$port_sfx is not reachable from containers; panel image links in"
    echo "         answers will not open in a browser. Set PMM_CALLBACK_URL to fix that and re-run."
  fi
fi
keep PMM_CALLBACK_URL "$PMM_CALLBACK_URL"
echo "         $PMM_CALLBACK_URL"

echo "==> 4/5  provision the PMM Assistant"
settings="{\"enabled\":true,\"url\":$(json_str "$HOLMES_URL"),\"tls_skip_verify\":true"
[ -n "$MODEL_NAME" ] && settings+=",\"chat_model\":$(json_str "${FAST_MODEL_NAME:-$MODEL_NAME}"),\"investigation_model\":$(json_str "$MODEL_NAME")"
api POST /v1/adre/settings "$settings}" >/dev/null
echo "         + Assistant enabled, Holmes at $HOLMES_URL (self-signed cert, TLS verify skipped)"
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
echo "==> done.  PMM: ${PMM_CALLBACK_URL/$PMM_CONTAINER:8443/<this-host>}   login: admin, password in .env (PMM_ADMIN_PASSWORD)"
if curl -ksf -u "admin:$PMM_ADMIN_PASSWORD" "$PMM_LOCAL/v1/adre/deployment" | grep -q '"models":\[\]'; then
  echo "    Next: add an LLM in PMM → Configuration → Assistant → Deployment → Models & Keys, then open Assistant."
fi
