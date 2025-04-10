# PMM Server deployment overview

PMM Server is the central component of Percona Monitoring and Management (PMM) that collects, analyzes, and visualizes monitoring data from your database environment.

## Before you begin
Before installing PMM Server, make sure to first:
{.power-number}

1. [Choose a deployment strategy](../plan-pmm-installation/choose-deployment.md) based on your environment needs. 
2. Verify your system meets the [hardware requirements](../plan-pmm-installation/hardware_and_system.md). 
3. [Configure your network](../plan-pmm-installation/network_and_firewall.md) for the required connections. 

## Deployment options

Install and run at least one PMM Server using one of the following deployment methods:

### [Deploy with Docker](../install-pmm-server/deployment-options/docker/index.md)
Container-based deployment, ideal for most environments
- Straightforward setup and management
- Flexible deployment on any system supporting Docker

### [Deploy with Podman](../install-pmm-server/deployment-options/podman/index.md)
Rootless container deployment for enhanced security
- Runs containers without `root` privileges
- Integrated with SystemD for service management

### [Deploy with Helm](../install-pmm-server/deployment-options/helm/index.md)
Orchestrated deployment for cloud-native environments
- Scalable deployment on Kubernetes clusters
- Managed through Helm charts

### [Deploy on Virtual Appliance](../install-pmm-server/deployment-options/virtual/index.md)
Pre-configured VM for traditional virtualization
- Ready-to-run OVA file for VMware or VirtualBox
- Includes all dependencies pre-configured

### [Deploy on Amazon AWS](../install-pmm-server/deployment-options/aws/aws.md)
Managed deployment on AWS infrastructure
- Quick launch from AWS Marketplace
- Seamless integration with AWS services

## Common installation process

While specific steps vary by deployment method, the general installation process includes:

1. Deploy PMM Server using your preferred method
2. Access the PMM web interface (default: https://server-address)
3. Log in with default credentials (username: `admin`, password: `admin`)
4. Change the default password
5. Configure PMM Server settings

## Next steps

After installing PMM Server:

- [Install PMM Client](../install-pmm-client/index.md) on hosts you want to monitor
- [Register Client nodes](../register-client-node/) with your PMM Server
- [Connect databases for monitoring](../install-pmm-client/connect-database/index.md)