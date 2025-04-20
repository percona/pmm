# PMM Client deployment installation

PMM Client is the component of Percona Monitoring and Management (PMM) that collects metrics from your database servers and sends them to PMM Server for analysis and visualization.

## Before you begin

Before installing PMM Client, make sure to first: 
{.power-number}

- [Check the prerequisites](prerequisites.md) to ensure your system meets the necessary requirements.
- [Install PMM Server](../install-pmm-server/index.md) and note its IP address or hostname.
- [Configure your network](../plan-pmm-installation/network_and_firewall.md) for the required connections.

## Deployment options

Install PMM Client using one of the following deployment methods:

| **Your setup** | **Recommended deployment** |
|----------------|----------------------------|
| **Production** environments on supported Linux distributions | **[Package Manager →](package_manager.md)** |
| Unsupported Linux distributions or **non-root** installation | **[Binary Package →](binary_package.md)** |
| **Containerized** environments or testing | **[Docker →](docker.md)** |

## Common installation process

While specific steps vary by deployment method, the general installation process includes: 
{.power-number}

1. Install PMM Client using your preferred method.
2. Register the Client node with your PMM Server.
3. Add database services for monitoring.
4. Verify monitoring data in the PMM web interface.

## Next steps

After installing PMM Client:

- [Register your Client node](../register-client-node/index.md) with PMM Server
- [Connect database services](connect-database/index.md) for monitoring
- [Configure optimization settings](connect-database/mysql/improve_perf.md) for specific database types
