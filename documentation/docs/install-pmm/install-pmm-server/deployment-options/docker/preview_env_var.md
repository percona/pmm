# Preview environment variables

!!! caution alert alert-warning "Preview features"
    The variables on this page are in preview and can change in future releases. Use them for testing only, not in production.

    For production-ready configuration options, see [Environment variables for PMM Server](../docker/env_var.md).

## External ClickHouse variables

| Variable                          | Description
| ----------------------------------| --------------------------------------------------------------------------
| `PMM_CLICKHOUSE_ADDR`             | Host and port of the external ClickHouse database instance
| `PMM_CLICKHOUSE_DATABASE`         | Database name of the external ClickHouse instance
| `PMM_CLICKHOUSE_USER`             | Database user
| `PMM_CLICKHOUSE_PASSWORD`         | Database user password
| `PMM_CLICKHOUSE_DATASOURCE_USER`  | Read-only database user for the Grafana ClickHouse data source
| `PMM_CLICKHOUSE_DATASOURCE_PASSWORD` | Read-only data source user password

## Testing external ClickHouse connections
The ClickHouse-related preview variables are useful for testing PMM Server with an external ClickHouse instance:
{.power-number}
    
1. Set up a test ClickHouse instance.
2. Configure the connection using the variables above.
3. Launch PMM Server and verify it connects and stores metrics correctly.
4. Monitor logs to validate how metrics display.

