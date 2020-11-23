# PostgreSQL

PMM follows the [postgresql.org EOL policy](https://www.postgresql.org/support/versioning/).

For specific details on supported platforms and versions, see
[Percona’s Software Platform Lifecycle page](https://www.percona.com/services/policies/percona-software-platform-lifecycle/).


To monitor PostgreSQL queries, you must install a database extension. There are two choices:

- `pg_stat_monitor`, a new extension created by Percona, based on `pg_stat_statements` and compatible with it.

- `pg_stat_statements`, the original extension created by PostgreSQL, part of the `postgres-contrib` package available on Linux.

`pg_stat_monitor` provides all the features of `pg_stat_statements`, but extends it to provide bucket-based data aggregation, a feature missing from `pg_stat_statements`. (`pg_stat_statements` accumulates data without providing aggregated statistics or histogram information.)

!!! note

    - `pg_stat_monitor` is the recommended option.

    - Although nothing prevents you from installing and using both, we don't recommend this as you will get duplicate metrics.

!!! caution

    `pg_stat_monitor` is beta software and currently unsupported.


## Prerequisites

We recommend that you create a PostgreSQL user for `SUPERUSER` level access. This lets you gather the most data with the least fuss.

This user must be able to connect to the `postgres` database where the extension was installed. The PostgreSQL user should have local password authentication enabled to access PMM. To do this, set `ident` to `md5` for the user in the `pg_hba.conf` configuration file.

To create a superuser:

```sql
CREATE USER pmm_user WITH SUPERUSER ENCRYPTED PASSWORD '******';
```

Or, if your database runs on Amazon RDS:

```sql
CREATE USER pmm_user WITH rds_superuser ENCRYPTED PASSWORD '******';
```

## `pg_stat_monitor`

`pg_stat_monitor` collects statistics and aggregates data in a data collection unit called a *bucket* linked together to form a *bucket chain*.

You can specify:

- the number of buckets (the length of the chain);
- how much space is available for all buckets;
- a time limit for each bucket's data collection (the *bucket expiry*).

When a bucket's expiration time is reached, accumulated statistics are reset and data is stored in the next available bucket in the chain.

When all buckets in the chain have been used, the first bucket is reused and its contents are overwritten.

If a bucket fills before its expiration time is reached, data is discarded.

### Compatibility

`pg_stat_monitor` has been tested with:

- PostgreSQL versions 11, 12.
- Percona Distribution for PostgreSQL versions 11, 12.

(It should also work with versions 13 of both, but hasn't been tested.)

### Install

This extension can be installed in two ways:

- For Percona Distribution for PostgreSQL: Using standard Linux package manager tools.

- For PostgreSQL or Percona Distribution for PostgreSQL: [download and compile the source code](https://github.com/percona/pg_stat_monitor#installation).

#### Install using Linux package manager

The `pg-stat-monitor` extension is included in *Percona Distribution for PostgreSQL*. This can be installed via the `percona-release` package.

This section reproduces parts of the following:

- [Configuring Percona Repositories with percona-release](https://www.percona.com/doc/percona-repo-config/percona-release.html)

- [Installing Percona Distribution for PostgreSQL](https://www.percona.com/doc/postgresql/LATEST/installing.html)

##### Debian

```sh
sudo apt-get install -y wget gnupg2 lsb-release
wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
sudo dpkg -i percona-release_latest.generic_all.deb

sudo percona-release setup ppg-12 # version 12 (others available)
sudo apt install -y percona-postgresql-12
```

##### Red Hat

```sh
sudo yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm

# If RHEL 8
sudo dnf module disable postgresql

# If RHEL 7
sudo yum install -y epel-release
sudo yum repolist

sudo percona-release setup ppg-12
sudo yum install -y percona-postgresql12-server
```

#### Install from source code

##### Debian

1. Install common packages

    ```sh
    sudo apt-get install -y curl git wget gnupg2 lsb-release
    sudo apt-get update -y
    ```

2. Install PostgreSQL development packages

    With Percona Distribution for PostgreSQL (version 12):

    ```sh
    wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
    sudo dpkg -i percona-release_latest.generic_all.deb
    sudo percona-release setup ppg-12
    sudo apt install -y percona-postgresql-server-dev-all
    ```

    With PostgreSQL:

    ```sh
    wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo  apt-key add -
    echo "deb http://apt.postgresql.org/pub/repos/apt/ `lsb_release -cs`-pgdg main" | sudo tee /etc/apt/sources.list.d/pgdg.list
    sudo apt install -y postgresql-server-dev-all
    ```

3. Download, compile, and install extension

    ```sh
    git clone git://github.com/percona/pg_stat_monitor.git && cd pg_stat_monitor
    sudo make USE_PGXS=1
    sudo make USE_PGXS=1 install
    ```

##### Red Hat

1. Install common packages

    ```sh
    sudo yum install -y centos-release-scl epel-release
    sudo yum update -y
    sudo yum install -y git gcc gcc-c++ llvm-toolset-7
    ```

2. Install PostgreSQL development packages

    With Percona Distribution for PostgreSQL (version 12):

    ```sh
    sudo yum install -y https://repo.percona.com/yum/percona-release-latest.noarch.rpm
    sudo percona-release setup ppg-12
    sudo yum install -y percona-postgresql12-devel
    ```

    With PostgreSQL version 12:

    ```sh
    sudo yum install -y https://download.postgresql.org/pub/repos/yum/reporpms/EL-7-x86_64/pgdg-redhat-repo-latest.noarch.rpm
    sudo yum install -y postgresql12-devel
    ```

3. Download, compile, and install extension

    ```sh
    git clone git://github.com/percona/pg_stat_monitor.git && cd pg_stat_monitor
    sudo make PG_CONFIG=/usr/pgsql-12/bin/pg_config USE_PGXS=1
    sudo make PG_CONFIG=/usr/pgsql-12/bin/pg_config USE_PGXS=1 install
    ```

### Configure

1. Set or change the value for `shared_preload_library` in your `postgresql.conf` file:

    ```ini
    shared_preload_libraries = 'pg_stat_monitor'
    ```

2. Set the value

pg_stat_monitor.pgsm_normalized_query

2. Start or restart your PostgreSQL instance.

3. In a `psql` session:

    ```sql
    CREATE EXTENSION pg_stat_monitor;
    ```


### Configuration Parameters

Here are the configuration parameters, available values ranges, and default values. All require a restart of PostgreSQL except for `pg_stat_monitor.pgsm_track_utility` and `pg_stat_monitor.pgsm_normalized_query`.

To make settings permanent, add them to your `postgresql.conf` file before starting your PostgreSQL instance.

`pg_stat_monitor.pgsm_max` (5000-2147483647 bytes) Default: 5000
:    Defines the limit of shared memory. Memory is used by buckets in a circular manner and is divided between buckets equally when PostgreSQL starts.

`pg_stat_monitor.pgsm_query_max_len` (1024-2147483647 bytes) Default: 1024
:    The maximum size of the query. Long queries are truncated to this length to avoid unnecessary usage of shared memory. This parameter must be set before PostgreSQL starts.

`pg_stat_monitor.pgsm_enable` (0-1) Default: 1 (true).
:    Enables or disables monitoring. A value of `Disable` means that `pg_stat_monitor` will not collect statistics for the entire cluster.

`pg_stat_monitor.pgsm_track_utility` (0-1) Default: 1 (true)
:    Controls whether utility commands (all except SELECT, INSERT, UPDATE and DELETE) are tracked.

`pg_stat_monitor.pgsm_normalized_query` (0-1) Default: 0 (false)
:    By default, a query shows the actual parameter instead of a placeholder. Set to 1 to change to showing value placeholders (as `$n` where `n` is an integer).

`pg_stat_monitor.pgsm_max_buckets` (1-10) Default: 10
:    Sets the maximum number of available data buckets.

`pg_stat_monitor.pgsm_bucket_time` (1-2147483647 seconds) Default: 60
:    Sets the lifetime of the bucket. The system switches between buckets on the basis of this value.

`pg_stat_monitor.pgsm_object_cache` (50-2147483647) Default: 50
:    The maximum number of objects in the information cache.

`pg_stat_monitor.pgsm_respose_time_lower_bound` (1-2147483647 milliseconds) Default: 1
:    Sets the lower bound of the execution time histogram.

`pg_stat_monitor.pgsm_respose_time_step` (1-2147483647 milliseconds) Default: 1
:    Sets the time value of the steps for the histogram.

`pg_stat_monitor.pgsm_query_shared_buffer` (500000-2147483647 bytes) Default: 500000
:   Sets the query shared_buffer size.

`pg_stat_monitor.pgsm_track_planning` (0-1) Default: 1 (true)
:   Whether to track planning statistics.


## `pg_stat_statements`

`pg_stat_statements` is included in the official PostgreSQL `postgresql-contrib` available from your Linux distribution package manager.

### Install

#### Debian

```sh
sudo apt-get install postgresql-contrib
```

#### Red Hat

```sh
sudo yum install -y postgresql-contrib
```

### Configure

1. Add these lines to your `postgresql.conf` file:

    ```sh
    shared_preload_libraries = 'pg_stat_statements'
    track_activity_query_size = 2048 # Increase tracked query string size
    pg_stat_statements.track = all   # Track all statements including nested
    ```

2. Restart your PostgreSQL instance.

3. Install the extension (run in the `postgres` database).

    ```sh
    CREATE EXTENSION pg_stat_statements SCHEMA public;
    ```

## Adding PostgreSQL queries and metrics monitoring

You add PostgreSQL metrics and queries monitoring with the following command:

```sh
pmm-admin add postgresql --username=<user name> --password=<password>
```

Where `<user name>` and `<password>` are the PostgreSQL user credentials.

Additionally, two positional arguments can be appended to the command line
flags: a service name to be used by PMM, and a service address. If not
specified, they are substituted automatically as `<node>-postgresql` and
`127.0.0.1:5432`.

The command line and the output of this command may look as follows:

```sh
pmm-admin add postgresql --username=pmm --password=pmm postgres 127.0.0.1:5432
PostgreSQL Service added.
Service ID  : /service_id/28f1d93a-5c16-467f-841b-8c014bf81ca6
Service name: postgres
```

If correct installed and set up, you should be able to see data in PostgreSQL Overview dashboard, and also Query Analytics should contain PostgreSQL queries.

Beside positional arguments shown above you can specify service name and service address with the following flags: `--service-name`, `--host` (the hostname or IP address of the service), and `--port` (the port number of the service). If both flag and positional argument are present, flag gains higher priority. Here is the previous example modified to use these flags:

```sh
pmm-admin add postgresql --username=pmm --password=pmm --service-name=postgres --host=127.0.0.1 --port=270175432
```

It is also possible to add a PostgreSQL instance using a UNIX socket with just the `--socket` flag followed by the path to a socket:

```sh
pmm-admin add postgresql --socket=/var/run/postgresql
```

Capturing read and write time statistics is possible only if `track_io_timing` setting is enabled. This can be done either in configuration file or with the following query executed on the running system:


```sh
ALTER SYSTEM SET track_io_timing=ON;
SELECT pg_reload_conf();
```

!!! seealso "See also"

    - `pg_stat_monitor Github repository <https://github.com/percona/pg_stat_monitor>`__

    - `PostgreSQL pg_stat_statements module <https://www.postgresql.org/docs/current/pgstatstatements.html>`__
