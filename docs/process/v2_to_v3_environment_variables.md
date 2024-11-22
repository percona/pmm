## Migrating PMM v2 environment variables to v3
We've renamed some environment variables used by PMM for consistency.
Below is a list of affected variables and their new names.

| PMM 2                                         | PMM 3                                      | Comments                                                     |
|-----------------------------------------------|--------------------------------------------|--------------------------------------------------------------|
| `DATA_RETENTION`                              | `PMM_DATA_RETENTION`                       |                                                              |
| `DISABLE_ALERTING`                            | `PMM_ENABLE_ALERTING`                      |                                                              |
| `DISABLE_UPDATES`                             | `PMM_ENABLE_UPDATES`                       |                                                              |
| `DISABLE_TELEMETRY`                           | `PMM_ENABLE_TELEMETRY`                     |                                                              |
| `PERCONA_PLATFORM_API_TIMEOUT`                | `PMM_DEV_PERCONA_PLATFORM_API_TIMEOUT`     |                                                              |
| `DISABLE_BACKUP_MANAGEMENT`                   | `PMM_ENABLE_BACKUP_MANAGEMENT`             | Note the reverted boolean                                    |
| `ENABLE_AZUREDISCOVER`                        | `PMM_ENABLE_AZURE_DISCOVER`                |                                                              |
| `ENABLE_RBAC`                                 | `PMM_ENABLE_ACCESS_CONTROL`                |                                                              |
| `LESS_LOG_NOISE`                              |                                            | Removed in PMM v3                                            |
| `METRICS_RESOLUTION`                          | `PMM_METRICS_RESOLUTION`                   |                                                              |
| `METRICS_RESOLUTION_HR`                       | `PMM_METRICS_RESOLUTION_HR`                |                                                              |
| `METRICS_RESOLUTION_LR`                       | `PMM_METRICS_RESOLUTION_LR`                |                                                              |
| `METRICS_RESOLUTION_MR`                       | `PMM_METRICS_RESOLUTION_MR`                |                                                              |
| `OAUTH_PMM_CLIENT_ID`                         | `PMM_DEV_OAUTH_CLIENT_ID`                  |                                                              |
| `OAUTH_PMM_CLIENT_SECRET`                     | `PMM_DEV_OAUTH_CLIENT_SECRET`              |                                                              |
| `PERCONA_TEST_AUTH_HOST`                      |                                            | Removed in PMM v3, use `PMM_DEV_PERCONA_PLATFORM_ADDRESS`    |
| `PERCONA_TEST_CHECKS_FILE`                    | `PMM_DEV_ADVISOR_CHECKS_FILE`              |                                                              |
| `PERCONA_TEST_CHECKS_HOST`                    |                                            | Removed in PMM v3, use `PMM_DEV_PERCONA_PLATFORM_ADDRESS`    |
| `PERCONA_TEST_CHECKS_INTERVAL`                |                                            | Removed in PMM v3 as it wasn't actually used.                |
| `PERCONA_TEST_CHECKS_PUBLIC_KEY`              |                                            | Removed in PMM v3, use `PMM_DEV_PERCONA_PLATFORM_PUBLIC_KEY` |
| `PERCONA_TEST_NICER_API`                      |                                            | Removed in PMM v3                                            |
| `PERCONA_TEST_PMM_CLICKHOUSE_ADDR`            | `PMM_CLICKHOUSE_ADDR`                      |                                                              |
| `PERCONA_TEST_PMM_CLICKHOUSE_BLOCK_SIZE`      |                                            | Removed in PMM v3, because of new clickhouse version.        |
| `PERCONA_TEST_PMM_CLICKHOUSE_DATABASE`        | `PMM_CLICKHOUSE_DATABASE`                  |                                                              |
| `PERCONA_TEST_PMM_CLICKHOUSE_DATASOURCE`      | `PMM_CLICKHOUSE_DATASOURCE`                |                                                              |
| `PERCONA_TEST_PMM_CLICKHOUSE_HOST`            | `PMM_CLICKHOUSE_HOST`                      |                                                              |
| `PERCONA_TEST_PMM_CLICKHOUSE_POOL_SIZE`       |                                            | Removed in PMM v3, because of new clickhouse version.        |
| `PERCONA_TEST_PMM_CLICKHOUSE_PORT`            | `PMM_CLICKHOUSE_PORT`                      |                                                              |
| `PERCONA_TEST_PMM_DISABLE_BUILTIN_CLICKHOUSE` | `PMM_DISABLE_BUILTIN_CLICKHOUSE`           |                                                              |
| `PERCONA_TEST_PMM_DISABLE_BUILTIN_POSTGRES`   | `PMM_DISABLE_BUILTIN_POSTGRES`             |                                                              |
| `PERCONA_TEST_INTERFACE_TO_BIND`              | `PMM_INTERFACE_TO_BIND`                    |                                                              |
| `PERCONA_TEST_PLATFORM_ADDRESS`               | `PMM_DEV_PERCONA_PLATFORM_ADDRESS`         |                                                              |
| `PERCONA_TEST_PLATFORM_INSECURE`              | `PMM_DEV_PERCONA_PLATFORM_INSECURE`        |                                                              |
| `PERCONA_TEST_PLATFORM_PUBLIC_KEY`            | `PMM_DEV_PERCONA_PLATFORM_PUBLIC_KEY`      |                                                              |
| `PERCONA_TEST_POSTGRES_ADDR`                  | `PMM_POSTGRES_ADDR`                        |                                                              |
| `PERCONA_TEST_POSTGRES_DBNAME`                | `PMM_POSTGRES_DBNAME`                      |                                                              |
| `PERCONA_TEST_POSTGRES_SSL_CA_PATH`           | `PMM_POSTGRES_SSL_CA_PATH`                 |                                                              |
| `PERCONA_TEST_POSTGRES_SSL_CERT_PATH`         | `PMM_POSTGRES_SSL_CERT_PATH`               |                                                              |
| `PERCONA_TEST_POSTGRES_SSL_KEY_PATH`          | `PMM_POSTGRES_SSL_KEY_PATH`                |                                                              |
| `PERCONA_TEST_POSTGRES_SSL_MODE`              | `PMM_POSTGRES_SSL_MODE`                    |                                                              |
| `PERCONA_TEST_POSTGRES_DBPASSWORD`            | `PMM_POSTGRES_DBPASSWORD`                  |                                                              |
| `PERCONA_TEST_SAAS_HOST`                      |                                            | Removed in PMM v3, use `PMM_DEV_PERCONA_PLATFORM_ADDRESS`    |
| `PERCONA_TEST_POSTGRES_USERNAME`              | `PMM_POSTGRES_USERNAME`                    |                                                              |
| `PERCONA_TEST_STARLARK_ALLOW_RECURSION`       | `PMM_DEV_ADVISOR_STARLARK_ALLOW_RECURSION` |                                                              |
| `PMM_TEST_TELEMETRY_DISABLE_SEND`             | `PMM_DEV_TELEMETRY_DISABLE_SEND`           |                                                              |
| `PERCONA_TEST_TELEMETRY_DISABLE_START_DELAY`  | `PMM_DEV_TELEMETRY_DISABLE_START_DELAY`    |                                                              |
| `PMM_TEST_TELEMETRY_FILE`                     | `PMM_DEV_TELEMETRY_FILE`                   |                                                              |
| `PERCONA_TEST_TELEMETRY_HOST`                 | `PMM_DEV_TELEMETRY_HOST`                   |                                                              |
| `PERCONA_TEST_TELEMETRY_INTERVAL`             | `PMM_DEV_TELEMETRY_INTERVAL`               |                                                              |
| `PERCONA_TEST_TELEMETRY_RETRY_BACKOFF`        | `PMM_DEV_TELEMETRY_RETRY_BACKOFF`          |                                                              |                 
| `PERCONA_TEST_VERSION_SERVICE_URL`            |                                            | Removed in PMM v3, use `PMM_DEV_PERCONA_PLATFORM_ADDRESS`    |
