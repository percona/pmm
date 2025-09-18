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
    PMM_DEBUG=true
    PMM_ENABLE_TELEMETRY=false
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



## Advanced configuration

### VictoriaMetrics tuning

Configure the embedded VictoriaMetrics instance:

```bash
# VictoriaMetrics memory settings
VM_retentionPeriod=90d
VM_memory.allowedPercent=60
```

For a complete list of available environment variables and their descriptions, see [Docker environment variables documentation](../docker/env_var.md).