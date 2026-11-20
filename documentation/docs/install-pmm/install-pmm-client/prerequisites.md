# Prerequisites for PMM Client

Before installing PMM Client, ensure your environment meets these requirements.

## Quick requirements checklist

✓ **Hardware**: 64-bit system (x86_64 or ARM64) with at least 100 MB storage  
✓ **OS**: Modern 64-bit Linux (Debian, Ubuntu, RHEL, Oracle Linux, Amazon Linux 2023)  
✓ **Network**: Connectivity to PMM Server (port 443)  
✓ **Software**: curl, gnupg, sudo, wget  
✓ **Database**: Appropriate monitoring user credentials  

## System requirements

PMM Client is designed to be lightweight but requires:

- **Architecture**: x86_64 or ARM64
- **RAM**: Minimal (100-200 MB per monitored database instance)
- **Storage**:

    - 100 MB for installation
    - VM Agent reserves 1 GB for caching during network outages

For comprehensive hardware specifications, see [Hardware and system requirements](../plan-pmm-installation/hardware_and_system.md#pmm-client).

## Network connectivity

PMM Client requires these network connections:

| Connection | Port | Purpose |
|------------|------|---------|
| PMM Client > PMM Server | 443 | Metrics reporting and management. Use port 8443 if your environment restricts privileged ports (<1024).  |
| PMM Client > Database instances | Varies by DB type | Collection of monitoring data |

For a complete list of ports and detailed network configuration options, see [Network and firewall requirements](../plan-pmm-installation/network_and_firewall.md).

## Required software

- Ensure these packages are installed before proceeding: curl, gnupg, sudo, wget.

- For Docker-based deployment, you'll also need [Docker Engine](https://docs.docker.com/get-started/get-docker/) properly installed and configured. 

## Database monitoring requirements

To ensure successful database monitoring with PMM, confirm the following:

- **Monitoring users**: Permissions for dashboard metrics — see database-specific setup guides
- **Query Analytics**: Depends on query source (Profiler/mongolog, slowlog/perfschema) — see the same guides

=== ":material-database: Core databases"

    - [MySQL monitoring requirements](../install-pmm-client/connect-database/mysql/mysql.md#create-a-database-account-for-pmm)  
    - [MongoDB monitoring requirements](../install-pmm-client/connect-database/mongodb.md#prerequisites)  
    - [PostgreSQL monitoring requirements](../install-pmm-client/connect-database/postgresql.md#create-a-database-account-for-pmm)

=== ":material-cloud: Cloud services"

    - [Amazon RDS / Aurora](../install-pmm-client/connect-database/aws.md#creating-an-iam-user-with-permission-to-access-amazon-rds-db-instances)
    - [Microsoft Azure](../install-pmm-client/connect-database/azure.md#required-settings)  
    - [Google Cloud Platform](../install-pmm-client/connect-database/google.md#mysql)

=== ":material-transit-connection-variant: Proxy services"

    - [ProxySQL monitoring requirements](../install-pmm-client/connect-database/proxysql.md)  
    - [HAProxy monitoring requirements](../install-pmm-client/connect-database/haproxy.md)

=== ":material-gauge: Additional services"

    - [External services monitoring](../install-pmm-client/connect-database/external.md)  
    - [Remote instances monitoring](../install-pmm-client/connect-database/remote.md#recommended-settings)

## Troubleshooting

If you encounter issues during installation or setup, see the [Troubleshooting checklist](../../troubleshoot/checklist.md).

## Next steps

After confirming your environment meets these prerequisites:
{.power-number}

- [Install PMM Client](../install-pmm-client/index.md) using your preferred method
- [Add database instances](../install-pmm-client/connect-database/index.md) for monitoring