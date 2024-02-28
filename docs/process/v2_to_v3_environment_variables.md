## Migrating PMM v2 environment variables to v3
We've renamed some environment variables used by PMM for consistency.
Below is a list of affected variables and their new names.

| PMM 2               | PMM 3              | Comments  |
|---------------------|--------------------|-----------|
|                     | `PMM_PERCONA_PLATFORM_API_TIMEOUT` |           |
|                     | `PMM_ENABLE_RBAC` |           |
|                     | `PMM_INTERFACE_TO_BIND` |           |
|                     | `PMM_PERCONA_PLATFORM_PUBLIC_KEY` |           |
|                     | `PMM_PERCONA_PLATFORM_ADDRESS` |           |
|                     | `PMM_PERCONA_PLATFORM_INSECURE` |           |
|                     | `PMM_DISABLE_UPDATES` |           |
|                     | `PMM_DISABLE_TELEMETRY` |           |
|                     | `PMM_DISABLE_ALERTING` |           |
|                     | `PMM_METRICS_RESOLUTION` |           |
|                     | `PMM_METRICS_RESOLUTION_MR` |           |
|                     | `PMM_METRICS_RESOLUTION_HR` |           |
|                     | `PMM_METRICS_RESOLUTION_LR` |           |
|                     | `PMM_DATA_RETENTION` |           |
|                     | `PMM_ENABLE_AZUREDISCOVER` |           |
|                     | `PMM_DEBUG` |           |
|                     | `PMM_TRACE` |           |
|                     | `PMM_TEST_VERSION_SERVICE_URL` |           |
|                     | `PMM_CLICKHOUSE_DATABASE` |           |
|                     | `PMM_CLICKHOUSE_BLOCK_SIZE` |           |
|                     | `PMM_CLICKHOUSE_ADDR` |           |
|                     | `PMM_CLICKHOUSE_POOL_SIZE` |           |
|                     | `PMM_DISABLE_UPDATES` |           |
|                     | `PMM_ENABLE_VM_CACHE` |           |
|                     | `PMM_DISABLE_BACKUP_MANAGEMENT` |           |
|                     | `PMM_TEST_AUTH_HOST` |           |
|                     | `PMM_TEST_CHECKS_HOST` |           |
|                     | `PMM_TEST_TELEMETRY_HOST` |           |
|                     | `PMM_TEST_SAAS_HOST` |           |
|                     | `PMM_TEST_CHECKS_PUBLIC_KEY` |           |
|                     | `PMM_VM_URL` |           |
|                     | `PMM_NO_PROXY` |           |
|                     | `PMM_HTTP_PROXY` |           |
|                     | `PMM_HTTPS_PROXY` |           |
|                     | `PMM_CONTAINER` |           |
|                     | `PMM_LESS_LOG_NOISE` |           |
|                     | `PMM_RELEASE_PATH` |           |
|                     | `PMM_TEST_TELEMETRY_INTERVAL` |           |
|                     | `PMM_TEST_TELEMETRY_DISABLE_START_DELAY` |           |
|                     | `PMM_TEST_TELEMETRY_RETRY_BACKOFF` |<br/>|                     | `PMM_ADVISORS_CHECKS_FILE` |           |
|                     | `PMM_POSTGRES_ADDR` |           |
|                     | `PMM_POSTGRES_DBNAME` |           |
|                     | `PMM_POSTGRES_USERNAME` |           |
|                     | `PMM_POSTGRES_DBPASSWORD` |           |
|                     | `PMM_POSTGRES_SSL_MODE` |           |
|                     | `PMM_POSTGRES_SSL_CA_PATH` |           |
|                     | `PMM_POSTGRES_SSL_KEY_PATH` |           |
|                     | `PMM_POSTGRES_SSL_CERT_PATH` |            |
|                     | `PMM_DISABLE_BUILTIN_POSTGRES` |           |