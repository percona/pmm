#!/bin/bash
set -o errexit

# init /srv if empty
DIST_FILE=/srv/pmm-distribution
if [ ! -f $DIST_FILE ]; then
    echo "File $DIST_FILE doesn't exist. Initialize /srv..."
    echo docker > $DIST_FILE
    mkdir -p /srv/{clickhouse,grafana,logs,postgres14,prometheus,nginx,victoriametrics}
    echo "Copying plugins and VERSION file"
    cp /usr/share/percona-dashboards/VERSION /srv/grafana/PERCONA_DASHBOARDS_VERSION
    cp -r /usr/share/percona-dashboards/panels/ /srv/grafana/plugins
    chown -R pmm:pmm /srv/grafana
    chown pmm:pmm /srv/{victoriametrics,prometheus,logs}
    chown pmm:pmm /srv/postgres14
    echo "Generating self-signed certificates for nginx"
    bash /var/lib/cloud/scripts/per-boot/generate-ssl-certificate
    echo "Initializing Postgres"
    /usr/pgsql-14/bin/initdb -D /srv/postgres14
    echo "Enable pg_stat_statements extension"
    /usr/pgsql-14/bin/pg_ctl start -D /srv/postgres14 -o "-c logging_collector=off"
    /usr/bin/psql postgres postgres -c 'CREATE EXTENSION pg_stat_statements SCHEMA public'
    /usr/pgsql-14/bin/pg_ctl stop -D /srv/postgres14
fi

# pmm-managed-init validates environment variables.
pmm-managed-init

# Start supervisor in foreground
exec supervisord -n -c /etc/supervisord.conf
