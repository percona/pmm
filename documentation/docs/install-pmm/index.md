# PMM installation overview

Installing Percona Monitoring and Management (PMM) involves setting up a central PMM Server and distributed PMM Clients that work together to monitor your database environment. 

PMM Server provides the web interface with dashboards and analytics, while PMM Clients collect data from your databases with minimal performance impact and send it back to PMM Server for analysis and visualization.

## What the installation involves

The PMM installation consists of three main steps that need to be completed in sequence: 
{.power-number}

1. [Install PMM Server](#1-install-pmm-server): centralized platform that collects, analyzes, and visualizes your monitoring data
2. [Install PMM Clients](#2-install-pmm-client): lightweight agents on each database host that collect metrics without impacting performance
3. [Configure monitoring services](#3-add-services-for-monitoring): connect PMM to your database instances, select which metrics to collect, and customize monitoring parameters

## Plan the installation

Before ou install PMM, ensure your environment is properly prepared:

- [Choose a deployment strategy](../install-pmm/plan-pmm-installation/choose-deployment.md) based on your environment needs.
- [Verify hardware requirements](../install-pmm/plan-pmm-installation/hardware_and_system.md) to ensure your system meets the necessary specifications.
- [Configure your network](../install-pmm/plan-pmm-installation/network_and_firewall.md) for the required connections.

### PMM Server deployment options

Compare the available deployment methods to choose what works best for your setup. For a fast evaluation setup, Docker is the quickest option. For production environments, consider your existing infrastructure stack and operational preferences when choosing between Docker, Kubernetes (Helm), or Virtual Appliance deployments:

| Deployment Method | Best for | Advantages | Considerations |
|-------------------|----------|------------|----------------|
| [Docker](../install-pmm/install-pmm-server/deployment-options/docker/index.md) | Quick setup, development environments | • Fast deployment<br>• Easy to manage<br>• Runs without root privileges<br>• Minimal resource overhead | • Requires Docker knowledge<br>• May need additional network configuration |
| [Podman](../install-pmm/install-pmm-server/deployment-options/podman/index.md) | Security-conscious environments | • Rootless by default<br>• Enhanced security<br>• Docker-compatible commands<br>• No daemon required | • Requires Podman installation<br>• Less common than Docker |
| [Helm](../install-pmm/install-pmm-server/deployment-options/helm/index.md) | Kubernetes environments | • Native Kubernetes deployment<br>• Scalable and orchestrated<br>• ConfigMap and Secret management<br>• Ingress controller support | • Requires Kubernetes cluster<br>• Helm knowledge needed<br>• More complex setup |
| [Virtual Appliance](../install-pmm/install-pmm-server/deployment-options/virtual/index.md) | Traditional VM environments | • Pre-configured virtual machine<br>• Works with VMware, VirtualBox<br>• No container knowledge required<br>• Isolated environment | • Larger resource footprint<br>• VM management overhead<br>• Less flexible than containers |
| [Amazon AWS](../install-pmm/install-pmm-server/deployment-options/aws/aws.md) | AWS cloud deployments | • Wizard-driven install<br>• Rootless deployment<br>• Integrated with AWS services | • Paid service, incurs infrastructure costs<br>• AWS-specific deployment |

## Installation steps 

### 1. Install PMM Server

Install and run at least one PMM Server using one of the following deployment methods. If you're not sure which deployment method is best for your environment, check out this [Choose a PMM deployment strategy](../install-pmm/plan-pmm-installation/choose-deployment.md) topic for a comparison of your options.

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

=== ":material-database: Database services"
    Monitor relational and NoSQL database instances:

    - [MySQL](../install-pmm/install-pmm-client/connect-database/mysql/mysql.md) and variants (Percona Server for MySQL, Percona XtraDB Cluster, MariaDB)
    - [MongoDB](../install-pmm/install-pmm-client/connect-database/mongodb.md)
    - [PostgreSQL](../install-pmm/install-pmm-client/connect-database/postgresql.md)

=== ":material-cloud: Cloud services"

    Monitor cloud-hosted database services and platforms:

    - [Microsoft Azure](../install-pmm/install-pmm-client/connect-database/azure.md)
    - [Google Cloud Platform](../install-pmm/install-pmm-client/connect-database/google.md)
    - [Amazon RDS](../install-pmm/install-pmm-client/connect-database/aws.md) 

=== ":material-server-network: System & infrastructure"

    Monitor system resources and infrastructure components:

    - [Linux systems](../install-pmm/install-pmm-client/connect-database/linux.md)
    - [Remote instances](../install-pmm/install-pmm-client/connect-database/remote.md)
    - [External services](../install-pmm/install-pmm-client/connect-database/external.md)

=== ":material-router-network: Proxy services"

    Monitor database proxy and load balancing services:

    - [ProxySQL](../install-pmm/install-pmm-client/connect-database/proxysql.md)
    - [HAProxy](../install-pmm/install-pmm-client/connect-database/haproxy.md)
