# PMM Server deployment overview

PMM Server is the central component of Percona Monitoring and Management (PMM) that collects, analyzes, and visualizes monitoring data from your database environment.

## Before you begin
Before installing PMM Server, make sure to first:
{.power-number}

- [Docker](../install-pmm-server/deployment-options/docker/index.md)
- [Podman](../install-pmm-server/deployment-options/podman/index.md)
- [Helm](../install-pmm-server/deployment-options/helm/index.md)
- [Virtual appliance](../install-pmm-server/deployment-options/virtual/index.md)
<!---- [Amazon AWS](../install-pmm-server/deployment-options/aws/aws.md) -->

## Deployment options

Install and run at least one PMM Server using one of the following deployment methods:

| **Your setup**                             | **Recommended deployment**                                          |
|--------------------------------------------|---------------------------------------------------------------------|
| Running in **AWS**                         | **[AWS Marketplace ->](../install-pmm-server/deployment-options/aws/aws.md)** |
| Cloud-native/**Kubernetes** environment  | **[Helm ->](../install-pmm-server/deployment-options/helm/index.md)**     |
| Traditional **virtual machines**           | **[Virtual Appliance ->](../install-pmm-server/deployment-options/virtual/index.md)** |
| **Fast** setup & flexibility              | **[Docker ->](../install-pmm-server/deployment-options/docker/index.md)** |
| Prioritizing **security** & rootless containers | **[Podman ->](../install-pmm-server/deployment-options/podman/index.md)** |

## Common installation process

While specific steps vary by deployment method, the general installation process includes:
{.power-number}

1. Deploy PMM Server using your preferred method.
2. Access the PMM web interface (default: `https://your-server-address`)
3. Log in with default credentials (username: `admin`, password: `admin`).
4. Change the default password.
5. Configure PMM Server settings.

## Next steps

After installing PMM Server:

- [Install PMM Client](../install-pmm-client/index.md) on hosts you want to monitor
- [Register Client nodes](../register-client-node/index.md) with your PMM Server
- [Connect databases for monitoring](../install-pmm-client/connect-database/index.md)