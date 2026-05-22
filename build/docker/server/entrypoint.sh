#!/bin/bash
set -o errexit

declare PMM_DISTRIBUTION_METHOD="${PMM_DISTRIBUTION_METHOD:-docker}"
declare CURRENT_GID CURRENT_UID CURRENT_USER

# Returns 0 (true) if the given variable is set to "1" or "true".
is_enabled() { [ "$1" = "1" ] || [ "$1" = "true" ]; }
declare POSTGRES_DATA_DIR="/srv/postgres14"
declare POSTGRES_PASSWORD_FILE="/srv/.postgres_password"

# Get current user info - handle cases where user doesn't exist in passwd
CURRENT_UID=$(id -u)
CURRENT_GID=$(id -g)
if whoami &> /dev/null; then
    CURRENT_USER=$(whoami)
else
    CURRENT_USER="user-${CURRENT_UID}"
fi
echo "Running as UID ${CURRENT_UID}"

if [ ! -w /srv ]; then
    echo "FATAL: /srv is not writable for ${CURRENT_USER} user." >&2
    echo "Please make sure that /srv is owned by uid ${CURRENT_UID} and gid ${CURRENT_GID} and try again." >&2
    echo "You can change ownership by running: sudo chown -R ${CURRENT_UID}:${CURRENT_GID} /srv" >&2
    exit 1
fi

if [ "$CURRENT_UID" != "1000" ] || [ "$CURRENT_GID" != "0" ]; then
    echo "Running as UID:GID $CURRENT_UID:$CURRENT_GID, setting up for arbitrary UID..."

    # Try NSS wrapper first if available
    declare NSS_WRAPPER_LIB="/usr/lib64/libnss_wrapper.so"
    if [ ! -f "$NSS_WRAPPER_LIB" ]; then
        echo "Fatal: NSS wrapper library not found at $NSS_WRAPPER_LIB, exiting..."
        exit 1
    fi

    echo "Setting up NSS wrapper..."
    declare NSS_WRAPPER_PASSWD NSS_WRAPPER_GROUP
    # Set up NSS wrapper for arbitrary UID support
    NSS_WRAPPER_PASSWD=$(mktemp)
    NSS_WRAPPER_GROUP=$(mktemp)
    export NSS_WRAPPER_PASSWD NSS_WRAPPER_GROUP

    # Cleanup temp files on exit
    cleanup_nss_wrapper() {
        [ -f "$NSS_WRAPPER_PASSWD" ] && rm -f "$NSS_WRAPPER_PASSWD"
        [ -f "$NSS_WRAPPER_GROUP" ] && rm -f "$NSS_WRAPPER_GROUP"
    }
    trap cleanup_nss_wrapper EXIT

    # Copy existing passwd and group entries
    cat /etc/passwd > "$NSS_WRAPPER_PASSWD"
    cat /etc/group > "$NSS_WRAPPER_GROUP"

    # Add current user if not exists (suppress errors if NSS wrapper is not yet active)
    if ! getent passwd "$CURRENT_UID" > /dev/null 2>&1; then
        echo "${CURRENT_USER}:x:${CURRENT_UID}:${CURRENT_GID}:PMM User:/srv:/bin/bash" >> "$NSS_WRAPPER_PASSWD"
    fi

    # Add current group if not exists (suppress errors if NSS wrapper is not yet active)
    if ! getent group "$CURRENT_GID" > /dev/null 2>&1; then
        echo "${CURRENT_USER}:x:${CURRENT_GID}:" >> "$NSS_WRAPPER_GROUP"
    fi

    # Fix LD_PRELOAD assignment to avoid leading colon
    if [ -n "$LD_PRELOAD" ]; then
        export LD_PRELOAD="$NSS_WRAPPER_LIB:$LD_PRELOAD"
    else
        export LD_PRELOAD="$NSS_WRAPPER_LIB"
    fi
    echo "NSS wrapper enabled with $NSS_WRAPPER_LIB"
fi

# Initialize /srv if empty
declare DIST_FILE=/srv/pmm-distribution
if [ ! -f "$DIST_FILE" ]; then
    echo -n "$PMM_DISTRIBUTION_METHOD" > "$DIST_FILE"
    echo "Initializing /srv..."
    mkdir -p /srv/{backup,clickhouse,grafana/plugins,logs,nginx,otelcol,prometheus/rules,victoriametrics}
    echo "Copying grafana plugins and the VERSION file..."
    cp -r /usr/share/percona-dashboards/panels/* /srv/grafana/plugins

    if is_enabled "$PMM_HA_ENABLE"; then
        echo "Skipping embedded PostgreSQL initialization in HA mode."
    elif is_enabled "$PMM_DISABLE_BUILTIN_POSTGRES"; then
        echo "Skipping embedded PostgreSQL initialization (builtin PostgreSQL is disabled)."
    else
        echo "Initializing Postgres..."
        install -d -m 750 "$POSTGRES_DATA_DIR"

        # Generate a random password for postgres superuser
        declare POSTGRES_PASSWORD
        POSTGRES_PASSWORD=$(openssl rand -hex 16)

        # Store the password securely with restricted permissions
        echo -n "$POSTGRES_PASSWORD" > "$POSTGRES_PASSWORD_FILE"
        chmod 600 "$POSTGRES_PASSWORD_FILE"

        # Initialize database with password authentication
        /usr/pgsql-14/bin/initdb -D "$POSTGRES_DATA_DIR" --auth-host=scram-sha-256 --auth-local=trust --username=postgres --pwfile="$POSTGRES_PASSWORD_FILE"

        echo "Enabling pg_stat_statements extension for PostgreSQL..."
        /usr/pgsql-14/bin/pg_ctl start -D "$POSTGRES_DATA_DIR" -o "-c logging_collector=off"
        PGPASSWORD="$POSTGRES_PASSWORD" /usr/bin/psql -U postgres -h /run/postgresql -d postgres -c 'CREATE EXTENSION pg_stat_statements SCHEMA public'
        /usr/pgsql-14/bin/pg_ctl stop -D "$POSTGRES_DATA_DIR"

        # Clean up password from environment
        unset POSTGRES_PASSWORD
    fi
fi

echo "Creating nginx temp directories..."
mkdir -p /srv/nginx/tmp/{client,proxy,fastcgi,uwsgi,scgi}

if [ ! -d "/srv/pmm-agent/tmp" ]; then
    echo "Creating pmm-agent temp directory..."
    install -d -m 770 /srv/pmm-agent/tmp
fi

if is_enabled "$PMM_HA_ENABLE"; then
    echo "Skipping embedded PostgreSQL migration in HA mode."
elif is_enabled "$PMM_DISABLE_BUILTIN_POSTGRES"; then
    echo "Skipping embedded PostgreSQL migration (builtin PostgreSQL is disabled)."
else
    mkdir -p /run/postgresql
    chmod 750 "$POSTGRES_DATA_DIR" || true
    bash /opt/ansible/roles/postgres/files/postgres-migration
fi

echo "Generating self-signed certificates for nginx..."
bash /var/lib/cloud/scripts/per-boot/generate-ssl-certificate > /dev/null 2>&1

echo "Checking nginx configuration..."
if ! nginx -t -e /dev/stdout; then
    echo "Nginx configuration test failed, exiting..."
    exit 1
fi

# pmm-managed-init validates environment variables.
pmm-managed-init

declare AGENT_CONFIG_DIR="/usr/local/percona/pmm/config"
declare AGENT_ID=pmm-server

if is_enabled "$PMM_HA_ENABLE"; then
    echo "High Availability mode is enabled."
    if [ -f "$AGENT_CONFIG_DIR/pmm-agent.yaml" ]; then
        rm -f "$AGENT_CONFIG_DIR/pmm-agent.yaml"
    fi

    AGENT_CONFIG_DIR="/srv/pmm-agent/config"
    if [ ! -d "$AGENT_CONFIG_DIR" ]; then
        echo "Creating pmm-agent config directory..."
        install -d -m 770 "$AGENT_CONFIG_DIR"
    fi

    AGENT_ID="$(uuidgen)"
fi

if [ ! -f "$AGENT_CONFIG_DIR/pmm-agent.yaml" ]; then
  echo "Creating pmm-agent configuration..."
  pmm-agent setup \
      --config-file="$AGENT_CONFIG_DIR/pmm-agent.yaml" \
      --skip-registration \
      --id="$AGENT_ID" \
      --paths-tempdir=/srv/pmm-agent/tmp \
      --paths-nomad-data-dir=/srv/nomad/data \
      --server-address=127.0.0.1:8443 \
      --server-insecure-tls
fi

unset AGENT_CONFIG_DIR AGENT_ID

# Start supervisor in foreground, i.e. as PID 1
exec supervisord -n -c /etc/supervisord.conf
