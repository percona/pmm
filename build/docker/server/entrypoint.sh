#!/bin/bash
set -o errexit

# Initialize /srv if empty
DIST_FILE=/srv/pmm-distribution
if [ ! -f $DIST_FILE ]; then
    echo "File $DIST_FILE doesn't exist. Initializing /srv..."
    su pmm
      # Run the commands below as pmm user
      echo docker > $DIST_FILE
      mkdir -p /srv/{backup,clickhouse,grafana,logs,nginx,postgres14,prometheus,victoriametrics}
      echo "Copying plugins and VERSION file"
      cp /usr/share/percona-dashboards/VERSION /srv/grafana/PERCONA_DASHBOARDS_VERSION
      cp -r /usr/share/percona-dashboards/panels/* /srv/grafana/plugins
      
      echo "Generating self-signed certificates for nginx..."
      bash /var/lib/cloud/scripts/per-boot/generate-ssl-certificate
      
      echo "Initializing Postgres..."
      /usr/pgsql-14/bin/initdb -D /srv/postgres14 --auth=trust --username=postgres
      
      echo "Enabling pg_stat_statements extension for PostgreSQL..."
      /usr/pgsql-14/bin/pg_ctl start -D /srv/postgres14 -o '-c logging_collector=off'
      # We create the postgres user with superuser privileges to not break the code that connects pmm-managed to postgres.
      /usr/pgsql-14/bin/createuser --echo --superuser --host=/run/postgresql --no-password postgres
      /usr/bin/psql postgres postgres -c 'CREATE EXTENSION pg_stat_statements SCHEMA public'
      /usr/pgsql-14/bin/pg_ctl stop -D /srv/postgres14
    exit
fi

# pmm-managed-init validates environment variables.
pmm-managed-init

# Start supervisor in foreground
exec supervisord -n -c /etc/supervisord.conf
