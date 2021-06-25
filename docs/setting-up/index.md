# Setting up

There are three stages to installing and setting up PMM.

```plantuml
@startuml
!include docs/_images/plantuml_styles.puml
rectangle "Set up a PMM Server" as SERVER
rectangle "Set up PMM Client(s)" as CLIENT
rectangle "Add services" as SERVICES
SERVER -right->> CLIENT
CLIENT -right->> SERVICES
@enduml
```

## Set up PMM Server

Install and run at least one PMM Server.

Choose from:

| Use                  | <i class="uil uil-thumbs-up"></i> **Benefits** | <i class="uil uil-thumbs-down"></i> **Drawbacks**
|----------------------|------------------------------------------------|-------------------------------------------------------------
| [Docker]             | Quick, simple                                  | Docker required, will have additional network configuration needs
| [Virtual appliance]  | Easily import into Hypervisor of your choice   | Requires more system resources compared to Docker footprint
| [Amazon AWS]         | Wizard-driven install                          | Non-free solution (infrastructure costs)

## Set up PMM Client

Install and run PMM Client on every node where there is a service you want to monitor.

The choices:

- With [Docker](client/index.md#docker)
- Natively, installed from:
    - [Linux package](client/index.md#package-manager) (installed with `apt`, `apt-get`, `dnf`, `yum`)
    - [Binary package](client/index.md#binary-package) (a downloaded `.tar.gz` file)

## Add services

On each PMM Client, you configure then add to PMM Server's inventory the node or service you want to monitor.

How you do this depends on the type of service. You can monitor:

- [MySQL] (and variants: Percona Server for MySQL, Percona XtraDB Cluster, MariaDB)
- [MongoDB]
- [PostgreSQL]
- [ProxySQL]
- [Amazon RDS]
- [Microsoft Azure]
- [Google Cloud Platform] (MySQL and PostgreSQL)
- [Linux]
- [External services]
- [HAProxy]
- [Remote instances]

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
[virtual appliance]: server/virtual-appliance.md
[Amazon AWS]: server/aws.md
[easy install]: server/easy-install.md
