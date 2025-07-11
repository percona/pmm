# PMM Client installation overview

PMM Client is the component of Percona Monitoring and Management (PMM) that collects metrics from your database servers and sends them to PMM Server for analysis and visualization.

## Common installation process

While specific steps vary by deployment method, the general installation process includes: 
{.power-number}

1. Install PMM Client using your preferred method.
2. Register the Client node with your PMM Server.
3. Add database services for monitoring.
4. Verify monitoring data in the PMM web interface.

## Before you begin

Complete these steps to prepare your system for PMM installation:

### Prerequisites

1. [Check system requirements](prerequisites.md) to ensure your environment meets the minimum criteria.

2. [Install and configure PMM Server](../install-pmm-server/index.md) using your preferred deployment method. You'll need PMM Server's IP address or hostname to configure PMM Client.

3. [Set up firewall rules](../plan-pmm-installation/network_and_firewall.md) to allow communication between PMM Client and PMM Server.

### Planning checklist

Before starting the installation, make sure:

- PMM Server is installed and reacheable
- Network connectivity is configured  
- Monitoring users for your database are created and have the necessary permissions
- You have administrator access to install PMM Client

## Deployment options

Install PMM Client using one of the following deployment methods:

| **Your setup** | **Recommended deployment** |
|----------------|----------------------------|
| **Production** environments on supported Linux distributions | **[Package Manager →](package_manager.md)** |
| Unsupported Linux distributions or **non-root** installation | **[Binary Package →](binary_package.md)** |
| **Containerized** environments or testing | **[Docker →](docker.md)** |

## Connect services

Each database service requires specific configuration parameters. Configure your service according to its service type:

- [MySQL](connect-database/mysql/mysql.md) (and variants Percona Server for MySQL, Percona XtraDB Cluster, MariaDB)
- [MongoDB](connect-database/mongodb.md)
- [PostgreSQL](connect-database/postgresql.md)
- [ProxySQL](connect-database/proxysql.md)
- [Amazon RDS](connect-database/aws.md)
- [Microsoft Azure](connect-database/azure.md)
- [Google Cloud Platform](connect-database/google.md) (MySQL and PostgreSQL)
- [Linux](connect-database/linux.md)
- [External services](connect-database/external.md)
- [HAProxy](connect-database/haproxy.md)
- [Remote instances](connect-database/remote.md)

### Modifying service configurations

If you need to modify the configuration of a service you've already added, you'll need to remove the service and re-add it with the new parameters.

## Next steps

- [Connect database services](connect-database/index.md) for monitoring
- [Configure optimization settings](connect-database/mysql/improve_perf.md) for specific database types
