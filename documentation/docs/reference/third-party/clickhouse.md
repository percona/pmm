# ClickHouse

You can use an external ClickHouse database instance outside the PMM Server container running on other hosts.

## Environment variables

PMM predefines certain flags that allow you to use ClickHouse parameters as environment variables.


To use ClickHouse as an external database instance, provide the following environment variables: 
 
`PMM_CLICKHOUSE_ADDR` -> hostname:port
:   Name of the host and port of the external ClickHouse database. 

`PMM_CLICKHOUSE_USER` -> username
:   Username to connect to the external ClickHouse database.

`PMM_CLICKHOUSE_PASSWORD` -> password
:   User password to connect to the external ClickHouse database.

**Optional environment variables**

`PMM_CLICKHOUSE_DATABASE` -> database name
:   Database name of the external ClickHouse database instance.

**Example**

To use ClickHouse as an external database instance, run PMM in docker or podman with the specified variables for external ClickHouse:
​​

```sh
-e PMM_CLICKHOUSE_ADDR=$ADDRESS:$PORT
-e PMM_CLICKHOUSE_DATABASE=$DB
-e PMM_CLICKHOUSE_USER=$CH_USER
-e PMM_CLICKHOUSE_PASSWORD=$CH_PASSWORD
```

## Troubleshooting

To troubleshoot issues, see the ClickHouse [troubleshooting documentation](https://clickhouse.com/docs/guides/troubleshooting).

