# ClickHouse

You can use an external ClickHouse database instance outside the PMM Server container running on other hosts.

## Environment variables

PMM predefines certain flags that allow you to use ClickHouse parameters as environment variables.


To use ClickHouse as an external database instance, provide the following environment variables: 
 
`PMM_CLICKHOUSE_ADDR` -> hostname:port
:   Hostname and port of the external ClickHouse database.

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

**Example**

To use ClickHouse as an external database instance, run PMM in docker or podman with the specified variables for external ClickHouse:

```sh
-e PMM_CLICKHOUSE_ADDR=$CH_HOST:$CH_PORT
-e PMM_CLICKHOUSE_DATABASE=$CH_DATABASE
-e PMM_CLICKHOUSE_USER=$CH_USER
-e PMM_CLICKHOUSE_PASSWORD=$CH_PASSWORD
-e PMM_DISABLE_BUILTIN_CLICKHOUSE=1
```

Alternatively, you can use the `PMM_CLICKHOUSE_HOST` and `PMM_CLICKHOUSE_PORT` variables instead of `PMM_CLICKHOUSE_ADDR`.

```sh
-e PMM_CLICKHOUSE_HOST=$CH_HOST
-e PMM_CLICKHOUSE_PORT=$CH_PORT
-e PMM_CLICKHOUSE_DATABASE=$CH_DATABASE
-e PMM_CLICKHOUSE_USER=$CH_USER
-e PMM_CLICKHOUSE_PASSWORD=$CH_PASSWORD
-e PMM_DISABLE_BUILTIN_CLICKHOUSE=1
```

## Troubleshooting

To troubleshoot issues, see the ClickHouse [troubleshooting documentation](https://clickhouse.com/docs/guides/troubleshooting).

