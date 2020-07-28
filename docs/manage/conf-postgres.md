# PostgreSQL

PMM provides both metrics and queries monitoring for PostgreSQL. Queries monitoring needs additional `pg_stat_statements` extension to be installed and enabled.

## Adding PostgreSQL extension for queries monitoring

The needed extension is `pg_stat_statements`. It is included in the official PostgreSQL contrib package, so you have to install this package first with your Linux distribution package manager. Particularly, on Debian-based systems it is done as follows:

```sh
sudo apt-get install postgresql-contrib
```

Now add/edit the following three lines in your `postgres.conf` file:

```ini
shared_preload_libraries = 'pg_stat_statements'
track_activity_query_size = 2048
pg_stat_statements.track = all
```

Besides making the appropriate module to be loaded, these edits will increase the maximum size of the query strings PostgreSQL records and will allow it to track all statements including nested ones. When the editing is over, restart PostgreSQL.

Finally, the following statement should be executed in the PostgreSQL shell to install the extension:

```sql
CREATE EXTENSION pg_stat_statements SCHEMA public;
```

!!! note

    `CREATE EXTENSION` statement should be run in the `postgres` database.

## Adding PostgreSQL queries and metrics monitoring

You can add PostgreSQL metrics and queries monitoring with the following command:

```sh
pmm-admin add postgresql --username=pmm --password=pmm
```

where username and password parameters should contain actual PostgreSQL user credentials.

Additionally, two positional arguments can be appended to the command line flags: a service name to be used by PMM, and a service address. If not specified, they are substituted automatically as `<node>-postgresql` and `127.0.0.1:5432`.

The command line and the output of this command may look as follows:

```sh
pmm-admin add postgresql --username=pmm --password=pmm postgres 127.0.0.1:5432
PostgreSQL Service added.
Service ID  : /service_id/28f1d93a-5c16-467f-841b-8c014bf81ca6
Service name: postgres
```

As a result, you should be able to see data in PostgreSQL Overview dashboard, and also Query Analytics should contain PostgreSQL queries, if the needed extension was installed and configured correctly.

Beside positional arguments shown above you can specify service name and service address with the following flags: `--service-name`, `--host` (the hostname or IP address of the service), and `--port` (the port number of the service). If both flag and positional argument are present, flag gains higher priority. Here is the previous example modified to use these flags:

```sh
pmm-admin add postgresql --username=pmm --password=pmm --service-name=postgres --host=127.0.0.1 --port=270175432
```

It is also possible to add a PostgreSQL instance using a UNIX socket with just the `--socket` flag followed by the path to a socket:

```sh
pmm-admin add postgresql --socket=/var/run/postgresql
```

Capturing read and write time statistics is possible only if `track_io_timing` setting is enabled. This can be done either in configuration file or with the following query executed on the running system:

```sql
ALTER SYSTEM SET track_io_timing=ON;
SELECT pg_reload_conf();
```

## Setting up the required user permissions and authentication

Percona recommends that a PostgreSQL user be configured for `SUPERUSER` level access, in order to gather the maximum amount of data with a minimum amount of complexity. This can be done with the following command for the standalone PostgreSQL installation:

```sql
CREATE USER pmm_user WITH SUPERUSER ENCRYPTED PASSWORD 'secret';
```

In case of monitoring a PostgreSQL database running on an Amazon RDS instance, the command should look as follows:

```sql
CREATE USER pmm_user WITH rds_superuser ENCRYPTED PASSWORD 'secret';
```

!!! note

    Specified PostgreSQL user should have enabled local password authentication to enable access for PMM. This can be set in the `pg_hba.conf` configuration file changing `ident` to `md5` for the correspondent user. Also, this user should be able to connect to the `postgres` database which we have installed the extension into.
