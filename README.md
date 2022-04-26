# Percona Monitoring and Management 2.x
[![build](https://github.com/percona/pmm/actions/workflows/ci.yml/badge.svg)](https://github.com/percona/pmm/actions/workflows/ci.yml)
[![CLA assistant](https://cla-assistant.percona.com/readme/badge/percona/pmm)](https://cla-assistant.percona.com/percona/pmm)

![PMM](https://www.percona.com/sites/default/files/pmm-logo.png)


## Percona Monitoring and Management

A 'single pane of glass' to easily view and monitor the performance of your MySQL, MongoDB, PostgreSQL, and MariaDB databases.

[Percona Monitoring and Management (PMM)](https://www.percona.com/software/database-tools/percona-monitoring-and-management) is a best-of-breed open source database monitoring solution. It helps you reduce complexity, optimize performance, and improve the security of your business-critical database environments, no matter where they are located or deployed.
PMM helps users to:
* Reduce Complexity
* Optimize Database Performance
* Improve Data Security


See the [PMM Documentation](https://www.percona.com/doc/percona-monitoring-and-management/2.x/index.html) for more information.

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

See stable API definitions [there](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/percona/pmm/main/api/swagger/swagger.json),
and all API definitions (including technical preview, development and testing APIs)
[there](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/percona/pmm/main/api/swagger/swagger-dev.json).


## Repositories

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
