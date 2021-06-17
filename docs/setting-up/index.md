# Setting up

There are three stages to installing and setting up PMM.

```mermaid
graph LR
1(Set up PMM Server)-->2(Set up PMM Client)-->3(Add services)
```

## Set up PMM Server {: #setting-up-pmm-server}

Set up at least one PMM Server as either:

- [a Docker container](server/docker.md),
- [a virtual appliance](server/virtual-appliance.md),
- or [an Amazon AWS instance](server/aws.md).

## Set up PMM Client {: #setting-up-pmm-client}

On every node where there is a service to be monitored,
set up PMM Client using:

- [Docker](client/index.md#docker),
- [a Linux package manager](client/index.md#package-manager) (`apt`, `apt-get`, `dnf`, `yum`),
- or [the binary package](client/index.md#binary-package) (a `.tar.gz` file).

## Add services {: #configure-add-services}

You must configure your services and add them to PMM Server's inventory of monitored systems.

You do this on each node/service being monitored.

The set up depends on which type of service you want to monitor:

- [MySQL and variants](client/mysql.md) (Percona Server for MySQL, Percona XtraDB Cluster, MariaDB)
- [MongoDB](client/mongodb.md)
- [PostgreSQL](client/postgresql.md)
- [ProxySQL](client/proxysql.md)
- [Amazon RDS](client/aws.md)
- [Microsoft Azure](client/azure.md)
- [Google Cloud Platform](client/google.md) (MySQL and PostgreSQL)
- [Linux](client/linux.md)
- [External services](client/external.md)
- [HAProxy](client/haproxy.md)

When you have configured everything correctly, you'll see data in the PMM user interface, in one of the [dashboards](../details/dashboards/) specific to the type of service.

Here's a graphical overview of the steps involved.

```plantuml
' Syntax: https://plantuml.com/activity-diagram-beta
@startuml "setting-up"
!include docs/_images/plantuml_styles.puml
skinparam partitionWidth 400
partition "<b>Stage 1:</b> Set up PMM Server\nChoices:" {
    split
        -[hidden]->
        :Docker or\nDocker compose;
    split again
        -[hidden]->
        :Virtual appliance;
    split again
        -[hidden]->
        :Amazon AWS marketplace;
    end split
}
partition "<b>Stage 2:</b> Set up PMM Client\nChoices:" {
    split
        :Docker or\nDocker compose;
    split again
        :Package manager;
    split again
        :Binary package;
    end split
}
partition "<b>Stage 3</b>" {
    :Register;
    :Add services;
}
@enduml
```


[MySQL and variants]: client/mysql.md
[MongoDB]: client/mongodb.md
[PostgreSQL]: client/postgresql.md
[ProxySQL]: client/proxysql.md
[Amazon RDS]: client/aws.md
[Microsoft Azure]: client/azure.md
[Google Cloud Platform]: client/google.md
[Linux]: client/linux.md
[External services]: client/external.md
[HAProxy]: client/haproxy.md
