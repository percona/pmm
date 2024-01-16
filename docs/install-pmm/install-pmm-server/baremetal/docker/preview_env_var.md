# Preview environment variables

!!! caution alert alert-warning "Warning"
     The `PERCONA_TEST_*` environment variables are experimental and subject to change. It is recommended that you use these variables for testing purposes only and not on production.

| Variable                                                                   | Description
| -------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------
| `PERCONA_TEST_SAAS_HOST`                                                   | SaaS server hostname.
| `PERCONA_TEST_PMM_CLICKHOUSE_ADDR`                                         | Name of the host and port of the external ClickHouse database instance.
| `PERCONA_TEST_PMM_CLICKHOUSE_DATABASE`                                     | Database name of the external ClickHouse database instance.
| `​​PERCONA_TEST_PMM_CLICKHOUSE_POOL_SIZE`                                    | The maximum number of threads in the current connection thread pool. This value cannot be bigger than max_thread_pool_size.
| `PERCONA_TEST_PMM_CLICKHOUSE_BLOCK_SIZE`                                   | The number of rows to load from tables in one block for this connection.
