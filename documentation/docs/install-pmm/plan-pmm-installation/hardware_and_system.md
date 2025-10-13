# Hardware and system requirements
To ensure optimal performance for your monitoring environment, check the appropriate hardware specifications before installing PMM.

For guidance on selecting the best deployment method based on these requirements, see the [choose your PMM deployment strategy](../plan-pmm-installation/choose-deployment.md) topic.

## PMM Server resource requirements
Resource requirements scale with the number of nodes and services monitored. Here are our recommendations for different deployment scales:

=== "Typical deployment (1-30 nodes)"

    This is the most common deployment scenario, suitable for small to medium-sized environments:

    - **CPU**: 4 cores
    - **Memory**: 8 GB  
    - **Storage**: 100 GB
    - **Example workloads**: Development environments, small businesses, initial deployments

=== "Medium (31-200 nodes)"

    Recommended for environments monitoring MySQL, PostgreSQL, or MongoDB at scale:

    - **CPU**: 8-16 cores
    - **Memory**: 16-32 GB
    - **Storage**: 200 GB
    - **CPU usage**: Expect 20-70% utilization
    - **Example workloads**: Production environments, mid-sized companies

=== "Large (200+ nodes)"

    Designed for extensive monitoring environments with high-node counts:

    - **CPU**: 16+ cores
    - **Memory**: 32+ GB
    - **Storage**: 500+ GB
    - **Example workloads**: Large enterprises, mission-critical database fleets

## Storage planning
Adjust storage calculations based on your data retention period and the number of metrics collected. To estimate storage requirements:

- **Base formula**: `nodes × retention_period_in_weeks × 1 GB`
- **Quick estimate**: for the default 30-day retention period, use the formula `number_of_nodes x 4 GB`
- **High-precision monitoring**: increase estimates by 20-50% when using 1-second collection intervals

### Storage optimization

To reduce storage usage, consider [disable table statistics](../install-pmm-client/connect-database/mysql/improve_perf.md#disable-table-statistics), which can significantly decrease the size of the VictoriaMetrics database.

## Architecture requirements

### PMM Server 

- **CPU**: must support the [`SSE4.2`](https://wikipedia.org/wiki/SSE4#SSE4.2), which is required for Query Analytics (QAN).
- **ARM64**: ensure your system uses a supported ARM64 architecture (such as ARMv8 or later). PMM Server is not currently available as a native ARM64 build. For ARM-based systems, use Docker or Podman to run x86_64 images via emulation. To explicitly force Docker to use the x86_64 image on an ARM system, use: `docker run --platform linux/amd64 ... <your_pmm_server_image>`. 

### PMM Client 

- **Installation storage**: Requires 100 MB of storage for installation
- **Cache storage**: 
    - VM Agent reserves 1 GB of disk space for caching during network outages
    - Query Analytics (QAN) uses RAM instead of disk storage for its cache
- **Architecture support**: Compatible with both x86_64 and ARM64 architectures
- **Operating systems**: Compatible with modern 64-bit Linux distributions including Debian, Ubuntu, Oracle Linux, and "Red Hat" derivatives

For specific version support details, see [Percona software support life cycle](https://www.percona.com/services/policies/percona-software-support-lifecycle#pt).

### ARM-specific considerations

- **Docker images**: If using Docker for PMM Client on ARM systems, ensure you're using the ARM64-compatible Docker images.
- **Performance testing**: Performance may vary across different ARM implementations. Conduct thorough testing to ensure optimal performance in your environment.
- **Software compatibility**: Ensure you're using ARM-compatible versions of any additional software or databases you're monitoring with PMM.
- **Resource monitoring**: Monitor resource usage closely on ARM systems, as it may differ from x86_64 systems. Adjust your configuration as needed for optimal performance.

## Next step

[Network and firewall requirements](../plan-pmm-installation/network_and_firewall.md){.md-button}
 
