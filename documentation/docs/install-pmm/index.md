# About PMM installation

Installing Percona Monitoring and Management (PMM) involves setting up a central PMM Server and distributed PMM Clients that work together to monitor your database environment. 

PMM Server provides the web interface with dashboards and analytics, while PMM Clients collect data from your databases with minimal performance impact and send it back to PMM Server for analysis and visualization.

## Installation overview
The PMM installation consists of three main steps that need to be completed in sequence: 
{.power-number}

1. **Deploy PMM Server**: centralized platform that collects, analyzes, and visualizes your monitoring data
2. **Install PMM Clients**: lightweight agents on each database host that collect metrics without impacting performance
3. **Configure monitoring services**: connect PMM to your database instances, select which metrics to collect, and customize monitoring parameters

## Architecture support
PMM provides flexible deployment options across different architectures:

- **PMM Server compatibility**: currently available as native x86_64 build only (not available as a native ARM64 build)
- **PMM Client compatibility**: available for both x86_64 and ARM64 architectures
- **ARM compatibility**: for ARM-based systems, use Docker or Podman to run PMM Server x86_64 images via emulation. When installing on ARM-based systems, performance may vary between architectures
- **Permissions**: both binary installation and Docker containers can be run without `root` permissions

## Prerequisites

Before starting the installation, make sure to check PMM's:

- [hardware and software requirements](../install-pmm/plan-pmm-installation/hardware_and_system.md)
- [network and firewall requirements](../install-pmm/plan-pmm-installation/network_and_firewall.md)

## Installation steps 

### 1. Install PMM Server

Install and run at least one PMM Server. Choose the deployment option that best fits your environment: 

#### Server deployment options

| **Method** | **Best for** | **Advantages** | **Considerations** |
|-----------|------------|---------------|--------------------|
| [**:material-docker: Docker**](../install-pmm/install-pmm-server/deployment-options/docker/index.md) | Development, testing & production | ✔  Quick setup<br>✔  Simple upgrades<br>✔  Works in various environments | ⚠ Requires Docker knowledge<br>⚠ May need additional configuration for production |
| [**:material-shield-lock: Podman**](../install-pmm/install-pmm-server/deployment-options/podman/index.md) | Security-focused setups | ✔ Rootless containers<br> ✔  Enhanced security<br> ✔  OCI-compatible | ⚠ Requires Podman installation & knowledge |
| [**:material-kubernetes: Helm**](../install-pmm/install-pmm-server/deployment-options/helm/index.md) | Cloud-native environments | ✔  Scalable & high availability<br> ✔  Kubernetes-native | ⚠ Requires existing Kubernetes cluster<br>⚠ More complex setup |
| [**:material-server: Virtual Appliance**](../install-pmm/install-pmm-server/deployment-options/virtual/index.md) | Traditional environments | ✔  Pre-configured with all dependencies<br>✔  Dedicated resources | ⚠ Larger resource footprint<br>⚠ Requires a hypervisor |


<!--| [Amazon AWS](../install-pmm/install-pmm-server/deployment-options/aws/aws.md) | AWS-based environments | Seamless AWS integration, easy provisioning | Monthly subscription costs, AWS infrastructure costs |-->

### 2. Install PMM Client

Install and run PMM Client on every node where there is a service you want to monitor. PMM Client supports both x86_64 and ARM64 architectures.

#### Client installation options

=== ":material-package-variant: With package manager"

    [Linux package](../install-pmm/install-pmm-client/package_manager.md): Use `apt`, `apt-get`, `dnf`, `yum`. 
    |The package manager automatically selects the correct version for your architecture.

=== ":material-archive: With binary package"

    [Binary package](../install-pmm/install-pmm-client/binary_package.md): Download the appropriate `.tar.gz` file for your architecture (x86_64 or ARM64).

=== ":material-docker: With Docker"

    [Running PMM Client as a Docker container](../install-pmm/install-pmm-client/docker.md) simplifies deployment across different architectures and automatically selects the appropriate image for your architecture (x86_64 or ARM64).

### 3. Add services for monitoring

After installing PMM Client, configure the nodes and services you want to monitor. 

PMM supports monitoring across the following database technologies, cloud services, proxy services, and system metrics:

=== ":material-database: Databases"
    Choose the database technology you want to monitor:
    
    | <small>*Database*</small> | <small>*Description*</small> | <small>*Setup guide*</small> |
    | ------------------------- | ---------------------------- | ---------------------------- |
    | :material-dolphin: **MySQL** | MySQL, Percona Server for MySQL, MariaDB | [**Connect to MySQL** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/mysql.md) |
    | :material-elephant: **PostgreSQL** | PostgreSQL database servers | [**Connect to PostgreSQL** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/postgresql.md) |
    | :material-leaf: **MongoDB** | MongoDB, Percona Server for MongoDB | [**Connect to MongoDB** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/mongodb.md) |

=== ":material-cloud: Cloud services"
    Configure monitoring for cloud-hosted database services:
    
    | <small>*Cloud provider*</small> | <small>*Supported services*</small> | <small>*Setup guide*</small> |
    | ------------------------------- | ----------------------------------- | ---------------------------- |
    | :material-aws: **Amazon Web Services** | Amazon RDS for MySQL/PostgreSQL, Aurora | [**Configure AWS monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/aws.md) |
    | :material-microsoft-azure: **Microsoft Azure** | Azure Database for MySQL/PostgreSQL, Cosmos DB | [**Configure Azure monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/azure.md) |
    | :material-google-cloud: **Google Cloud** | Cloud SQL for MySQL/PostgreSQL | [**Configure Google Cloud monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/google.md) |

=== ":material-server: System monitoring"
    Monitor operating system and remote services:
    
    | <small>*Monitoring type*</small> | <small>*Description*</small> | <small>*Setup guide*</small> |
    | -------------------------------- | ---------------------------- | ---------------------------- |
    | :simple-linux: **Linux systems** | CPU, memory, disk, network metrics | [**Configure Linux monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/linux.md) |
    | :material-network: **Remote MySQL** | Monitor MySQL across network segments | [**Configure remote MySQL** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/remote.md#mysql-remote) |
    | :material-network: **Remote PostgreSQL** | Monitor PostgreSQL across network segments | [**Configure remote PostgreSQL** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/remote.md#postgresql-remote) |
    | :material-network: **Remote MongoDB** | Monitor MongoDB across network segments | [**Configure remote MongoDB** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/remote.md#mongodb-remote) |

=== ":material-transit-connection-variant: Proxy services"
    Monitor database proxy and load balancing solutions:
    
    | <small>*Proxy service*</small> | <small>*Description*</small> | <small>*Setup guide*</small> |
    | ------------------------------ | ---------------------------- | ---------------------------- |
    | :material-database-cog: **ProxySQL** | MySQL database proxy | [**Configure ProxySQL monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/proxysql.md) |
    | :material-scale-balance: **HAProxy** | High-availability load balancer | [**Configure HAProxy monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/haproxy.md) |

=== ":material-gauge: External services"
    Extend PMM with external exporters and third-party metrics:
    
    | <small>*Service type*</small> | <small>*Description*</small> | <small>*Setup guide*</small> |
    | ----------------------------- | ---------------------------- | ---------------------------- |
    | :material-chart-line: **Custom exporters** | Add custom Prometheus exporters | [**Configure external exporters** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/external.md#adding-exporters) |
    | :material-api: **Third-party sources** | Connect external metrics sources | [**Extend monitoring capabilities** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/external.md#extended-monitoring) |
    | :material-application-cog: **General external services** | Generic external service monitoring | [**Configure external services** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/external.md) |