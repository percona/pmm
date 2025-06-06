#!/bin/bash
set -o errexit

PMM_DISTRIBUTION_METHOD="${PMM_DISTRIBUTION_METHOD:-docker}"

# Get current user info - handle cases where user doesn't exist in passwd
CURRENT_UID=$(id -u)
CURRENT_GID=$(id -g)
if whoami &> /dev/null; then
    CURRENT_USER=$(whoami)
else
    CURRENT_USER="user-${CURRENT_UID}"
    echo "Running as UID ${CURRENT_UID} (user not in passwd file)"
fi

if [ ! -w /srv ]; then
    echo "FATAL: /srv is not writable for ${CURRENT_USER} user." >&2
    echo "Please make sure that /srv is owned by uid ${CURRENT_UID} and gid ${CURRENT_GID} and try again." >&2
    echo "You can change ownership by running: sudo chown -R ${CURRENT_UID}:${CURRENT_GID} /srv" >&2
    exit 1
fi

if [ "$CURRENT_UID" != "1000" ] || [ "$CURRENT_GID" != "1000" ]; then
    echo "Running as UID:GID $CURRENT_UID:$CURRENT_GID, setting up for arbitrary UID..."

    # Try NSS wrapper first if available
    NSS_WRAPPER_LIB=""
    for lib_path in "/usr/lib64/libnss_wrapper.so" "/usr/lib/x86_64-linux-gnu/libnss_wrapper.so" "/usr/lib/libnss_wrapper.so"; do
        if [ -f "$lib_path" ]; then
            NSS_WRAPPER_LIB="$lib_path"
            break
        fi
    done
    
    if [ -n "$NSS_WRAPPER_LIB" ]; then
        echo "Setting up NSS wrapper..."
        # Set up NSS wrapper for arbitrary UID support
        export NSS_WRAPPER_PASSWD=$(mktemp)
        export NSS_WRAPPER_GROUP=$(mktemp)
        
        # Copy existing passwd and group entries
        cat /etc/passwd > $NSS_WRAPPER_PASSWD
        cat /etc/group > $NSS_WRAPPER_GROUP
        
        # Add current user if not exists
        if ! getent passwd $CURRENT_UID > /dev/null 2>&1; then
            echo "pmm:x:${CURRENT_UID}:${CURRENT_GID}:PMM User:/srv:/bin/bash" >> $NSS_WRAPPER_PASSWD
        fi
        
        # Add current group if not exists  
        if ! getent group $CURRENT_GID > /dev/null 2>&1; then
            echo "pmm:x:${CURRENT_GID}:" >> $NSS_WRAPPER_GROUP
        fi
        
        export LD_PRELOAD="${NSS_WRAPPER_LIB}:${LD_PRELOAD:-}"
        echo "NSS wrapper enabled with $NSS_WRAPPER_LIB"
    else
        echo "NSS wrapper not available, using fallback approach..."
        # Fallback: just ensure the UID can access necessary directories
        # Most applications don't actually need user resolution to work
    fi
    
    # Fix ownership of PostgreSQL directories if needed
    if [ -d "/srv/postgres14" ]; then
        chown -R "$CURRENT_UID:$CURRENT_GID" /srv/postgres14 2>/dev/null || true
    fi
    if [ -d "/run/postgresql" ]; then
        chown -R "$CURRENT_UID:$CURRENT_GID" /run/postgresql 2>/dev/null || true
    fi

    # Fix permissions
    if [ -d "/srv/postgres14" ]; then
        chmod 700 /srv/postgres14 2>/dev/null || true
    fi
fi

# Initialize /srv if empty
DIST_FILE=/srv/pmm-distribution
if [ ! -f $DIST_FILE ]; then
    echo $PMM_DISTRIBUTION_METHOD > $DIST_FILE
    echo "Initializing /srv..."
    mkdir -p /srv/{backup,clickhouse,grafana,logs,nginx,postgres14,prometheus,victoriametrics}
    echo "Copying grafana plugins and the VERSION file..."
    mkdir -p /srv/grafana/plugins
    cp -r /usr/share/percona-dashboards/panels/* /srv/grafana/plugins
    
    echo "Generating self-signed certificates for nginx..."
    bash /var/lib/cloud/scripts/per-boot/generate-ssl-certificate
    
    echo "Initializing Postgres..."
    /usr/pgsql-14/bin/initdb -D /srv/postgres14 --auth=trust --username=postgres
    
    echo "Enabling pg_stat_statements extension for PostgreSQL..."
    /usr/pgsql-14/bin/pg_ctl start -D /srv/postgres14 -o '-c logging_collector=off'
    /usr/bin/psql postgres postgres -c 'CREATE EXTENSION pg_stat_statements SCHEMA public'
    /usr/pgsql-14/bin/pg_ctl stop -D /srv/postgres14
fi

# pmm-managed-init validates environment variables.
pmm-managed-init

# Start supervisor in foreground
exec supervisord -n -c /etc/supervisord.conf
