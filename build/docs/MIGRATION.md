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

### Migration steps for PostgreSQL

1. Stop the following processes:
    - pmm-agent (will stop all exporters)
    - pmm-managed
    - grafana
    - postgres

2. Backup the databases
```
  /usr/pgsql-14/bin/pg_dump --host=/run/postgresql --username=postgres --file=/srv/backup/grafana.sql --dbname grafana
  /usr/pgsql-14/bin/pg_dump --host=/run/postgresql --username=postgres --file=/srv/backup/pmm-managed.sql --dbname pmm-managed
```

3. Move the database directory to /srv/backup/posgres14
```
  mv /srv/posgres14 /srv/backup/
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
  /usr/pgsql-14/bin/pg_restore --host=/run/postgresql --username=postgres --file=/srv/backup/postgres.sql -S postgres
  /usr/pgsql-14/bin/pg_restore --host=/run/postgresql --username=postgres --file=/srv/backup/grafana.sql -S postgres
```

8. Start the following processes:
    - postgres
    - grafana
    - pmm-managed
    - pmm-agent

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
