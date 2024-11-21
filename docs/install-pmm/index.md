# About PMM installation

??? info "Summary"

    !!! summary alert alert-info ""
        1. [Install PMM Server](#install-pmm-server).
        2. [Install PMM Client(s)](#install-pmm-client).
        3. [Add services](#add-services).

## Install PMM Server

Install and run at least one PMM Server.

Choose from:

| Use | <i class="uil uil-thumbs-up"></i> **Benefits** | <i class="uil uil-thumbs-down"></i> **Drawbacks**|
|---|---|---
| [Docker] | 1. Quick<br>2. Simple<br> 3. Rootless |  Additional network configuration required.
| [Podman] | 1. Quick<br>2. Simple<br>3. Rootless |Podman installation required.
| [Helm] (Technical Preview) | 1. Quick<br>2. Simple<br>3. Cloud <br> 4. Rootless| Requires running Kubernetes cluster.
| [Virtual appliance]  | 1. Easily import into Hypervisor of your choice <br> 2. Rootless| More system resources compared to Docker footprint.
| [Amazon AWS] | 1. Wizard-driven install. <br>  2. Rootless| Non-free solution (infrastructure costs).

## Install PMM Client

Install and run PMM Client on every node where there is a service you want to monitor.

The choices are:

- With [Docker](client/index.md#docker);
- Natively, installed from:
    - [Linux package](client/index.md#package-manager) (installed with `apt`, `apt-get`, `dnf`, `yum`);
    - [Binary package](client/index.md#binary-package) (a downloaded `.tar.gz` file).

!!! hint alert "Binary is only way to install PMM client without root permissions"

## Add services
On each PMM Client instance, configure the nodes and services you want to monitor. 
??? info "Which services you can monitor?"

    - [MySQL] (and variants: Percona Server for MySQL, Percona XtraDB Cluster, MariaDB);
    - [MongoDB];
    - [PostgreSQL];
    - [ProxySQL];
    - [Amazon RDS];
    - [Microsoft Azure];
    - [Google Cloud Platform] (MySQL and PostgreSQL);
    - [Linux];
    - [External services];
    - [HAProxy];
    - [Remote instances].

[MySQL]: client/mysql.md
[MongoDB]: client/mongodb.md
[PostgreSQL]: client/postgresql.md
[ProxySQL]: client/proxysql.md
[Amazon RDS]: client/aws.md
[Microsoft Azure]: client/azure.md
[Google Cloud Platform]: client/google.md
[Linux]: client/linux.md
[External services]: client/external.md
[HAProxy]: client/haproxy.md
[Remote instances]: client/remote.md
[dashboards]: ../details/dashboards/
[Docker]: ../install-pmm/install-pmm-server/baremetal/docker/index.md
[Podman]: ../install-pmm/install-pmm-server/baremetal/podman/index.md
[Helm]: ../install-pmm/install-pmm-server/baremetal/helm/index.md
[virtual appliance]: ../install-pmm/install-pmm-server/baremetal/virtual/index.md
[Amazon AWS]: ../install-pmm/install-pmm-server/aws/aws.md
[easy install]: ../install-pmm/install-pmm-server/baremetal/easy-install.md
