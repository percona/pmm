# Prerequisites for PMM Server

Before installing PMM Server, ensure your environment meets these requirements.

## Quick requirements checklist

✓ **Hardware**: CPU with SSE4.2 support, 4+ cores, 8+ GB RAM, 100+ GB storage  
✓ **OS**: Modern 64-bit Linux or container platform (Docker/Podman/Kubernetes)  
✓ **Network**: Ports 80/443 accessible to PMM Clients and users  
✓ **Container runtime**: Docker 17.03+ or Podman (for containerized deployments)  
✓ **Storage**: Persistent storage solution for data retention  

## System requirements

PMM Server requirements scale with the number of monitored nodes:

### Hardware specifications by deployment size

=== "Small (1-30 nodes)"

    - **CPU**: 4 cores with SSE4.2 support
    - **Memory**: 8 GB RAM
    - **Storage**: 100 GB
    - **Use cases**: Development, small businesses, initial deployments

=== "Medium (31-200 nodes)"

    - **CPU**: 8-16 cores with SSE4.2 support
    - **Memory**: 16-32 GB RAM
    - **Storage**: 200+ GB
    - **Use cases**: Production environments, mid-sized companies

=== "Large (200+ nodes)"
w1
    - **CPU**: 16+ cores with SSE4.2 support
    - **Memory**: 32+ GB RAM
    - **Storage**: 500+ GB
    - **Use cases**: Large enterprises, mission-critical database fleets

For detailed sizing calculations, see [Hardware and system requirements](../plan-pmm-installation/hardware_and_system.md).

### Architecture requirements

- **CPU**: Must support SSE4.2 instruction set (required for Query Analytics)
- **x86_64**: Native support for optimal performance
- **ARM64**: Supported via Docker emulation using `--platform linux/amd64`

## Storage planning

PMM Server stores metrics data, requiring persistent storage:

- **Storage type**: Use persistent volumes (Docker volumes, cloud storage or host directories)
- **Capacity planning**: `nodes × retention_period_in_weeks × 1 GB`
- **Quick estimate**: `number_of_nodes × 4 GB` for default 30-day retention
- **Performance**: SSD recommended for better I/O performance

## Network connectivity

PMM Server requires these network connections:

| Connection | Port | Purpose | Required |
|------------|------|---------|----------|
| Users > PMM Server | 443 (HTTPS) | Web interface access | Essential |
| Users > PMM Server | 80 (HTTP) | Web interface (insecure) | Optional |
| PMM Clients > PMM Server | 443/80 | Metrics reporting | Essential |
| PMM Server > Internet | 443 | Updates, telemetry | Optional |

For complete port specifications, see [Network and firewall requirements](../plan-pmm-installation/network_and_firewall.md).

## Deployment-specific prerequisites

Choose your deployment method and ensure it meets these specific requirements:

=== ":simple-docker: Docker"
    **Version requirements:**
    - Docker version 17.03 or higher
    - CPU with `x86-64-v2` support
    
    **System resources:**
    - Recommended: 2+ CPU cores, 4+ GB RAM, 100+ GB disk space
    
    **Optional but recommended:**
    - Watchtower for UI-based updates
    - Docker Compose for multi-container setups
    
    **Security considerations for Watchtower:**
    - Limit Watchtower's access to Docker network or localhost
    - Configure network to ensure only PMM Server is exposed externally
    - Secure Docker socket access for Watchtower
    - Place both Watchtower and PMM Server on the same Docker network

=== ":simple-podman: Podman"
    **Version requirements:**
    - Recent Podman version with systemd support
    - Rootless Podman configuration
    
    **System configuration:**
    - Allow non-root users to bind to privileged ports (port 443)
    - Podman socket enabled for Watchtower integration
    - systemd user services enabled
    
    **Required setup commands:**
    ```sh
    # Configure privileged port access
    echo "net.ipv4.ip_unprivileged_port_start=443" | sudo tee /etc/sysctl.d/99-pmm.conf
    sudo sysctl -p /etc/sysctl.d/99-pmm.conf
    
    # Enable Podman socket
    systemctl --user enable --now podman.socket
    ```
    
    **Security advantages:**
    - Enhanced security isolation with rootless containers
    - Better systemd integration for service management
    - Fine-grained permission control

=== ":simple-kubernetes: Kubernetes"
    **Cluster requirements:**
    - Kubernetes 1.19+ with supported version
    - kubectl configured to communicate with your cluster
    
    **Helm requirements:**
    - Helm v3 installed and configured
    - Access to Percona Helm charts repository
    
    **Storage requirements:**
    - Storage driver with snapshot support (for backups)
    - Dynamic provisioning capability
    - Persistent Volume support
    
    **Production considerations:**
    - Separate PMM Server from monitored systems
    - High availability configuration for continuous monitoring
    - Workload separation through node configurations and affinity rules

=== ":material-harddisk: Virtual Appliance (OVA)"
    **Hypervisor compatibility:**
    - VMware ESXi 6.0+, Workstation 12.0+, Fusion 10.0+
    - VirtualBox 6.0+
    
    **VM specifications (default):**
    - OS: Oracle Linux 9.3
    - CPU: 1 (adjustable after deployment)
    - Memory: 4096 MB (adjustable after deployment)
    - Disk 1: 40 GB (system)
    - Disk 2: 400 GB (data)
    
    **Network access:**
    - Outbound internet access for updates (optional)
    - Access to monitored database instances
    - Access from client browsers to PMM web interface
    
    **Security note:**
    - Default users: `admin/admin` and `root/percona`
    - **Must change default passwords immediately after installation**

## Security considerations

### SSL/TLS certificates
- **Self-signed**: PMM Server generates these automatically
- **Custom certificates**: Prepare SSL certificates for production deployments
- **Certificate authority**: Consider using trusted CA certificates for public access

### Access control
- **Admin credentials**: Plan initial admin username/password strategy
- **Default passwords**: Change immediately (especially for OVA deployment)
- **User management**: Consider integration with existing authentication systems
- **Network security**: Plan firewall rules and network segmentation

### Data protection
- **Backup strategy**: Plan for PMM Server data backup and recovery
- **Encryption**: Consider encryption for data at rest and in transit
- **Compliance**: Ensure deployment meets organizational security requirements

## Before you install

Complete these preparation steps:
{.power-number}

1. Choose deployment method based on your [environment and requirements](../plan-pmm-installation/choose-deployment.md)
2. Prepare infrastructure (servers, storage, networking)
3. Plan capacity using sizing guidelines above
4. Configure security (certificates, firewall rules, access controls)
5. Prepare persistent storage for data retention
6. Install required tools (Docker/Podman/Helm/kubectl as needed)
7. Document configuration for maintenance and disaster recovery

## Next steps

After confirming your environment meets these prerequisites:
{.power-number}

1. [Choose your deployment method](../plan-pmm-installation/choose-deployment.md) based on your infrastructure
2. Install PMM Server  using your selected method:
   - [Docker installation](../install-pmm-server/deployment-options/docker/index.md)
   - [Podman installation](../install-pmm-server/deployment-options/podman/index.md)
   - [Kubernetes/Helm installation](../install-pmm-server/deployment-options/helm/index.md)
   - [Virtual Appliance deployment](../install-pmm-server/deployment-options/virtual/index.md)
3. [Configure security settings](../../admin/security/index.md) for production use
4. [Install PMM Clients](../install-pmm-client/index.md) on systems you want to monitor
