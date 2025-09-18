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

| Variable | Default | Description | AWS-specific notes |
|----------|---------|-------------|-------------------|
| `PMM_DATA_RETENTION` | `30d` | Duration to retain metrics data | Consider EBS volume size |
| `PMM_ENABLE_UPDATES` | `true` | Allow version checks and updates | Requires internet access |
| `PMM_ENABLE_TELEMETRY` | `true` | Enable usage data collection | Requires outbound access |
| `PMM_PUBLIC_ADDRESS` | Auto-detected | External DNS/IP for PMM Server | Set to Elastic IP if used |
| `PMM_DEBUG` | `false` | Enable verbose logging | Can increase CloudWatch costs |
| `PMM_METRICS_RESOLUTION` | `1s` | Base metrics collection interval | Affects instance performance |

## AWS-specific configuration examples

### Production deployment with Elastic IP

For production environments using Elastic IP addresses:

```bash
# Set static public address
PMM_PUBLIC_ADDRESS=203.0.113.10

# Performance optimizations for production
PMM_DATA_RETENTION=90d
PMM_METRICS_RESOLUTION=5s
PMM_METRICS_RESOLUTION_HR=10s
PMM_METRICS_RESOLUTION_MR=30s
PMM_METRICS_RESOLUTION_LR=300s

# Security settings
PMM_ENABLE_TELEMETRY=false
```

### Multi-AZ deployment with internal addressing

For deployments using internal IP addresses and load balancers:

```bash
# Use internal IP for cluster communication
PMM_PUBLIC_ADDRESS=10.0.1.100

# Configure for high availability
PMM_DATA_RETENTION=30d
PMM_ENABLE_BACKUP_MANAGEMENT=true

# Optimize for network performance
PMM_METRICS_RESOLUTION=10s
```

### Development and testing

For development environments with cost optimization:

```bash
# Short retention to minimize storage costs
PMM_DATA_RETENTION=7d

# Enable debugging for troubleshooting
PMM_DEBUG=true
PMM_TRACE=true

# Disable external features to reduce costs
PMM_ENABLE_TELEMETRY=false
PMM_ENABLE_UPDATES=false
```

### Compliance-focused deployment

For environments with strict compliance requirements:

```bash
# Disable all external communication
PMM_ENABLE_UPDATES=false
PMM_ENABLE_TELEMETRY=false

# Enable audit logging
PMM_DEBUG=true

# Set specific network configuration
PMM_PUBLIC_ADDRESS=172.31.1.100
```

## AWS integration

### Instance metadata

You can use AWS instance metadata in environment variables:

```bash
# Dynamic public IP detection
PMM_PUBLIC_ADDRESS=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)

# Use instance ID in configuration
INSTANCE_ID=$(curl -s http://169.254.169.254/latest/meta-data/instance-id)
```

### CloudWatch integration

Configure environment variables for enhanced CloudWatch monitoring:

```bash
# Enable detailed logging (increases CloudWatch costs)
PMM_DEBUG=true
PMM_TRACE=false

# Optimize for CloudWatch log shipping
PMM_METRICS_RESOLUTION=30s
```

### EBS volume optimization

Configure PMM for your EBS volume setup:

```bash
# For larger EBS volumes, increase retention
PMM_DATA_RETENTION=180d

# Optimize for GP3 volumes
PMM_METRICS_RESOLUTION=5s

# For io2 volumes, use higher resolution
PMM_METRICS_RESOLUTION=1s
PMM_METRICS_RESOLUTION_HR=1s
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

## Advanced AWS configuration

### External RDS integration

Configure PMM to work with external RDS instances:

```bash
# PostgreSQL RDS connection
PMM_POSTGRES_HOST=pmm-db.cluster-xyz.us-east-1.rds.amazonaws.com
PMM_POSTGRES_PORT=5432
PMM_POSTGRES_USER=pmm
PMM_POSTGRES_PASSWORD=secure-password
PMM_POSTGRES_DATABASE=pmm
```

### Cross-region monitoring

For cross-region deployments:

```bash
# Set region-specific settings
AWS_REGION=us-west-2
PMM_PUBLIC_ADDRESS=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)

# Optimize for cross-region latency
PMM_METRICS_RESOLUTION=10s
PMM_DATA_RETENTION=30d
```

### Auto Scaling Group considerations

For instances in Auto Scaling Groups:

```bash
# Use dynamic addressing
PMM_PUBLIC_ADDRESS=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)

# Configure for ephemeral instances
PMM_DATA_RETENTION=7d
PMM_ENABLE_BACKUP_MANAGEMENT=true
```

For a complete list of available environment variables and their descriptions, see [Docker environment variables documentation](../docker/env_var.md).

## Cost optimization

### Monitor resource usage

Environment variable settings directly impact AWS costs:

| Variable | Cost Impact | Recommendation |
|----------|-------------|----------------|
| `PMM_DATA_RETENTION` | EBS storage costs | Set based on compliance needs |
| `PMM_DEBUG` | CloudWatch Logs costs | Disable in production |
| `PMM_METRICS_RESOLUTION` | CPU/Memory usage | Balance detail vs. performance |

### Example cost-optimized configuration

```bash
# Minimize storage costs
PMM_DATA_RETENTION=14d

# Reduce CPU usage
PMM_METRICS_RESOLUTION=30s
PMM_METRICS_RESOLUTION_HR=60s

# Minimize CloudWatch costs
PMM_DEBUG=false
PMM_TRACE=false

# Disable unnecessary features
PMM_ENABLE_TELEMETRY=false
```