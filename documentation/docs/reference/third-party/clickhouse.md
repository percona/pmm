# ClickHouse

You can use an external ClickHouse database instance outside the PMM Server container running on other hosts.

## Environment variables

PMM predefines certain flags that allow you to use ClickHouse parameters as environment variables.

To use ClickHouse as an external database instance, use the following environment variables: 
 
`PMM_CLICKHOUSE_ADDR` -> hostname:port
:   Name of the host and port of the external ClickHouse database instance. 

**Optional environment variables**

`PMM_CLICKHOUSE_DATABASE` -> database name
:   Database name of the external ClickHouse database instance.
 
**Example**

To use ClickHouse as an external database instance, start the PMM docker with the specified variables for external ClickHouse:

```sh
-e PMM_CLICKHOUSE_ADDR=$ADDRESS:$PORT
-e PMM_CLICKHOUSE_DATABASE=$DB
```

## Troubleshooting

To troubleshoot issues, see the ClickHouse [troubleshooting documentation](https://clickhouse.com/docs/en/guides/troubleshooting/).

