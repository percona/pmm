# About PMM installation

??? info "Summary"

    !!! summary alert alert-info ""
        1. [Install PMM Server](#install-pmm-server).
        2. [Install PMM Client(s)](#install-pmm-client).
        3. [Add services](#add-services).

## Install PMM Server

Install and run at least one PMM Server. Choose from the following options:

!!! summary alert alert-info "ARM support"
     PMM Server is not currently available as a native ARM64 build. For ARM-based systems, consider using the Docker or Podman installation methods, which can run x86_64 images via emulation on ARM platforms.


| Use | :material-thumb-up: **Benefits** | :material-thumb-down: **Drawbacks**|
|---|---|---
| [Docker](../install-pmm/install-pmm-server/baremetal/docker/index.md) | 1. Quick<br>2. Simple<br> 3. Rootless |  Additional network configuration required.
| [Podman](../install-pmm/install-pmm-server/baremetal/podman/index.md) | 1. Quick<br>2. Simple<br>3. Rootless | Podman installation required.
| [Helm](../install-pmm/install-pmm-server/baremetal/helm/index.md) (Technical Preview) | 1. Quick<br>2. Simple<br>3. Cloud-compatible <br> 4. Rootless| Requires running a Kubernetes cluster.
| [Virtual appliance](../install-pmm/install-pmm-server/baremetal/virtual/index.md)  | 1. Easily import into Hypervisor of your choice <br> 2. Rootless| More system resources compared to Docker footprint.
| [Amazon AWS](../install-pmm/install-pmm-server/aws/aws.md) | 1. Wizard-driven install. <br>  2. Rootless| Paid, incurs infrastructure costs.

## Install PMM Client

Install and run PMM Client on every node where there is a service you want to monitor. PMM Client now supports both x86_64 and ARM64 architectures.

The installation choices are:

=== "With Docker"

    [Running PMM Client as a Docker container](../install-pmm/install-pmm-client/docker.md) simplifies deployment across different architectures and automatically selects the appropriate image for your architecture (x86_64 or ARM64).

=== "With package manager"

    [Linux package](../install-pmm/install-pmm-client/package_manager.md): Use `apt`, `apt-get`, `dnf`, `yum`. The package manager automatically selects the correct version for your architecture.

=== "With binary package"

    [Binary package](../install-pmm/install-pmm-client/binary_package.md): Download the appropriate `.tar.gz` file for your architecture (x86_64 or ARM64).


!!! hint alert "Tips"
    Both binary installation and Docker containers can be run without root permissions. When installing on ARM-based systems, ensure you're using ARM64-compatible versions. Performance may vary between architectures.

## Add services

On each PMM Client instance, configure the nodes and services you want to monitor. 

??? info "Which services you can monitor?"

    - [MySQL](../install-pmm/install-pmm-client/connect-database/mysql.md) and variants: Percona Server for MySQL, Percona XtraDB Cluster, MariaDB
    - [MongoDB](../install-pmm/install-pmm-client/connect-database/mongodb.md)
    - [PostgreSQL](../install-pmm/install-pmm-client/connect-database/postgresql.md)
    - [ProxySQL](../install-pmm/install-pmm-client/connect-database/proxysql.md)
    - [Amazon RDS](../install-pmm/install-pmm-client/connect-database/aws.md)
    - [Microsoft Azure](../install-pmm/install-pmm-client/connect-database/azure.md)
    - [Google Cloud Platform](../install-pmm/install-pmm-client/connect-database/google.md)
    - [Linux](../install-pmm/install-pmm-client/connect-database/linux.md)
    - [External services](../install-pmm/install-pmm-client/connect-database/external.md)
    - [HAProxy](../install-pmm/install-pmm-client/connect-database/haproxy.md)
    - [Remote instances](../install-pmm/install-pmm-client/connect-database/remote)
