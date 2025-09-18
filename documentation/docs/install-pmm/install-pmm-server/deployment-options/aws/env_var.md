# Configure environment variables for PMM Server (AMI/AWS)

Configure PMM Server behavior on AWS AMI instances by setting environment variables in the systemd environment file. This allows you to customize performance, storage, features, and other settings without modifying the container directly.

## Using environment variables

PMM Server on AWS AMI runs as a systemd user service that launches a Podman container. Environment variables are configured through a dedicated environment file on the EC2 instance.

### Configure environment variables

To set environment variables for PMM Server on AWS AMI deployments:

{.power-number}

1. Connect to your PMM Server EC2 instance via SSH using the `admin` user and your key pair:

    ```bash
    ssh -i your-key.pem admin@<ec2-instance-public-ip>
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
    PMM_PUBLIC_ADDRESS=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)
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

For detailed information about available environment variables, see the [Docker environment variables documentation](../docker/env_var.md). All variables documented for Docker deployments are supported in AMI deployments.

### Key variables for AWS AMI deployments

These variables are particularly useful for AWS AMI deployments:

| Variable | Default | Description |
|----------|---------|-------------|
| `PMM_DATA_RETENTION` | `30d` | Duration to retain metrics data |
| `PMM_ENABLE_UPDATES` | `true` | Allow version checks and updates |
| `PMM_ENABLE_TELEMETRY` | `true` | Enable usage data collection |
| `PMM_PUBLIC_ADDRESS` | Auto-detected | External DNS/IP for PMM Server |
| `PMM_DEBUG` | `false` | Enable verbose logging |
| `PMM_METRICS_RESOLUTION` | `1s` | Base metrics collection interval |

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

### AWS-specific troubleshooting

#### Check instance metadata access

Verify that the instance can access metadata:

```bash
curl -s http://169.254.169.254/latest/meta-data/instance-id
```

#### Verify security group settings

Ensure your security group allows the necessary ports:

```bash
# Check if PMM ports are accessible
netstat -tlnp | grep :443
netstat -tlnp | grep :80
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
PMM_DISTRIBUTION_METHOD=ami
```

Then restart the service:

```bash
systemctl --user restart pmm-server
```

## Advanced configuration

For a complete list of available environment variables and their descriptions, see [Docker environment variables documentation](../docker/env_var.md).