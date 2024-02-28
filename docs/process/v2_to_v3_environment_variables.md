## Migrating PMM v2 environment variables to v3
We've renamed some environment variables used by PMM for consistency.
Below is a list of affected variables and their new names.

| PMM 2                                        | PMM 3                                    | Comments |
|----------------------------------------------|------------------------------------------|------|
| `PERCONA_PLATFORM_API_TIMEOUT`               | `PMM_PERCONA_PLATFORM_API_TIMEOUT`       |      |
| `ENABLE_RBAC`                                | `PMM_ENABLE_RBAC`                        |      |
| `PERCONA_TEST_INTERFACE_TO_BIND`             | `PMM_INTERFACE_TO_BIND`                  |      |
| `PERCONA_TEST_PLATFORM_PUBLIC_KEY`           | `PMM_PERCONA_PLATFORM_PUBLIC_KEY`        |      |
| `PERCONA_TEST_PLATFORM_ADDRESS`              | `PMM_PERCONA_PLATFORM_ADDRESS`           |      |
| `PERCONA_TEST_PLATFORM_INSECURE`             | `PMM_PERCONA_PLATFORM_INSECURE`          |      |
| `DISABLE_UPDATES`                            | `PMM_DISABLE_UPDATES`                    |      |
| `DISABLE_TELEMETRY`                          | `PMM_DISABLE_TELEMETRY`                  |      |
| `DISABLE_ALERTING`                           | `PMM_DISABLE_ALERTING`                   |      |
| `METRICS_RESOLUTION`                         | `PMM_METRICS_RESOLUTION`                 |      |
| `METRICS_RESOLUTION_MR`                      | `PMM_METRICS_RESOLUTION_MR`              |      |
| `METRICS_RESOLUTION_HR`                      | `PMM_METRICS_RESOLUTION_HR`              |      |
| `METRICS_RESOLUTION_LR`                      | `PMM_METRICS_RESOLUTION_LR`              |      |
| `DATA_RETENTION`                             | `PMM_DATA_RETENTION`                     |      |
| `ENABLE_AZUREDISCOVER`                       | `PMM_ENABLE_AZUREDISCOVER`               |      |
| `PERCONA_TEST_PMM_CLICKHOUSE_DATABASE`       | `PMM_CLICKHOUSE_DATABASE`                |      |
| `PERCONA_TEST_PMM_CLICKHOUSE_BLOCK_SIZE`     | `PMM_CLICKHOUSE_BLOCK_SIZE`              |      |
| `PERCONA_TEST_PMM_CLICKHOUSE_ADDR`           | `PMM_CLICKHOUSE_ADDR`                    |      |
| `PERCONA_TEST_PMM_CLICKHOUSE_POOL_SIZE`      | `PMM_CLICKHOUSE_POOL_SIZE`               |      |
| `DISABLE_UPDATES`                            | `PMM_DISABLE_UPDATES`                    |      |
| `ENABLE_VM_CACHE`                            | `PMM_ENABLE_VM_CACHE`                    |      |
| `DISABLE_BACKUP_MANAGEMENT`                  | `PMM_DISABLE_BACKUP_MANAGEMENT`          |      |
| `NO_PROXY`                                   | `PMM_NO_PROXY`                           |      |
| `HTTP_PROXY`                                 | `PMM_HTTP_PROXY`                         |      |
| `HTTPS_PROXY`                                | `PMM_HTTPS_PROXY`                        |      |
| `CONTAINER`                                  | `PMM_CONTAINER`                          |      |
| `LESS_LOG_NOISE`                             | `PMM_LESS_LOG_NOISE`                     |      |
| `PERCONA_TEST_CHECKS_FILE`                   | `PMM_ADVISORS_CHECKS_FILE`               |           |
| `PERCONA_TEST_STARLARK_ALLOW_RECURSION`      | `PMM_ADVISORS_STARLARK_ALLOW_RECURSION`               |           |
| `PERCONA_TEST_POSTGRES_ADDR`                 | `PMM_POSTGRES_ADDR`                      |      |
| `PERCONA_TEST_POSTGRES_DBNAME`               | `PMM_POSTGRES_DBNAME`                    |      |
| `PERCONA_TEST_POSTGRES_USERNAME`             | `PMM_POSTGRES_USERNAME`                  |      |
| `PERCONA_TEST_POSTGRES_DBPASSWORD`           | `PMM_POSTGRES_DBPASSWORD`                |      |
| `PERCONA_TEST_POSTGRES_SSL_MODE`             | `PMM_POSTGRES_SSL_MODE`                  |      |
| `PERCONA_TEST_POSTGRES_SSL_CA_PATH`          | `PMM_POSTGRES_SSL_CA_PATH`               |      |
| `PERCONA_TEST_POSTGRES_SSL_KEY_PATH`         | `PMM_POSTGRES_SSL_KEY_PATH`              |      |
| `PERCONA_TEST_POSTGRES_SSL_CERT_PATH`        | `PMM_POSTGRES_SSL_CERT_PATH`             |       |
| `PERCONA_TEST_PMM_DISABLE_BUILTIN_POSTGRES`  | `PMM_DISABLE_BUILTIN_POSTGRES`           |      |
| `PERCONA_TEST_NICER_API`  | `PMM_NICER_API`           |      |
| `PERCONA_TEST_AUTH_HOST`                     |  `PMM_TEST_AUTH_HOST`                    |      |
| `PERCONA_TEST_CHECKS_HOST`                   | `PMM_TEST_CHECKS_HOST`                   |      |
| `PERCONA_TEST_TELEMETRY_INTERVAL`            | `PMM_TEST_TELEMETRY_INTERVAL`            |      |
| `PERCONA_TEST_TELEMETRY_DISABLE_START_DELAY` | `PMM_TEST_TELEMETRY_DISABLE_START_DELAY` |      |
| `PERCONA_TEST_TELEMETRY_HOST`                | `PMM_TEST_TELEMETRY_HOST`                |      |
| `PERCONA_TEST_SAAS_HOST`                     | `PMM_TEST_SAAS_HOST`                     |      |
| `PERCONA_TEST_VERSION_SERVICE_URL`           | `PMM_TEST_VERSION_SERVICE_URL`           |      |
| `PERCONA_TEST_TELEMETRY_RETRY_BACKOFF`       | `PMM_TEST_TELEMETRY_RETRY_BACKOFF`       | |                 
| `PERCONA_TEST_CHECKS_PUBLIC_KEY`             | `PMM_TEST_CHECKS_PUBLIC_KEY`             |      |