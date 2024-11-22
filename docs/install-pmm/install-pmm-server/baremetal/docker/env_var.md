# Environment variables in PMM

Configure PMM Server by setting Docker container environment variables using the `-e var=value` syntax:

```bash
docker run -e PMM_DATA_RETENTION=720h -e PMM_DEBUG=true perconalab/pmm-server:3.0.0-beta
```

## Core configuration variables

### Performance & storage

| Variable | Default | Description | Example |
|----------|---------|-------------|----------|
| `PMM_DATA_RETENTION` | `30d` | Duration to retain metrics data. Must be in multiples of 24h. | `720h` (30 days) |
| `PMM_METRICS_RESOLUTION` | `1s` | Base metrics collection interval | `5s` |
| `PMM_METRICS_RESOLUTION_HR` | `5s` | High-resolution metrics interval | `10s` |
| `PMM_METRICS_RESOLUTION_MR` | `10s` | Medium-resolution metrics interval | `30s` |
| `PMM_METRICS_RESOLUTION_LR` | `60s` | Low-resolution metrics interval | `300s` |

### Feature flags

| Variable | Default | Effect when enabled |
|----------|---------|-------------------|
| `PMM_ENABLE_UPDATES` | `true` | Allows version checks and UI updates |
| `PMM_ENABLE_TELEMETRY` | `true` | Enables usage data collection |
| `PMM_ENABLE_ALERTING` | `true` | Enables Percona Alerting system |
| `PMM_ENABLE_BACKUP_MANAGEMENT` | `true` | Enables backup features |
| `PMM_ENABLE_AZURE_DISCOVER` | `false` | Enables Azure database discovery |

### Debugging

| Variable | Default | Purpose |
|----------|---------|---------|
| `PMM_DEBUG` | `false` | Enables verbose logging |
| `PMM_TRACE` | `false` | Enables detailed trace logging |

## Advanced configuration

### Networking

| Variable | Description |
|----------|-------------|
| `PMM_PUBLIC_ADDRESS` | External DNS/IP for PMM server |
| `PMM_INTERFACE_TO_BIND` | Network interface binding |

### Database connections

| Variable | Purpose |
|----------|----------|
| `PMM_CLICKHOUSE_*` | ClickHouse connection settings |
| `PMM_POSTGRES_*` | PostgreSQL connection settings |


### Supported external variables

- **Grafana**: All `GF_*` variables
- **VictoriaMetrics**: All `VM_*` variables
- **Kubernetes**: All `KUBERNETES_*` variables
- **System**: Standard variables like `HOME`, `PATH`, etc.

## Variables for migrating from PMM v2 to PMM v3

When migrating from PMM v2 to PMM v3, you'll need to update your environment variables to match the new naming convention. This is because PMM v3 introduces several important changes to improve consistency and clarity:

- environment variables now use `PMM_` prefix
- some boolean flags reversed (e.g., `DISABLE_` → `ENABLE_`)
- removed deprecated variables

### Examples

```bash
# PMM v2
-e DISABLE_UPDATES=true -e DATA_RETENTION=720h

# PMM v3 equivalent
-e PMM_ENABLE_UPDATES=false -e PMM_DATA_RETENTION=720h
```

### Migration reference table

??? note "Click to expand migration reference table"

    #### Configuration variables
    | PMM 2                          | PMM 3                              | Comments                      |
    |---------------------------------|------------------------------------|------------------------------|
    | `DATA_RETENTION`                | `PMM_DATA_RETENTION`               |                              |
    | `DISABLE_ALERTING`              | `PMM_ENABLE_ALERTING`              |                              |
    | `DISABLE_UPDATES`               | `PMM_ENABLE_UPDATES`               |                              |
    | `DISABLE_TELEMETRY`             | `PMM_ENABLE_TELEMETRY`             |                              |
    | `DISABLE_BACKUP_MANAGEMENT`      | `PMM_ENABLE_BACKUP_MANAGEMENT`     | Note the reverted boolean   |
    | `ENABLE_AZUREDISCOVER`          | `PMM_ENABLE_AZURE_DISCOVER`        |                              |
    | `ENABLE_RBAC`                   | `PMM_ENABLE_ACCESS_CONTROL`        |                              |
    | `LESS_LOG_NOISE`                |                                    | Removed in PMM v3            |
    
    #### Metrics configuration
    | PMM 2                          | PMM 3                              | 
    |---------------------------------|------------------------------------|
    | `METRICS_RESOLUTION`            | `PMM_METRICS_RESOLUTION`           | 
    | `METRICS_RESOLUTION_HR`         | `PMM_METRICS_RESOLUTION_HR`        | 
    | `METRICS_RESOLUTION_LR`         | `PMM_METRICS_RESOLUTION_LR`        | 
    | `METRICS_RESOLUTION_MR`         | `PMM_METRICS_RESOLUTION_MR`        |    
    
    
    #### ClickHouse configuration
    | PMM 2                               | PMM 3                              | Comments                 |
    |-------------------------------------|------------------------------------|--------------------------|
    | `PERCONA_TEST_PMM_CLICKHOUSE_ADDR`  | `PMM_CLICKHOUSE_ADDR`              |                          |
    | `PERCONA_TEST_PMM_CLICKHOUSE_DATABASE` | `PMM_CLICKHOUSE_DATABASE`         |                        |
    | `PERCONA_TEST_PMM_CLICKHOUSE_DATASOURCE` | `PMM_CLICKHOUSE_DATASOURCE`      |                       |
    | `PERCONA_TEST_PMM_CLICKHOUSE_HOST`  | `PMM_CLICKHOUSE_HOST`              |                          |
    | `PERCONA_TEST_PMM_CLICKHOUSE_PORT`  | `PMM_CLICKHOUSE_PORT`              |                          |
    | `PERCONA_TEST_PMM_DISABLE_BUILTIN_CLICKHOUSE` | `PMM_DISABLE_BUILTIN_CLICKHOUSE` |                  |
    | `PERCONA_TEST_PMM_CLICKHOUSE_BLOCK_SIZE` |                                    | Removed in PMM v3, new version|
    | `PERCONA_TEST_PMM_CLICKHOUSE_POOL_SIZE`  |                                    | Removed in PMM v3, new version|
    
    #### PostgreSQL configuration
    | PMM 2                               | PMM 3                              | 
    |-------------------------------------|------------------------------------|
    | `PERCONA_TEST_POSTGRES_ADDR`        | `PMM_POSTGRES_ADDR`                |
    | `PERCONA_TEST_POSTGRES_DBNAME`      | `PMM_POSTGRES_DBNAME`              |
    | `PERCONA_TEST_POSTGRES_USERNAME`    | `PMM_POSTGRES_USERNAME`            | 
    | `PERCONA_TEST_POSTGRES_DBPASSWORD`  | `PMM_POSTGRES_DBPASSWORD`          |  
    | `PERCONA_TEST_POSTGRES_SSL_CA_PATH` | `PMM_POSTGRES_SSL_CA_PATH`         | 
    | `PERCONA_TEST_POSTGRES_SSL_CERT_PATH` | `PMM_POSTGRES_SSL_CERT_PATH`      | 
    | `PERCONA_TEST_POSTGRES_SSL_KEY_PATH` | `PMM_POSTGRES_SSL_KEY_PATH`       |   
    | `PERCONA_TEST_POSTGRES_SSL_MODE`    | `PMM_POSTGRES_SSL_MODE`            |  
    | `PERCONA_TEST_PMM_DISABLE_BUILTIN_POSTGRES` | `PMM_DISABLE_BUILTIN_POSTGRES` |   
   
    #### Telemetry & development
    | PMM 2                               | PMM 3                              | 
    |-------------------------------------|------------------------------------|
    | `PMM_TEST_TELEMETRY_DISABLE_SEND`   | `PMM_DEV_TELEMETRY_DISABLE_SEND`   |                
    | `PERCONA_TEST_TELEMETRY_DISABLE_START_DELAY` | `PMM_DEV_TELEMETRY_DISABLE_START_DELAY` | 
    | `PMM_TEST_TELEMETRY_FILE`           | `PMM_DEV_TELEMETRY_FILE`           |   
    | `PERCONA_TEST_TELEMETRY_HOST`       | `PMM_DEV_TELEMETRY_HOST`           |   
    | `PERCONA_TEST_TELEMETRY_INTERVAL`   | `PMM_DEV_TELEMETRY_INTERVAL`       |      
    | `PERCONA_TEST_TELEMETRY_RETRY_BACKOFF` | `PMM_DEV_TELEMETRY_RETRY_BACKOFF` |   
    | `PERCONA_TEST_VERSION_SERVICE_URL`  | `PMM_DEV_VERSION_SERVICE_URL`      |         
    | `PERCONA_TEST_STARLARK_ALLOW_RECURSION` | `PMM_DEV_ADVISOR_STARLARK_ALLOW_RECURSION` |       
    
    #### Removed variables
    | PMM 2                               | PMM 3                              | Comments                     |
    |-------------------------------------|------------------------------------|------------------------------|
    | `PERCONA_TEST_AUTH_HOST`            |                                    | Removed, use `PMM_DEV_PERCONA_PLATFORM_ADDRESS` |
    | `PERCONA_TEST_CHECKS_HOST`          |                                    | Removed, use `PMM_DEV_PERCONA_PLATFORM_ADDRESS` |
    | `PERCONA_TEST_CHECKS_INTERVAL`      |                                    | Removed, not used            |
    | `PERCONA_TEST_CHECKS_PUBLIC_KEY`    |                                    | Removed, use `PMM_DEV_PERCONA_PLATFORM_PUBLIC_KEY` |
    | `PERCONA_TEST_NICER_API`            |                                    | Removed in PMM v3            |
    | `PERCONA_TEST_SAAS_HOST`            |                                    | Removed, use `PMM_DEV_PERCONA_PLATFORM_ADDRESS` |