# Welcome

!!! alert alert-success "This is the technical documentation for the latest release: [PMM {{release}}](release-notes/{{release}}.md)"

Percona Monitoring and Management (PMM) is a free, open-source database and system monitoring tool for MySQL, PostgreSQL, MongoDB, and ProxySQL, and the servers they run on.

PMM helps you improve the performance of database instances, simplify their management, and strengthen their security. With it, you can:

- Visualize a wide range of out-of-the-box system performance metrics
- Collect and analyze data across complex multi-vendor system topologies
- Drill-down and discover the cause of inefficiencies
- Anticipate performance issues, troubleshoot existing ones
- Watch for potential security issues and remedy them

PMM is efficient, quick to set up and easy to use. It runs in cloud, on-prem, or across hybrid platforms. It is supported by [Percona's legendary expertise](https://www.percona.com/services) in open source databases, and by a [vibrant developer and user community](https://www.percona.com/forums/questions-discussions/percona-monitoring-and-management).

!!! alert alert-info "Try the online demo at <a href='https://pmmdemo.percona.com/' target='_blank'>pmmdemo.percona.com</a>"

PMM is a client/server application built by Percona with their own and third-party open-source tools. We provide packages for both PMM Server and PMM Client.

```plantuml source="_resources/diagrams/1_PMM_Context.puml"
```

(See more in [Architecture](details/architecture.md).)

**PMM Server**

PMM Server is the heart of PMM. It receives data from clients, collates it and stores it. Metrics are drawn as tables, charts and graphs within [*dashboards*](details/dashboards/), each a part of the web-based [user interface](using/interface.md). This is an example of the home dashboard from [pmmdemo](https://pmmdemo.percona.com/):

![PMM Server user interface home page](_images/PMM_Home_Dashboard_TALL.jpg)

**PMM Client**

PMM Client runs on every database host or node you want to monitor. The client collects server metrics, general system metrics, and query analytics data, and sends it to the server.

**Percona Enterprise Platform**

[Percona Enterprise Platform](using/platform/) (in development) provides value-added services for PMM.

- [Security Threat Tool](using/platform/security-threat-tool.md): checks registered database instances for a range of common security issues.

## Setting up

!!! alert alert-info "Quickstart installation <{{ extra.quickstart }}>"

To get PMM running, you must:

- [Set up a PMM Server](setting-up/server/index.md) that communicates with clients, receiving metrics data and presenting it in a web-based user interface. PMM Server can run as:
	- [A Docker container](setting-up/server/docker.md);
	- An [OVA/OVF virtual appliance](setting-up/server/virtual-appliance.md) running on VirtualBox, VMware and other hypervisors;
	- An [Amazon AWS EC2 instance](setting-up/server/aws.md).
- [Set up PMM Client](setting-up/client/index.md) on all hosts you want to monitor according to the type of system:
	- Databases
		- [MySQL, Percona Server, MariaDB](setting-up/client/mysql.md)
		- [MongoDB](setting-up/client/mongodb.md)
		- [PostgreSQL](setting-up/client/postgresql.md)
		- [Amazon RDS](setting-up/client/aws.md)
		- [Microsoft Azure](setting-up/client/azure.md)
	- Services
		- [ProxySQL](setting-up/client/proxysql.md)
		- [Linux](setting-up/client/linux.md)
		- [External services](setting-up/client/external.md)
		- [HAProxy](setting-up/client/haproxy.md)

	The PMM Client package provides exporters for different database and system types, and administration tools and agents.

## Documentation site map

```plantuml format="svg_object" width="90%" height="90%" source="_resources/diagrams/Map.puml"
```
