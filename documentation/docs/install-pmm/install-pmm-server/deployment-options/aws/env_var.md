# Configure environment variables for PMM Server on AWS

Set up environment variables in the `systemd` environment file to customize performance, storage, features, and other settings without modifying the container directly.

Your PMM Server on AWS runs as a `systemd` user service that launches a Podman container, with environment variables configured through a dedicated environment file.

## Configure environment variables

To set environment variables for PMM Server on AWS:
{.power-number}

1. Connect to your PMM Server instance via SSH using the `admin` user:
    ```bash
    ssh -i your-key.pem admin@<pmm-server-ip>
    ```

2. Open `/home/admin/.config/systemd/user/pmm-server.env` and edit the environment variables file. This file is automatically loaded by the `systemd` service:

    ```bash
    nano ~/.config/systemd/user/pmm-server.env
    ```

3. Add your desired environment variables in `KEY=VALUE` format:

    ```bash
    PMM_DEBUG=true
    PMM_ENABLE_ACCESS_CONTROL=true
    ```

4. Restart the PMM Server service to apply the changes:

    ```bash
    systemctl --user restart pmm-server
    ```

5. Verify the service is running with the new configuration:

    ```bash
    systemctl --user status pmm-server
    ```

## Available variables

Unlike Docker deployments that use `-e` flags, AWS AMI instances configure PMM Server using the systemd environment file instructions above. However, PMM uses the same [list of environment variables](../docker/env_var.md) across all deployment methods.

### Common examples for AWS AMI

These variables are particularly useful for AWS AMI deployments:

| Variable | Default | Description |
|----------|---------|-------------|
| `PMM_DATA_RETENTION` | `30d` | Duration to retain metrics data |
| `PMM_ENABLE_UPDATES` | `true` | Allow version checks and updates |
| `PMM_ENABLE_ACCESS_CONTROL` | `true` | Enable label-based access control (LBAC) |
| `PMM_PUBLIC_ADDRESS` | Auto-detected | External DNS/IP for PMM Server |
| `PMM_DEBUG` | `true` | Enable verbose logging |
| `PMM_METRICS_RESOLUTION` | `1s` | Base metrics collection interval |

### VictoriaMetrics performance tuning

PMM Server uses VictoriaMetrics as its metrics storage engine. For high-volume environments or extended retention periods, you may need to tune VictoriaMetrics settings to optimize performance and resource usage. To do this:
{.power-number}

1. Add the following variables to your `pmm-server.env` file:

    ```bash
    # Configure disk space limit per client during network outages
    VMAGENT_remoteWrite_maxDiskUsagePerURL=52428800

    # Configure temporary data storage path
    VMAGENT_remoteWrite_tmpDataPath=/tmp/custom-vmagent

    # Configure logging verbosity level
    VMAGENT_loggerLevel=DEBUG

    # Configure maximum scrape size per target
    VMAGENT_promscrape_maxScrapeSize=128MiB
    ```

2. Restart the PMM Server service after changing environment variables for VictoriaMetrics and monitor disk space when extending retention periods.

When to configure these settings:

 - Large deployments (>100 monitored nodes): increase query timeout and adjust memory limits
 - Extended retention (>30 days): Set `VM_retentionPeriod` independently from `PMM_DATA_RETENTION` for finer control
 - Memory-constrained environments: reduce `VM_memory.allowedPercent` to prevent OOM issues
 - High write volume: add merge speed optimization for better ingestion performance
