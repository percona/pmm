# Welcome to Percona Monitoring and Management (PMM)!

We're glad that you would like to become a Percona community member and participate in keeping open source open. [Percona Monitoring and Management (PMM)](https://www.percona.com/software/database-tools/percona-monitoring-and-management) is a open source database monitoring solution. It allows you to monitor your databases, different services (HAProxy, ProxySQL and etc) as well as Nodes, Kubernetes clusters and containers.

## Project repos structure
This project is built from several repositories:

### APIs

* [percona/pmm](https://github.com/percona/pmm)
* [percona-platform/saas](https://github.com/percona-platform/saas)
* [percona-platform/dbaas-api](https://github.com/percona-platform/dbaas-api)

### PMM Server

* [percona/pmm-managed](https://github.com/percona/pmm-managed)
* [percona-platform/dbaas-controller](https://github.com/percona-platform/dbaas-controller)
* [percona/qan-api2](https://github.com/percona/qan-api2)
* [percona/pmm-update](https://github.com/percona/pmm-update)
* [percona/grafana-dashboards](https://github.com/percona/grafana-dashboards)
* [percona-platform/grafana](https://github.com/percona-platform/grafana)

### PMM Client

* [percona/pmm-agent](https://github.com/percona/pmm-agent)
* [percona/pmm-admin](https://github.com/percona/pmm-admin)
* [percona/node_exporter](https://github.com/percona/node_exporter)
* [percona/mysqld_exporter](https://github.com/percona/mysqld_exporter)
* [percona/mongodb_exporter](https://github.com/percona/mongodb_exporter)
* [percona/postgres_exporter](https://github.com/percona/postgres_exporter)
* [percona/proxysql_exporter](https://github.com/percona/proxysql_exporter)
* [percona/rds_exporter](https://github.com/percona/rds_exporter)
* [percona/azure_exporter](https://github.com/percona/azure_metrics_exporter)
* [Percona-Lab/clickhouse_exporter](https://github.com/Percona-Lab/clickhouse_exporter)
* [percona/percona-toolkit](https://github.com/percona/percona-toolkit)

### Building and Packaging

* [percona/pmm-server](https://github.com/percona/pmm-server)
* [Percona-Lab/pmm-submodules](https://github.com/Percona-Lab/pmm-submodules)
* [Percona-Lab/jenkins-pipelines](https://github.com/Percona-Lab/jenkins-pipelines)

### QA, Testing and Documentation
* [percona/pmm-ui-tests](https://github.com/percona/pmm-ui-tests)
* [percona/pmm-qa](https://github.com/percona/pmm-qa)
* [percona/pmm-doc](https://github.com/percona/pmm-doc)

## APIs

See API definitions [here](https://percona-pmm.readme.io/reference/introduction).

## Prerequisites

Before submitting code or documentation contributions, you should first complete the following prerequisites.



### 1. Sign the CLA

Before you can contribute, we kindly ask you to sign our [Contributor License Agreement](https://cla-assistant.percona.com/<linktoCLA>) (CLA). You can do this using your GitHub account and one click.

### 2. Code of Conduct

Please make sure to read and agree to our [Code of Conduct](https://github.com/percona/community/blob/main/content/contribute/coc.md).

## Submitting a Bug

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

## Setup your local development environment
<explanation of local setup>

## Tests

<include section about how to test and how to write tests>

## Submitting a Pull Request

<include information of what you expect from a PR to be successfully merged - code standards, branching, etc>

### Code Reviews

<explain how the project code is reviewed and how to raise questions, if reviewers need to be added etc>

## After your Pull Request is merged

Once your pull request is merged, you are an official Percona Community Contributor. Welcome to the community!

We're looking forward to your contributions and hope to hear from you soon on our [Forums](https://forums.percona.com) and [Discord](https://discord.gg/mQEyGPkNbR).
