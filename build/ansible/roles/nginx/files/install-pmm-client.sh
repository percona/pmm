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
  --force                       Pass --force to pmm-admin config (removes existing node name and its services on the server, then registers again). When omitted, pmm-admin config is skipped automatically if pmm-agent is already set up on this node; if PMM Server then rejects the agent's stored token (e.g. an expired install token), it is refreshed from --pmm-server-url non-destructively (existing services kept) and the add is retried. Use --force only for a full re-registration.

Generic DB options (mapped per technology):
  --db-user USER
  --db-password PASSWORD
  --db-host HOST
  --db-port PORT
  --db-name NAME                DB name for PostgreSQL
  --db-address HOST:PORT        Explicit service address
  --db-service-name NAME        PMM service name (default: <hostname>-<tech>, with -<port> or -<socket-id> suffix when the port is non-default, --db-port is set, or a socket path is used)
  --db-auth-db NAME             MongoDB auth database
  --db-socket PATH              Socket path for MySQL/PostgreSQL/MongoDB/Valkey
  --db-query-source SOURCE      MySQL QAN source: slowlog, perfschema, or none (passed to pmm-admin add mysql as --query-source; default: pmm-admin default slowlog)

Environment variables are also supported.
Priority is: flags > env vars > interactive prompt.
When stdin is a terminal, database prompts are skipped if credentials are already
set from flags or environment (DB_USER / DB_PASSWORD and per-tech MYSQL_*,
POSTGRESQL_* / … after apply_generic_inputs). MYSQL_QUERY_SOURCE overrides
DB_QUERY_SOURCE for MySQL. Use sudo -E bash … when running as root so your
exports reach the script.

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
PMM_CONFIG_SKIPPED=0
# Combined stdout+stderr of the last `pmm-admin add` so we can tell a PMM Server
# authentication failure (expired/invalid agent token) apart from a database error.
ADD_OUTPUT=""
# Set to 1 once a non-destructive token refresh has actually been attempted, so error
# messages don't claim the auto-refresh "did not help" on paths where it never ran.
PMM_TOKEN_REFRESH_ATTEMPTED=0

DB_USER="${DB_USER:-}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_HOST="${DB_HOST:-}"
DB_PORT="${DB_PORT:-}"
DB_NAME="${DB_NAME:-}"
DB_ADDRESS="${DB_ADDRESS:-}"
DB_SERVICE_NAME="${DB_SERVICE_NAME:-}"
DB_AUTH_DB="${DB_AUTH_DB:-}"
DB_SOCKET="${DB_SOCKET:-}"
DB_QUERY_SOURCE="${DB_QUERY_SOURCE:-}"

MYSQL_USERNAME="${MYSQL_USERNAME:-}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-}"
MYSQL_HOST="${MYSQL_HOST:-}"
MYSQL_PORT="${MYSQL_PORT:-}"
MYSQL_ADDRESS="${MYSQL_ADDRESS:-}"
MYSQL_SERVICE_NAME="${MYSQL_SERVICE_NAME:-}"
MYSQL_SOCKET="${MYSQL_SOCKET:-}"
MYSQL_QUERY_SOURCE="${MYSQL_QUERY_SOURCE:-}"

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
    --db-query-source)
      DB_QUERY_SOURCE="${2:-}"
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
    log "Created empty pmm-agent config: ${PMM_AGENT_CONFIG_FILE}"
  fi
  chmod 0660 "${PMM_AGENT_CONFIG_FILE}" || true
  if id -u pmm-agent >/dev/null 2>&1; then
    chown pmm-agent:pmm-agent "${PMM_AGENT_CONFIG_FILE}" 2>/dev/null || \
      chown pmm-agent "${PMM_AGENT_CONFIG_FILE}" 2>/dev/null || true
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
  # the nohup fallback from being a privilege regression vs. systemd.
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
# pmm-admin config/add.
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

# True when pmm-agent is running and has registered with PMM Server locally.
# Returns false when --force is set (caller should run pmm-admin config).
pmm_agent_already_configured() {
  if [ "${PMM_CONFIG_FORCE}" = "1" ] || [ "${PMM_CONFIG_FORCE}" = "true" ]; then
    return 1
  fi
  if ! command -v pmm-admin >/dev/null 2>&1; then
    return 1
  fi
  if ! pmm_agent_listening; then
    return 1
  fi
  local agent_id
  agent_id="$(pmm-admin status 2>/dev/null | sed -n 's/^Agent ID *: *\(.*\)/\1/p' | head -1 | tr -d '[:space:]')"
  [ -n "${agent_id}" ]
}

# Port from a host:port address (empty when no colon is present).
extract_port_from_address() {
  local address="$1"
  case "${address}" in
    *:*)
      printf '%s' "${address##*:}"
      ;;
  esac
}

# Disambiguator for default service names, e.g. "-3307" or "-mysql2" (socket basename).
service_name_disambiguator() {
  local default_port="$1"
  local port="$2"
  local port_explicit="$3"
  local socket="$4"
  local effective="${port:-${default_port}}"

  if [ -n "${socket}" ]; then
    local sock_id="${socket##*/}"
    sock_id="${sock_id%.sock}"
    printf '-%s' "${sock_id}"
    return
  fi

  if [ "${port_explicit}" = "1" ] || [ "${effective}" != "${default_port}" ]; then
    printf '-%s' "${effective}"
  fi
}

default_db_service_name() {
  local tech_label="$1"
  local default_port="$2"
  local port="$3"
  local port_explicit="$4"
  local socket="$5"
  local suffix
  suffix="$(service_name_disambiguator "${default_port}" "${port}" "${port_explicit}" "${socket}")"
  printf '%s-%s%s' "$(detect_node_hostname)" "${tech_label}" "${suffix}"
}

configure_pmm_agent() {
  prompt_if_empty TECH "Technology to add (mysql/postgresql/mongodb/valkey)"

  if pmm_agent_already_configured; then
    log "pmm-agent is already configured with PMM Server; skipping re-registration to preserve existing services."
    # We intentionally keep the agent's existing (durable) token instead of overwriting
    # it on every run. If that stored token turns out to be rejected by PMM Server (e.g.
    # an old, short-lived install token that has since expired), add_service triggers a
    # non-destructive refresh from --pmm-server-url and retries — see should_attempt_token_refresh.
    if [ -n "${PMM_SERVER_URL}" ]; then
      log "If PMM Server rejects the stored token, it will be refreshed from --pmm-server-url automatically (no services removed)."
    fi
    log "(Use --force only to fully re-register this node; that removes the node and ALL its services on the server.)"
    PMM_CONFIG_SKIPPED=1
    return 0
  fi

  prompt_if_empty PMM_SERVER_URL "PMM server URL (example: https://service_token:GLSA_TOKEN@pmm.example.com:443)" 1

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
  MYSQL_QUERY_SOURCE="${MYSQL_QUERY_SOURCE:-${DB_QUERY_SOURCE}}"

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

validate_mysql_query_source() {
  case "${MYSQL_QUERY_SOURCE}" in
    slowlog|perfschema|none)
      ;;
    '')
      ;;
    *)
      error "Unsupported MySQL query source '${MYSQL_QUERY_SOURCE}'. Supported: slowlog, perfschema, none."
      ;;
  esac
}

# Run `pmm-admin add ...` while keeping a copy of its combined output in ADD_OUTPUT
# (still streamed live to the user). The captured text lets us classify failures —
# a PMM Server auth error reads very differently from a DB credentials error — and
# never contains the DB password (that's only on the command line, not echoed back).
run_pmm_admin_add() {
  local out_file rc
  if out_file="$(mktemp 2>/dev/null)"; then
    # mktemp gives a fresh 0600 file, so there is no predictable-path/symlink race.
    "$@" 2>&1 | tee "${out_file}"
    rc=${PIPESTATUS[0]}
    ADD_OUTPUT="$(cat "${out_file}" 2>/dev/null || true)"
    rm -f "${out_file}"
  else
    # mktemp unavailable (extremely rare on a real distro): run directly with normal
    # stdout/stderr rather than redirecting everything to one stream. ADD_OUTPUT stays
    # empty, so the auth-failure auto-refresh degrades to the generic error path for this
    # edge case — acceptable, and the output the user sees is unchanged.
    ADD_OUTPUT=""
    "$@"
    rc=$?
  fi
  return "${rc}"
}

# True when the last `pmm-admin add` failed because PMM Server rejected the agent's
# token, not because of bad DB credentials or a duplicate service name. We match only
# PMM-Server-specific markers: the Grafana phrase "Auth method is not service account
# token", and ". Please check username and password." which pmm-admin appends to any
# HTTP 401 from PMM Server (non-JSON mode). We deliberately do NOT match a bare
# "Unauthorized" / "401": database drivers (e.g. MongoDB) emit those for DB-credential
# failures, which would misclassify a DB error as a server-token error.
add_failure_is_auth() {
  case "${ADD_OUTPUT}" in
    *"service account token"*|*"Please check username and password"*)
      return 0
      ;;
  esac
  return 1
}

# Decide whether to auto-recover from an add failure by refreshing the token.
# Only when: the failure is a server-side auth error, the user did not ask for a
# (destructive) --force, and we actually have a --pmm-server-url to read a fresh
# token from. When config was just run (not skipped) and still auth-failed, the
# supplied token itself is bad, so refreshing with it again would not help.
should_attempt_token_refresh() {
  add_failure_is_auth || return 1
  if [ "${PMM_CONFIG_FORCE}" = "1" ] || [ "${PMM_CONFIG_FORCE}" = "true" ]; then
    return 1
  fi
  [ "${PMM_CONFIG_SKIPPED}" = "1" ] || return 1
  [ -n "${PMM_SERVER_URL}" ] || return 1
  return 0
}

# Update the agent's stored PMM Server token from --pmm-server-url WITHOUT
# re-registering the node. PMM_AGENT_SETUP_SKIP_REGISTRATION makes the `pmm-agent
# setup` that `pmm-admin config` execs only rewrite credentials and reload — it does
# not delete and recreate the node, so existing services on this node are preserved
# (unlike --force). The freshly supplied token then authenticates the retried add.
#
# Limitation: because registration is skipped, the agent does NOT receive a durable
# node service-account token from PMM Server here — it adopts whatever token is in
# --pmm-server-url. With the one-step UI that token is short-lived (~15 min). On a PMM
# Server that mints a durable node token at registration (current behaviour), this only
# runs as a recovery and the durable token is what normally authenticates the agent. On
# an older server the agent is left with the short-lived token again, so a much later
# re-add may need a freshly generated command. This is still strictly better than the
# previous behaviour, where the only recovery was a destructive --force.
refresh_pmm_agent_token() {
  log "PMM Server rejected the agent's stored token (it may have expired)."
  log "Refreshing the token from --pmm-server-url without removing existing services..."
  local refresh_cmd=(pmm-admin config "--server-url=${PMM_SERVER_URL}")
  if [ "${PMM_SERVER_INSECURE_TLS}" = "1" ] || [ "${PMM_SERVER_INSECURE_TLS}" = "true" ]; then
    refresh_cmd+=(--server-insecure-tls)
  fi
  PMM_AGENT_SETUP_SKIP_REGISTRATION=1 "${refresh_cmd[@]}"
}

add_mysql() {
  local db_cred_hint='Use --db-user and --db-password, or set DB_USER and DB_PASSWORD (MYSQL_* overrides if set). If you use sudo env, list DB_USER and DB_PASSWORD there; exports in your shell are not passed to the script.'
  prompt_if_empty MYSQL_USERNAME "MySQL username" 0 "${db_cred_hint}"
  prompt_if_empty MYSQL_PASSWORD "MySQL password" 1 "${db_cred_hint}"
  local port_explicit=0
  if [ -n "${DB_PORT}" ] || [ -n "${MYSQL_PORT}" ]; then
    port_explicit=1
  fi
  if [ -z "${MYSQL_PORT}" ] && [ -n "${MYSQL_ADDRESS}" ]; then
    MYSQL_PORT="$(extract_port_from_address "${MYSQL_ADDRESS}")"
  fi
  MYSQL_ADDRESS="${MYSQL_ADDRESS:-${MYSQL_HOST:-127.0.0.1}:${MYSQL_PORT:-3306}}"
  if [ -z "${MYSQL_PORT}" ]; then
    MYSQL_PORT="$(extract_port_from_address "${MYSQL_ADDRESS}")"
  fi
  MYSQL_SERVICE_NAME="${MYSQL_SERVICE_NAME:-$(default_db_service_name mysql 3306 "${MYSQL_PORT}" "${port_explicit}" "${MYSQL_SOCKET}")}"
  validate_mysql_query_source
  local cmd=(pmm-admin add mysql "${MYSQL_SERVICE_NAME}" "${MYSQL_ADDRESS}" "--username=${MYSQL_USERNAME}" "--password=${MYSQL_PASSWORD}")
  if [ -n "${MYSQL_SOCKET}" ]; then
    cmd+=("--socket=${MYSQL_SOCKET}")
  fi
  if [ -n "${MYSQL_QUERY_SOURCE}" ]; then
    cmd+=("--query-source=${MYSQL_QUERY_SOURCE}")
  fi
  log "Running pmm-admin add mysql..."
  run_pmm_admin_add "${cmd[@]}"
}

add_postgresql() {
  local db_cred_hint='Use --db-user and --db-password, or set DB_USER and DB_PASSWORD (POSTGRESQL_* overrides if set). If you use sudo env, list DB_USER and DB_PASSWORD there; exports in your shell are not passed to the script.'
  prompt_if_empty POSTGRESQL_USERNAME "PostgreSQL username" 0 "${db_cred_hint}"
  prompt_if_empty POSTGRESQL_PASSWORD "PostgreSQL password" 1 "${db_cred_hint}"
  local port_explicit=0
  if [ -n "${DB_PORT}" ] || [ -n "${POSTGRESQL_PORT}" ]; then
    port_explicit=1
  fi
  if [ -z "${POSTGRESQL_PORT}" ] && [ -n "${POSTGRESQL_ADDRESS}" ]; then
    POSTGRESQL_PORT="$(extract_port_from_address "${POSTGRESQL_ADDRESS}")"
  fi
  POSTGRESQL_ADDRESS="${POSTGRESQL_ADDRESS:-${POSTGRESQL_HOST:-127.0.0.1}:${POSTGRESQL_PORT:-5432}}"
  if [ -z "${POSTGRESQL_PORT}" ]; then
    POSTGRESQL_PORT="$(extract_port_from_address "${POSTGRESQL_ADDRESS}")"
  fi
  POSTGRESQL_SERVICE_NAME="${POSTGRESQL_SERVICE_NAME:-$(default_db_service_name postgresql 5432 "${POSTGRESQL_PORT}" "${port_explicit}" "${POSTGRESQL_SOCKET}")}"
  local cmd=(pmm-admin add postgresql "${POSTGRESQL_SERVICE_NAME}" "${POSTGRESQL_ADDRESS}" "--username=${POSTGRESQL_USERNAME}" "--password=${POSTGRESQL_PASSWORD}")
  if [ -n "${POSTGRESQL_DATABASE}" ]; then
    cmd+=("--database=${POSTGRESQL_DATABASE}")
  fi
  if [ -n "${POSTGRESQL_SOCKET}" ]; then
    cmd+=("--socket=${POSTGRESQL_SOCKET}")
  fi
  log "Running pmm-admin add postgresql..."
  run_pmm_admin_add "${cmd[@]}"
}

add_mongodb() {
  local db_cred_hint='Use --db-user and --db-password, or set DB_USER and DB_PASSWORD (MONGODB_* overrides if set). If you use sudo env, list DB_USER and DB_PASSWORD there; exports in your shell are not passed to the script.'
  prompt_if_empty MONGODB_USERNAME "MongoDB username" 0 "${db_cred_hint}"
  prompt_if_empty MONGODB_PASSWORD "MongoDB password" 1 "${db_cred_hint}"
  local port_explicit=0
  if [ -n "${DB_PORT}" ] || [ -n "${MONGODB_PORT}" ]; then
    port_explicit=1
  fi
  if [ -z "${MONGODB_PORT}" ] && [ -n "${MONGODB_ADDRESS}" ]; then
    MONGODB_PORT="$(extract_port_from_address "${MONGODB_ADDRESS}")"
  fi
  MONGODB_ADDRESS="${MONGODB_ADDRESS:-${MONGODB_HOST:-127.0.0.1}:${MONGODB_PORT:-27017}}"
  if [ -z "${MONGODB_PORT}" ]; then
    MONGODB_PORT="$(extract_port_from_address "${MONGODB_ADDRESS}")"
  fi
  MONGODB_SERVICE_NAME="${MONGODB_SERVICE_NAME:-$(default_db_service_name mongodb 27017 "${MONGODB_PORT}" "${port_explicit}" "${MONGODB_SOCKET}")}"
  local cmd=(pmm-admin add mongodb "${MONGODB_SERVICE_NAME}" "${MONGODB_ADDRESS}" "--username=${MONGODB_USERNAME}" "--password=${MONGODB_PASSWORD}")
  if [ -n "${MONGODB_AUTH_DB}" ]; then
    cmd+=("--authentication-database=${MONGODB_AUTH_DB}")
  fi
  if [ -n "${MONGODB_SOCKET}" ]; then
    cmd+=("--socket=${MONGODB_SOCKET}")
  fi
  log "Running pmm-admin add mongodb..."
  run_pmm_admin_add "${cmd[@]}"
}

add_valkey() {
  local db_cred_hint='Use --db-password or DB_PASSWORD (VALKEY_PASSWORD overrides if set). If you use sudo env, list DB_PASSWORD there; exports in your shell are not passed to the script.'
  prompt_if_empty VALKEY_PASSWORD "Valkey password" 1 "${db_cred_hint}"
  local port_explicit=0
  if [ -n "${DB_PORT}" ] || [ -n "${VALKEY_PORT}" ]; then
    port_explicit=1
  fi
  if [ -z "${VALKEY_PORT}" ] && [ -n "${VALKEY_ADDRESS}" ]; then
    VALKEY_PORT="$(extract_port_from_address "${VALKEY_ADDRESS}")"
  fi
  VALKEY_ADDRESS="${VALKEY_ADDRESS:-${VALKEY_HOST:-127.0.0.1}:${VALKEY_PORT:-6379}}"
  if [ -z "${VALKEY_PORT}" ]; then
    VALKEY_PORT="$(extract_port_from_address "${VALKEY_ADDRESS}")"
  fi
  VALKEY_SERVICE_NAME="${VALKEY_SERVICE_NAME:-$(default_db_service_name valkey 6379 "${VALKEY_PORT}" "${port_explicit}" "${VALKEY_SOCKET}")}"
  local cmd=(pmm-admin add valkey "${VALKEY_SERVICE_NAME}" "${VALKEY_ADDRESS}" "--password=${VALKEY_PASSWORD}")
  if [ -n "${VALKEY_USERNAME}" ]; then
    cmd+=("--username=${VALKEY_USERNAME}")
  fi
  if [ -n "${VALKEY_SOCKET}" ]; then
    cmd+=("--socket=${VALKEY_SOCKET}")
  fi
  log "Running pmm-admin add valkey..."
  run_pmm_admin_add "${cmd[@]}"
}

add_service() {
  # IMPORTANT: keep this list in sync with:
  #   - SUPPORTED_TECHNOLOGIES in ui/apps/pmm/src/api/installToken.ts
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

# Print a tailored recovery hint when `pmm-admin add` fails.
report_add_service_failure() {
  local exit_code="$1"
  echo >&2
  log "ERROR: 'pmm-admin add ${TECH}' failed (exit ${exit_code})."

  # Server-side authentication failure: this is NOT a database problem, and --force
  # would only make things worse by deleting this node and all of its services.
  if add_failure_is_auth; then
    log "       PMM Server rejected the agent's token (authentication error) — this is NOT a"
    log "       database credentials problem."
    if [ "${PMM_TOKEN_REFRESH_ATTEMPTED}" = "1" ]; then
      log "       The automatic non-destructive token refresh did not resolve it, so the token"
      log "       supplied via --pmm-server-url is most likely expired too (install tokens are short-lived)."
    else
      log "       The token is most likely expired (install tokens are short-lived, ~15 min)."
    fi
    log "       To recover WITHOUT losing other services already on this node:"
    log "         * Generate a fresh install command/token in the PMM UI and re-run this script"
    log "           with the new --pmm-server-url."
    log "       Do NOT use --force unless you intend to remove this node and ALL its services from PMM Server."
    exit "${exit_code}"
  fi

  log "       Most common causes:"
  log "         * Wrong DB credentials → fix DB_USER / DB_PASSWORD (or --db-user / --db-password) and re-run."
  if [ "${PMM_CONFIG_SKIPPED}" = "1" ]; then
    log "         * Service name already in use on this node → set a unique --db-service-name or a"
    log "           different --db-port so the script default name includes a port suffix."
    log "           Do not use --force here; that removes the node and all existing services."
  else
    log "         * The node was registered but no service was added → fix credentials or service"
    log "           options and re-run. To replace the node registration entirely, use --force"
    log "           (removes the node and its services on PMM Server before re-registering)."
  fi
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
  # If the add failed only because PMM Server rejected the agent's stored token,
  # refresh that token from --pmm-server-url (non-destructively, keeping existing
  # services) and retry once — instead of pushing the user toward a destructive --force.
  if [ "${rc}" -ne 0 ] && should_attempt_token_refresh; then
    PMM_TOKEN_REFRESH_ATTEMPTED=1
    refresh_pmm_agent_token
    local refresh_rc=$?
    if [ "${refresh_rc}" -eq 0 ]; then
      log "Retrying 'pmm-admin add ${TECH}' with the refreshed token..."
      add_service
      rc=$?
    fi
  fi
  set -e
  if [ "${rc}" -ne 0 ]; then
    report_add_service_failure "${rc}"
  fi
  if [ "${PMM_TOKEN_REFRESH_ATTEMPTED}" = "1" ]; then
    log "Note: the agent's token was refreshed from --pmm-server-url. If a later re-add fails"
    log "      with an authentication error, generate a fresh install command in the PMM UI and re-run."
  fi
  log "PMM client setup completed successfully."
}

main "$@"
