# Install PMM with Docker HA (Basic)

Docker's built-in restart capabilities combined with PMM's client-side data buffering provide basic availability improvements for single-server deployments. This approach is perfect for development, testing, and environments where brief monitoring interruptions are acceptable.

!!! note "This is not true high availability"
    Docker HA provides automatic restart after container failures, but cannot protect against host-level failures. For true high availability with zero-downtime failover, see [Kubernetes HA Clustered](HA-clustered.md).

## What is Docker HA?

Docker HA leverages Docker's automatic container recovery to restart the PMM Server container after crashes or system reboots. Combined with PMM Client's built-in data buffering, this ensures no metrics are lost during brief outages.

### Key benefits

- **Automatic recovery**: Docker restarts PMM Server automatically after container failures
- **Zero data loss**: PMM Clients cache metrics locally during outages
- **Simple setup**: No orchestration platform required
- **Minimal overhead**: Single container deployment with no additional infrastructure
- **Production-ready**: Stable and tested for years

### How it works

When you launch PMM Server with the `--restart=always` flag, Docker monitors the container and automatically restarts it if the:

- PMM Server process crashes
- container stops unexpectedly
- host system reboots

During any downtime, PMM Clients automatically:

- detect the connection failure
- begin caching metrics data locally
- continue attempting to reconnect
- transfer all cached data once the connection is restored

**Typical recovery time**: 1-3 minutes

### Limitations

- **Single point of failure**: Host-level failures require manual intervention
- **Brief monitoring gaps**: 1-3 minutes of downtime during container restarts
- **No load balancing**: Single PMM Server handles all monitoring traffic
- **Manual failover**: No automatic failover to backup servers

This solution works well for environments where brief interruptions are acceptable and post-incident analysis is more important than real-time availability.

## Prerequisites

Storage requirements increase with the number of monitored services and data retention period. See [PMM Server system requirements](../install-pmm/install-pmm-server/prerequisites.md#system-requirements) for detailed sizing.

=== "Required software"

    - **Docker**: 20.10 or higher
    - **Docker Compose** (optional): 2.0 or higher for easier management

=== "System requirements"

    **Minimum resources:**

    - **CPU**: 2 cores
    - **Memory**: 4 GB RAM
    - **Storage**: 20+ GB available disk space

## Installation

Choose the installation method that fits your needs and launch PMM Server with automatic restart enabled.

=== "Quick start"

    Get PMM Server running in under a minute with minimal configuration:
    {.power-number}

    1. Launch PMM Server:
      ```sh
        docker run -d \
          --name pmm-server \
          --restart=always \
          -p 443:8443 \
          -v pmm-data:/srv \
          percona/pmm-server:3
      ```

    2. Access PMM UI at `https://localhost` and log in with default credentials: `admin`/`admin` (change immediately after first login).

=== "Recommended"

    Add security and performance options for best practices:
    {.power-number}

    1. Launch PMM Server with recommended configuration:
      ```sh
        docker run -d \
          --name pmm-server \
          --restart=always \
          -p 443:8443/tcp \
          -v pmm-data:/srv \
          -e PMM_ENABLE_UPDATES=0 \
          --ulimit=nofile=1000000:1000000 \
          percona/pmm-server:3
      ```

        where:

        - `--restart=always`: Ensures automatic container restart after failures or reboots
        - `-p 443:8443`: Exposes HTTPS port for secure web access
        - `-v pmm-data:/srv`: Persists PMM data across container restarts
        - `-e PMM_ENABLE_UPDATES=0`: Disables automatic PMM updates (control updates manually)
        - `--ulimit=nofile=1000000:1000000`: Increases file descriptor limit for large deployments

    2. Access PMM UI at `https://localhost` and log in with default credentials: `admin`/`admin` (change immediately after first login).

=== "Docker Compose"

    Use Docker Compose for easier management and reproducible deployments:
    {.power-number}

    1. Create a `docker-compose.yml` file:
      ```yaml
        services:
          pmm-server:
            image: percona/pmm-server:3
            container_name: pmm-server
            restart: always
            ports:
              - "443:8443"
            volumes:
              - pmm-data:/srv
            environment:
              - PMM_ENABLE_UPDATES=0
            ulimits:
              nofile:
                soft: 1000000
                hard: 1000000

        volumes:
          pmm-data:
      ```

    2. Launch PMM Server:
    ```sh
        docker-compose up -d
    ```

    3. Access PMM UI at `https://localhost` and log in with default credentials: `admin`/`admin` (change immediately after first login).

### Verify installation

Regardless of which method you chose, verify that PMM Server is running correctly:
```sh
# Check container status
docker ps | grep pmm-server

# View container logs
docker logs pmm-server

# Check restart policy
docker inspect pmm-server | grep -A 3 RestartPolicy
```

Expected output should show:
```
"RestartPolicy": {
    "Name": "always",
    "MaximumRetryCount": 0
},
```

## Configuration

### Change admin password

After first login, immediately change the default admin password:
{.power-number}

1. Log in to PMM UI at `https://localhost`
2. Go to **Account > Change password**
3. Click **Change Password**
4. Enter current password (`admin`) and new secure password

### Enable external access

By default, PMM Server listens on all interfaces. To restrict access:
```sh
# Bind only to localhost
docker run -d \
  --name pmm-server \
  --restart=always \
  -p 127.0.0.1:443:8443 \
  -v pmm-data:/srv \
  percona/pmm-server:3
```

For production deployments, use a reverse proxy (nginx, Apache) or firewall rules to control access.

### Custom SSL certificates

Replace self-signed certificates with your own:
```sh
# Copy certificates to container
docker cp certificate.crt pmm-server:/srv/nginx/certificate.crt
docker cp certificate.key pmm-server:/srv/nginx/certificate.key
docker cp ca-certs.pem pmm-server:/srv/nginx/ca-certs.pem

# Restart nginx
docker exec pmm-server supervisorctl restart nginx
```

## Operations

### Connect monitoring clients

To monitor your databases, install PMM Client on each database host and connect it to PMM Server:
{.power-number}

1. [Install PMM Client](../install-pmm/install-pmm-client/index.md) on your database hosts.

2. Connect to PMM Server and add your database:
```sh
pmm-admin config --server-url=https://admin:password@pmm-server:443
```
3. [Add a database service](../install-pmm/install-pmm-client/add-services.md) for monitoring.

### Test automatic restart

Verify that Docker restarts PMM Server automatically:
```sh
# Stop the container
docker stop pmm-server

# Wait a few seconds
sleep 5

# Check if Docker restarted it
docker ps | grep pmm-server
```

The container should restart automatically within seconds.

### Simulate container crash

Test recovery from container failures:
```sh
# Kill the main process inside the container
docker exec pmm-server pkill -9 supervisord

# Watch Docker restart the container
docker logs -f pmm-server
```

Recovery should complete in 1-2 minutes.

### Monitor PMM Server health

Check PMM Server status and resource usage:
```sh
# View container stats
docker stats pmm-server

# Check disk usage
docker exec pmm-server df -h /srv

# View all services status
docker exec pmm-server supervisorctl status
```

### Backup and restore

#### Create backup
```sh
# Stop PMM Server to ensure consistent backup
docker stop pmm-server

# Backup data volume
docker run --rm \
  -v pmm-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/pmm-backup-$(date +%Y%m%d).tar.gz /data

# Restart PMM Server
docker start pmm-server
```

#### Restore from backup
```sh
# Stop and remove existing container
docker stop pmm-server
docker rm pmm-server

# Remove old data
docker volume rm pmm-data

# Restore backup
docker run --rm \
  -v pmm-data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/pmm-backup-YYYYMMDD.tar.gz -C /

# Start new container
docker run -d \
  --name pmm-server \
  --restart=always \
  -p 443:8443 \
  -v pmm-data:/srv \
  percona/pmm-server:3
```

### Upgrade PMM Server

Perform manual upgrades with zero data loss:
```sh
# Pull the latest image
docker pull percona/pmm-server:3

# Stop and remove old container
docker stop pmm-server
docker rm pmm-server

# Start new container with same data volume
docker run -d \
  --name pmm-server \
  --restart=always \
  -p 443:8443 \
  -v pmm-data:/srv \
  percona/pmm-server:3
```

The data volume (`pmm-data`) persists all monitoring data, dashboards, and configurations across upgrades.

## Troubleshooting

### Container won't start

**Problem**: Container starts but immediately exits

**Solution**: Check container logs for errors:
```sh
docker logs pmm-server

# Common issues:
# - Port already in use: Change port mapping (-p 443:8443)
# - Insufficient memory: Increase Docker memory limits
# - Corrupted data: Remove volume and start fresh
```

### High memory usage

**Problem**: PMM Server consuming excessive memory

**Solution**: Reduce retention period or check resource usage:

```sh
# Check container resource usage
docker stats pmm-server --no-stream

# Check which services are monitored
docker exec pmm-server pmm-admin list
```

### Clients can't connect

**Problem**: PMM Clients fail to connect to server

**Solution**: Verify network connectivity and firewall rules:
```sh
# Test from client host
curl -k https://127.0.0.1:443/ping

# Check firewall rules
sudo iptables -L | grep 443

# Verify Docker port mapping
docker port pmm-server
```

### Data not persisting

**Problem**: Data lost after container restart

**Solution**: Verify volume is properly mounted:
```sh
# Check volume exists
docker volume ls | grep pmm-data

# Inspect volume mount
docker inspect pmm-server | grep -A 5 Mounts

# Ensure volume is specified in docker run command
```

## Limitations and when to upgrade

### When Docker HA is sufficient

- **Development and testing environments** where availability isn't critical
- **Small deployments** monitoring fewer than 50 database instances
- **Environments with maintenance windows** where 1-3 minutes of downtime is acceptable
- **Teams without Kubernetes expertise** who want simple deployment

### When to consider Kubernetes HA

Consider upgrading to [Kubernetes HA Single-Instance](HA-kubernetes-single-instance.md) when you:

- need automatic recovery from host-level failures
- are already using Kubernetes for other infrastructure
- want automated pod rescheduling across nodes
- need better resource management and scheduling

### When to consider Kubernetes HA Clustered

Consider [Kubernetes HA Clustered](HA-clustered.md) when you:

- require zero-downtime monitoring (< 30 second failover)
- can tolerate Tech Preview status and known issues
- have expert Kubernetes skills
- test for future production HA requirements

## Get help

- [Percona Community Forum](https://forums.percona.com/c/percona-monitoring-and-management-pmm/)
- [Percona Support](https://www.percona.com/services/support) 
- [Docker deployment guide](../install-pmm/install-pmm-server/deployment-options/docker/index.md)
