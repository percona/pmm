# Welcome

Percona Monitoring and Management (PMM) is an open-source platform
for managing and monitoring MySQL, PostgreSQL, MongoDB, and ProxySQL performance.
It is developed by Percona in collaboration with experts
in the field of managed database services, support and consulting.

!!! alert alert-success "This documentation covers the latest release: PMM {{release}}"

## What is *Percona Monitoring and Management*?

PMM is a free and open-source solution
that you can run in your own environment
for maximum security and reliability.
It provides thorough time-based analysis for MySQL, PostgreSQL and MongoDB servers
to ensure that your data works as efficiently as possible.

## Architecture

The PMM platform is based on a client-server model that enables scalability. It includes the following modules:

* [PMM Client](#pmm-client) installed on every database host that you want to monitor. It collects server metrics, general system metrics, and Query Analytics data for a complete performance overview.

* [PMM Server](#pmm-server) is the central part of PMM that aggregates collected data and presents it in the form of tables, dashboards, and graphs in a web interface.

* [Percona Platform](#percona-platform) provides value-added services for PMM.

![image](_images/diagram.pmm.client-server-platform.png)

The modules are packaged for easy installation and usage. It is assumed that the user should not need to understand what are the exact tools that make up each module and how they interact. However, if you want to leverage the full potential of PMM, the internal structure is important.

PMM is a collection of tools designed to seamlessly work together.  Some are developed by Percona and some are third-party open-source tools.

!!! alert alert-info "Note"
    The overall client-server model is not likely to change, but the set of tools that make up each component may evolve with the product.

The following sections illustrates how PMM is currently structured.

## PMM Client

![image](_images/diagram.pmm.client-architecture.png)

Each PMM Client collects various data about general system and database performance, and sends this data to the corresponding PMM Server.

The PMM Client package consist of the following:

* `pmm-admin` is a command-line tool for managing PMM Client, for example, adding and removing database instances that you want to monitor. For more information, see [pmm-admin - PMM Administration Tool](details/commands/pmm-admin.md).

* `pmm-agent` is a client-side component a minimal command-line interface, which is a central entry point in charge for bringing the client functionality: it carries on client’s authentication, gets the client configuration stored on the PMM Server, manages exporters and other agents.

* `node_exporter` is an exporter that collects general system metrics.

* `mysqld_exporter` is an exporter that collects MySQL server metrics.

* `mongodb_exporter` is an exporter that collects MongoDB server metrics.

* `postgres_exporter` is an exporter that collects PostgreSQL performance metrics.

* `proxysql_exporter` is an exporter that collects ProxySQL performance metrics.

To make data transfer from PMM Client to PMM Server secure, all exporters are able to use SSL/TLS encrypted connections, and their communication with the PMM server is protected by the HTTP basic authentication.

!!! alert alert-info "Note"

    Credentials used in communication between the exporters and the PMM Server are the following ones:

    * login is `pmm`
    * password is equal to Agent ID, which can be seen e.g. on the Inventory Dashboard.

## PMM Server

![image](_images/PMM_Architecture_Client_Server.jpg)

PMM Server runs on the machine that will be your central monitoring host. It is distributed as an appliance via the following:

* Docker image that you can use to run a container

* OVA (Open Virtual Appliance) that you can run in VirtualBox or another hypervisor

* AMI (Amazon Machine Image) that you can run via Amazon Web Services

PMM Server includes the following tools:

* Query Analytics (QAN) enables you to analyze MySQL query performance over periods of time. In addition to the client-side QAN agent, it includes the following:

    * QAN API is the backend for storing and accessing query data collected by the QAN agent running on a PMM Client.

    * QAN Web App is a web application for visualizing collected Query Analytics data.

* Metrics Monitor provides a historical view of metrics that are critical to a MySQL or MongoDB server instance. It includes the following:

    - [VictoriaMetrics](https://github.com/VictoriaMetrics/VictoriaMetrics), a scalable time-series database. (Replaces [Prometheus](https://prometheus.io).)

    - [ClickHouse](https://clickhouse.tech/) is a third-party column-oriented database that facilitates the Query Analytics functionality.

    - [Grafana](http://docs.grafana.org/) is a third-party dashboard and graph builder for visualizing data aggregated (by VictoriaMetrics or Prometheus) in an intuitive web interface.

    * Percona Dashboards is a set of dashboards for Grafana developed by Percona.

All tools can be accessed from the PMM Server [web interface](using/interface.md).

## Percona Platform

Percona Platform provides the following value-added services to PMM.

### Security Threat Tool

Security Threat Tool checks registered database instances for a range of common security issues. This service requires the *Telemetry* setting to be on.



## Contact Us

*Percona Monitoring and Management* is an open source product.  We provide ways for anyone to contact developers and experts directly, submit bug reports and feature requests, and contribute to source code directly.

### Contacting the developers

Use the [community forum](https://www.percona.com/forums/questions-discussions/percona-monitoring-and-management) to ask questions about using PMM.  Developers and experts will try to help with problems that you experience.

### Reporting bugs

Use the [PMM project in JIRA](https://jira.percona.com/projects/PMM) to report bugs and request features.  Please register and search for similar issues before submitting a bug or feature request.

### Contributing to development

To explore source code and suggest contributions, see the [PMM repository list](https://github.com/percona/pmm/blob/PMM-2.0/README.md).

You can fork and clone any Percona repositories, but to have your source code patches accepted please sign the Contributor License Agreement (CLA).
