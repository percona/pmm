# Environment variables in PMM

Configure PMM Server by setting Docker container environment variables using the `-e var=value` syntax:

```bash
docker run -e PMM_DATA_RETENTION=720h -e PMM_DEBUG=true percona/pmm-server:3
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

### Variables for migrating from PMM v2 to PMM v3

When migrating from PMM v2 to PMM v3, you'll need to update your environment variables to match the new naming convention. This is because PMM v3 introduces several important changes to improve consistency and clarity.

 To see the full lists of variable name changes between PMM v2 and PMM v3, see the [Migration guide](../../../../pmm-upgrade/migrating_from_pmm_2.md#variables-for-migrating-from-pmm-v2-to-pmm-v3).