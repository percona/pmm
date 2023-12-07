# Migration from PMM v2 to v3 [draft]

## Preface
The migration from PMM v2 to v3 is a complex process that requires a lot of manual work. This document describes the process of migration and the steps that need to be taken to complete it.

One of the goals of the migration is to run all processes as an unprivileged user. This will allow us to run PMM Server in a container without the need to run it as root. This will also allow us to run PMM Client, and therefore all exporters, as an unprivileged user. The benefit lies primarily in increased security, but also in the ability to make PMM Server compatible with systems like Kubernetes, Podman etc.

## General migration steps

1. Update PMM Server to v2.41.x
The first step is to update PMM Server to the latest version of v2.41.x. This is necessary because the migration process requires the latest version of the PMM Server v2.41.x.

2. Stop all PMM Server processes
The next step is to stop all PMM Server processes. This includes shutting down supervisord process as well.


## Migration steps for PostgreSQL

1. Stop the following processes (all that connect to the database):
```
  - pmm-agent
  - pmm-managed
  - grafana
  - postgres
```

2. Backup the databases
```
  - /usr/pgsql-14/bin/pg_dump --host=/run/postgresql --username=postgres --file=/srv/backup/grafana.sql --dbname grafana
  - /usr/pgsql-14/bin/pg_dump --host=/run/postgresql --username=postgres --file=/srv/backup/pmm-managed.sql --dbname pmm-managed
  - /usr/pgsql-14/bin/pg_dump --host=/run/postgresql --username=postgres --file=/srv/backup/postgres.sql --dbname postgres
```

3. Move the database directory to /srv/backup/posgres14
```
  - mv /srv/posgres14 /srv/backup/
```

4. Recreate the following files or directories setting the ownership to `pmm` user
```
  - /srv/postgres14 (0700)
  - /run/postgresql (0755)
  - /srv/logs/postgresql14.log (0744)
```

5. Start a v3 instance
Remember to pass the data volume to the instance so it can bootstrap the database.

6. Shut down the following processes
```
  - pmm-agent
  - pmm-managed
  - grafana
```

7. Restore the databases from the backup
```
  - /usr/pgsql-14/bin/pg_restore --host=/run/postgresql --username=postgres --file=/srv/backup/postgres.sql -S postgres
  - /usr/pgsql-14/bin/pg_restore --host=/run/postgresql --username=postgres --file=/srv/backup/grafana.sql -S postgres
  - /usr/pgsql-14/bin/pg_restore --host=/run/postgresql --username=postgres --file=/srv/backup/pmm-managed.sql -S postgres
```

8. Start the following processes
```
  - postgres
  - grafana
  - pmm-managed
  - pmm-agent
```
