# Preview environment variables

!!! caution alert alert-warning "Warning"
     The `PERCONA_TEST_*` environment variables are experimental and subject to change. These variables are intended for testing purposes only and should not be used in production environments.

     For stable, production-ready configuration options, see the main [Environment variables for PMM Server](../docker/env_var.md) documentation.

## Available preview variables

| Variable                                                      | Description
| ------------------------------------------------------------- | --------------------------------------------------------------------------
| `PERCONA_TEST_SAAS_HOST`                                      | SaaS server hostname.
| `PMM_CLICKHOUSE_ADDR`                                         | Name of the host and port of the external ClickHouse database instance
| `PMM_CLICKHOUSE_DATABASE`                                     | Database name of the external ClickHouse database instance
| `​​PMM_CLICKHOUSE_USER`                                         | Database username
| `PMM_CLICKHOUSE_PASSWORD`                                     | Database username password


## Using preview variables

Add preview variables to your `docker run` command for testing purposes:

```sh 
docker run -d \
  --name pmm-server \
  -e PERCONA_TEST_PMM_CLICKHOUSE_ADDR=clickhouse-test:9000 \
  -e PERCONA_TEST_PMM_CLICKHOUSE_DATABASE=pmm_test \
  percona/pmm-server:3
```

## Testing external ClickHouse connections
The ClickHouse-related preview variables are useful for testing PMM Server with an external ClickHouse instance:
{.power-number}
    
1. Set up a test ClickHouse instance.
2. Configure the connection using the variables above.
3. Launch PMM Server and verify it connects and stores metrics correctly.
4. Monitor logs to validate how metrics display.

