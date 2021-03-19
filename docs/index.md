# Welcome

**Percona Monitoring and Management (PMM) is a free, open-source monitoring tool for MySQL, PostgreSQL, MongoDB, and ProxySQL, and the servers they run on.** PMM helps you improve the performance of databases, simplify their management, and strengthen their security.

<div class="alert alert-success">
This documentation is for the latest release: <a href="release-notes/{{release}}.html">PMM {{release}}</a>
</div>

With PMM, you can:

- Visualize a wide range of out-of-the-box system performance metrics
- Collect and analyze data across complex multi-vendor system topologies
- Drill-down and discover the cause of inefficiencies, anticipate performance issues, or troubleshoot existing ones
- Watch for potential security issues and remedy them

> Try the live demo: <a href='https://pmmdemo.percona.com/' target='_blank'>pmmdemo.percona.com</a>

PMM is efficient, quick to set up and easy to use. It runs in cloud, on-prem, or across hybrid platforms. It is supported by [our legendary expertise][PERCONA_SERVICES] in open source databases, and by a [vibrant developer and user community][PMM_FORUM].

## Setting up

[PMM Server](setting-up/server/index.md) can run as:

- [a Docker container](setting-up/server/docker.md)
- a [virtual machine](setting-up/server/virtual-appliance.md)
- an [Amazon AWS EC2 instance](setting-up/server/aws.md)

[PMM Client](setting-up/client/index.md) runs on all hosts you want to monitor. The setup varies according to the type of system:

- [MySQL and variants](setting-up/client/mysql.md)
- [MongoDB](setting-up/client/mongodb.md)
- [PostgreSQL](setting-up/client/postgresql.md)
- [Amazon RDS](setting-up/client/aws.md)
- [Microsoft Azure](setting-up/client/azure.md)
- [Google Cloud Platform](setting-up/client/google.md)
- [ProxySQL](setting-up/client/proxysql.md)
- [Linux](setting-up/client/linux.md)
- [External services](setting-up/client/external.md)
- [HAProxy](setting-up/client/haproxy.md)

> [**Quickstart installation**][PMM_QUICKSTART]

## How it works

PMM is a client/server application built by us with our own and third-party open-source tools. (Read more in [Architecture](details/architecture.md).)


```plantuml source="_resources/diagrams/1_PMM_Context.puml"
```

**PMM Server**

PMM Server is the heart of PMM. It receives data from clients, collates it and stores it. Metrics are drawn as tables, charts and graphs within [*dashboards*](details/dashboards/), each a part of the web-based [user interface](using/interface.md).

This is the home dashboard from [pmmdemo][PMMDEMO]:

![PMM Server user interface home page](_images/PMM_Home_Dashboard_TALL.jpg)

**PMM Client**

PMM Client runs on every database host or node you want to monitor. The client collects server metrics, general system metrics, and query analytics data, and sends it to the server.

**Percona Enterprise Platform**

[Percona Enterprise Platform](using/platform/) (in development) provides value-added services for PMM.

## Documentation site map

```plantuml format="svg_object" width="90%" height="90%" source="_resources/diagrams/Map.puml"
```


[PERCONA_SERVICES]: https://www.percona.com/services
[PMM_FORUM]: https://www.percona.com/forums/questions-discussions/percona-monitoring-and-management
[PMM_QUICKSTART]: https://www.percona.com/software/pmm/quickstart
[PMMDEMO]: https://pmmdemo.percona.com/
