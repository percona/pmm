#!/usr/bin/env bash

set -euo pipefail

log() {
  echo "[install-pmm-client] $*"
}

error() {
  echo "[install-pmm-client] ERROR: $*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Usage: install-pmm-client.sh [options]

Global options:
  --pmm-server-url URL          PMM server URL (supports service_token userinfo)
  --pmm-server-insecure-tls     Use --server-insecure-tls for pmm-admin config
  --tech TECH                   One of: mysql, postgresql, mongodb, valkey
  --node-name NAME              Node name for pmm-admin config
  --node-address ADDRESS        Node address for pmm-admin config
  --force                       Pass --force to pmm-admin config (removes existing node name and its services on the server, then registers again)

Generic DB options (mapped per technology):
  --db-user USER
  --db-password PASSWORD
  --db-host HOST
  --db-port PORT
  --db-name NAME                DB name for PostgreSQL
  --db-address HOST:PORT        Explicit service address
  --db-service-name NAME        PMM service name
  --db-auth-db NAME             MongoDB auth database
  --db-socket PATH              Socket path for MySQL/PostgreSQL/MongoDB/Valkey

Environment variables are also supported.
Priority is: flags > env vars > interactive prompt.
When stdin is a terminal, database prompts are skipped if credentials are already
set from flags or environment (DB_USER / DB_PASSWORD and per-tech MYSQL_*,
POSTGRESQL_* / … after apply_generic_inputs). Use sudo -E bash … when running
as root so your exports reach the script.

pmm-agent runtime knobs (env only):
  PMM_AGENT_CONFIG_FILE         Path to pmm-agent.yaml (default: /usr/local/percona/pmm/config/pmm-agent.yaml)
  PMM_AGENT_LISTEN_HOST         Host the local API binds to (default: 127.0.0.1)
  PMM_AGENT_LISTEN_PORT         Port the local API binds to (default: 7777)
  PMM_AGENT_LOG_FILE            Log file when started without systemd (default: /var/log/pmm-agent.log)
  PMM_AGENT_START_TIMEOUT_SECS  Seconds to wait for the local API after start (default: 30)
EOF
}

PMM_SERVER_URL="${PMM_SERVER_URL:-}"
PMM_SERVER_INSECURE_TLS="${PMM_SERVER_INSECURE_TLS:-0}"
TECH="${TECH:-}"
NODE_NAME="${NODE_NAME:-}"
NODE_ADDRESS="${NODE_ADDRESS:-}"
PMM_CONFIG_FORCE="${PMM_CONFIG_FORCE:-0}"

DB_USER="${DB_USER:-}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_HOST="${DB_HOST:-}"
DB_PORT="${DB_PORT:-}"
DB_NAME="${DB_NAME:-}"
DB_ADDRESS="${DB_ADDRESS:-}"
DB_SERVICE_NAME="${DB_SERVICE_NAME:-}"
DB_AUTH_DB="${DB_AUTH_DB:-}"
DB_SOCKET="${DB_SOCKET:-}"

MYSQL_USERNAME="${MYSQL_USERNAME:-}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-}"
MYSQL_HOST="${MYSQL_HOST:-}"
MYSQL_PORT="${MYSQL_PORT:-}"
MYSQL_ADDRESS="${MYSQL_ADDRESS:-}"
MYSQL_SERVICE_NAME="${MYSQL_SERVICE_NAME:-}"
MYSQL_SOCKET="${MYSQL_SOCKET:-}"

POSTGRESQL_USERNAME="${POSTGRESQL_USERNAME:-}"
POSTGRESQL_PASSWORD="${POSTGRESQL_PASSWORD:-}"
POSTGRESQL_HOST="${POSTGRESQL_HOST:-}"
POSTGRESQL_PORT="${POSTGRESQL_PORT:-}"
POSTGRESQL_ADDRESS="${POSTGRESQL_ADDRESS:-}"
POSTGRESQL_SERVICE_NAME="${POSTGRESQL_SERVICE_NAME:-}"
POSTGRESQL_DATABASE="${POSTGRESQL_DATABASE:-}"
POSTGRESQL_SOCKET="${POSTGRESQL_SOCKET:-}"

MONGODB_USERNAME="${MONGODB_USERNAME:-}"
MONGODB_PASSWORD="${MONGODB_PASSWORD:-}"
MONGODB_HOST="${MONGODB_HOST:-}"
MONGODB_PORT="${MONGODB_PORT:-}"
MONGODB_ADDRESS="${MONGODB_ADDRESS:-}"
MONGODB_SERVICE_NAME="${MONGODB_SERVICE_NAME:-}"
MONGODB_AUTH_DB="${MONGODB_AUTH_DB:-}"
MONGODB_SOCKET="${MONGODB_SOCKET:-}"

VALKEY_USERNAME="${VALKEY_USERNAME:-}"
VALKEY_PASSWORD="${VALKEY_PASSWORD:-}"
VALKEY_HOST="${VALKEY_HOST:-}"
VALKEY_PORT="${VALKEY_PORT:-}"
VALKEY_ADDRESS="${VALKEY_ADDRESS:-}"
VALKEY_SERVICE_NAME="${VALKEY_SERVICE_NAME:-}"
VALKEY_SOCKET="${VALKEY_SOCKET:-}"

# pmm-agent runtime knobs. Defaults match the Debian/RPM package layout.
# Override via env if the package places things elsewhere or you need to
# bind the local API on a non-default host/port.
PMM_AGENT_CONFIG_FILE="${PMM_AGENT_CONFIG_FILE:-/usr/local/percona/pmm/config/pmm-agent.yaml}"
PMM_AGENT_LISTEN_HOST="${PMM_AGENT_LISTEN_HOST:-127.0.0.1}"
PMM_AGENT_LISTEN_PORT="${PMM_AGENT_LISTEN_PORT:-7777}"
PMM_AGENT_LOG_FILE="${PMM_AGENT_LOG_FILE:-/var/log/pmm-agent.log}"
PMM_AGENT_START_TIMEOUT_SECS="${PMM_AGENT_START_TIMEOUT_SECS:-30}"

while [ $# -gt 0 ]; do
  case "$1" in
    --help|-h)
      usage
      exit 0
      ;;
    --pmm-server-url)
      PMM_SERVER_URL="${2:-}"
      shift 2
      ;;
    --pmm-server-insecure-tls)
      PMM_SERVER_INSECURE_TLS=1
      shift
      ;;
    --tech)
      TECH="${2:-}"
      shift 2
      ;;
    --node-name)
      NODE_NAME="${2:-}"
      shift 2
      ;;
    --node-address)
      NODE_ADDRESS="${2:-}"
      shift 2
      ;;
    --force)
      PMM_CONFIG_FORCE=1
      shift
      ;;
    --db-user)
      DB_USER="${2:-}"
      shift 2
      ;;
    --db-password)
      DB_PASSWORD="${2:-}"
      shift 2
      ;;
    --db-host)
      DB_HOST="${2:-}"
      shift 2
      ;;
    --db-port)
      DB_PORT="${2:-}"
      shift 2
      ;;
    --db-name)
      DB_NAME="${2:-}"
      shift 2
      ;;
    --db-address)
      DB_ADDRESS="${2:-}"
      shift 2
      ;;
    --db-service-name)
      DB_SERVICE_NAME="${2:-}"
      shift 2
      ;;
    --db-auth-db)
      DB_AUTH_DB="${2:-}"
      shift 2
      ;;
    --db-socket)
      DB_SOCKET="${2:-}"
      shift 2
      ;;
    *)
      error "Unknown option: $1. Use --help for usage."
      ;;
  esac
done

require_root() {
  if [ "${EUID}" -ne 0 ]; then
    error "Run this script as root (for package installation). Example: curl -fsSLk ... | sudo -E env ... bash -s --"
  fi
}

prompt_if_empty() {
  local var_name="$1"
  local prompt_label="$2"
  local secret="${3:-0}"
  local hint="${4:-}"
  local value="${!var_name:-}"

  if [ -n "${value}" ]; then
    return
  fi

  if [ "${secret}" = "1" ]; then
    read -r -s -p "${prompt_label}: " value
    echo
  else
    read -r -p "${prompt_label}: " value
  fi

  if [ -z "${value}" ]; then
    if [ -n "${hint}" ]; then
      error "${prompt_label} is required. ${hint}"
    else
      error "${prompt_label} is required."
    fi
  fi

  printf -v "${var_name}" '%s' "${value}"
}

detect_os_family() {
  if [ -f /etc/os-release ]; then
    # shellcheck source=/dev/null
    . /etc/os-release
    case "${ID:-}" in
      debian|ubuntu)
        echo "debian"
        return
        ;;
      rhel|ol|amzn|rocky|almalinux|centos|fedora)
        echo "el"
        return
        ;;
    esac
  fi
  if [ -f /etc/redhat-release ] || [ -f /etc/oracle-release ]; then
    echo "el"
    return
  fi
  if [ -f /etc/debian_version ]; then
    echo "debian"
    return
  fi
  error "Unsupported OS. Supported 64-bit Linux: Debian, Ubuntu (DEB) and RHEL, Oracle Linux, Amazon Linux (RPM)."
}

install_percona_repo_el() {
  if command -v percona-release >/dev/null 2>&1; then
    return
  fi
  if command -v dnf >/dev/null 2>&1; then
    dnf install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
  elif command -v yum >/dev/null 2>&1; then
    yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
  else
    error "Neither dnf nor yum was found."
  fi
}

install_percona_repo_debian() {
  if command -v percona-release >/dev/null 2>&1; then
    return
  fi
  apt-get update -y
  apt-get install -y curl gnupg lsb-release
  local deb_path="/tmp/percona-release_latest.generic_all.deb"
  curl -fsSL -o "${deb_path}" "https://repo.percona.com/apt/percona-release_latest.generic_all.deb"
  apt-get install -y "${deb_path}"
  rm -f "${deb_path}"
}

install_pmm_client() {
  if command -v pmm-admin >/dev/null 2>&1; then
    log "pmm-admin already installed; skipping package install."
    return
  fi

  local os_family
  os_family="$(detect_os_family)"
  log "Detected OS family: ${os_family}"

  if [ "${os_family}" = "el" ]; then
    install_percona_repo_el
    percona-release enable pmm3-client release || true
    if command -v dnf >/dev/null 2>&1; then
      dnf install -y pmm-client
    else
      yum install -y pmm-client
    fi
    return
  fi

  install_percona_repo_debian
  percona-release enable pmm3-client release || true
  apt-get update -y
  apt-get install -y pmm-client
}

# Returns 0 if a real systemd is the init (i.e. systemctl can actually start units).
# Mere presence of systemctl is not enough — Docker images often ship the binary
# without PID 1 being systemd, and `systemctl start` then no-ops or fails.
systemd_is_running() {
  [ -d /run/systemd/system ] && command -v systemctl >/dev/null 2>&1
}

# Cheap TCP probe via bash builtins; no curl/nc dependency. We only need to know
# the local API socket is bound — pmm-admin will do the actual HTTP handshake.
# Anything else listening on that port (very unlikely on 7777) would falsely report
# success; that case fails clearly at the next pmm-admin step.
pmm_agent_listening() {
  (exec 3<>"/dev/tcp/${PMM_AGENT_LISTEN_HOST}/${PMM_AGENT_LISTEN_PORT}") >/dev/null 2>&1
}

wait_for_pmm_agent() {
  local i=0
  while [ "$i" -lt "$PMM_AGENT_START_TIMEOUT_SECS" ]; do
    if pmm_agent_listening; then
      return 0
    fi
    sleep 1
    i=$((i + 1))
  done
  return 1
}

# pmm-agent refuses to start without a config file. The Debian/RPM packages
# create an empty 0660 file at install time; recreate it if something deleted it
# (e.g. after a manual cleanup) so the daemon at least has a path to write to.
# Resolve the local node's hostname without assuming the `hostname(1)` binary
# is installed. Minimal RHEL/UBI/Alpine images often ship without it, and a
# `$(hostname)` call there fails with "command not found", which under
# `set -euo pipefail` aborts the whole script. Order: bash's $HOSTNAME (set
# automatically via gethostname() syscall) → uname -n → /etc/hostname → "node".
detect_node_hostname() {
  if [ -n "${HOSTNAME:-}" ]; then
    printf '%s' "${HOSTNAME}"
    return
  fi
  if command -v hostname >/dev/null 2>&1; then
    hostname
    return
  fi
  if command -v uname >/dev/null 2>&1; then
    uname -n
    return
  fi
  if [ -r /etc/hostname ]; then
    head -n 1 /etc/hostname
    return
  fi
  printf 'node'
}

ensure_pmm_agent_config_file() {
  local dir
  dir="$(dirname "${PMM_AGENT_CONFIG_FILE}")"
  if [ ! -d "${dir}" ]; then
    mkdir -p "${dir}"
  fi
  if [ ! -e "${PMM_AGENT_CONFIG_FILE}" ]; then
    : > "${PMM_AGENT_CONFIG_FILE}"
    chmod 0660 "${PMM_AGENT_CONFIG_FILE}" || true
    log "Created empty pmm-agent config: ${PMM_AGENT_CONFIG_FILE}"
  fi
}

start_pmm_agent_systemd() {
  if ! systemctl cat pmm-agent.service >/dev/null 2>&1; then
    return 1
  fi
  log "Starting pmm-agent via systemd..."
  systemctl daemon-reload >/dev/null 2>&1 || true
  if ! systemctl enable --now pmm-agent.service; then
    return 1
  fi
}

start_pmm_agent_nohup() {
  if ! command -v pmm-agent >/dev/null 2>&1; then
    error "pmm-agent binary not found in PATH; cannot start it manually."
  fi
  mkdir -p "$(dirname "${PMM_AGENT_LOG_FILE}")" 2>/dev/null || true

  # Drop privileges to the pmm-agent system user when it exists (created by the
  # package's postinst). The systemd unit runs as that user too, so this keeps
  # the nohup fallback from being a privilege regression vs. systemd. If the
  # user is missing (very minimal images, broken postinst), fall back to root —
  # the agent still works, just with a wider blast radius if it's ever exploited.
  local runner=()
  if id -u pmm-agent >/dev/null 2>&1 && command -v runuser >/dev/null 2>&1; then
    runner=(runuser -u pmm-agent --)
    log "Starting pmm-agent as user pmm-agent (no usable systemd); logging to ${PMM_AGENT_LOG_FILE}"
  else
    log "Starting pmm-agent as root (no pmm-agent user or no runuser binary); logging to ${PMM_AGENT_LOG_FILE}"
  fi

  nohup "${runner[@]}" pmm-agent --config-file="${PMM_AGENT_CONFIG_FILE}" \
    >>"${PMM_AGENT_LOG_FILE}" 2>&1 &
  disown 2>/dev/null || true
}

# Make sure pmm-agent is up and listening on its local API before we hand off to
# pmm-admin config/add. The script previously assumed the package's postinst had
# already started the daemon via systemd; that breaks in containers (no systemd)
# and on hosts where the service is masked or stopped.
ensure_pmm_agent_running() {
  if pmm_agent_listening; then
    log "pmm-agent already listening on ${PMM_AGENT_LISTEN_HOST}:${PMM_AGENT_LISTEN_PORT}."
    return
  fi

  log "pmm-agent is not running; attempting to start it."
  ensure_pmm_agent_config_file

  local started=0
  if systemd_is_running; then
    if start_pmm_agent_systemd; then
      started=1
    else
      log "No pmm-agent.service unit found; falling back to nohup."
    fi
  fi

  if [ "${started}" -eq 0 ]; then
    start_pmm_agent_nohup
  fi

  if ! wait_for_pmm_agent; then
    log "pmm-agent did not bind ${PMM_AGENT_LISTEN_HOST}:${PMM_AGENT_LISTEN_PORT} within ${PMM_AGENT_START_TIMEOUT_SECS}s."
    if [ -f "${PMM_AGENT_LOG_FILE}" ]; then
      log "Last 20 lines of ${PMM_AGENT_LOG_FILE}:"
      tail -n 20 "${PMM_AGENT_LOG_FILE}" >&2 || true
    elif systemd_is_running; then
      log "Try: journalctl -u pmm-agent -n 50 --no-pager"
    fi
    error "pmm-agent failed to start."
  fi

  log "pmm-agent is up on ${PMM_AGENT_LISTEN_HOST}:${PMM_AGENT_LISTEN_PORT}."
}

# When stdin is not a terminal (e.g. curl ... | bash), prompts cannot be used for DB
# credentials. Fail before pmm-admin config so we do not register the node and then fail on add.
# Caller must have already invoked apply_generic_inputs.
#
# Contract with the PMM UI's "Prompt on node" credentials mode:
#   The UI deliberately renders a TWO-STEP command in that mode:
#     1) curl -fsSL[k] -o /tmp/install-pmm-client.sh '<url>'
#     2) sudo -E bash /tmp/install-pmm-client.sh --pmm-server-url ... --tech ... [...]
#   Step 2 reads the script from disk (not from a pipe), so stdin stays attached
#   to the user's TTY through sudo, [ -t 0 ] is true, and prompt_if_empty /
#   read -r -s in add_mysql / add_postgresql / add_mongodb / add_valkey can ask
#   for DB user and password interactively — unless DB_USER / DB_PASSWORD (or
#   per-tech MYSQL_* / …) are already set; sudo -E preserves those exports.
#   This guard therefore never trips
#   when the user followed the UI's prompt-mode command — it only protects the
#   curl | bash pipeline from registering a half-configured node.
require_db_creds_before_config_if_noninteractive() {
  if [ -t 0 ]; then
    return 0
  fi

  local hint='This install is non-interactive (stdin is not a terminal, e.g. curl ... | bash), so database credentials cannot be prompted. Either pass them up front (--db-user/--db-password or DB_USER/DB_PASSWORD; with sudo env, use sudo -E to preserve exports), or switch to the UI'\''s "Prompt on node" mode which renders a download-then-run command: curl -fsSLk -o /tmp/install-pmm-client.sh '\''<url>'\''; sudo -E bash /tmp/install-pmm-client.sh --pmm-server-url ... --tech ...'

  case "${TECH}" in
    mysql)
      if [ -z "${MYSQL_USERNAME}" ] || [ -z "${MYSQL_PASSWORD}" ]; then
        error "MySQL username and password are required for non-interactive runs. ${hint}"
      fi
      ;;
    postgresql)
      if [ -z "${POSTGRESQL_USERNAME}" ] || [ -z "${POSTGRESQL_PASSWORD}" ]; then
        error "PostgreSQL username and password are required for non-interactive runs. ${hint}"
      fi
      ;;
    mongodb)
      if [ -z "${MONGODB_USERNAME}" ] || [ -z "${MONGODB_PASSWORD}" ]; then
        error "MongoDB username and password are required for non-interactive runs. ${hint}"
      fi
      ;;
    valkey)
      if [ -z "${VALKEY_PASSWORD}" ]; then
        error "Valkey password is required for non-interactive runs (use --db-password or DB_PASSWORD / VALKEY_PASSWORD). ${hint}"
      fi
      ;;
  esac
}

configure_pmm_agent() {
  prompt_if_empty PMM_SERVER_URL "PMM server URL (example: https://service_token:GLSA_TOKEN@pmm.example.com:443)" 1
  prompt_if_empty TECH "Technology to add (mysql/postgresql/mongodb/valkey)"

  require_db_creds_before_config_if_noninteractive

  local config_cmd=(pmm-admin config "--server-url=${PMM_SERVER_URL}")
  if [ "${PMM_SERVER_INSECURE_TLS}" = "1" ] || [ "${PMM_SERVER_INSECURE_TLS}" = "true" ]; then
    config_cmd+=(--server-insecure-tls)
  fi
  # pmm-admin config positionals are [<node-address>] [<node-type>] [<node-name>].
  # NODE_NAME without NODE_ADDRESS would shift "generic" into the address slot.
  if [ -n "${NODE_NAME}" ]; then
    local node_address="${NODE_ADDRESS:-$(detect_node_hostname)}"
    config_cmd+=("${node_address}" "generic" "${NODE_NAME}")
  elif [ -n "${NODE_ADDRESS}" ]; then
    config_cmd+=("${NODE_ADDRESS}")
  fi
  if [ "${PMM_CONFIG_FORCE}" = "1" ] || [ "${PMM_CONFIG_FORCE}" = "true" ]; then
    config_cmd+=(--force)
  fi

  log "Running pmm-admin config..."
  "${config_cmd[@]}"
}

apply_generic_inputs() {
  MYSQL_USERNAME="${MYSQL_USERNAME:-${DB_USER}}"
  MYSQL_PASSWORD="${MYSQL_PASSWORD:-${DB_PASSWORD}}"
  MYSQL_HOST="${MYSQL_HOST:-${DB_HOST}}"
  MYSQL_PORT="${MYSQL_PORT:-${DB_PORT}}"
  MYSQL_ADDRESS="${MYSQL_ADDRESS:-${DB_ADDRESS}}"
  MYSQL_SERVICE_NAME="${MYSQL_SERVICE_NAME:-${DB_SERVICE_NAME}}"
  MYSQL_SOCKET="${MYSQL_SOCKET:-${DB_SOCKET}}"

  POSTGRESQL_USERNAME="${POSTGRESQL_USERNAME:-${DB_USER}}"
  POSTGRESQL_PASSWORD="${POSTGRESQL_PASSWORD:-${DB_PASSWORD}}"
  POSTGRESQL_HOST="${POSTGRESQL_HOST:-${DB_HOST}}"
  POSTGRESQL_PORT="${POSTGRESQL_PORT:-${DB_PORT}}"
  POSTGRESQL_ADDRESS="${POSTGRESQL_ADDRESS:-${DB_ADDRESS}}"
  POSTGRESQL_SERVICE_NAME="${POSTGRESQL_SERVICE_NAME:-${DB_SERVICE_NAME}}"
  POSTGRESQL_DATABASE="${POSTGRESQL_DATABASE:-${DB_NAME}}"
  POSTGRESQL_SOCKET="${POSTGRESQL_SOCKET:-${DB_SOCKET}}"

  MONGODB_USERNAME="${MONGODB_USERNAME:-${DB_USER}}"
  MONGODB_PASSWORD="${MONGODB_PASSWORD:-${DB_PASSWORD}}"
  MONGODB_HOST="${MONGODB_HOST:-${DB_HOST}}"
  MONGODB_PORT="${MONGODB_PORT:-${DB_PORT}}"
  MONGODB_ADDRESS="${MONGODB_ADDRESS:-${DB_ADDRESS}}"
  MONGODB_SERVICE_NAME="${MONGODB_SERVICE_NAME:-${DB_SERVICE_NAME}}"
  MONGODB_AUTH_DB="${MONGODB_AUTH_DB:-${DB_AUTH_DB}}"
  MONGODB_SOCKET="${MONGODB_SOCKET:-${DB_SOCKET}}"

  VALKEY_USERNAME="${VALKEY_USERNAME:-${DB_USER}}"
  VALKEY_PASSWORD="${VALKEY_PASSWORD:-${DB_PASSWORD}}"
  VALKEY_HOST="${VALKEY_HOST:-${DB_HOST}}"
  VALKEY_PORT="${VALKEY_PORT:-${DB_PORT}}"
  VALKEY_ADDRESS="${VALKEY_ADDRESS:-${DB_ADDRESS}}"
  VALKEY_SERVICE_NAME="${VALKEY_SERVICE_NAME:-${DB_SERVICE_NAME}}"
  VALKEY_SOCKET="${VALKEY_SOCKET:-${DB_SOCKET}}"
}

add_mysql() {
  local db_cred_hint='Use --db-user and --db-password, or set DB_USER and DB_PASSWORD (MYSQL_* overrides if set). If you use sudo env, list DB_USER and DB_PASSWORD there; exports in your shell are not passed to the script.'
  prompt_if_empty MYSQL_USERNAME "MySQL username" 0 "${db_cred_hint}"
  prompt_if_empty MYSQL_PASSWORD "MySQL password" 1 "${db_cred_hint}"
  MYSQL_ADDRESS="${MYSQL_ADDRESS:-${MYSQL_HOST:-127.0.0.1}:${MYSQL_PORT:-3306}}"
  MYSQL_SERVICE_NAME="${MYSQL_SERVICE_NAME:-$(detect_node_hostname)-mysql}"
  local cmd=(pmm-admin add mysql "${MYSQL_SERVICE_NAME}" "${MYSQL_ADDRESS}" "--username=${MYSQL_USERNAME}" "--password=${MYSQL_PASSWORD}")
  if [ -n "${MYSQL_SOCKET}" ]; then
    cmd+=("--socket=${MYSQL_SOCKET}")
  fi
  log "Running pmm-admin add mysql..."
  "${cmd[@]}"
}

add_postgresql() {
  local db_cred_hint='Use --db-user and --db-password, or set DB_USER and DB_PASSWORD (POSTGRESQL_* overrides if set). If you use sudo env, list DB_USER and DB_PASSWORD there; exports in your shell are not passed to the script.'
  prompt_if_empty POSTGRESQL_USERNAME "PostgreSQL username" 0 "${db_cred_hint}"
  prompt_if_empty POSTGRESQL_PASSWORD "PostgreSQL password" 1 "${db_cred_hint}"
  POSTGRESQL_ADDRESS="${POSTGRESQL_ADDRESS:-${POSTGRESQL_HOST:-127.0.0.1}:${POSTGRESQL_PORT:-5432}}"
  POSTGRESQL_SERVICE_NAME="${POSTGRESQL_SERVICE_NAME:-$(detect_node_hostname)-postgresql}"
  local cmd=(pmm-admin add postgresql "${POSTGRESQL_SERVICE_NAME}" "${POSTGRESQL_ADDRESS}" "--username=${POSTGRESQL_USERNAME}" "--password=${POSTGRESQL_PASSWORD}")
  if [ -n "${POSTGRESQL_DATABASE}" ]; then
    cmd+=("--database=${POSTGRESQL_DATABASE}")
  fi
  if [ -n "${POSTGRESQL_SOCKET}" ]; then
    cmd+=("--socket=${POSTGRESQL_SOCKET}")
  fi
  log "Running pmm-admin add postgresql..."
  "${cmd[@]}"
}

add_mongodb() {
  local db_cred_hint='Use --db-user and --db-password, or set DB_USER and DB_PASSWORD (MONGODB_* overrides if set). If you use sudo env, list DB_USER and DB_PASSWORD there; exports in your shell are not passed to the script.'
  prompt_if_empty MONGODB_USERNAME "MongoDB username" 0 "${db_cred_hint}"
  prompt_if_empty MONGODB_PASSWORD "MongoDB password" 1 "${db_cred_hint}"
  MONGODB_ADDRESS="${MONGODB_ADDRESS:-${MONGODB_HOST:-127.0.0.1}:${MONGODB_PORT:-27017}}"
  MONGODB_SERVICE_NAME="${MONGODB_SERVICE_NAME:-$(detect_node_hostname)-mongodb}"
  local cmd=(pmm-admin add mongodb "${MONGODB_SERVICE_NAME}" "${MONGODB_ADDRESS}" "--username=${MONGODB_USERNAME}" "--password=${MONGODB_PASSWORD}")
  if [ -n "${MONGODB_AUTH_DB}" ]; then
    cmd+=("--authentication-database=${MONGODB_AUTH_DB}")
  fi
  if [ -n "${MONGODB_SOCKET}" ]; then
    cmd+=("--socket=${MONGODB_SOCKET}")
  fi
  log "Running pmm-admin add mongodb..."
  "${cmd[@]}"
}

add_valkey() {
  local db_cred_hint='Use --db-password or DB_PASSWORD (VALKEY_PASSWORD overrides if set). If you use sudo env, list DB_PASSWORD there; exports in your shell are not passed to the script.'
  prompt_if_empty VALKEY_PASSWORD "Valkey password" 1 "${db_cred_hint}"
  VALKEY_ADDRESS="${VALKEY_ADDRESS:-${VALKEY_HOST:-127.0.0.1}:${VALKEY_PORT:-6379}}"
  VALKEY_SERVICE_NAME="${VALKEY_SERVICE_NAME:-$(detect_node_hostname)-valkey}"
  local cmd=(pmm-admin add valkey "${VALKEY_SERVICE_NAME}" "${VALKEY_ADDRESS}" "--password=${VALKEY_PASSWORD}")
  if [ -n "${VALKEY_USERNAME}" ]; then
    cmd+=("--username=${VALKEY_USERNAME}")
  fi
  if [ -n "${VALKEY_SOCKET}" ]; then
    cmd+=("--socket=${VALKEY_SOCKET}")
  fi
  log "Running pmm-admin add valkey..."
  "${cmd[@]}"
}

add_service() {
  # IMPORTANT: keep this list in sync with:
  #   - installTokenTechnologies in managed/services/management/install_token.go
  #   - the Technology union in ui/apps/pmm/src/pages/install-client/InstallClientPage.utils.ts
  # If you add a tech here, also add a matching add_<tech> function above and the require_*
  # branch in require_db_creds_before_config_if_noninteractive.
  case "${TECH}" in
    mysql)
      add_mysql
      ;;
    postgresql)
      add_postgresql
      ;;
    mongodb)
      add_mongodb
      ;;
    valkey)
      add_valkey
      ;;
    *)
      error "Unsupported TECH '${TECH}'. Supported values: mysql, postgresql, mongodb, valkey."
      ;;
  esac
}

# Print a tailored recovery hint when `pmm-admin add` fails after `pmm-admin config`
# has already registered the node. The most common cause we see in the field is
# wrong DB credentials; the second most common is leftover state from a previous
# attempt. Either way the user wants `--force` on the next run + corrected creds.
report_add_service_failure() {
  local exit_code="$1"
  echo >&2
  log "ERROR: 'pmm-admin add ${TECH}' failed (exit ${exit_code}) after the node was already registered with PMM Server."
  log "       The node is now visible on PMM Server but no service is attached to it."
  log "       Most common causes:"
  log "         * Wrong DB credentials → fix DB_USER / DB_PASSWORD (or --db-user / --db-password) and re-run."
  log "         * Service already attached from a prior attempt → re-run with --force (or PMM_CONFIG_FORCE=1)"
  log "           which removes the previous node registration and its services on the server before re-registering."
  log "       For MongoDB also check --db-auth-db / DB_AUTH_DB; for PostgreSQL check --db-name / DB_NAME."
  exit "${exit_code}"
}

main() {
  require_root
  install_pmm_client
  apply_generic_inputs
  ensure_pmm_agent_running
  configure_pmm_agent
  # Disable -e for the add step so we can intercept its non-zero exit, print a
  # helpful recovery message, and propagate the original status. `set -E` would
  # work too but only on bash >= 4 and changes broader trap semantics.
  set +e
  add_service
  local rc=$?
  set -e
  if [ "${rc}" -ne 0 ]; then
    report_add_service_failure "${rc}"
  fi
  log "PMM client setup completed successfully."
}

main "$@"
