#!/usr/bin/env bash
#
# One-command PMM + HolmesGPT (PMM Assistant) install on a single Docker or Podman host.
#
#   mkdir pmm-assistant && cd pmm-assistant
#   curl -fsSLO https://raw.githubusercontent.com/percona/pmm/tibi-holmes/dev/adre/deploy/install.sh
#   bash install.sh                     # PMM Server, the image renderer and HolmesGPT
#   bash install.sh --existing-server   # only the renderer and HolmesGPT, next to a PMM Server that already runs
#   bash install.sh --remove            # remove what this script started here
#
# Nothing is required up front for a new install. Optional settings come from the environment or ./.env (see
# .env.example); the environment wins. Set MODEL_NAME + LITELLM_MODEL + LLM_API_KEY to add a model now, or add it later in
# PMM → Configuration → Assistant → Deployment → Models & Keys. Re-runnable: the cert, admin password and provisioning are kept,
# and models are upserted, never removed.
set -euo pipefail
cd "$(dirname "$0")"

usage() {
  cat <<'EOF'
Usage: bash install.sh [--existing-server[=CONTAINER]] [--remove]

  (no option)                    Start PMM Server, the image renderer and HolmesGPT (Docker, or rootless Podman).
  --existing-server[=CONTAINER]  Use a PMM Server container that is already running (default: pmm-server).
                                 Checks it has what HolmesGPT and the renderer need, then starts only those two.
                                 Needs that server's PMM_ADMIN_PASSWORD (environment or ./.env); it is never changed.
  --remove                       Remove the containers this script started from this directory. PMM's data volume,
                                 ./.env and ./holmes-config stay; a server it did not start is left running.
EOF
}
REMOVE=''
for arg in "$@"; do
  case $arg in
    --existing-server) export PMM_EXISTING_SERVER=pmm-server ;;
    --existing-server=?*) export PMM_EXISTING_SERVER=${arg#*=} ;;
    --remove) REMOVE=1 ;;
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
if [ -n "$EXISTING" ] && [ -z "$REMOVE" ] && [ -z "${PMM_ADMIN_PASSWORD:-}" ]; then
  echo "ERROR: --existing-server needs that server's admin password: set PMM_ADMIN_PASSWORD in ./.env or the environment."; exit 1
fi
# The images this version installs: a new build changes DEFAULT_PMM_IMAGE here and nowhere else. Fully qualified, so
# Podman never asks which registry a short name means.
DEFAULT_PMM_IMAGE=docker.io/perconalab/pmm-server-fb:PR-4259-19150b4
PMM_IMAGE="${PMM_IMAGE:-$DEFAULT_PMM_IMAGE}"
HOLMES_IMAGE="${HOLMES_IMAGE:-docker.io/robustadev/holmes:0.45.0}"
RENDERER_IMAGE="${RENDERER_IMAGE:-docker.io/grafana/grafana-image-renderer:latest}"
HOLMES_CONFIG_DIR="${HOLMES_CONFIG_DIR:-./holmes-config}"
PMM_HTTPS_PORT="${PMM_HTTPS_PORT:-443}"             # new install: host port PMM's HTTPS (8443) is published on
PMM_VOLUME="${PMM_VOLUME:-pmm-data}"                # new install: PMM's data volume
PMM_ADMIN_PASSWORD="${PMM_ADMIN_PASSWORD:-$(openssl rand -base64 18 | tr -d '/+=')}"
HOLMES_URL="${HOLMES_URL:-https://holmesgpt:5050}"  # how PMM reaches Holmes (container name on the shared network)
PMM_LOCAL="${PMM_LOCAL:-}"                          # how THIS script reaches PMM; default https://localhost on its published port
PMM_PORT=443
NET=pmm
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
# Docker when this user can talk to its daemon, else Podman (rootless for a non-root user), else Docker through sudo.
if [ -z "${DOCKER:-}" ]; then
  if command -v docker >/dev/null && docker info >/dev/null 2>&1; then DOCKER=docker
  elif command -v podman >/dev/null; then DOCKER=podman
  elif command -v docker >/dev/null; then DOCKER="sudo docker"
  else echo "ERROR: Docker or Podman is required."; exit 1
  fi
fi
PODMAN='' ROOTLESS='' SYSTEMCTL=systemctl
if $DOCKER --version 2>/dev/null | grep -qi podman; then
  PODMAN=1
  [ "$($DOCKER info --format '{{.Host.Security.Rootless}}' 2>/dev/null)" != true ] || { ROOTLESS=1; SYSTEMCTL='systemctl --user'; }
fi

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

# docker.io/x:t and x:t name the same image.
image_key() { local i=${1#docker.io/}; printf '%s' "${i#library/}"; }

# The directory whose install.sh started container $1 (older versions of this script used docker compose).
owner_dir() {
  $DOCKER container inspect -f \
    '{{index .Config.Labels "pmm-assistant.dir"}}{{index .Config.Labels "com.docker.compose.project.working_dir"}}' "$1" 2>/dev/null
}
mine() { [ "$(owner_dir "$1")" = "$(pwd)" ]; }

# Succeeds, printing where it comes from, when container $1 exists but this directory did not start it.
managed_elsewhere() {
  local wd
  $DOCKER container inspect "$1" >/dev/null 2>&1 || return 1
  wd=$(owner_dir "$1")
  [ "$wd" != "$(pwd)" ] || return 1
  printf '%s' "${wd:-outside this script}"
}

networks_of() {
  $DOCKER container inspect -f '{{range $k, $v := .NetworkSettings.Networks}}{{println $k}}{{end}}' "$1" | grep . || true
}
# Default networks give containers no DNS names for each other, so HolmesGPT, the renderer and PMM share another one.
shared_network() { networks_of "$1" | { grep -vxE 'bridge|podman|host|none|default' || true; } | head -n1; }

join_network() {
  networks_of "$1" | grep -x "$NET" >/dev/null \
    || { $DOCKER network connect "$NET" "$1"; echo "         + connected '$1' to the '$NET' network"; }
}

ensure_network() {
  $DOCKER network inspect "$NET" >/dev/null 2>&1 || $DOCKER network create --label "pmm-assistant.dir=$(pwd)" "$NET" >/dev/null
}

pull_missing() {
  $DOCKER image inspect "$1" >/dev/null 2>&1 && return
  echo "         pulling $1"
  $DOCKER pull -q "$1" >/dev/null
}

# run_container NAME FORCE ARGS... — (re)creates NAME from ARGS (its image and command last) unless it runs with the
# same ARGS already; FORCE=1 always recreates. Extra networks of a replaced container are joined again.
run_container() {
  local name=$1 force=$2 hash old_nets='' n
  shift 2
  hash=$(printf '%s\0' "$@" | sha256sum | cut -c1-16)
  if where=$(managed_elsewhere "$name"); then
    echo "ERROR: a $name container from $where exists; remove it ($DOCKER rm -f $name) or run install.sh where it"
    echo "       was installed."; exit 1
  fi
  if $DOCKER container inspect "$name" >/dev/null 2>&1; then
    if [ "$force" != 1 ] && [ "$($DOCKER container inspect -f '{{.State.Running}}|{{index .Config.Labels "pmm-assistant.config"}}' "$name")" = "true|$hash" ]; then
      echo "         $name: running, keeping it"
      return
    fi
    old_nets=$(networks_of "$name")
    # A long stop timeout lets PMM's PostgreSQL shut down cleanly before the container goes.
    $DOCKER stop -t 120 "$name" >/dev/null 2>&1 || true
    $DOCKER rm -f "$name" >/dev/null
  fi
  $DOCKER run -d --name "$name" --label "pmm-assistant.dir=$(pwd)" --label "pmm-assistant.config=$hash" "$@" >/dev/null
  for n in $old_nets; do
    case $n in "$NET" | bridge | podman | host | none | default) continue ;; esac
    $DOCKER network connect "$n" "$name" 2>/dev/null && echo "         + reconnected '$name' to the '$n' network"
  done
  echo "         $name: started"
}

remove_install() {
  local c vol
  for c in holmesgpt grafana-image-renderer; do
    if mine "$c"; then $DOCKER rm -f "$c" >/dev/null; echo "removed $c"; fi
  done
  if [ -z "$EXISTING" ] && mine pmm-server; then
    vol=$($DOCKER container inspect -f '{{range .Mounts}}{{if eq .Destination "/srv"}}{{.Name}}{{end}}{{end}}' pmm-server)
    $DOCKER stop -t 120 pmm-server >/dev/null 2>&1 || true
    $DOCKER rm -f pmm-server >/dev/null
    echo "removed pmm-server; its data stays in the volume '$vol' (delete it with: $DOCKER volume rm $vol)"
  fi
  if [ "$($DOCKER network inspect -f '{{index .Labels "pmm-assistant.dir"}}' "$NET" 2>/dev/null)" = "$(pwd)" ]; then
    if [ -n "$EXISTING" ] && networks_of "$EXISTING" 2>/dev/null | grep -x "$NET" >/dev/null; then
      $DOCKER network disconnect "$NET" "$EXISTING" && echo "disconnected '$EXISTING' from the '$NET' network"
    fi
    $DOCKER network rm "$NET" >/dev/null 2>&1 && echo "removed the '$NET' network"
  fi
  echo "./.env and ./holmes-config stay; the Assistant settings stay in PMM."
}

if [ -n "$REMOVE" ]; then
  remove_install
  exit 0
fi

# Docker and Podman fix a container's env, mounts and networking when it is created, so an existing PMM Server is
# checked before anything is changed; what only a recreate can fix is listed with the options that fix it.
check_existing_server() {
  local c=$EXISTING running env cfg_dir mount drop_mount='' callback ip port where image new_image='' unit unit_file server_net=''
  local env_file opts added user_env rerun e
  local -a recreate=() envs=() runopts=()
  running=$($DOCKER container inspect -f '{{.State.Running}}' "$c" 2>/dev/null) || { echo "ERROR: there is no container named '$c'."; exit 1; }
  [ "$running" = true ] || { echo "ERROR: container '$c' is not running."; exit 1; }
  env=$($DOCKER container inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$c")
  getenv() { sed -n "s/^$1=//p" <<<"$env" | tail -n1; }
  # A systemd unit that runs `podman run` (PMM's Podman guide) recreates the container on every start. Generated and
  # Quadlet units label it; a hand-written one does not, but its conmon stays in the unit's cgroup.
  unit=''
  if [ -n "$PODMAN" ]; then
    unit=$($DOCKER container inspect -f '{{index .Config.Labels "PODMAN_SYSTEMD_UNIT"}}' "$c")
    [ -n "$unit" ] || unit=$(sed -n 's#^[0-9]*:[^:]*:.*/\([^/]*\.service\)$#\1#p' \
      "/proc/$($DOCKER container inspect -f '{{.State.ConmonPid}}' "$c")/cgroup" 2>/dev/null | { grep -v '^user@' || true; } | head -n1)
  fi

  if [ "$($DOCKER container inspect -f '{{.HostConfig.NetworkMode}}' "$c")" = host ]; then
    recreate+=("it uses host networking, so it cannot share a network with HolmesGPT and the renderer")
    runopts+=(--network pmm -p 443:8443)
  else
    port=$($DOCKER container inspect -f '{{with index .NetworkSettings.Ports "8443/tcp"}}{{(index . 0).HostPort}}{{end}}' "$c")
    if [ -n "$port" ]; then
      PMM_PORT=$port
      PMM_LOCAL=${PMM_LOCAL:-https://localhost:$port}
    elif [ -z "$PMM_LOCAL" ]; then
      recreate+=("its HTTPS port 8443 is not published, so this script cannot reach PMM (or set PMM_LOCAL)")
      runopts+=(-p 443:8443)
    fi
    NET=$(shared_network "$c")
    server_net=$NET
    # Docker connects a running container to the new network in step 2; Podman cannot on its default network.
    if [ -z "$NET" ]; then
      NET=pmm
      if [ -n "$PODMAN" ]; then
        recreate+=("it is only on Podman's default network, where HolmesGPT and the renderer cannot reach it by name")
        runopts+=(--network pmm)
      fi
    fi
  fi

  if where=$(managed_elsewhere holmesgpt); then
    echo "ERROR: a holmesgpt container from $where exists; remove it ($DOCKER rm -f holmesgpt) or run install.sh"
    echo "       where it was installed."; exit 1
  fi

  if [ -n "$PMM_LOCAL" ]; then
    wait_ready "$DOCKER logs $c"
    case $(curl -ks -o /dev/null -w '%{http_code}' -u "admin:$PMM_ADMIN_PASSWORD" "$PMM_LOCAL/v1/server/version") in
      200) ;;
      401) echo "ERROR: PMM_ADMIN_PASSWORD is not the admin password of '$c' (install.sh never changes it on an existing server)."; exit 1 ;;
      *) echo "ERROR: $PMM_LOCAL does not answer like PMM Server; set PMM_LOCAL to its HTTPS URL."; exit 1 ;;
    esac
    case $(curl -ks -o /dev/null -w '%{http_code}' -u "admin:$PMM_ADMIN_PASSWORD" "$PMM_LOCAL/v1/adre/settings") in
      200)
        image=$($DOCKER container inspect -f '{{.Config.Image}}' "$c")
        if [ "$(image_key "$image")" != "$(image_key "$PMM_IMAGE")" ]; then
          echo "         NOTE: '$c' runs $image; this script's build is $PMM_IMAGE. To move to it, recreate '$c'"
          echo "         from that image with the same options (data in the volume is kept)."
        fi ;;
      404)
        new_image=$PMM_IMAGE
        recreate+=("its image has no PMM Assistant (stock PMM); it has to run $new_image") ;;
      *) echo "ERROR: $PMM_LOCAL/v1/adre/settings failed; check: $DOCKER logs $c"; exit 1 ;;
    esac
  fi

  cfg_dir=$(getenv ADRE_CONFIG_DIR)
  cfg_dir=${cfg_dir:-/holmes-config}
  mount=$($DOCKER container inspect -f '{{range .Mounts}}{{.Type}}|{{.Source}}|{{.Destination}}{{println}}{{end}}' "$c" |
    awk -F'|' -v d="$cfg_dir" '$3 == d')
  if [ "${mount%%|*}" = bind ]; then
    HOLMES_CONFIG_DIR=$(cut -d'|' -f2 <<<"$mount")
  else
    recreate+=("PMM renders the HolmesGPT config into $cfg_dir, which is not a host directory HolmesGPT can mount too")
    runopts+=(-v "$(realpath -m "$HOLMES_CONFIG_DIR"):$cfg_dir:z")
    drop_mount=$cfg_dir
    [ -n "$(getenv ADRE_CONFIG_DIR)" ] || envs+=("ADRE_CONFIG_DIR=$cfg_dir")
  fi

  if [ -n "$(getenv GF_RENDERING_SERVER_URL)" ]; then
    RENDERER_URL=$(getenv GF_RENDERING_SERVER_URL)
  else
    recreate+=("GF_RENDERING_SERVER_URL is not set, so Grafana cannot render panel images for answers")
    envs+=("GF_RENDERING_SERVER_URL=$RENDERER_URL")
  fi
  callback=$(getenv GF_RENDERING_CALLBACK_URL)
  if [ -z "$callback" ]; then
    recreate+=("GF_RENDERING_CALLBACK_URL is not set, so the renderer cannot load panels back from Grafana")
    envs+=("GF_RENDERING_CALLBACK_URL=http://$c:8080/graph/")
  else
    # The renderer calls back over the network, so a port bound to 127.0.0.1 (Grafana's :3000) never answers it.
    ip=$($DOCKER container inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}' "$c" | awk '{print $1}')
    if [ -n "$ip" ] && [ "$($DOCKER exec "$c" curl -sk -o /dev/null --max-time 5 -w '%{http_code}' \
        "$(sed -E "s#^([a-zA-Z]+://)[^/:]+#\1$ip#" <<<"$callback")")" = 000 ]; then
      recreate+=("GF_RENDERING_CALLBACK_URL ($callback) does not answer on the container's network address, so the renderer cannot load panels")
      envs+=("GF_RENDERING_CALLBACK_URL=http://$c:8080/graph/")
    fi
  fi

  case $(getenv PMM_ENABLE_UPDATES) in
    false | 0) ;;
    *)
      echo "         WARNING: in-app updates are on; they can replace this build with a stock PMM release, which has no"
      echo "         Assistant. Add PMM_ENABLE_UPDATES=false when you next recreate '$c'."
      envs+=(PMM_ENABLE_UPDATES=false) ;;
  esac

  [ ${#recreate[@]} -eq 0 ] && return
  image=$($DOCKER container inspect -f '{{.Config.Image}}' "$c")
  rerun="bash install.sh --existing-server"
  [ "$c" = pmm-server ] || rerun+="=$c"
  echo "ERROR: '$c' has to be recreated; a restart is not enough, because the container's env, mounts and networking"
  echo "       are fixed when it is created. To fix:"
  printf '         - %s\n' "${recreate[@]}"
  if [ -n "$new_image" ]; then
    echo "       WARNING: moving its data to $new_image is one-way. The prototype adds its own database changes, so"
    echo "       going back to stock PMM or upgrading to a later official release is not supported. Use a test server,"
    echo "       or back up the data volume first."
  fi

  if [ -n "$unit" ]; then
    # The unit recreates the container on every start, so the fix goes into the unit and its env file.
    unit_file=$($SYSTEMCTL show -p FragmentPath --value "$unit" 2>/dev/null || true)
    env_file=$($SYSTEMCTL cat "$unit" 2>/dev/null | grep -o -- '--env-file=[^ ]*' | head -n1 | cut -d= -f2- || true)
    env_file=${env_file//%h/$HOME}
    echo "       It is started by the systemd unit $unit${unit_file:+ ($unit_file)}, which recreates it on every"
    echo "       start, so change the unit rather than the container:"
    echo "       1. Set these in ${env_file:-the env file the unit passes with --env-file} (replace a line for the same name):"
    [ -z "$new_image" ] || echo "            PMM_IMAGE=$new_image"
    for e in "${envs[@]}"; do echo "            $e"; done
    if [ ${#runopts[@]} -gt 0 ]; then
      echo "       2. Add to the unit's podman run line: ${runopts[*]}"
      [[ " ${runopts[*]} " != *" --network pmm "* ]] || echo "          and create that network first: $DOCKER network create pmm"
    else
      echo "       2. Nothing to change in the unit's podman run line."
    fi
    [ -z "$new_image" ] || echo "          The unit must take the image from PMM_IMAGE (\${PMM_IMAGE}), as in PMM's Podman guide."
    echo "       3. $SYSTEMCTL daemon-reload && $SYSTEMCTL restart $unit"
    echo "       Then run: $rerun"
    echo "       Nothing else was changed."
    exit 2
  fi

  # One option per line so a config-dir mount being replaced can be dropped; values of -e options are not printed.
  opts=$($DOCKER container inspect -f '{{with .HostConfig.RestartPolicy.Name}}{{if ne . "no"}}--restart {{.}}{{println}}{{end}}{{end}}
{{- range $p, $b := .HostConfig.PortBindings}}{{range $b}}-p {{$ip := printf "%s" .HostIP}}{{if and $ip (ne $ip "invalid IP")}}{{$ip}}:{{end}}{{.HostPort}}:{{$p}}{{println}}{{end}}{{end}}
{{- range .Mounts}}{{if eq .Type "volume"}}-v {{.Name}}:{{.Destination}}{{else}}-v {{.Source}}:{{.Destination}}{{end}}{{println}}{{end}}' "$c" |
    { grep -v -e '^$' ${drop_mount:+-e "^-v .*:$drop_mount\$"} || true; } | paste -sd' ')
  # The -e options it was started with (not from the image, not set by Podman, not replaced above) go to an env
  # file, so the new container keeps them and no value is printed.
  added=$(printf '%s\n' "${envs[@]}" HOME HOSTNAME container | sed 's/=.*//')
  user_env=$(comm -23 <(sed 's/=.*//' <<<"$env" | grep . | sort -u) \
    <({ $DOCKER image inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$image" 2>/dev/null | sed 's/=.*//'; echo "$added"; } |
      grep . | sort -u))
  if [ -n "$user_env" ]; then
    (umask 077; for n in $user_env; do grep "^$n=" <<<"$env" | tail -n1; done > "$c.env")
    opts+=" --env-file $(pwd)/$c.env"
  fi
  for e in "${envs[@]}"; do runopts+=(-e "$e"); done
  if [ -n "$new_image" ]; then
    echo "       Recreate it from $new_image instead of $image, with the same data volume and ports, adding: ${runopts[*]}"
  else
    echo "       Recreate it with the same image, data volume and ports, adding: ${runopts[*]}"
  fi
  [ -z "$user_env" ] || echo "       Its other -e options ($(paste -sd' ' <<<"$user_env")) are saved in ./$c.env (mode 0600)."
  echo "       For example (data in the volume is kept):"
  [ -z "$server_net" ] || opts+=" --network $server_net"
  [[ " ${runopts[*]} " != *" --network pmm "* ]] || echo "         $DOCKER network inspect pmm >/dev/null 2>&1 || $DOCKER network create pmm"
  echo "         $DOCKER stop -t 120 $c && $DOCKER rm $c"
  echo "         $DOCKER run -d --name $c $opts ${runopts[*]} ${new_image:-$image}"
  echo "       Then run: $rerun"
  echo "       Nothing else was changed."
  exit 2
}

if [ -n "$EXISTING" ]; then
  echo "==> check the running PMM Server '$EXISTING' ($DOCKER)"
  check_existing_server
  echo "         OK: reachable at $PMM_LOCAL, shares $HOLMES_CONFIG_DIR, renders through $RENDERER_URL, network '$NET'"
else
  if where=$(managed_elsewhere pmm-server); then
    echo "ERROR: a pmm-server container from $where exists. To add HolmesGPT and the renderer next to it, run:"
    echo "       bash install.sh --existing-server"; exit 1
  fi
  for c in holmesgpt grafana-image-renderer; do
    if where=$(managed_elsewhere "$c"); then
      echo "ERROR: a $c container from $where exists; remove it ($DOCKER rm -f $c) or run install.sh where it was"
      echo "       installed."; exit 1
    fi
  done
  if [ "$(image_key "$PMM_IMAGE")" != "$(image_key "$DEFAULT_PMM_IMAGE")" ]; then
    echo "NOTE: PMM_IMAGE is $PMM_IMAGE (from .env or the environment); this script's build is"
    echo "      $DEFAULT_PMM_IMAGE. Remove PMM_IMAGE from .env and re-run to move to it."
  fi
  if [ -n "$ROOTLESS" ] && [ "$PMM_HTTPS_PORT" -lt "$(cat /proc/sys/net/ipv4/ip_unprivileged_port_start 2>/dev/null || echo 1024)" ]; then
    echo "ERROR: rootless Podman cannot publish port $PMM_HTTPS_PORT. Allow it as PMM's Podman guide does:"
    echo "         echo net.ipv4.ip_unprivileged_port_start=$PMM_HTTPS_PORT | sudo tee /etc/sysctl.d/99-pmm.conf"
    echo "         sudo sysctl -p /etc/sysctl.d/99-pmm.conf"
    echo "       or use another port: PMM_HTTPS_PORT=8443 bash install.sh"; exit 1
  fi
  PMM_PORT=$PMM_HTTPS_PORT
  [ "$PMM_PORT" = 443 ] && PMM_LOCAL=${PMM_LOCAL:-https://localhost} || PMM_LOCAL=${PMM_LOCAL:-https://localhost:$PMM_PORT}
fi

# Persist what re-runs need; keep anything the user already put in .env. The images are not saved, so a newer
# install.sh moves the install to the build it names.
umask 077
touch .env
keep() { grep -q "^$1=" .env || printf '%s=%s\n' "$1" "$2" >> .env; }
keep DOCKER "$DOCKER"
if [ -n "$EXISTING" ]; then keep PMM_EXISTING_SERVER "$EXISTING"; else keep PMM_HTTPS_PORT "$PMM_HTTPS_PORT"; fi
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
# Bind mounts need an absolute path; :z lets SELinux hosts share the directory between containers.
CFG_ABS=$(realpath "$HOLMES_CONFIG_DIR")

# Grafana Image Renderer: turns panels into PNGs for answers. Reached by name on the shared network, no published port.
renderer_args=(--network "$NET" --restart always -e HTTP_HOST=0.0.0.0 -e HTTP_PORT=8081 "$RENDERER_IMAGE")

if [ -n "$EXISTING" ]; then
  echo "==> 2/5  start the image renderer next to '$EXISTING' (network '$NET')"
  ensure_network
  join_network "$EXISTING"
  if [ "$(sed -E 's#^[a-zA-Z]+://([^:/]+).*#\1#' <<<"$RENDERER_URL")" != grafana-image-renderer ]; then
    echo "         '$EXISTING' uses its own renderer at $RENDERER_URL"
  elif [ "$($DOCKER container inspect -f '{{.State.Running}}' grafana-image-renderer 2>/dev/null)" = true ] && ! mine grafana-image-renderer; then
    echo "         grafana-image-renderer: already running, keeping it"
  else
    pull_missing "$RENDERER_IMAGE"
    run_container grafana-image-renderer 0 "${renderer_args[@]}"
  fi
  [ "$($DOCKER exec "$EXISTING" curl -s -o /dev/null --max-time 5 -w '%{http_code}' "${RENDERER_URL%/render}/")" != 000 ] \
    || echo "         WARNING: '$EXISTING' cannot reach the renderer at $RENDERER_URL; answers will have no panel images."
else
  echo "==> 2/5  start PMM Server ($PMM_IMAGE) and the image renderer ($DOCKER${ROOTLESS:+, rootless})"
  # A server this directory started earlier keeps its data volume (older versions named it <dir>_pmm-data).
  if mine pmm-server; then
    PMM_VOLUME=$($DOCKER container inspect -f '{{range .Mounts}}{{if eq .Destination "/srv"}}{{.Name}}{{end}}{{end}}' pmm-server)
  fi
  ensure_network
  server_args=(--network "$NET" -p "$PMM_HTTPS_PORT:8443" --restart always --stop-timeout 120
    -v "$PMM_VOLUME:/srv" -v "$CFG_ABS:/holmes-config:z"
    # PMM renders the Holmes config into the shared directory.
    -e ADRE_CONFIG_DIR=/holmes-config
    # The in-app updater would replace this build with a stock PMM release, which has no Assistant.
    -e PMM_ENABLE_UPDATES=false
    # Grafana itself listens on 127.0.0.1:3000 only, so the renderer calls back through nginx's plain-HTTP port 8080.
    -e GF_RENDERING_SERVER_URL="$RENDERER_URL" -e GF_RENDERING_CALLBACK_URL=http://pmm-server:8080/graph/)
  # As in PMM's Podman guide: PMM's uid 1000 is this user on the host, so files it writes stay this user's.
  [ -z "$ROOTLESS" ] || server_args+=("--userns=keep-id:uid=1000,gid=1000" --cap-add=net_admin --cap-add=net_raw)
  pull_missing "$PMM_IMAGE"
  run_container pmm-server 0 "${server_args[@]}" "$PMM_IMAGE"
  pull_missing "$RENDERER_IMAGE"
  run_container grafana-image-renderer 0 "${renderer_args[@]}"
  wait_ready "$DOCKER logs pmm-server"
  # A new data volume starts with admin/admin; set the password this install keeps in .env.
  if ! curl -ksf -o /dev/null -u "admin:$PMM_ADMIN_PASSWORD" "$PMM_LOCAL/v1/server/version"; then
    $DOCKER exec pmm-server change-admin-password "$PMM_ADMIN_PASSWORD" >/dev/null
    echo "         admin password set"
  fi
  require_assistant PMM_IMAGE
fi

echo "==> 3/5  pick the PMM URL HolmesGPT calls back on"
# It must work from the Holmes container AND from browsers (answers embed panel image links), so prefer this
# host's IP over the container-only https://pmm-server:8443; fall back to that when the IP is not reachable.
pull_missing "$HOLMES_IMAGE"
if [ -z "${PMM_CALLBACK_URL:-}" ]; then
  host_ip=$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{for (i = 1; i < NF; i++) if ($i == "src") print $(i + 1)}')
  [ -n "$host_ip" ] || host_ip=$(hostname -I 2>/dev/null | awk '{print $1}')
  port_sfx=''
  [ "$PMM_PORT" = 443 ] || port_sfx=":$PMM_PORT"
  PMM_CALLBACK_URL=https://$PMM_CONTAINER:8443
  if [ -n "$host_ip" ] && [ "$($DOCKER run --rm --network "$NET" --entrypoint curl "$HOLMES_IMAGE" \
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
holmes_args=(--network "$NET" -p 127.0.0.1:5050:5050 --restart always
  -v "$CFG_ABS:/config:z"
  -e HOLMES_CONFIGPATH_DIR=/config -e MODEL_LIST_FILE_LOCATION=/config/model_list.yaml -e CUSTOM_SKILL_PATHS=/config/skills
  # PMM hot-reloads the config on save through POST /api/admin/reload.
  -e ENABLE_ADMIN_API=true
  # Strict tool calls make the model send every optional parameter as null, and Holmes 0.45's Prometheus tools then
  # fail on timeout=None. Leave TOOL_SCHEMA_NO_PARAM_OBJECT_IF_NO_PARAMS unset: it breaks OpenAI's strict mode.
  -e HOLMES_DISABLE_STRICT_TOOL_CALLS=true
  # Native HTTPS on 5050; PMM's client sends no client cert, so no mTLS (HOLMES_SSL_CA_CERTS).
  -e HOLMES_SSL_CERTFILE=/config/certs/tls.crt -e HOLMES_SSL_KEYFILE=/config/certs/tls.key
  # Source the PMM-rendered bootstrap .env (PMM_URL, PMM_API_TOKEN, HOLMES_API_KEY) at start.
  --entrypoint /bin/sh "$HOLMES_IMAGE" -c 'set -a; [ -f /config/.env ] && . /config/.env; set +a; exec python3 server.py')
# Recreate so Holmes reads the .env PMM just rendered.
run_container holmesgpt 1 "${holmes_args[@]}"
for i in $(seq 1 40); do
  curl -sk --max-time 3 https://127.0.0.1:5050/healthz >/dev/null 2>&1 && break
  [ "$i" = 40 ] && { echo "ERROR: HolmesGPT did not start — check: $DOCKER logs holmesgpt"; exit 1; }
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
server_image=$($DOCKER container inspect -f '{{.Config.Image}}' "$PMM_CONTAINER")
if [[ $server_image == *perconalab/pmm-server-fb:* ]]; then
  echo "    PMM Client for database hosts, same build: perconalab/pmm-client-fb:${server_image##*:} or"
  echo "    https://s3.us-east-2.amazonaws.com/pmm-build-cache/PR-BUILDS/pmm-client/pmm-client-${server_image##*:}.tar.gz"
fi
if curl -ksf -u "admin:$PMM_ADMIN_PASSWORD" "$PMM_LOCAL/v1/adre/deployment" | grep -q '"models":\[\]'; then
  echo "    Next: add an LLM in PMM → Configuration → Assistant → Deployment → Models & Keys, then open Assistant."
fi
if [ -n "$ROOTLESS" ]; then
  echo "    Rootless Podman starts these containers again after a reboot only with:"
  echo "      loginctl enable-linger $USER && systemctl --user enable podman-restart.service"
fi
