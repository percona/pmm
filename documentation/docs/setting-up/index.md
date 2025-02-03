# Setting up

There are three stages to installing and setting up PMM.

!!! summary alert alert-info "Summary"
    1. [Set up a PMM Server](#set-up-pmm-server).
    2. [Set up PMM Client(s)](#set-up-pmm-client).
    3. [Add services](#add-services).

## Set up PMM Server

Install and run at least one PMM Server.

Choose from:

| Use | <i class="uil uil-thumbs-up"></i> **Benefits** | <i class="uil uil-thumbs-down"></i> **Drawbacks**|
|---|---|---
| [Docker] | 1. Quick.<br>2. Simple. | 1. Docker installation required.<br>2. Additional network configuration required.
| [Podman] | 1. Quick.<br>2. Simple.<br>3. Rootless. | 1. Podman installation required.
| [Helm] | 1. Quick.<br>2. Simple.<br>3. Cloud. | 1. Requires running Kubernetes cluster.
| [Virtual appliance]  | 1. Easily import into Hypervisor of your choice | 1. More system resources compared to Docker footprint.
| [Amazon AWS] | 1. Wizard-driven install. | 1. Non-free solution (infrastructure costs).

## Set up PMM Client

Install and run PMM Client on every node where there is a service you want to monitor.

The choices:

- With [Docker](client/index.md#docker);
- Natively, installed from:
    - [Linux package](client/index.md#package-manager) (installed with `apt`, `apt-get`, `dnf`, `yum`);
    - [Binary package](client/index.md#binary-package) (a downloaded `.tar.gz` file).

!!! hint alert "Binary is only way to install PMM client without root permissions"

## Add services

On each PMM Client, you configure then add to PMM Server's inventory the node or service you want to monitor.

How you do this depends on the type of service. You can monitor:

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
[Docker]: server/docker.md
[Podman]: server/podman.md
[Helm]: server/helm.md
[virtual appliance]: server/virtual-appliance.md
[Amazon AWS]: server/aws.md
[easy install]: server/easy-install.md
