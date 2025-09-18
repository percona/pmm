# Configure environment variables for PMM Server (OVF/Virtual Appliance)

Configure PMM Server behavior on virtual appliances (OVF) by setting environment variables in the systemd environment file. This allows you to customize performance, storage, features, and other settings without modifying the container directly.

## Using environment variables

PMM Server in virtual appliances runs as a systemd user service that launches a Podman container. Environment variables are configured through a dedicated environment file.

### Configure environment variables

To set environment variables for PMM Server on OVF deployments:

{.power-number}

1. Connect to your PMM Server virtual machine via SSH using the `admin` user:

    ```bash
    ssh admin@<pmm-server-ip>
    ```

2. Edit the environment variables file:

    ```bash
    nano ~/.config/systemd/user/pmm-server.env
    ```

3. Add your desired environment variables in `KEY=VALUE` format:

    ```bash
    PMM_DATA_RETENTION=720h
    PMM_DEBUG=false
    PMM_ENABLE_TELEMETRY=true
    PMM_METRICS_RESOLUTION=5s
    ```

4. Restart the PMM Server service to apply the changes:

    ```bash
    systemctl --user restart pmm-server
    ```

5. Verify the service is running with the new configuration:

    ```bash
    systemctl --user status pmm-server
    ```

!!! note "File location"
    The environment file is located at `/home/admin/.config/systemd/user/pmm-server.env` and is automatically loaded by the systemd service.

## Core configuration variables

For detailed information about available environment variables, see the [Docker environment variables documentation](../docker/env_var.md). All variables documented for Docker deployments are supported in OVF deployments.

### Key variables for virtual appliances

These variables are particularly useful for virtual appliance deployments:

| Variable | Default | Description |
|----------|---------|-------------|
| `PMM_DATA_RETENTION` | `30d` | Duration to retain metrics data |
| `PMM_ENABLE_UPDATES` | `true` | Allow version checks and updates |
| `PMM_ENABLE_TELEMETRY` | `true` | Enable usage data collection |
| `PMM_DEBUG` | `false` | Enable verbose logging |
| `PMM_METRICS_RESOLUTION` | `1s` | Base metrics collection interval |
| `PMM_PUBLIC_ADDRESS` | Auto-detected | External DNS/IP for PMM Server |

## Example configurations

### High-performance production server

For production environments with high monitoring loads:

```bash
# Performance optimizations
PMM_DATA_RETENTION=90d
PMM_METRICS_RESOLUTION=5s
PMM_METRICS_RESOLUTION_HR=10s
PMM_METRICS_RESOLUTION_MR=30s
PMM_METRICS_RESOLUTION_LR=300s

# Disable non-essential features
PMM_ENABLE_TELEMETRY=false
PMM_DEBUG=false
```

### Development environment

For development and testing environments:

```bash
# Short retention for testing
PMM_DATA_RETENTION=7d

# Enable debugging
PMM_DEBUG=true
PMM_TRACE=true

# Disable telemetry
PMM_ENABLE_TELEMETRY=false
```

### Security-focused deployment

For environments with strict security requirements:

```bash
# Disable external communication features
PMM_ENABLE_UPDATES=false
PMM_ENABLE_TELEMETRY=false

# Set specific network binding
PMM_PUBLIC_ADDRESS=10.0.1.100
```

## Troubleshooting

### View current environment variables

To see the current environment variables being used by the PMM Server service:

```bash
systemctl --user show pmm-server --property=Environment
```

### Check service logs

If PMM Server fails to start after changing environment variables:

```bash
journalctl --user -u pmm-server -f
```

### Reset to defaults

To reset environment variables to defaults, edit the file and remove custom variables:

```bash
nano ~/.config/systemd/user/pmm-server.env
```

Keep only the required system variables:

```bash
PMM_WATCHTOWER_HOST=http://watchtower:8080
PMM_WATCHTOWER_TOKEN=123
PMM_IMAGE=docker.io/percona/pmm-server:3
PMM_DISTRIBUTION_METHOD=ovf
```

Then restart the service:

```bash
systemctl --user restart pmm-server
```

## Advanced configuration

### External database connections

Configure connections to external ClickHouse or PostgreSQL instances:

```bash
# External ClickHouse
PMM_CLICKHOUSE_ADDR=clickhouse.example.com:9000
PMM_CLICKHOUSE_DATABASE=pmm
PMM_CLICKHOUSE_USER=pmm
PMM_CLICKHOUSE_PASSWORD=secret
PMM_DISABLE_BUILTIN_CLICKHOUSE=true
```

### Grafana customization

Pass environment variables to the embedded Grafana instance:

```bash
# Grafana security settings
GF_SECURITY_ADMIN_PASSWORD=secure-password
GF_SECURITY_ADMIN_USER=admin
GF_SECURITY_DISABLE_GRAVATAR=true
```

### VictoriaMetrics tuning

Configure the embedded VictoriaMetrics instance:

```bash
# VictoriaMetrics memory settings
VM_retentionPeriod=90d
VM_memory.allowedPercent=60
```

For a complete list of available environment variables and their descriptions, see [Docker environment variables documentation](../docker/env_var.md).