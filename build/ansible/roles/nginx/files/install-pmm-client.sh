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

while [[ $# -gt 0 ]]; do
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
  if [[ "${EUID}" -ne 0 ]]; then
    error "Run this script as root (for package installation). Example: curl -fsSLk ... | sudo -E env ... bash -s --"
  fi
}

prompt_if_empty() {
  local var_name="$1"
  local prompt_label="$2"
  local secret="${3:-0}"
  local hint="${4:-}"
  local value="${!var_name:-}"

  if [[ -n "${value}" ]]; then
    return
  fi

  if [[ "${secret}" == "1" ]]; then
    read -r -s -p "${prompt_label}: " value
    echo
  else
    read -r -p "${prompt_label}: " value
  fi

  if [[ -z "${value}" ]]; then
    if [[ -n "${hint}" ]]; then
      error "${prompt_label} is required. ${hint}"
    else
      error "${prompt_label} is required."
    fi
  fi

  printf -v "${var_name}" '%s' "${value}"
}

detect_os_family() {
  if [[ -f /etc/os-release ]]; then
    # shellcheck source=/dev/null
    . /etc/os-release
    case "${ID:-}" in
      debian|ubuntu|linuxmint|kali|pop|astra)
        echo "debian"
        return
        ;;
      fedora|rhel|centos|rocky|almalinux|ol|amzn|virtuozzo|vzlinux|mageia)
        echo "el"
        return
        ;;
    esac
    case ",${ID_LIKE:-}," in
      *,debian*|*,ubuntu*)
        echo "debian"
        return
        ;;
      *,rhel*|*,fedora*|*,centos*)
        echo "el"
        return
        ;;
    esac
  fi
  if [[ -f /etc/redhat-release ]] || [[ -f /etc/centos-release ]] || [[ -f /etc/fedora-release ]]; then
    echo "el"
    return
  fi
  if [[ -f /etc/debian_version ]]; then
    echo "debian"
    return
  fi
  error "Unsupported OS. Supported families: RHEL/CentOS/Rocky/Fedora/Amazon Linux/Oracle Linux (RPM) and Debian/Ubuntu (DEB)."
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

  if [[ "${os_family}" == "el" ]]; then
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

# When stdin is not a terminal (e.g. curl ... | bash), prompts cannot be used for DB
# credentials. Fail before pmm-admin config so we do not register the node and then fail on add.
require_db_creds_before_config_if_noninteractive() {
  if [[ -t 0 ]]; then
    return 0
  fi

  apply_generic_inputs

  local hint='This install is non-interactive (stdin is not a terminal, e.g. curl ... | bash), so database credentials cannot be prompted. Set them before the agent registers with PMM Server: use --db-user and --db-password, or DB_USER and DB_PASSWORD (include them in sudo env if you use sudo env; use sudo -E to preserve exports).'

  case "${TECH}" in
    mysql)
      if [[ -z "${MYSQL_USERNAME}" || -z "${MYSQL_PASSWORD}" ]]; then
        error "MySQL username and password are required for non-interactive runs. ${hint}"
      fi
      ;;
    postgresql)
      if [[ -z "${POSTGRESQL_USERNAME}" || -z "${POSTGRESQL_PASSWORD}" ]]; then
        error "PostgreSQL username and password are required for non-interactive runs. ${hint}"
      fi
      ;;
    mongodb)
      if [[ -z "${MONGODB_USERNAME}" || -z "${MONGODB_PASSWORD}" ]]; then
        error "MongoDB username and password are required for non-interactive runs. ${hint}"
      fi
      ;;
    valkey)
      if [[ -z "${VALKEY_PASSWORD}" ]]; then
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
  if [[ "${PMM_SERVER_INSECURE_TLS}" == "1" || "${PMM_SERVER_INSECURE_TLS}" == "true" ]]; then
    config_cmd+=(--server-insecure-tls)
  fi
  if [[ -n "${NODE_ADDRESS}" ]]; then
    config_cmd+=("${NODE_ADDRESS}")
  fi
  if [[ -n "${NODE_NAME}" ]]; then
    config_cmd+=("generic" "${NODE_NAME}")
  fi
  if [[ "${PMM_CONFIG_FORCE}" == "1" || "${PMM_CONFIG_FORCE}" == "true" ]]; then
    config_cmd+=(--force)
  fi

  log "Running pmm-admin config..."
  "${config_cmd[@]}"
}

apply_generic_inputs() {
  DB_USER="${DB_USER:-}"
  DB_PASSWORD="${DB_PASSWORD:-}"
  DB_HOST="${DB_HOST:-}"
  DB_PORT="${DB_PORT:-}"
  DB_ADDRESS="${DB_ADDRESS:-}"
  DB_SERVICE_NAME="${DB_SERVICE_NAME:-}"
  DB_AUTH_DB="${DB_AUTH_DB:-}"
  DB_SOCKET="${DB_SOCKET:-}"

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
  MYSQL_SERVICE_NAME="${MYSQL_SERVICE_NAME:-$(hostname)-mysql}"
  local cmd=(pmm-admin add mysql "${MYSQL_SERVICE_NAME}" "${MYSQL_ADDRESS}" "--username=${MYSQL_USERNAME}" "--password=${MYSQL_PASSWORD}")
  if [[ -n "${MYSQL_SOCKET}" ]]; then
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
  POSTGRESQL_SERVICE_NAME="${POSTGRESQL_SERVICE_NAME:-$(hostname)-postgresql}"
  local cmd=(pmm-admin add postgresql "${POSTGRESQL_SERVICE_NAME}" "${POSTGRESQL_ADDRESS}" "--username=${POSTGRESQL_USERNAME}" "--password=${POSTGRESQL_PASSWORD}")
  if [[ -n "${POSTGRESQL_DATABASE}" ]]; then
    cmd+=("--database=${POSTGRESQL_DATABASE}")
  fi
  if [[ -n "${POSTGRESQL_SOCKET}" ]]; then
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
  MONGODB_SERVICE_NAME="${MONGODB_SERVICE_NAME:-$(hostname)-mongodb}"
  local cmd=(pmm-admin add mongodb "${MONGODB_SERVICE_NAME}" "${MONGODB_ADDRESS}" "--username=${MONGODB_USERNAME}" "--password=${MONGODB_PASSWORD}")
  if [[ -n "${MONGODB_AUTH_DB}" ]]; then
    cmd+=("--authentication-database=${MONGODB_AUTH_DB}")
  fi
  if [[ -n "${MONGODB_SOCKET}" ]]; then
    cmd+=("--socket=${MONGODB_SOCKET}")
  fi
  log "Running pmm-admin add mongodb..."
  "${cmd[@]}"
}

add_valkey() {
  local db_cred_hint='Use --db-password or DB_PASSWORD (VALKEY_PASSWORD overrides if set). If you use sudo env, list DB_PASSWORD there; exports in your shell are not passed to the script.'
  prompt_if_empty VALKEY_PASSWORD "Valkey password" 1 "${db_cred_hint}"
  VALKEY_ADDRESS="${VALKEY_ADDRESS:-${VALKEY_HOST:-127.0.0.1}:${VALKEY_PORT:-6379}}"
  VALKEY_SERVICE_NAME="${VALKEY_SERVICE_NAME:-$(hostname)-valkey}"
  local cmd=(pmm-admin add valkey "${VALKEY_SERVICE_NAME}" "${VALKEY_ADDRESS}" "--password=${VALKEY_PASSWORD}")
  if [[ -n "${VALKEY_USERNAME}" ]]; then
    cmd+=("--username=${VALKEY_USERNAME}")
  fi
  if [[ -n "${VALKEY_SOCKET}" ]]; then
    cmd+=("--socket=${VALKEY_SOCKET}")
  fi
  log "Running pmm-admin add valkey..."
  "${cmd[@]}"
}

add_service() {
  apply_generic_inputs

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

main() {
  require_root
  install_pmm_client
  configure_pmm_agent
  add_service
  log "PMM client setup completed successfully."
}

main "$@"
