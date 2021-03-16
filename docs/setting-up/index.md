# Setting up

This section is an overview of the PMM setting-up process broken into three key stages:

1. [Setting up at least one PMM Server](#setting-up-pmm-server)
2. [Setting up one or more PMM Clients](#setting-up-pmm-client)
3. [Configuring and adding services for monitoring](#configure-add-services)

```plantuml source="_resources/diagrams/Setting-Up.puml"
```

## 1. Setting up PMM Server {: #setting-up-pmm-server}

You must set up at least one PMM Server.

A server can run as:

- [a Docker container](server/docker.md);
- [a virtual appliance](server/virtual-appliance.md);
- [an Amazon AWS EC2 instance](server/aws.md).

## 2. Setting up PMM Client {: #setting-up-pmm-client}

You must [set up PMM Client](client/index.md) on each node where there is a service to be monitored.

You can do this:

1. [with a package manager (`apt`, `apt-get`, `dnf`, `yum`)](client/index.md#package-manager);
1. [by manually downloading and installing `.deb` or `.rpm` packages](client/index.md#manual-package);
1. [by manually downloading and unpacking a binary package (`.tar.gz`)](client/index.md#binary-package);
1. [with a Docker image](client/index.md#docker).

## 3. Configure and add services {: #configure-add-services}

You must configure your services and adding them to PMM Server's inventory of monitored systems. This is different for each type of service:

- [MySQL and variants (Percona Server for MySQL, Percona XtraDB Cluster, MariaDB)](client/mysql.md)
- [MongoDB](client/mongodb.md)
- [PostgreSQL](client/postgresql.md)
- [ProxySQL](client/proxysql.md)
- [Amazon RDS](client/aws.md)
- [Microsoft Azure](client/azure.md)
- [Linux](client/linux.md)
- [External services](client/external.md)
- [HAProxy](client/haproxy.md)

You do this on each node/service being monitored.

If you have configured everything correctly, you'll see data in the PMM user interface, in one of the dashboards specific to the type of service.
