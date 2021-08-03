# PostgreSQL

How to set up PMM to monitor a [PostgreSQL] or [Percona Distribution for PostgreSQL] database instance.

!!! summary alert alert-info "Summary"
    - Create PMM account and set permissions.
    - Choose, install and configure an extension:
        - `pg_stat_statements`, or,
        - `pg_stat_monitor`.
    - Add service.
    - Check service.

## Before you start

Check that:

- [PMM Server is installed](../server/index.md) and running with a known IP address accessible from the client node.
- [PMM Client is installed](index.md) and the [node is registered with PMM Server](index.md#register).
- You have superuser (root) access on the client host.
- You have superuser access to any database servers that you want to monitor.

(PMM follows [PostgreSQL's end-of-life policy][POSTGRESQL_VERSIONING]. For specific details on supported platforms and versions, see [Percona’s Software Platform Lifecycle page][PERCONA_LIFECYCLE].)

## Create a database account for PMM

We recommend creating a PMM database account that can connect to the `postgres` database with the `SUPERUSER` role.

1. Create a user. This example uses `pmm`. (Replace `******` with a strong password of your choice.)

    ```sql
    CREATE USER pmm WITH SUPERUSER ENCRYPTED PASSWORD '******';
    ```

    If your database runs on Amazon RDS:

    ```sql
    CREATE USER pmm WITH rds_superuser ENCRYPTED PASSWORD '******';
    ```

2. PMM must be able to log in locally as this user to the PostgreSQL instance. To enable this, edit the `pg_hba.conf` file. If not already enabled by an existing rule, add:

    ```conf
    local   all             pmm                                md5
    # TYPE  DATABASE        USER        ADDRESS                METHOD
    ```

    (Ignore the second line. It is a comment to show field alignment.)

3. Reload the configuration:

    ```sh
    su - postgres
    psql -c "select pg_reload_conf()"
    ```

4. Check local login.

    ```sh
    psql postgres pmm -c "\conninfo"
    ```

5. Enter the password for the `pmm` user when prompted.

## Choose and configure an extension

Decide which database extension to use, and configure your database server for it. The choices are:

1. [`pg_stat_statements`](#pg_stat_statements), the original extension created by PostgreSQL, part of the `postgresql-contrib` package available on Linux.

2. [`pg_stat_monitor`](#pg_stat_monitor) is a new extension created by Percona. It is based on and compatible with `pg_stat_statements`. `pg_stat_monitor` has all the features of `pg_stat_statements`, but adds *bucket-based data aggregation*.

We recommend choosing only one of these. **If you use both, you will get duplicate metrics.**

!!! caution alert alert-warning "Caution"
    While we recommend use of the newer `pg_stat_monitor` extension, be aware it is currently in beta phase and unsupported.

Here are the benefits and drawbacks of each.

|                      | <i class="uil uil-thumbs-up"></i> Benefits     | <i class="uil uil-thumbs-down"></i> Drawbacks
|----------------------|------------------------------------------------|---------------------------------------------------
| `pg_stat_statements` | 1. Part of official `postgresql-contrib` package. | 1. No aggregated statistics or histograms.<br>2. No Query Examples.
| `pg_stat_monitor`    | 1. Builds on `pg_stat_monitor` features.<br>2. Bucket-based aggregation. | 1. Beta software.

!!! note alert alert-primary "Bucket-based data aggregation"
    `pg_stat_monitor` collects statistics and aggregates data in a data collection unit called a *bucket*. These are linked together to form a *bucket chain*.

    You can specify:

    - the number of buckets (the length of the chain);
    - how much space is available for all buckets;
    - a time limit for each bucket's data collection (the *bucket expiry*).

    When a bucket's expiration time is reached, accumulated statistics are reset and data is stored in the next available bucket in the chain.

    When all buckets in the chain have been used, the first bucket is reused and its contents are overwritten.

    If a bucket fills before its expiration time is reached, data is discarded.

### `pg_stat_statements`

#### Install

- Debian/Ubuntu

    !!! hint "Root permissions"
        ```sh
        apt install -y postgresql-contrib
        ```

- Red Hat/CentOS

    !!! hint "Root permissions"
        ```sh
        yum install -y postgresql-contrib
        ```

#### Configure

1. Add these lines to your `postgresql.conf` file:

    ```conf
    shared_preload_libraries = 'pg_stat_statements'
    track_activity_query_size = 2048 # Increase tracked query string size
    pg_stat_statements.track = all   # Track all statements including nested
    track_io_timing = on             # Capture read/write stats
    ```

2. Restart the database server.

3. Install the extension.

    ```sh
    psql postgres postgres -c "CREATE EXTENSION pg_stat_statements SCHEMA public"
    ```

You can now [add the service](#add-service).

### `pg_stat_monitor`

!!! caution alert alert-warning "Caution"
    `pg_stat_monitor` is currently in beta phase and is unsupported.

`pg_stat_monitor` has been tested with:

- PostgreSQL versions 11, 12, 13.
- Percona Distribution for PostgreSQL versions 11, 12, 13.

#### Install

- If you use *Percona Distribution for PostgreSQL*, you can install the extension with your Linux package manager. See [Installing Percona Distribution for PostgreSQL][PERCONA_POSTGRESQL_INSTALL].

- If you use *PostgreSQL* you can install by downloading and compiling the source code. See [Installing `pg_stat_monitor`][PG_STAT_MONITOR_INSTALL].

#### Configure

1. Set or change the value for `shared_preload_library`.

    In your `postgresql.conf` file:

    ```ini
    shared_preload_libraries = 'pg_stat_monitor'
    ```

2. Set configuration values.

    You can get a list of available settings with `SELECT * FROM pg_stat_monitor_settings;`.

    !!! note alert alert-primary ""
        See [`pg_stat_monitor` GitHub repository](https://github.com/percona/pg_stat_monitor/blob/master/docs/USER_GUIDE.md#configuration) for details about available parameters.

3. Set bucket time to 60 seconds.

    ```sql
    ALTER SYSTEM SET pg_stat_monitor.pgsm_bucket_time=60;
    ```

4. Start or restart your PostgreSQL instance.

5. In a `psql` session:

    ```sql
    CREATE EXTENSION pg_stat_monitor;
    ```

6. Check the version.

    ```sql
    SELECT pg_stat_monitor_version();
    ```

## Add service

When you have configured your database server, you can add a PostgreSQL service with the user interface or on the command line.

### With the user interface

1. Select <i class="uil uil-cog"></i> *Configuration* → {{icon.inventory}} *PMM Inventory* → {{icon.addinstance}} *Add Instance*.

2. Select *PostgreSQL -- Add a remote instance*.

3. Enter or select values for the fields.

4. Click *Add service*.

![!](../../_images/PMM_Add_Instance_PostgreSQL.jpg)

### On the command line

Add the database server as a service using one of these example commands. If successful, PMM Client will print `PostgreSQL Service added` with the service's ID and name. Use the `--environment` and `-custom-labels` options to set tags for the service to help identify them.

### Examples

Add instance with default node (`<node>-postgresql`) and service name.

```sh
pmm-admin add postgresql \
--username=pmm \
--password=password \
--server-url=https://admin:admin@X.X.X.X:443 \
--server-insecure-tls
```

- `<user name>`: The PostgreSQL PMM user
- `<password>`: The PostgreSQL user credentials.

The service name and service ID will be automatically chosen.

Add instance with specified node and service name.

```sh
pmm-admin add postgresql \
--username=pmm \
--password=password \
--server-url=https://admin:admin@X.X.X.X:443 \
--server-insecure-tls \
```

Add instance to connect with a UNIX socket.

```sh
pmm-admin add postgresql --socket=/var/run/postgresql
```

Add instance to connect with SSL/TLS.

```sh
pmm-admin add postgresql \
--username=pmm \
--password=password \
--tls \
--tls-skip-verify
```

where:

- `--tls`: Use TLS to connect to the database.
- `--tls-skip-verify`: Skip TLS certificates validation.

## Check the service

### Check service - PMM user interface

1. Select <i class="uil uil-cog"></i> *Configuration* → {{icon.inventory}} *PMM Inventory* → {{icon.inventory}} *Inventory list*.

2. Look in the *Services* tab for a matching *Service Type* (PostgreSQL), *Service name*, *Addresses*, and any other details entered in the form.

3. Look in the *Agents* tab to check the desired data source is being used.

### Check service - Command line

Look for your service in the output of this command.

```sh
pmm-admin inventory list services
```

!!! hint alert alert-success ""
    If using Docker, use `docker exec pmm-client pmm-admin inventory list services`

### Check data

1. Open the *PostgreSQL Instance Summary* dashboard.

2. Set the *Service Name* to the newly-added service.

!!! seealso alert alert-info "See also"
    - [`pmm-admin` man page for `pmm-admin add postgresql`](../../details/commands/pmm-admin.md#postgresql)
    - [Configuring Percona Repositories with percona-release][PERCONA_RELEASE]

[PostgreSQL]: https://www.postgresql.org/
[Percona Distribution for PostgreSQL]: https://www.percona.com/software/postgresql-distribution
[POSTGRESQL_VERSIONING]: https://www.postgresql.org/support/versioning/
[PERCONA_LIFECYCLE]: https://www.percona.com/services/policies/percona-software-platform-lifecycle/
[PERCONA_RELEASE]: https://www.percona.com/doc/percona-repo-config/percona-release.html
[PERCONA_POSTGRESQL_INSTALL]: https://www.percona.com/doc/postgresql/LATEST/installing.html
[PG_STAT_MONITOR_INSTALL]: https://github.com/percona/pg_stat_monitor#installation
[PMM_ADMIN]: ../../details/pmm-admin.md
