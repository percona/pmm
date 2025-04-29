# Prerequisites for PMM Client

Before installing PMM Client, ensure your environment meets these requirements.

## Quick requirements checklist

✓ **Hardware**: 64-bit system (x86_64 or ARM64) with at least 100 MB storage  
✓ **OS**: Modern 64-bit Linux (Debian, Ubuntu, RHEL, Oracle Linux, Amazon Linux 2023)  
✓ **Network**: Connectivity to PMM Server (ports 80/443)  
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
| PMM Client > PMM Server | 443 (or 80) | Metrics reporting and management |
| PMM Client > Database instances | Varies by DB type | Collection of monitoring data |

For a complete list of ports and detailed network configuration options, see [Network and firewall requirements](../plan-pmm-installation/network_and_firewall.md).

## Required software

- Ensure these packages are installed before proceeding:

    ```
    curl gnupg sudo wget
    ```

- For Docker-based deployment, you'll also need [Docker Engine](https://docs.docker.com/get-started/get-docker/) properly installed and configured. 

## Database monitoring requirements

To monitor database instances, you'll need:

- **Monitoring users**: Database accounts with appropriate permissions
- **Log access**: File system access to database logs where applicable
- **Performance schema**: Enabled for MySQL monitoring (recommended)

## Database monitoring requirements

To ensure successful database monitoring with PMM, confirm the following:

- **Monitoring users**: Create database accounts with the required permissions  
- **Log access**: Enable file system access to database logs (where applicable)  
- **Performance Schema**: Recommended for enhanced MySQL monitoring  

=== ":material-database: Core databases"

    - [MySQL monitoring requirements](../install-pmm-client/connect-database/mysql/mysql.md#create-a-database-account-for-pmm)  
    - [MongoDB monitoring requirements](../install-pmm-client/connect-database/mongodb.md#create-a-database-account-and-set-permissions)  
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


## Before you install

Complete these steps before installing PMM Client:
{.power-number}

1. **Install PMM Server** using your [preferred deployment method](../install-pmm-server/index.md)
2. **Note the PMM Server address** (hostname or IP)
3. **Select your PMM Client deployment approach** based on your [environment needs](../plan-pmm-installation/choose-deployment.md)
4. **Prepare database credentials** for monitoring users
5. **Verify firewall rules** allow necessary connections

## Next steps

After confirming your environment meets these prerequisites:
{.power-number}

1. [Install PMM Client](../install-pmm-client/index.md) using your preferred method
2. [Register the Client node](../register-client-node/index.md) with your PMM Server
3. [Add database instances](../install-pmm-client/connect-database/index.md) for monitoring

## Troubleshooting

If you encounter issues during installation or setup, refer to the [Troubleshooting checklist](../../troubleshoot/checklist.md).