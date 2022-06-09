# Percona Monitoring and Management 2.x
[![Build](https://github.com/percona/pmm/actions/workflows/ci.yml/badge.svg)](https://github.com/percona/pmm/actions/workflows/ci.yml)
[![CLA assistant](https://cla-assistant.percona.com/readme/badge/percona/pmm)](https://cla-assistant.percona.com/percona/pmm)
[![Code coverage](https://codecov.io/gh/percona/pmm/branch/main/graph/badge.svg)](https://codecov.io/gh/percona/pmm)
[![Go Report Card](https://goreportcard.com/badge/github.com/percona/pmm)](https://goreportcard.com/report/github.com/percona/pmm)

![PMM](https://www.percona.com/sites/default/files/pmm-logo.png)

## Percona Monitoring and Management

A 'single pane of glass' to easily view and monitor the performance of your MySQL, MongoDB, PostgreSQL, and MariaDB databases.

[Percona Monitoring and Management (PMM)](https://www.percona.com/software/database-tools/percona-monitoring-and-management) is a best-of-breed open source database monitoring solution. It helps you reduce complexity, optimize performance, and improve the security of your business-critical database environments, no matter where they are located or deployed.
PMM helps users to:
* Reduce Complexity
* Optimize Database Performance
* Improve Data Security


See the [PMM Documentation](https://www.percona.com/doc/percona-monitoring-and-management/2.x/index.html) for more information.

## Use Cases

* Monitor your database performance with customizable dashboards and real-time alerting.
* Spot critical performance issues faster, understand the root cause of incidents better and troubleshoot them more efficiently.
* Zoom-in, drill-down database performance from node to single query levels. Perform in-depth troubleshooting and performance optimization.
* Built-in Advisors run regular checks of the databases connected to PMM. The checks identify and alert you of potential security threats, performance degradation, data loss and data corruption.
* DBaaS: Create and configure database clusters no matter where the infrastructure is deployed.
* Backup and restore databases up to a specific moment with Point-in-Time-Recovery.

## Architecture

Please check our [Documentation](https://docs.percona.com/percona-monitoring-and-management/details/architecture.html) for the actual architecture.

![Overal Architecture](https://docs.percona.com/percona-monitoring-and-management/_images/C_S_Architecture.jpg "Client Server Architecture")


![PMM Conponents](https://docs.percona.com/percona-monitoring-and-management/_images/PMM_Architecture_Client_Server.jpg "PMM Server Architecture")


## Installation

There are numbers of installation methods, please check our [Setting Up](https://docs.percona.com/percona-monitoring-and-management/setting-up/index.html) documentation page.

But in a nutshell:
```bash
$ docker pull percona/pmm-server:2

$ docker create --volume /srv \
--name pmm-data \
percona/pmm-server:2 /bin/true

$ docker run --detach --restart always \
--publish 443:443 \
--volumes-from pmm-data \
--name pmm-server \
percona/pmm-server:2

```

## Submitting Bug Reports

If you find a bug in Percona Monitoring and Management  or one of the related projects, you should submit a report to that project's [JIRA](https://jira.percona.com) issue tracker. Some of related project also have GitHub Issues enabled, so you also could submit there.

Your first step should be [to search](https://jira.percona.com/issues/?jql=project=PMM) the existing set of open tickets for a similar report. If you find that someone else has already reported your problem, then you can upvote that report to increase its visibility.

If there is no existing report, submit a report following these steps:

1. [Sign in to Percona JIRA.](https://jira.percona.com/login.jsp) You will need to create an account if you do not have one.
2. [Go to the Create Issue screen and select the relevant project.](https://jira.percona.com/secure/CreateIssueDetails!init.jspa?pid=11600&issuetype=1&priority=3)
3. Fill in the fields of Summary, Description, Steps To Reproduce, and Affects Version to the best you can. If the bug corresponds to a crash, attach the stack trace from the logs.

An excellent resource is [Elika Etemad's article on filing good bug reports.](http://fantasai.inkedblade.net/style/talks/filing-good-bugs/).

As a general rule of thumb, please try to create bug reports that are:

- *Reproducible.* Include steps to reproduce the problem.
- *Specific.* Include as much detail as possible: which version, what environment, etc.
- *Unique.* Do not duplicate existing tickets.


## Licensing

Percona is dedicated to **keeping open source open**. Wherever possible, we strive to include permissive licensing for both our software and documentation. For this project, we are using the [GNU AGPLv3](https://github.com/percona/pmm/blob/main/LICENSE) license.

## How to get involved

First please check [contribution guidelines](CONTRIBUTING.md) for this repo to get started.

We encourage contributions and are always looking for new members that are as dedicated to serving the community as we are.

If you’re looking for information about how you can contribute, we have [contribution guidelines](CONTRIBUTING.md) across all our repositories in `CONTRIBUTING.md` files. Some of them may just link to the main project’s repository’s contribution guidelines.

We're looking forward to your contributions and hope to hear from you soon on our [Forums](https://forums.percona.com) and [Discord](https://discord.gg/mQEyGPkNbR).
