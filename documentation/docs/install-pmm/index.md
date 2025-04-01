# About PMM installation

Installing PMM involves setting up a PMM Server, installing PMM Clients on your database hosts, and configuring the services you want to monitor:
{.power-number}

1. **Deploy PMM Server** - centralized platform that collects, analyzes, and visualizes your monitoring data
2. **Install PMM Clients** - lightweight agents on each database host that collect metrics without impacting performance
3. **Configure Monitoring Services** - Select exactly which database instances and metrics to track based on your specific needs


## Prerequisites

Before starting the installation, check the [hardware and software requirements](../install-pmm/plan-pmm-installation/hardware_and_system.md) and the [network and firewall requirements](../install-pmm/plan-pmm-installation/network_and_firewall.md).

## Install PMM Server

Install and run at least one PMM Server. Choose from the following options:

### ARM compatibility

PMM Server is not currently available as a native ARM64 build. For ARM-based systems, consider using the Docker or Podman installation methods, which can run x86_64 images via emulation on ARM platforms.

### Deployment options

| Method | Best for | Advantages | Considerations |
|--------|----------|------------|----------------|
| [Docker](../install-pmm/install-pmm-server/deployment-options/docker/index.md) | Development and testing environments | Quick setup, simple upgrades, minimal system requirements | Requires Docker knowledge, additional network configuration for production |
| [Podman](../install-pmm/install-pmm-server/deployment-options/podman/index.md)| Security-focused environments | Rootless containers, enhanced security, OCI-compatible | Requires Podman installation and knowledge |
| [Helm](../install-pmm/install-pmm-server/deployment-options/helm/index.md) | Cloud-native environments | Scalability, high availability, cloud-compatible | Requires existing Kubernetes cluster, more complex setup |
| [Virtual appliance](../install-pmm/install-pmm-server/deployment-options/virtual/index.md) | Traditional environments | Pre-configured with all dependencies, dedicated resources | Larger resource footprint, hypervisor requirement |
<!--| [Amazon AWS](../install-pmm/install-pmm-server/deployment-options/aws/aws.md) | AWS-based environments | Seamless AWS integration, easy provisioning | Monthly subscription costs, AWS infrastructure costs |-->

## Install PMM Client

Install and run PMM Client on every node where there is a service you want to monitor. PMM Client supports both x86_64 and ARM64 architectures.

## Client installation options


=== "With package manager"

    [Linux package](../install-pmm/install-pmm-client/package_manager.md): Use `apt`, `apt-get`, `dnf`, `yum`. The package manager automatically selects the correct version for your architecture.

=== "With binary package"

    [Binary package](../install-pmm/install-pmm-client/binary_package.md): Download the appropriate `.tar.gz` file for your architecture (x86_64 or ARM64).

=== "With Docker"

    [Running PMM Client as a Docker container](../install-pmm/install-pmm-client/docker.md) simplifies deployment across different architectures and automatically selects the appropriate image for your architecture (x86_64 or ARM64).

## Compatibility 

Both binary installation and Docker containers can be run without root permissions. 

When installing on ARM-based systems, ensure you're using ARM64-compatible versions. Performance may vary between architectures.

## Add services for monitoring

After installing PMM Client, configure the nodes and services you want to monitor. PMM supports monitoring across the following database technologies, cloud services, proxy services, and system metrics:

=== "Databases"
- [MySQL](../install-pmm/install-pmm-client/connect-database/mysql.md) (including Percona Server for MySQL, Percona XtraDB Cluster, and MariaDB)
- [MongoDB](../install-pmm/install-pmm-client/connect-database/mongodb.md)
- [PostgreSQL](../install-pmm/install-pmm-client/connect-database/postgresql.md)

=== "Cloud services"
- [Amazon RDS & Aurora](../install-pmm/install-pmm-client/connect-database/aws.md)
- [Microsoft Azure](../install-pmm/install-pmm-client/connect-database/azure.md)
- [Google Cloud SQL](../install-pmm/install-pmm-client/connect-database/google.md)

=== "Proxy services"
- [ProxySQL](../install-pmm/install-pmm-client/connect-database/proxysql.md)
- [HAProxy](../install-pmm/install-pmm-client/connect-database/haproxy.md)

=== "System monitoring"
- [Linux system metrics](../install-pmm/install-pmm-client/connect-database/linux.md)
- [External services](../install-pmm/install-pmm-client/connect-database/external.md) (via exporters)

=== "Remote services"
- [Remote instances](../install-pmm/install-pmm-client/connect-database/remote.md) (for monitoring across network segments)
