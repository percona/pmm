# Terminology Reference

[TOC]

## Data retention

By default, Prometheus stores time-series data for 30 days, and QAN stores query data for 8 days.

Depending on available disk space and your requirements, you may need to adjust data retention time.

You can control data retention by passing the [`METRICS_RETENTION`](#metrics-retention) and [`QUERIES_RETENTION`](#queries-retention) environment variables when creating and running the PMM Server container.

## Data Source Name

A database server attribute found on the QAN page. It informs how PMM connects to the selected database.

## Default ports

See [Ports](#ports).

## DSN

See [Data Source Name](#data-source-name)

## External Monitoring Service

A monitoring service which is not provided by PMM directly. It is bound to a running Prometheus exporter. As soon as such an service is added, you can set up the [Metrics Monitor](#metrics-monitor) to display its graphs.

## Grand Total Time

Grand Total Time (percent of grand total time) is the percentage of time that the database server spent running a specific query, compared to the total time it spent running all queries during the selected period of time.

## %GTT

See [Grand Total Time](#grand-total-time)

## Metrics

A series of data which are visualized in PMM.

## Metrics Monitor (MM)

Component of PMM Server that provides a historical view of metrics critical to a MySQL server instance.

## Monitoring service

A special service which collects information from the database instance where [PMM Client](#pmm-client) is installed.

To add a monitoring service, use the **pmm-admin add** command.

!!! seealso "See also"

    Passing parameters to a monitoring service
    : [Passing options to the exporter](pmm-admin.md#pmm-pmm-admin-monitoring-service-pass-parameter)

## Orchestrator

The topology manager for MySQL. By default it is disabled for the [PMM Server](#pmm-server). To enable it, set the [`ORCHESTRATOR_ENABLED`](#orchestrator-enabled).

## PMM

Percona Monitoring and Management

## pmm-admin

A program which changes the configuration of the [PMM Client](#pmm-client). See detailed documentation in the [Managing PMM Client](pmm-admin.md#pmm-admin) section.

## PMM annotation

A feature of PMM Server which adds a special mark to all dashboards and signifies an important event in your application. Annotations are added on the PMM Client by using the **pmm-admin annotate** command.

!!! seealso "See also"

    Grafana Documentation: Annotations
    : <http://docs.grafana.org/reference/annotations/>

## PMM Docker Image

A docker image which enables installing the PMM Server by using **docker**.

!!! seealso "See also"
    Installing PMM Server using Docker
    : [Running PMM Server via Docker](deploy/server/docker.md#run-server-docker)

## PMM Client

Collects MySQL server metrics, general system metrics, and query analytics data for a complete performance overview.

The collected data is sent to [PMM Server](#pmm-server).

For more information, see Overview of [Percona Monitoring and Management Architecture](architecture.md).

## PMM Home Page

The starting page of the PMM portal from which you can have an overview of your environment, open the tools of PMM, and browse to online resources.

On the PMM home page, you can also find the version number and a button to update your PMM Server (see [PMM Version](#pmm-version)).

## PMM Server

Aggregates data collected by PMM Client and presents it in the form of tables, dashboards, and graphs in a web interface.

PMM Server combines the backend API and storage for collected data with a frontend for viewing time-based graphs and performing thorough analysis of your MySQL and MongoDB hosts through a web interface.

Run PMM Server on a host that you will use to access this data.

!!! seealso "See also"
    PMM Architecture
    : [Overview of Percona Monitoring and Management Architecture](architecture.md#pmm-architecture)

## PMM Server Version

If [PMM Server](#pmm-server) is installed via Docker, you can check the current PMM Server version by running **docker exec**:

Run this command as root or by using the **sudo** command

```
$ docker exec -it pmm-server head -1 /srv/update/main.yml
# v1.5.3
```

## PMM user permissions for AWS

When creating a [IAM user](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/CHAP_SettingUp.html#CHAP_SettingUp.IAM) for Amazon RDS DB instance that you intend to monitor in PMM, you need to set all required permissions properly. For this, you may copy the following JSON for your IAM user:

```
{ "Version": "2012-10-17",
  "Statement": [{ "Sid": "Stmt1508404837000",
                  "Effect": "Allow",
                  "Action": [ "rds:DescribeDBInstances",
                              "cloudwatch:GetMetricStatistics",
                              "cloudwatch:ListMetrics"],
                              "Resource": ["*"] },
                 { "Sid": "Stmt1508410723001",
                   "Effect": "Allow",
                   "Action": [ "logs:DescribeLogStreams",
                               "logs:GetLogEvents",
                               "logs:FilterLogEvents" ],
                               "Resource": [ "arn:aws:logs:*:*:log-group:RDSOSMetrics:*" ]}
               ]
}
```

!!! seealso "See also"
    Creating an IAM user
    : [Creating an IAM user](amazon-rds.md#pmm-amazon-rds-iam-user-creating)

## PMM Version

The version of PMM appears at the bottom of the PMM server home page.

![](_images/pmm.home-page.1.png)

To update your PMM Server, click the *Check for Updates Manually* button located next to the version number.

!!! seealso "See also"
    Checking the version of PMM Server
    : [PMM Server Version](#pmm-server-version)

## Ports

The following ports must be open to enable communication between the PMM Server and PMM clients.

42000
: For PMM to collect general system metrics.

42002
: For PMM to collect MySQL server metrics.

42003
: For PMM to collect MongoDB server metrics.

42004
: For PMM to collect ProxySQL server metrics.

42005
: For PMM to collect PostgreSQL server metrics.

Also PMM Server should keep ports 80 or 443 ports open for computers where PMM Client is installed to access the PMM web interface and the QAN agent.

!!! seealso "See also"

    Setting up a firewall on CentOS
    : <https://www.digitalocean.com/community/tutorials/how-to-set-up-a-firewall-using-firewalld-on-centos-7>

    Setting up a firewall on Ubuntu
    : <https://www.digitalocean.com/community/tutorials/how-to-set-up-a-firewall-with-ufw-on-ubuntu-16-04>

## QAN

See [Query Analytics (QAN)](#query-analytics-qan)

## Query Abstract

Query pattern with placeholders. This term appears in [Query Analytics (QAN)](#query-analytics-qan) as an attribute of queries.

## Query Analytics (QAN)

Component of PMM Server that enables you to analyze MySQL query performance over periods of time.

## Query ID

A [query fingerprint](#query-fingerprint) which groups similar queries.

## Query Fingerprint

See [Query Abstract](#query-abstract)

## Query Load

The percentage of time that the MySQL server spent executing a specific query.

## Query Metrics Summary Table

An element of [Query Analytics (QAN)](#query-analytics-qan) which displays the available metrics for the selected query.

## Query Metrics Table

A tool within QAN which lists metrics applicable to the query selected in the query summary table.

## Query Summary Table

A tool within QAN which lists the queries which were run on the selected database server during the [selected time or date range](#selected-time-or-date-range).

## Quick ranges

Predefined time periods which are used by QAN to collect metrics for queries. The following quick ranges are available:

* last hour
* last three hours
* last five hours
* last twelve hours
* last twenty four hours
* last five days

## Selected Time or Date Range

A predefined time period (see [Quick ranges](#quick-ranges)), such as 1 hour, or a range of dates that QAN uses to collects metrics.

## Telemetry

Percona may collect some statistics about the machine where PMM is running.

This statistics includes the following information:

* PMM Server unique ID
* PMM version
* The name and version of the operating system, AMI or virtual appliance
* MySQL version
* Perl version

You may disable telemetry by passing an additional parameter to Docker.

```
$ docker run ... -e DISABLE_TELEMETRY=true ... percona/pmm-server:1
```

## Version

A database server attribute found on the QAN page. it informs the full version of the monitored database server, as well as the product name, revision and release number.
