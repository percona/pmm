# Percona Monitoring and Management 2.x

[![Build Status](https://travis-ci.com/percona/pmm.svg?branch=PMM-2.0)](https://travis-ci.com/percona/pmm)
[![CLA assistant](https://cla-assistant.percona.com/readme/badge/percona/pmm)](https://cla-assistant.percona.com/percona/pmm)

![PMM](https://www.percona.com/sites/default/files/pmm-logo.png)

See the [PMM 2.x docs](https://www.percona.com/doc/percona-monitoring-and-management/2.x/index.html) for more information.

## Submitting Bug Reports

If you find a bug in Percona Monitoring and Management  or one of the related projects, you should submit a report to that project's [JIRA](https://jira.percona.com) issue tracker.

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

## APIs

See stable API definitions [there](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/percona/pmm/PMM-2.0/api/swagger/swagger.json),
and all API definitions (including technical preview, development and testing APIs)
[there](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/percona/pmm/PMM-2.0/api/swagger/swagger-dev.json).


## Repositories

### APIs

* [percona/pmm](https://github.com/percona/pmm/tree/PMM-2.0)
* [percona-platform/saas](https://github.com/percona-platform/saas)
* [percona-platform/dbaas-api](https://github.com/percona-platform/dbaas-api)

### PMM Server

* [percona/pmm-managed](https://github.com/percona/pmm-managed/tree/PMM-2.0)
* [percona-platform/dbaas-controller](https://github.com/percona-platform/dbaas-controller)
* [percona/qan-api2](https://github.com/percona/qan-api2)
* [percona/pmm-update](https://github.com/percona/pmm-update/tree/PMM-2.0)
* [percona/percona-toolkit](https://github.com/percona/percona-toolkit/tree/3.0)
* [percona/grafana-dashboards](https://github.com/percona/grafana-dashboards/tree/PMM-2.0)
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
* [Percona-Lab/clickhouse_exporter](https://github.com/Percona-Lab/clickhouse_exporter)

### Building and Packaging

* [percona/pmm-server](https://github.com/percona/pmm-server/tree/PMM-2.0)
* [percona/pmm-server-packaging](https://github.com/percona/pmm-server-packaging/tree/PMM-2.0)
* [Percona-Lab/pmm-submodules](https://github.com/Percona-Lab/pmm-submodules/tree/PMM-2.0)
* [Percona-Lab/jenkins-pipelines](https://github.com/Percona-Lab/jenkins-pipelines)
* [Percona-Lab/percona-images](https://github.com/Percona-Lab/percona-images)

### QA, Testing and Documentation

* [percona/pmm-qa](https://github.com/percona/pmm-qa/tree/PMM-2.0)
* [Percona-Lab/pmm-api-tests](https://github.com/Percona-Lab/pmm-api-tests)
* [percona/pmm-doc](https://github.com/percona/pmm-doc)
