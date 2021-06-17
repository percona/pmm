# Setting up

There are three stages to installing and setting up PMM.

```plantuml
@startuml
skinparam rectangle {
    roundCorner 25
}
skinparam defaultFontName Chivo

rectangle "Set up PMM Server" as SERVER
rectangle "Set up PMM Client" as CLIENT
rectangle "Add services" as SERVICES
SERVER -right->> CLIENT
CLIENT -right->> SERVICES
@enduml
```

## Set up PMM Server

Choose how you want to run PMM Server:

- With [Docker]
- As a [virtual appliance]
- On [Amazon AWS]

## Set up PMM Client

Choose how you want to run PMM Client:

- With [Docker](client/index.md#docker)
- Natively, installed from:
    - [Linux package](client/index.md#package-manager) (installed with `apt`, `apt-get`, `dnf`, `yum`)
    - [Binary package](client/index.md#binary-package) (a downloaded `.tar.gz` file)

## Add services

You must configure services and add them to PMM Server's inventory of monitored systems for each node/service being monitored. How you do this depends on the type of service.

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