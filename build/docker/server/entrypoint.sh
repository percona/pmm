#!/bin/bash
set -o errexit

# init /srv if empty
DIST_FILE=/srv/pmm-distribution
if [ ! -f $DIST_FILE ]; then
    echo "File $DIST_FILE doesn't exist. Initizlize /srv..."
    echo docker > $DIST_FILE
    mkdir -p /srv/{clickhouse,grafana,logs,postgres,prometheus,nginx,victoriametrics}
    echo "Copy plugins and VERSION file"
    cp /usr/share/percona-dashboards/VERSION /srv/grafana/PERCONA_DASHBOARDS_VERSION
    cp -r /usr/share/percona-dashboards/panels/ /srv/grafana/plugins
    chown -R grafana:grafana /srv/grafana
    chown pmm:pmm /srv/{victoriametrics,prometheus,logs}
    chown postgres:postgres /srv/postgres
    echo "Generate self-signed certificates for nginx"
    bash /var/lib/cloud/scripts/per-boot/generate-ssl-certificate
    echo "Init Postgres"
    su postgres -c "/usr/pgsql-11/bin/initdb -D /srv/postgres"
    echo "Temporary start postgres and enable pg_stat_statements"
    su postgres -c "/usr/pgsql-11/bin/pg_ctl start -D /srv/postgres"
    su postgres -c "psql postgres postgres -c 'CREATE EXTENSION pg_stat_statements SCHEMA public'"
    su postgres -c "/usr/pgsql-11/bin/pg_ctl stop -D /srv/postgres"
fi

# pmm-managed-init validates environment variables.
pmm-managed-init

# Start supervisor in foreground
exec supervisord -n -c /etc/supervisord.conf
