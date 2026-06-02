# Use external ClickHouse with PMM

You can use an external ClickHouse database instance outside the PMM Server container running on other hosts.

## Environment variables

PMM predefines certain flags that allow you to use ClickHouse parameters as environment variables.


To use ClickHouse as an external database instance, provide the following environment variables: 
 
`PMM_CLICKHOUSE_ADDR` -> hostname:port
:   Name of the host and port of the external ClickHouse database instance. 

`PMM_CLICKHOUSE_HOST` -> hostname
:   Hostname of the external ClickHouse database.

`PMM_CLICKHOUSE_PORT` -> port
:   Port of the external ClickHouse database.

`PMM_CLICKHOUSE_USER` -> username
:   Username to connect to the external ClickHouse database.

`PMM_CLICKHOUSE_PASSWORD` -> password
:   User password to connect to the external ClickHouse database.

`PMM_DISABLE_BUILTIN_CLICKHOUSE` -> 1
:   Disables the built-in ClickHouse database instance.

**Optional environment variables**

`PMM_CLICKHOUSE_DATABASE` -> database name
:   Database name of the external ClickHouse database instance.

`PMM_CLICKHOUSE_POOL_SIZE` -> number of connections
:   Maximum number of ClickHouse connections used by Query Analytics (QAN), shared by data ingestion and analytics. Default: `20`. Applies to both the built-in and an external ClickHouse instance. See [Tune the ClickHouse connection pool size](#tune-the-clickhouse-connection-pool-size).
 
**Example**

To use ClickHouse as an external database instance, run PMM in docker or podman with the specified variables for external ClickHouse:

```sh
-e PMM_CLICKHOUSE_ADDR=<hostname>:<port>
-e PMM_CLICKHOUSE_DATABASE=<database-name>
-e PMM_CLICKHOUSE_USER=<username>
-e PMM_CLICKHOUSE_PASSWORD=<password>
-e PMM_DISABLE_BUILTIN_CLICKHOUSE=1
```

Alternatively, you can use the `PMM_CLICKHOUSE_HOST` and `PMM_CLICKHOUSE_PORT` variables instead of `PMM_CLICKHOUSE_ADDR`.

```sh
-e PMM_CLICKHOUSE_HOST=<hostname>
-e PMM_CLICKHOUSE_PORT=<port>
-e PMM_CLICKHOUSE_DATABASE=<database-name>
-e PMM_CLICKHOUSE_USER=<username>
-e PMM_CLICKHOUSE_PASSWORD=<password>
-e PMM_DISABLE_BUILTIN_CLICKHOUSE=1
```

## Tune the ClickHouse connection pool size

Query Analytics (QAN) keeps a pool of connections to ClickHouse that is shared between two workloads:

- **Data ingestion** — one connection used to write incoming query metrics.
- **Analytics** — each QAN report or filter panel a user opens runs several ClickHouse queries in parallel (up to 4 per page).

Use `PMM_CLICKHOUSE_POOL_SIZE` (default: `20`) to set the maximum number of connections in this pool. If the pool is too small, QAN requests queue and can time out under concurrent use; if it is too large, too many heavy ClickHouse queries can run at once and exhaust ClickHouse CPU or memory.

### Formula

```text
PMM_CLICKHOUSE_POOL_SIZE = 1 + (C × 4)
```

where:

- `1` — one connection reserved for QAN data ingestion.
- `C` — the peak number of users or browser tabs loading QAN reports at the same time.
- `4` — the maximum number of queries a single QAN page runs in parallel (fixed internally).

**Example:** to support 4 concurrent QAN users, set `1 + (4 × 4) = 17`, rounded up to the default of `20`.

### Guidelines

- **Minimum `5`** — enough for one QAN page (4 queries) plus ingestion.
- **Upper bound** — a larger pool only helps up to what ClickHouse can serve. QAN report queries are heavy scans, so as a rule of thumb keep the pool at or below the number of ClickHouse CPU cores, unless you have confirmed there is enough memory for that many concurrent aggregations.
- Raise the value only if you observe QAN requests queuing during peak usage **and** ClickHouse has spare CPU and memory; otherwise the default is sufficient for a typical single-host deployment.

Set it like any other PMM environment variable, for example:

```sh
docker run -d ... -e PMM_CLICKHOUSE_POOL_SIZE=30 ... percona/pmm-server:3
```

## Enhance ClickHouse security for PMM

When configuring PMM to use an external ClickHouse instance, make sure to enforce robust security practices to protect sensitive data and prevent unauthorized access:

- Enable SSL/TLS encryption for all connections
- Ensure that your ClickHouse instance is properly secured and monitored
- Disable empty passwords and plain text passwords
- Define all ClickHouse users explicitly, including permissions, to prevent automatic creation of unsecured users without passwords.
- Generate strong, random passwords for the dedicated PMM ClickHouse user. Use the following commands to generate a password and its SHA256 hash (useful for advanced ClickHouse configurations):

    ```sh
    PASSWORD=$(base64 < /dev/urandom | head -c12)
    echo "$PASSWORD" # note it down
    echo -n "$PASSWORD" | sha256sum | tr -d '-'
    ```

For more details, see the [ClickHouse user and roles settings](https://clickhouse.com/docs/operations/settings/settings-users).

## Troubleshooting

To troubleshoot issues, see the ClickHouse [troubleshooting documentation](https://clickhouse.com/docs/guides/troubleshooting).
