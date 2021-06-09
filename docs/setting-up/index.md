# Setting up

The PMM setting-up process can be broken into three key stages:

1. [Setting up at least one PMM Server](#setting-up-pmm-server)
2. [Setting up one or more PMM Clients](#setting-up-pmm-client)
3. [Configuring and adding services for monitoring](#configure-add-services)

```plantuml
@startuml "setting-up"
!include docs/_images/plantuml_styles.puml
skinparam partitionWidth 400
title Setting up PMM\nOverview\n
partition "Set up PMM Server as one of: " {
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
partition "Set up PMM Client for each node via: " {
    split
        partition "Package manager" {
            :Set up ""percona-release"";
            :""apt/yum install pmm2-client"";
        }
    split again
        partition "Manual package install" {
            :Download "".deb""/"".rpm"";
            :Install
            <code>
            dpkg -i *.deb # Debian
            dnf localinstall *.rpm # RHEL
            </code>;
        }
    split again
        partition "Binary package install" {
            :Download "".tar.gz"";
            :<code>
            tar xfz ...
            pmm-agent setup ...
            </code>;
        }
    split again
        partition "Docker" {
            :<code>
            docker pull
            percona/pmm-client:2
            </code>;
        }
    end split
}
:Configure and add service;
@enduml
```

## Setting up PMM Server {: #setting-up-pmm-server}

You must set up at least one PMM Server. A server can run:

- [with Docker](server/docker.md)
- [as a virtual appliance](server/virtual-appliance.md)
- [on an Amazon AWS EC2 instance](server/aws.md)

## Setting up PMM Client {: #setting-up-pmm-client}

You must [set up PMM Client](client/index.md) on each node where there is a service to be monitored. You can do this:

1. [with a package manager (`apt`, `apt-get`, `dnf`, `yum`)](client/index.md#package-manager)
1. [by manually downloading and installing `.deb` or `.rpm` packages](client/index.md#manual-package)
1. [by manually downloading and unpacking a binary package (`.tar.gz`)](client/index.md#binary-package)
1. [with a Docker image](client/index.md#docker)

## Configure and add services {: #configure-add-services}

You must configure your services and add them to PMM Server's inventory of monitored systems. This is different for each type of service:

- [MySQL and variants (Percona Server for MySQL, Percona XtraDB Cluster, MariaDB)](client/mysql.md)
- [MongoDB](client/mongodb.md)
- [PostgreSQL](client/postgresql.md)
- [ProxySQL](client/proxysql.md)
- [Amazon RDS](client/aws.md)
- [Microsoft Azure](client/azure.md)
- [Google Cloud Platform (MySQL and PostgreSQL)](client/google.md)
- [Linux](client/linux.md)
- [External services](client/external.md)
- [HAProxy](client/haproxy.md)

You do this on each node/service being monitored.

If you have configured everything correctly, you'll see data in the PMM user interface, in one of the dashboards specific to the type of service.
