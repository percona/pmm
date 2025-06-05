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
