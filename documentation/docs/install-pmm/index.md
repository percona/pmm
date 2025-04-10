# About PMM installation

Installing Percona Monitoring and Management (PMM) involves setting up a central PMM Server and distributed PMM Clients that work together to monitor your database environment. 

PMM Server provides the web interface with dashboards and analytics, while PMM Clients collect data from your databases with minimal performance impact and send it back to PMM Server for analysis and visualization.

## Installation overview
The PMM installation consists of three main steps that need to be completed in sequence: 
{.power-number}

1. **[Deploy PMM Server](#1-install-pmm-server)**: centralized platform that collects, analyzes, and visualizes your monitoring data
2. **[Install PMM Clients](#2-install-pmm-client)**: lightweight agents on each database host that collect metrics without impacting performance
3. **[Configure monitoring services](#3-add-services-for-monitoring)**: connect PMM to your database instances, select which metrics to collect, and customize monitoring parameters

## Prerequisites

Before you begin the installation, make sure to:

- Review the [deployment strategy options](../install-pmm/plan-pmm-installation/choose-deployment.md) to determine the best fit for your environment.  
- Confirm that your system meets the [hardware and software requirements](../install-pmm/plan-pmm-installation/hardware_and_system.md).  
- Ensure your environment satisfies the [network and firewall requirements](../install-pmm/plan-pmm-installation/network_and_firewall.md) for proper connectivity.


## Installation steps 

### 1. Install PMM Server

Install and run at least one PMM Server using one of the following deployment methods. If you're sure which deployment method is best for your environment, check out this [Choose a PMM deployment strategy](../install-pmm/plan-pmm-installation/choose-deployment.md) topic for a comparison of your options:

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
    
    | **Database** | **Description** | **Setup guide** |
    | ------------------------- | ---------------------------- | ---------------------------- |
    | :material-dolphin: **MySQL** | MySQL, Percona Server for MySQL, MariaDB | [**Connect to MySQL** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/mysql.md) |
    | :material-elephant: **PostgreSQL** | PostgreSQL database servers | [**Connect to PostgreSQL** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/postgresql.md) |
    | :material-leaf: **MongoDB** | MongoDB, Percona Server for MongoDB | [**Connect to MongoDB** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/mongodb.md) |

=== ":material-cloud: Cloud services"
    Configure monitoring for cloud-hosted database services:
    
    | **Cloud provider** | **Supported services** | **Setup guide** |
    | ------------------------------- | ----------------------------------- | ---------------------------- |
    | :material-aws: **Amazon Web Services** | Amazon RDS for MySQL/PostgreSQL, Aurora | [**Configure AWS monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/aws.md) |
    | :material-microsoft-azure: **Microsoft Azure** | Azure Database for MySQL/PostgreSQL, Cosmos DB | [**Configure Azure monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/azure.md) |
    | :material-google-cloud: **Google Cloud** | Cloud SQL for MySQL/PostgreSQL | [**Configure Google Cloud monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/google.md) |

=== ":material-server: System monitoring"
    Monitor operating system and remote services:
    
    | **Monitoring type**| **Description**|**Setup guide** |
    | -------------------------------- | ---------------------------- | ---------------------------- |
    | :simple-linux: **Linux systems** | CPU, memory, disk, network metrics | [**Configure Linux monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/linux.md) |
    | :material-network: **Remote MySQL** | Monitor MySQL across network segments | [**Configure remote MySQL** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/remote.md#mysql-remote) |
    | :material-network: **Remote PostgreSQL** | Monitor PostgreSQL across network segments | [**Configure remote PostgreSQL** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/remote.md#postgresql-remote) |
    | :material-network: **Remote MongoDB** | Monitor MongoDB across network segments | [**Configure remote MongoDB** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/remote.md#mongodb-remote) |

=== ":material-transit-connection-variant: Proxy services"
    Monitor database proxy and load balancing solutions:
    
    | **Proxy service** | **Description**| **Setup guide** |
    | ------------------------------ | ---------------------------- | ---------------------------- |
    | :material-database-cog: **ProxySQL** | MySQL database proxy | [**Configure ProxySQL monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/proxysql.md) |
    | :material-scale-balance: **HAProxy** | High-availability load balancer | [**Configure HAProxy monitoring** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/haproxy.md) |

=== ":material-gauge: External services"
    Extend PMM with external exporters and third-party metrics:
    
    | **Service type** | **Description** | **Setup guide** |
    | ----------------------------- | ---------------------------- | ---------------------------- |
    | :material-chart-line: **Custom exporters** | Add custom Prometheus exporters | [**Configure external exporters** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/external.md#adding-exporters) |
    | :material-api: **Third-party sources** | Connect external metrics sources | [**Extend monitoring capabilities** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/external.md#extended-monitoring) |
    | :material-application-cog: **General external services** | Generic external service monitoring | [**Configure external services** :material-arrow-right:](../install-pmm/install-pmm-client/connect-database/external.md) |

## What's next

After completing the installation, you can:

- [Configure alerts](../alert/index.md) to notify you of critical events
- [Set up backup management](../backup/index.mdd) to protect your data
- [Explore dashboards](../use/dashboards-panels/index.md) to monitor your database performance
- [Analyze query performance](../use/qan/index.md) to identify and optimize slow queries