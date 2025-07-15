#!/bin/bash
set -o errexit

declare PMM_DISTRIBUTION_METHOD="${PMM_DISTRIBUTION_METHOD:-docker}"
declare CURRENT_GID CURRENT_UID CURRENT_USER

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

    if [ -n "$NSS_WRAPPER_LIB" ]; then
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
fi

# Check /usr/share/pmm-server directory on every start
echo "Checking /usr/share/pmm-server directory structure..."
# Still ensure critical directories exist, but don't create empty ones
if [ ! -d "/usr/share/pmm-server/nginx" ]; then 
    echo "Creating nginx temp directories..."
    mkdir -p /usr/share/pmm-server/nginx/{client_temp,proxy_temp,fastcgi_temp,uwsgi_temp,scgi_temp}
fi

if [ ! -d "/srv/pmm-agent/tmp" ]; then
    echo "Creating pmm-agent temp directory..."
    install -d -m 770 /srv/pmm-agent/tmp
fi

# Initialize /srv if empty
declare DIST_FILE=/srv/pmm-distribution
if [ ! -f "$DIST_FILE" ]; then
    echo -n "$PMM_DISTRIBUTION_METHOD" > "$DIST_FILE"
    echo "Initializing /srv..."
    mkdir -p /srv/{backup,clickhouse,grafana,logs,nginx,prometheus,victoriametrics}
    echo "Copying grafana plugins and the VERSION file..."
    mkdir -p /srv/grafana/plugins
    cp -r /usr/share/percona-dashboards/panels/* /srv/grafana/plugins

    echo "Initializing Postgres..."
    install -d -m 750 /srv/postgres14
    /usr/pgsql-14/bin/initdb -D /srv/postgres14 --auth=trust --username=postgres

    echo "Enabling pg_stat_statements extension for PostgreSQL..."
    /usr/pgsql-14/bin/pg_ctl start -D /srv/postgres14
    /usr/bin/psql postgres postgres -c 'CREATE EXTENSION pg_stat_statements SCHEMA public'
    /usr/pgsql-14/bin/pg_ctl stop -D /srv/postgres14
fi

echo "Generating self-signed certificates for nginx..."
bash /var/lib/cloud/scripts/per-boot/generate-ssl-certificate

# Ensure /srv/postgres14 has the correct permissions
chmod 750 /srv/postgres14 || true

echo "Checking nginx configuration..."
if ! nginx -t; then
    echo "Nginx configuration test failed, exiting..."
    exit 1
fi

# pmm-managed-init validates environment variables.
pmm-managed-init

# Start supervisor in foreground
exec supervisord -n -c /etc/supervisord.conf
