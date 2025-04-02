# About PMM installation

Installing Percona Monitoring and Management (PMM) involves setting up a central PMM Server and distributed PMM Clients that work together to monitor your database environment. 

PMM Server provides the web interface with dashboards and analytics, while PMM Clients collect data from your databases with minimal performance impact and send it back to PMM Server for analysis and visualization.

## Installation overview
The PMM installation consists of three main steps that need to be completed in sequence: 
{.power-number}

1. **Deploy PMM Server** - centralized platform that collects, analyzes, and visualizes your monitoring data
2. **Install PMM Clients** - lightweight agents on each database host that collect metrics without impacting performance
3. **Configure monitoring services** - connect PMM to your database instances, select which metrics to collect, and customize monitoring parameters

## Architecture support and compatibility
PMM provides flexible deployment options across different architectures:

- **PMM Server**: currently available as native x86_64 build only (not available as a native ARM64 build)
- **PMM Client**: available for both x86_64 and ARM64 architectures
- **ARM compatibility**: for ARM-based systems, use Docker or Podman to run PMM Server x86_64 images via emulation. When installing on ARM-based systems, performance may vary between architectures
- **Permissions**: both binary installation and Docker containers can be run without `root` permissions

## Prerequisites

Before starting the installation, review PMM's:

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
    The package manager automatically selects the correct version for your architecture.

=== ":material-archive: With binary package"

    [Binary package](../install-pmm/install-pmm-client/binary_package.md): Download the appropriate `.tar.gz` file for your architecture (x86_64 or ARM64).

=== ":material-docker: With Docker"

    [Running PMM Client as a Docker container](../install-pmm/install-pmm-client/docker.md) simplifies deployment across different architectures and automatically selects the appropriate image for your architecture (x86_64 or ARM64).

## 3. Add services for monitoring

After installing PMM Client, configure the nodes and services you want to monitor. PMM supports monitoring across the following database technologies, cloud services, proxy services, and system metrics:

=== ":material-dolphin: MySQL"
    To set up PMM Client for MySQL, check the type of host that you have for your database and follow the instructions:

    | <small>*Host*</small> | <small>*Recommended setup*</small> | <small>*Other advanced options*</small> |
    | --------------------- | ----------------------------------- | ------------------------------ |
    | **Self-hosted / AWS EC2** | [**Install PMM Client using Percona Repositories** :material-arrow-right:](./install-pmm-client/percona-repositories.md) | [Using a PMM Client Docker image](./install-pmm/install-pmm-client/docker.md)<br><br>[Download and install PMM Client files](./install-pmm/install-pmm-client/binary_package.md) |
    | **AWS RDS / AWS Aurora** | [**Configure AWS settings** :material-arrow-right:](./install-pmm-client/connect-database/aws.md) | |
    | **Azure Database for MySQL** | [**Configure Azure Settings** :material-arrow-right:](./install-pmm/install-pmm-client/connect-database/azure.md) | |
    | **Google Cloud SQL for MySQL** | [**Configure Google Cloud Settings** :material-arrow-right:](./install-pmm/install-pmm-client/connect-database/google.md) | |
    | **Other hosts / No access to the node** | [**Remote monitoring** :material-arrow-right:](./install-pmm/install-pmm-client/connect-database/remote.md) | |

=== ":material-elephant: PostgreSQL"
    For PostgreSQL databases:

    | <small>*Host*</small> | <small>*Recommended setup*</small> | <small>*Other advanced options*</small> |
    | --------------------- | ----------------------------------- | ------------------------------ |
    | **Self-hosted / AWS EC2** | [**Install PMM Client using Percona Repositories** :material-arrow-right:](../install-pmm/install-pmm-client/package_manager.md) | [Using a PMM Client Docker image](../install-pmm/install-pmm-client/docker.md)<br><br>[Download and install PMM Client files](../install-pmm/install-pmm-client/binary_package.md) |
    | **AWS RDS / Aurora for PostgreSQL** | [**Configure AWS settings** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/aws.md) | |
    | **Azure Database for PostgreSQL** | [**Configure Azure Settings** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/azure.md) | |
    | **Google Cloud SQL for PostgreSQL** | [**Configure Google Cloud Settings** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/google.md) | |
    | **Other hosts / No access to the node** | [**Remote monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/remote.md) | |

=== ":material-leaf: MongoDB"
    For MongoDB databases:

    | <small>*Host*</small> | <small>*Recommended setup*</small> | <small>*Other advanced options*</small> |
    | --------------------- | ----------------------------------- | ------------------------------ |
    | **Self-hosted / AWS EC2** | [**Install PMM Client using Percona Repositories** :material-arrow-right:](../install-pmm/install-pmm-client/package_manager.md) | [Using a PMM Client Docker image](../install-pmm/install-pmm-client/docker.md)<br><br>[Download and install PMM Client files](../install-pmm/install-pmm-client/binary_package.md) |
    | **MongoDB Atlas / DocumentDB** | [**Configure MongoDB cloud monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/mongodb.md) | |
    | **Azure Cosmos DB (MongoDB API)** | [**Configure Azure Settings** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/azure.md) | |
    | **Other hosts / No access to the node** | [**Remote monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/remote.md) | |

=== ":material-database: ProxySQL"
    For ProxySQL:

    | <small>*Host*</small> | <small>*Recommended setup*</small> | <small>*Other advanced options*</small> |
    | --------------------- | ----------------------------------- | ------------------------------ |
    | **Self-hosted / AWS EC2** | [**Install PMM Client using Percona Repositories** :material-arrow-right:](../install-pmm/install-pmm-client/package_manager.md) | [Using a PMM Client Docker image](../install-pmm/install-pmm-client/docker.md)<br><br>[Download and install PMM Client files](../install-pmm/install-pmm-client/binary_package.md) |
    | **ProxySQL with MySQL backend** | [**Configure ProxySQL monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/proxysql.md) | |
    | **Other hosts / No access to the node** | [**Remote monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/remote.md) | |

=== ":material-database: HAProxy"
    For HAProxy:

    | <small>*Host*</small> | <small>*Recommended setup*</small> | <small>*Other advanced options*</small> |
    | --------------------- | ----------------------------------- | ------------------------------ |
    | **Self-hosted / AWS EC2** | [**Install PMM Client using Percona Repositories** :material-arrow-right:](../install-pmm/install-pmm-client/package_manager.md) | [Using a PMM Client Docker image](../install-pmm/install-pmm-client/docker.md)<br><br>[Download and install PMM Client files](../install-pmm/install-pmm-client/binary_package.md) |
    | **HAProxy with database backend** | [**Configure HAProxy monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/haproxy.md) | |
    | **Other hosts / No access to the node** | [**Remote monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/remote.md) | |

=== ":simple-linux: Linux"
    For Linux system monitoring:

    | <small>*Host*</small> | <small>*Recommended setup*</small> | <small>*Other advanced options*</small> |
    | --------------------- | ----------------------------------- | ------------------------------ |
    | **Physical/VM Linux servers** | [**Install PMM Client using Percona Repositories** :material-arrow-right:](../install-pmm/install-pmm-client/package_manager.md) | [Using a PMM Client Docker image](../install-pmm/install-pmm-client/docker.md)<br><br>[Download and install PMM Client files](../install-pmm/install-pmm-client/binary_package.md) |
    | **Linux with containerized applications** | [**Configure Linux monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/linux.md) | |
    | **Other hosts / No access to the node** | [**Remote monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/remote.md) | |

=== ":material-cloud: Remote services"
    For remote monitoring scenarios:

    | <small>*Host*</small> | <small>*Recommended setup*</small> | <small>*Other advanced options*</small> |
    | --------------------- | ----------------------------------- | ------------------------------ |
    | **Network segments without direct access** | [**Configure remote monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/remote.md) | |
    | **Cross-network database instances** | [**Remote MySQL monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/remote.md#mysql-remote) | |
    | **Databases behind firewalls** | [**Remote PostgreSQL monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/remote.md#postgresql-remote) | |
    | **Cloud services without direct agent access** | [**Remote MongoDB monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/remote.md#mongodb-remote) | |

=== ":material-gauge: External services"
    For external services and custom exporters:

    | <small>*Service type*</small> | <small>*Recommended setup*</small> | <small>*Other advanced options*</small> |
    | --------------------- | ----------------------------------- | ------------------------------ |
    | **Custom Prometheus exporters** | [**Configure external services** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/external.md) | |
    | **Third-party metrics sources** | [**Add external exporters** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/external.md#adding-exporters) | |
    | **Additional database features** | [**Extend monitoring capabilities** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/external.md#extended-monitoring) | |


<!--
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
    - [Remote instances](../install-pmm/install-pmm-client/connect-database/remote.md) (for monitoring across network segments) |-->

