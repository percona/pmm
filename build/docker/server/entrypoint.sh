#!/bin/bash
set -o errexit

PMM_DISTRIBUTION_METHOD="${PMM_DISTRIBUTION_METHOD:-docker}"

if [ ! -w /srv ]; then
    echo "FATAL: /srv is not writable for $(whoami) user." >&2
    echo "Please make sure that /srv is owned by uid $(id -u) and gid $(id -g) and try again." >&2
    echo "You can change ownership by running: sudo chown -R $(id -u):$(id -g) /srv" >&2
    exit 1
fi

# Initialize /srv if empty
DIST_FILE=/srv/pmm-distribution
if [ ! -f $DIST_FILE ]; then
    echo $PMM_DISTRIBUTION_METHOD > $DIST_FILE
    echo "Initializing /srv..."
    mkdir -p /srv/{backup,clickhouse,grafana,logs,nginx,postgres14,prometheus,victoriametrics,supervisord.d}
    echo "Copying grafana plugins and the VERSION file..."
    mkdir -p /srv/grafana/plugins
    cp -r /usr/share/percona-dashboards/panels/* /srv/grafana/plugins

    mkdir -p /srv/nginx/{client_body_temp,proxy_temp,fastcgi_temp,uwsgi_temp,scgi_temp}
    chmod 700 /srv/nginx/client_body_temp /srv/nginx/proxy_temp /srv/nginx/fastcgi_temp /srv/nginx/uwsgi_temp /srv/nginx/scgi_temp
    
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
