# Configure environment variables for PMM Server
Configure PMM Server behavior by setting environment variables when running the Docker container. This allows you to customize performance, storage, features, and other settings without modifying configuration files.

## Using environment variables

Configure PMM Server by setting Docker container environment variables using the `-e var=value` syntax:

```bash
docker run -e PMM_DATA_RETENTION=720h -e PMM_DEBUG=true percona/pmm-server:3
```

## Core configuration variables

### Performance & storage
Fine-tune data retention and collection intervals to balance monitoring detail with resource usage:

| Variable | Default | Description | Example |
|----------|---------|-------------|----------|
| `PMM_DATA_RETENTION` | `30d` | Duration to retain metrics data (must be in multiples of 24h) | `720h` (30 days) |
| `PMM_METRICS_RESOLUTION` | `1s` | Base metrics collection interval | `5s` |
| `PMM_METRICS_RESOLUTION_HR` | `5s` | High-resolution metrics interval | `10s` |
| `PMM_METRICS_RESOLUTION_MR` | `10s` | Medium-resolution metrics interval | `30s` |
| `PMM_METRICS_RESOLUTION_LR` | `60s` | Low-resolution metrics interval | `300s` |

!!! tip "Performance impact"
    Higher resolution (lower values) provides more detailed metrics but increases storage requirements and system load. For high-traffic production environments, consider increasing these values.

### Feature controls
Enable or disable specific PMM features:

| Variable | Default | Effect when enabled |
|----------|---------|-------------------|
| `PMM_ENABLE_UPDATES` | `true` | Allows version checks and UI updates |
| `PMM_ENABLE_TELEMETRY` | `true` | Enables usage data collection |
| `PMM_ENABLE_ALERTING` | `true` | Enables Percona Alerting system |
| `PMM_ENABLE_BACKUP_MANAGEMENT` | `true` | Enables backup features |
| `PMM_ENABLE_AZURE_DISCOVER` | `false` | Enables Azure database discovery |

### Debugging and troubleshooting
Use these variables when diagnosing issues with PMM Server:

| Variable | Default | Purpose |
|----------|---------|---------|
| `PMM_DEBUG` | `false` | Enables verbose logging |
| `PMM_TRACE` | `false` | Enables detailed trace logging |

!!! warning "Production use"
    Debug and trace logging can significantly impact performance and generate large log volumes. Use only temporarily when troubleshooting issues.

## PMM Client configuration variables

Instead of configuring each client individually, set environment variables once on the PMM Server to automatically apply settings across your entire monitoring infrastructure.

This centralized approach lets you manage disk usage limits, logging levels, and resource constraints from a single location. 

For instance, PMM Clients can consume up to 1GB of disk space by default, which can quickly exhaust shared volumes in containerized environments like Kubernetes.

To avoid storage issues and maintain consistency, configure disk usage limits on the PMM Server that apply to all clients automatically. For most deployments, consider reducing disk usage to 50–100MB per client.

### Configure `vmagent` settings

Control `vmagent` behavior on all PMM Clients by setting `VMAGENT_*` environment variables on PMM Server. 

This gives you centralized control over resource usage, logging, and storage settings without touching individual client deployments.

#### Essential variables

These environment variables prevent the most common deployment challenges. For all available configuration options, see [Victoria metrics command-line flags](https://docs.victoriametrics.com/#list-of-command-line-flags)


| Variable | Purpose | Example | Default | Notes |
|----------|---------|---------|---------|-------|
| `VMAGENT_remoteWrite_maxDiskUsagePerURL` | Disk space limit per client during network outages | `52428800` (50MB) | `1073741824` (1GB) | Critical for Kubernetes |
| `VMAGENT_remoteWrite_tmpDataPath` | Temporary data storage path | `/tmp/custom-vmagent` | `/tmp/vmagent-temp-dir` | |
| `VMAGENT_loggerLevel` | Logging verbosity level | `DEBUG`, `WARN`, `ERROR` | `INFO` | |
| `VMAGENT_promscrape_maxScrapeSize` | Maximum scrape size per target | `128MiB`, `32MiB` | `64MiB` | Controls memory usage |
| `VMAGENT_remoteWrite_basicAuth_username` | Basic auth username for external VictoriaMetrics | `admin` | Template-based | Auto-extracted from URL |
| `VMAGENT_remoteWrite_basicAuth_password` | Basic auth password for external VictoriaMetrics | `secret` | Template-based | Auto-extracted from URL |
| `VMAGENT_remoteWrite_tlsInsecureSkipVerify` | Skip TLS certificate verification | `true`, `false` | `false` | Security setting |

### Convert flags to environment variables

`Vmagent` uses command-line flags for configuration. 

To manage these settings centrally through PMM Server, convert the flags to environment variables using this pattern:

- add `VMAGENT_ prefix`
- replace hyphens (`-`)with underscores (`_`)
- replace dots (`.`) with underscores (_)

#### Examples

- `remoteWrite.maxDiskUsagePerURL` > `VMAGENT_remoteWrite_maxDiskUsagePerURL`

- `loggerLevel` > `VMAGENT_loggerLevel`

- `httpListenAddr` > `VMAGENT_httpListenAddr`

### Common deployment scenarios

Here are some examples for typical production configurations for different environments. 

Set these variables on PMM Server startup to automatically configure all connected clients.

#### Kubernetes with shared storage

Optimize for disk space constraints:

```sh
docker run \
  -e VMAGENT_remoteWrite_maxDiskUsagePerURL=52428800 \
  -e VMAGENT_promscrape_maxScrapeSize=32MiB \
  -e VMAGENT_loggerLevel=WARN \
  percona/pmm-server:3
```

#### High-traffic environments
Balance resource usage with monitoring detail:

```sh
docker run \
  -e VMAGENT_remoteWrite_maxDiskUsagePerURL=104857600 \
  -e VMAGENT_promscrape_maxScrapeSize=128MiB \
  -e VMAGENT_loggerLevel=ERROR \
  percona/pmm-server:3
```

### Development and debugging
Enable detailed logging for troubleshooting:

```sh
docker run \
  -e VMAGENT_loggerLevel=DEBUG \
  -e VMAGENT_remoteWrite_maxDiskUsagePerURL=26214400 \
  -e VMAGENT_promscrape_maxScrapeSize=64MiB \
  percona/pmm-server:3
```

Configuration changes apply to all connected PMM Clients automatically without requiring restarts.

## Verify configuration
Confirm that the environment variables set on PMM Server are being applied to the `vmagent` processes: 

### Check if `vmagent` is running

```sh
pidof vmagent
# Should return a process ID
```

### Check configuration is applied
Check that configuration parameters are present:

```sh
ps aux | grep vmagent | grep -v grep | wc -w
# Should return > 10 (indicating multiple arguments)
```

### Verify network connectivity
Check that `vmagent` is listening on expected port:

```sh
netstat -tulpn | grep $(pidof vmagent)
```

## Advanced configuration

### Network configuration
Control how PMM Server presents itself on the network:

| Variable | Description |
|----------|-------------|
| `PMM_PUBLIC_ADDRESS` | External DNS/IP for PMM server |
| `PMM_INTERFACE_TO_BIND` | Network interface binding |

### Database connections
Configure connections to external database services:

| Variable | Purpose |
|----------|----------|
| `PMM_CLICKHOUSE_*` | ClickHouse connection settings |
| `PMM_POSTGRES_*` | PostgreSQL connection settings |

### Supported external variables
PMM Server passes these variables to integrated components:

- **Grafana**: All `GF_*` variables (e.g., `GF_SECURITY_ADMIN_PASSWOR`)
- **VictoriaMetrics**: All `VM_*` variables (e.g., VM_retentionPeriod)
- **Kubernetes**: All `KUBERNETES_*` variables
- **System variables**: Standard variables like `HOME`, `PATH`, etc.


## Experimental variables

PMM includes experimental environment variables prefixed with `PERCONA_TEST_*` that are under development and subject to change. To see the complete list and details of experimental variables, see [Preview environment variables](preview_env_var.md).


!!! caution "For testing only"
    Experimental variables are not supported for production use. Use these variables for testing purposes only.


### Variables for migrating from PMM v2 to PMM v3

PMM v3 introduces several important changes to improve consistency and clarity. When migrating from PMM v2 to PMM v3, you'll need to update your environment variables to match the new naming convention: 

For example:

- `METRICS_RESOLUTION` → `PMM_METRICS_RESOLUTION`
- `METRICS_RESOLUTION_HR` → `PMM_METRICS_RESOLUTION_HR`

To see the full lists of variable name changes between PMM v2 and PMM v3, see the [Migration guide](../../../../pmm-upgrade/migrating_from_pmm_2.md#variables-for-migrating-from-pmm-v2-to-pmm-v3).

## Common configuration examples

### High-traffic production server
For environments with many monitored systems, increase collection intervals and retention:

``` sh
docker run \
 -e PMM_DATA_RETENTION=45d \
 -e PMM_METRICS_RESOLUTION=5s \
 -e PMM_METRICS_RESOLUTION_HR=30s \
 -e PMM_METRICS_RESOLUTION_MR=60s \
 -e PMM_METRICS_RESOLUTION_LR=300s \
 percona/pmm-server:3
```

### Development environment
For testing and development, you might want debugging enabled:

```sh
docker run \
 -e PMM_DATA_RETENTION=7d \
 -e PMM_DEBUG=true \
 -e PMM_ENABLE_TELEMETRY=false \
 percona/pmm-server:3
``` 

### Restricted features
To disable certain features for security or policy reasons:

```sh
docker run \
 -e PMM_ENABLE_UPDATES=false \
 -e PMM_ENABLE_TELEMETRY=false \
 -e PMM_ENABLE_BACKUP_MANAGEMENT=false \
 percona/pmm-server:3
```

## Related topic
[Preview environment variables](../docker/preview_env_var.md)