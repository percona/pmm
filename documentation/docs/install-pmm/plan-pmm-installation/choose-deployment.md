# Choose a PMM deployment strategy

Whether you're monitoring a single database or managing hundreds across your organization, it's important to select the appropriate deployment approach for both PMM Server and PMM Client components.

Plan your PMM architecture to align with your infrastructure, growth expectations, and operational needs.

## PMM architecture overview

PMM components can be deployed in various combinations depending on your needs: 
{.power-number}

1. [PMM Server](../install-pmm-server/index.md): The central component that stores, analyzes, and visualizes monitoring data
2. [PMM Client](../install-pmm-client/index.md): The distributed component installed on database hosts to collect metrics

## Planning considerations

### Hardware and network requirements

For detailed hardware and network specifications, see:

- [hardware and system requirements](../install-pmm/plan-pmm-installation/hardware_and_system.md) 
- [network and firewall requirements](../../install-pmm/plan-pmm-installation/network_and_firewall.md)

### Architecture considerations

- Consider network segmentation and access controls
- Plan user authentication and authorization strategy
- Evaluate TLS certificate requirements (self-signed vs custom certificates)
- For high-security environments, consider Podman's rootless container capabilities
- Both binary installation and Docker containers can be run without `root` privileges, enhancing security

For information on PMM's architecture, see [PMM architecture](../../reference/index.md). 

## PMM Server deployment options

| **Method** | **Best for** | **Advantages** | **Considerations** |
|-----------|------------|---------------|--------------------|
| [**:material-docker: Docker**](../install-pmm-server/deployment-options/docker/index.md) | Development, testing & production | ✔  Quick setup<br>✔  Simple upgrades<br>✔  Works in various environments | ⚠ Requires Docker knowledge<br>⚠ May need additional configuration for production |
| [**:material-shield-lock: Podman**](../install-pmm-server/deployment-options/podman/index.md) | Security-focused setups | ✔ Rootless containers<br> ✔  Enhanced security<br> ✔  OCI-compatible | ⚠ Requires Podman installation & knowledge |
| [**:material-kubernetes: Helm**](../install-pmm-server/deployment-options/helm/index.md) | Cloud-native environments | ✔  Scalable & high availability<br> ✔  Kubernetes-native | ⚠ Requires existing Kubernetes cluster<br>⚠ More complex setup |
| [**:material-server: Virtual Appliance**](../install-pmm-server/deployment-options/virtual/index.md) | Traditional environments | ✔  Pre-configured with all dependencies<br>✔  Dedicated resources | ⚠ Larger resource footprint<br>⚠ Requires a hypervisor |
<!--| [Amazon AWS](../install-pmm/install-pmm-server/deployment-options/aws/aws.md) | AWS-based environments | Seamless AWS integration, easy provisioning | Monthly subscription costs, AWS infrastructure costs |-->

## PMM Client deployment options

| Deployment method | Best for | Advantages | Considerations |
|-------------------|----------|------------|----------------|
| [**Package Manager**](../install-pmm-client/package_manager.md) | Standard Linux environments | • Easy install<br>• Native to OS | • OS-specific<br>• Requires repo access |
| [**Binary Package**](../install-pmm-client/binary_package.md) | Custom/isolated environments | • Portable<br>• Minimal dependencies | • Manual install & updates |
| [**Docker**](../install-pmm-client/docker.md) | Containerized hosts | • Consistent environment<br>• Easy to manage | • Requires Docker<br>• Needs access to host metrics |

## Recommended deployment patterns

Based on the scale and environment of your monitoring needs, we recommend different deployment patterns:

=== "Small-scale (1-30 database instances)"

    - **PMM Server**: Docker or Virtual Appliance
    - **PMM Client**: Package Manager
    - **Implementation tips**:
        - for Docker, use the easy install script for quick setup
        - for Virtual Appliance, use the pre-configured OVA file
        - consider backup options early, even for small deployments
    - **Ideal for**: Small businesses, development environments, initial deployments

=== "Medium (31-200)"

    - **PMM Server**: Docker with volume storage or Kubernetes
    - **PMM Client**: Package Manager or Docker
    - **Implementation tips**:
        - use Docker volumes instead of host directories for better data management
        - consider setting up high availability for production environments
        - implement regular backup procedures for monitoring data
    - **Ideal for**: Mid-sized companies, production environments

=== "Large (200+)"

    - **PMM Server**: Kubernetes with proper resource allocation
    - **PMM Client**: Automated deployment via package manager
    - **Implementation tips**:
        - use infrastructure as code to manage deployments
        - consider distributed monitoring architecture
        - implement proper monitoring of the PMM Server itself
    - **Ideal for**: Large enterprises, mission-critical database fleets

=== "Cloud-based database monitoring"

    - **PMM Server**: AWS Marketplace (for AWS) or Kubernetes (for other clouds)
    - **PMM Client**: Package Manager or automated cloud deployment
    - **PMM Remote**: For monitoring cloud database services (RDS, Azure DB, Cloud SQL)
    - **Implementation tips**:
        - use cloud-native storage options for better performance
        - leverage auto-scaling groups for handling variable loads
        - consider network costs when planning your architecture
    - **Ideal for**: Cloud-native companies, hybrid cloud environments

## Decision factors

When selecting your deployment strategy, consider the following:
{.power-number}

1. **Existing infrastructure** to align with what your team already manages.
2. **Team expertise** to use tools and platforms your team is comfortable with.
3. **Scalability** to plan but for future growth, not just for today.
4. **Security policies** to match the deployment to your organization’s security practices.
5. **Maintenance strategy**  choose what support easy upgrades and backups.

## Deployment planning checklist

✓ Inventory of systems and environments to monitor  
✓ Estimate monitoring scope (instances, metric volume)  
✓ Size the PMM Server based on expected load  
✓ Choose a PMM Server deployment method  
✓ Select the best PMM Client install method per system type  
✓ Verify network and firewall access  
✓ Plan for data backups and disaster recovery  
✓ Define upgrade and patching process for all PMM components

This deployment planning ensures that your PMM environment is efficient, secure, and scalable from day one.

## Next step

[Hardware and system requirements](../plan-pmm-installation/hardware_and_system.md){.md-button} 



