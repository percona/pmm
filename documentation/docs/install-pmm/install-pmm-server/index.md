# PMM Server installation overview

PMM Server is the central component of Percona Monitoring and Management (PMM) that collects, analyzes, and visualizes monitoring data from your database environment.

??? info "Common installation process at a glance"
    While specific steps vary by deployment method, the general installation process includes:
    {.power-number}

    1. Deploy PMM Server using your preferred method.
    2. Access the PMM web interface (default: `https://your-server-address`)
    3. Log in with default credentials (username: `admin`, password: `admin`).
    4. Change the default password.
    5. Configure PMM Server settings. 

## Before you begin
Before installing PMM Server, make sure to first read the [Hardware and system requirements](../plan-pmm-installation/hardware_and_system.md) and the [Network and firewall requirements](../plan-pmm-installation/network_and_firewall.md).

## Deployment options

Install and run at least one PMM Server using one of the following deployment methods:

| Environment/Requirement       | Recommended Method          | Documentation Link                                                                 |
|-------------------------------|-----------------------------|-----------------------------------------------------------------------------------|
| **Kubernetes** environments   | Helm chart                  | [Helm installation guide →](../install-pmm-server/deployment-options/helm/index.md) |
| **Virtual machines**          | Virtual appliance           | [VM installation →](../install-pmm-server/deployment-options/virtual/index.md)     |
| **Quick setup** needs         | Docker container            | [Docker guide →](../install-pmm-server/deployment-options/docker/index.md)         |
| **Security-focused** setups   | Podman (rootless containers)| [Podman instructions →](../install-pmm-server/deployment-options/podman/index.md)  |

|  **AWS cloud** deployments     | AWS Marketplace             | [AWS option →](../install-pmm-server/deployment-options/aws/deploy_aws.md)|


## Next steps

After installing PMM Server:

- [Install PMM Client](../install-pmm-client/index.md) on hosts you want to monitor
- [Connect databases for monitoring](../install-pmm-client/connect-database/index.md)