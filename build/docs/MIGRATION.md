# Migration from PMM v2 to v3 [early draft]

## Preface
The migration from PMM v2 to v3 is a complex process that requires a lot of manual work. This document describes the process of migration and the steps that need to be taken to complete it.

One of the goals of the migration is to run all processes as an unprivileged user. This will allow us to run PMM Server in a container without the need to run it as root. This will also allow us to run PMM Client, and therefore all exporters, as an unprivileged user. The benefit lies primarily in increased security, but also in the ability to make PMM Server compatible with systems like Kubernetes, Podman etc.

## General migration steps

1. Upgrade PMM Server to v2.41.x
The first step is to upgrade PMM Server to the latest version of v2.41.x. This is necessary because the migration from v2 to v3 requires the latest version of PMM Server v2.41.x.

2. Stop all PMM Server processes
The next step is to stop all PMM Server processes. This includes shutting down supervisord process as well.

3. Backup the data
This involves backing up the `/srv` directory since all the databases, log files, some user-facing config files and plugins are stored there.

4. Run PMM Server v3
The next step is to run PMM Server v3. The data volume should be mounted to PMM's `/srv` directory.

When PMM v3 starts, it will detect the need for migration and proceed with it. 

5. Migrate the data
The next step is migration of data from v2 to v3. This involves running an ansible migration playbook, that will migrate the data from v2 to v3. It will also create the necessary users and set the correct permissions on the files and directories. 

The process is automatic and does not require any manual intervention. Upon completion, the UI will display a message with the migration summary.

The migration process will also start all the processes that were running prior to it.

6. Migrate PMM Clients
PMM Clients version 2.x and earlier are not compatible with PMM Server v3.x. There is a good number of breaking changes in v3, which make it impossible to use v1/v2 clients along with PMM Server v3. 

The migration of clients involves the installation of PMM Clients v3.x. The process is manual and requires the user to install the new clients on all monitored hosts. Once the client is installed, it should connect to PMM Server v3 and start sending data.

7. Post-migration steps
The last step is to perform some post-migration checks to be sure that the migration was successful. These checks are manual. They include:
  - verification of PMM settings, users and permissions
  - verification of data integrity and consistency
  - verification of inventory to make sure all nodes, agents and services run as before

We suggest to keep the old data for some time in case you need to roll PMM Server back to v2. Once you are sure that the migration was successful, you can remove the old data.


## Migration steps for individual components
The following sections describe the migration steps for individual components. They are meant to be used as a reference for the migration process so that the user can understand what is happening during the migration. 

### Migration steps for PostgreSQL (v11 to v14)

1. Stop the following processes:
    - pmm-agent (will stop all exporters)
    - pmm-managed
    - grafana
    - postgres

2. Backup the databases
```
  # Read the postgres password from the secure file
  PGPASSWORD=$(cat /srv/.postgres_password)
  PGPASSWORD="$PGPASSWORD" /usr/pgsql-14/bin/pg_dump --host=/run/postgresql --username=postgres --file=/srv/backup/grafana.sql --dbname grafana
  PGPASSWORD="$PGPASSWORD" /usr/pgsql-14/bin/pg_dump --host=/run/postgresql --username=postgres --file=/srv/backup/pmm-managed.sql --dbname pmm-managed
  unset PGPASSWORD
```

3. Move the database directory to /srv/backup/postgres14
```
  mv /srv/postgres14 /srv/backup/
```

4. Recreate the following files or directories setting the ownership to `pmm` user:
    - /srv/postgres14 (0750)
    - /run/postgresql (0775)
    - /srv/logs/postgresql14.log (0664)

5. Start a v3 instance
Remember to pass the data volume to the instance so it can bootstrap the database. This is normally done by passing the `-v pmm-data:/srv` option to the `docker run` command, where `pmm-data` is the name of the volume.

6. Shut down the following processes:
    - pmm-agent
    - pmm-managed
    - grafana
    - postgres

7. Restore the databases from the backup
```
  # Read the postgres password from the secure file
  PGPASSWORD=$(cat /srv/.postgres_password)
  PGPASSWORD="$PGPASSWORD" /usr/pgsql-14/bin/pg_restore --host=/run/postgresql --username=postgres --file=/srv/backup/pmm-managed.sql -S postgres
  PGPASSWORD="$PGPASSWORD" /usr/pgsql-14/bin/pg_restore --host=/run/postgresql --username=postgres --file=/srv/backup/grafana.sql -S postgres
  unset PGPASSWORD
```

8. Start the following processes:
    - postgres
    - grafana
    - pmm-managed
    - pmm-agent

### Migration steps for PostgreSQL (v14 to v18)

Unlike the sections above, this one describes an upgrade between PMM v3 releases. Earlier PMM v3 releases used PostgreSQL 14 for the embedded database, while newer ones use PostgreSQL 18. Since the on-disk format is not compatible between major PostgreSQL releases, the data directory cannot simply be reused.

The upgrade is performed automatically on the first start of a PMM Server that ships PostgreSQL 18, so no manual intervention is required. It is driven by `build/docker/server/entrypoint.sh`, which runs `build/ansible/roles/postgres/files/postgres-migration` before supervisord starts. The steps below document what that script does.

The upgrade runs only when `/srv/postgres14` exists and `/srv/postgres18` does not, which makes it a one-time operation. The new cluster is built in `/srv/postgres18.new` and moved into place only once it is fully restored, so `/srv/postgres18` existing always means a finished cluster. An interrupted attempt leaves nothing but a staging directory, which the next start discards before retrying. It is skipped entirely when the embedded PostgreSQL is not in use, i.e. when `PMM_HA_ENABLE` or `PMM_DISABLE_BUILTIN_POSTGRES` is enabled. In those cases the external database has to be upgraded by its owner.

Both the PostgreSQL 14 and 18 binaries are shipped in the image, so the upgrade is a logical dump and restore rather than an in-place `pg_upgrade`. This also means the upgrade path is only available as long as the PostgreSQL 14 binaries remain in the image; upgrading from an older PMM v3 release after they are dropped requires an intermediate upgrade. Make sure the volume holding `/srv` has enough free space for a plain-text dump of both databases plus a second data directory. The dumps are removed once the upgrade completes, so that part of the requirement is transient.

1. Dump the databases from PostgreSQL 14
The old server is started on the socket in `/run/postgresql`, and each database is dumped to `/srv/backup`:
```
  PGPASSWORD=$(cat /srv/.postgres_password)
  export PGPASSWORD
  /usr/pgsql-14/bin/pg_ctl start -D /srv/postgres14 -o "-c logging_collector=off" -w
  /usr/pgsql-14/bin/pg_dump -h /run/postgresql -U postgres -F p -f /srv/backup/pg18-upgrade-pmm-managed.sql pmm-managed
  /usr/pgsql-14/bin/pg_dump -h /run/postgresql -U postgres -F p -f /srv/backup/pg18-upgrade-grafana.sql grafana
  /usr/pgsql-14/bin/pg_ctl stop -D /srv/postgres14 -w
```

Only databases that are actually present are dumped. The `grafana` one is skipped when `GF_DATABASE_URL` or `GF_DATABASE_HOST` is set, because Grafana then keeps its data in an external database, and also when the old cluster never held it — a PMM 2 installation may not have.

2. Initialize the PostgreSQL 18 data directory
The new cluster is created in `/srv/postgres18.new` with the same authentication settings as a fresh installation, reusing the existing superuser password. Data directories that predate PMM 3.7 have no `/srv/.postgres_password`, in which case a password is generated first, since `initdb` seeds the new cluster's `postgres` role from that file. An existing password file is left untouched:
```
  install -d -m 750 /srv/postgres18.new
  /usr/pgsql-18/bin/initdb -D /srv/postgres18.new --auth-host=scram-sha-256 --auth-local=trust --username=postgres --pwfile=/srv/.postgres_password
```

3. Recreate the roles and databases, then restore the dumps
The dumps contain no role definitions, so each role is recreated and made the owner of its database. Ownership is what matters here: since PostgreSQL 15 the `public` schema belongs to `pg_database_owner`, so the owner can create tables in it while a plain `GRANT ALL PRIVILEGES ON DATABASE` cannot. This is done for every database dumped in step 1:
```
  /usr/pgsql-18/bin/pg_ctl start -D /srv/postgres18.new -o "-c logging_collector=off" -w
  /usr/pgsql-18/bin/psql -h /run/postgresql -U postgres -d postgres \
      -c "CREATE ROLE \"pmm-managed\" LOGIN PASSWORD 'pmm-managed'" \
      -c "CREATE DATABASE \"pmm-managed\" OWNER \"pmm-managed\""
  /usr/pgsql-18/bin/psql -h /run/postgresql -U postgres -d pmm-managed -f /srv/backup/pg18-upgrade-pmm-managed.sql
```

4. Recreate the pg_stat_statements extension
The extension is registered per database, so it has to be created again in the new cluster:
```
  /usr/pgsql-18/bin/psql -h /run/postgresql -U postgres -d postgres -c "CREATE EXTENSION IF NOT EXISTS pg_stat_statements SCHEMA public"
  /usr/pgsql-18/bin/pg_ctl stop -D /srv/postgres18.new -w
  unset PGPASSWORD
```

5. Publish the new cluster, keep the old directory for rollback and discard the dumps
```
  mv /srv/postgres18.new /srv/postgres18
```

That move is the point at which the upgrade counts as done. `/srv/postgres14` is then renamed to `/srv/postgres14.old` rather than removed. The rename also prevents the upgrade from running a second time. That directory is the rollback artifact, so the `pg18-upgrade-*.sql` dumps are deleted once it is in place; only those files are touched, since `/srv/backup` also holds user backups. A failed upgrade never reaches this point, so its dumps are left behind to inspect.

To roll back, shut PMM Server down, remove `/srv/postgres18`, rename `/srv/postgres14.old` back to `/srv/postgres14` and start the previous PMM Server image. Once the upgrade is confirmed to be successful, `/srv/postgres14.old` can be removed to reclaim disk space.

6. Start the processes
Supervisord starts the new server as `/usr/pgsql-18/bin/postgres -D /srv/postgres18` and writes its log to `/srv/logs/postgresql.log`. The log file is no longer named after the major version, so a `/srv/logs/postgresql14.log` left over from the previous release can be deleted. The remaining processes (grafana, pmm-managed, pmm-agent) are started as usual, and pmm-managed applies its schema migrations on the restored database.

### Migration steps for ClickHouse

1. Stop the following processes:
    - grafana (its UI requires QAN API)
    - qan-api2
    - clickhouse

2. Change ownership of the following directories (recursive) to `pmm` user:
    - /srv/clickhouse (0755) - the data directory

3. Start the following processes:
    - clickhouse
    - qan-api2
    - grafana

Please note, that data migration to v3 is done by PMM.
