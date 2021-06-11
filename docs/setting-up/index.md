# Setting up

There are three stages when installing and setting up PMM.

## 1. Set up at least one PMM Server {: #setting-up-pmm-server}

You have these options:

- [Pull and run our Docker container](server/docker.md).
- [Download and import our Virtual Appliance](server/virtual-appliance.md) (as an `.ovf` file).
- [Use your Amazon AWS account and our marketplace offering](server/aws.md).

## 2. Set up one or more PMM Clients {: #setting-up-pmm-client}

You must set up PMM Client on each node where there is a service to be monitored.

You have these options:

- [Use a package manager](client/index.md#package-manager) (`apt`, `apt-get`, `dnf`, `yum`).
- [Manually download and install packages](client/index.md#manual-package) (as `.deb` or `.rpm` files).
- [Manually download and unpack a binary package](client/index.md#binary-package) (a `.tar.gz` file).
- [Pull and run our Docker image](client/index.md#docker).

## 3. Configure and add services {: #configure-add-services}

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
title Setting up PMM\nOverview\n
partition "<b>Stage 1:</b> Set up PMM Server. Choose from:" {
    split
        -[hidden]->
        :Docker container;
    split again
        -[hidden]->
        :Virtual appliance;
    split again
        -[hidden]->
        :Amazon AWS marketplace;
    end split
}
partition "<b>Stage 2:</b> Set up PMM Client on each node. Choose from:" {
    split
        partition "Package\nmanager" {
            :Set up ""percona-release"";
            :""apt/yum install pmm2-client"";
        }
    split again
        partition "Manual\npackage\ninstall" {
            :Download "".deb"" / "".rpm"";
            :Install:
            <code>
            dpkg -i *.deb
            </code>
            or
            <code>
            dnf localinstall *.rpm
            </code>;
        }
    split again
        partition "Binary\npackage\ninstall" {
            :Download "".tar.gz"";
            :<code>
            tar xfz ...
            pmm-agent setup ...
            </code>;
        }
    split again
        partition "Docker" {
            :<code>
            docker pull percona/pmm-client:2
            </code>;
        }
    end split
}
partition "<b>Stage 3</b>" {
    :Configure and add service(s);
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
