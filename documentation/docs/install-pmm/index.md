# PMM installation overview

Installing Percona Monitoring and Management (PMM) involves setting up a central PMM Server and distributed PMM Clients that work together to monitor your database environment. 

PMM Server provides the web interface with dashboards and analytics, while PMM Clients collect data from your databases with minimal performance impact and send it back to PMM Server for analysis and visualization.

## What the installation involves

The PMM installation consists of three main steps that need to be completed in sequence: 
{.power-number}

1. [Install PMM Server](#1-install-pmm-server): centralized platform that collects, analyzes, and visualizes your monitoring data
2. [Install PMM Clients](#2-install-pmm-client): lightweight agents on each database host that collect metrics without impacting performance
3. [Configure monitoring services](#3-add-services-for-monitoring): connect PMM to your database instances, select which metrics to collect, and customize monitoring parameters

## Planning the installation

| Use | :material-thumb-up: **Benefits** | :material-thumb-down: **Drawbacks**|
|---|---|---
| [Docker](../install-pmm/install-pmm-server/deployment-options/docker/index.md) | 1. Quick<br>2. Simple<br> 3. Rootless |  Additional network configuration required.
| [Podman](../install-pmm/install-pmm-server/deployment-options/podman/index.md) | 1. Quick<br>2. Simple<br>3. Rootless | Podman installation required.
| [Helm](../install-pmm/install-pmm-server/deployment-options/helm/index.md) (Technical Preview) | 1. Quick<br>2. Simple<br>3. Cloud-compatible <br> 4. Rootless| Requires running a Kubernetes cluster.
| [Virtual appliance](../install-pmm/install-pmm-server/deployment-options/virtual/index.md)  | 1. Easily import into Hypervisor of your choice <br> 2. Rootless| More system resources compared to Docker footprint.
<!---| [Amazon AWS](../install-pmm/install-pmm-server/deployment-options/aws/aws.md) | 1. Wizard-driven install. <br>  2. Rootless| Paid, incurs infrastructure costs. --->

- [Choose a deployment strategy](../install-pmm/plan-pmm-installation/choose-deployment.md) based on your environment needs.
- [Verify hardware requirements](../install-pmm/plan-pmm-installation/hardware_and_system.md) to ensure your system meets the necessary specifications.
- [Configure your network](../install-pmm/plan-pmm-installation/network_and_firewall.md) for the required connections.

## Installation steps 

### 1. Install PMM Server

Install and run at least one PMM Server using one of the following deployment methods: 

=== ":material-docker: Docker"
    Run PMM Server as a Docker container
    
    [**Get started with Docker deployment** :material-arrow-right:](../install-pmm/install-pmm-server/deployment-options/docker/index.md)

=== ":material-shield-lock: Podman"
    Run PMM Server as a rootless Podman container
    
    [**Get started with Podman deployment** :material-arrow-right:](../install-pmm/install-pmm-server/deployment-options/podman/index.md)

=== ":material-kubernetes: Helm"
    Deploy PMM Server on a Kubernetes cluster
    
    [**Get started with Kubernetes deployment** :material-arrow-right:](../install-pmm/install-pmm-server/deployment-options/helm/index.md)

=== ":material-server: Virtual Appliance"
    Run PMM Server as a pre-configured virtual machine
    
    [**Get started with Virtual Appliance** :material-arrow-right:](../install-pmm/install-pmm-server/deployment-options/virtual/index.md)

=== ":material-aws: AWS Marketplace"
    Deploy PMM Server from AWS Marketplace
    
    [**Get started with AWS deployment** :material-arrow-right:](../install-pmm/install-pmm-server/deployment-options/aws/aws.md)

If you're sure which deployment method is best for your environment, check out this [Choose a PMM deployment strategy](../install-pmm/plan-pmm-installation/choose-deployment.md) topic for a comparison of your options.

### 2. Install PMM Client

Install and run PMM Client on every node where there is a service you want to monitor. Choose the installation method that best fits your environment:

#### Client installation options

=== ":material-package-variant: With package manager"

    [Linux package](../install-pmm/install-pmm-client/package_manager.md): Use `apt`, `apt-get`, `dnf`, `yum`. 
    The package manager automatically selects the correct version for your architecture.

=== ":material-archive: With binary package"

    [Binary package](../install-pmm/install-pmm-client/binary_package.md): Download the appropriate `.tar.gz` file for your architecture (x86_64 or ARM64).

=== ":material-docker: With Docker"

    [Running PMM Client as a Docker container](../install-pmm/install-pmm-client/docker.md) simplifies deployment across different architectures and automatically selects the appropriate image for your architecture (x86_64 or ARM64).

### 3. Add services for monitoring

After installing PMM Client, configure the nodes and services you want to monitor. 

PMM supports monitoring across the following database technologies, cloud services, proxy services, and system metrics:

    - [MySQL](../install-pmm/install-pmm-client/connect-database/mysql.md) and variants: Percona Server for MySQL, Percona XtraDB Cluster, MariaDB
    - [MongoDB](../install-pmm/install-pmm-client/connect-database/mongodb.md)
    - [PostgreSQL](../install-pmm/install-pmm-client/connect-database/postgresql.md)
    - [ProxySQL](../install-pmm/install-pmm-client/connect-database/proxysql.md)
 <!---   - [Amazon RDS](../install-pmm/install-pmm-client/connect-database/aws.md)--->
    - [Microsoft Azure](../install-pmm/install-pmm-client/connect-database/azure.md)
    - [Google Cloud Platform](../install-pmm/install-pmm-client/connect-database/google.md)
    - [Linux](../install-pmm/install-pmm-client/connect-database/linux.md)
    - [External services](../install-pmm/install-pmm-client/connect-database/external.md)
    - [HAProxy](../install-pmm/install-pmm-client/connect-database/haproxy.md)
    - [Remote instances](../install-pmm/install-pmm-client/connect-database/remote.md)
