# Welcome to PMM!

We would be happy to see you become a Percona community member and participate in keeping open source open.

## Prerequisites

Before submitting code or documentation contributions, you should first complete the following prerequisites.

### 1. Sign the CLA

Before you can contribute, we kindly ask you to sign our [Contributor License Agreement](https://cla-assistant.percona.com/percona/pmm) (CLA). You can do this using your GitHub account and one click.

### 2. Code of Conduct

Please make sure to read and agree to our [Code of Conduct](https://github.com/percona/pmm/blob/main/code-of-conduct.md).

## Submitting Bug Reports

If you find a bug in Percona Monitoring and Management or one of the related projects, you can submit a bug report to that project's [JIRA](https://jira.percona.com) issue tracker.

Your first step should be [to search](https://jira.percona.com/issues/?jql=project=PMM) for existing ticket on the same subject. If you find that someone else has already reported your problem, then you can upvote that bug report to attract our attention to it.

If there is no existing bug report, you can submit one following these steps:

1. [Sign in to Percona JIRA.](https://jira.percona.com/login.jsp) You will need to create an account if you do not already have one.
2. [Go to the Create Issue screen and select the relevant project.](https://jira.percona.com/secure/CreateIssueDetails!init.jspa?pid=11600&issuetype=1&priority=3)
3. Fill in the fields of *Summary*, *Description*, *Steps To Reproduce*, and *Affects Version* the best you can. If the bug corresponds to a crash, attach the stack trace from the logs.

An excellent resource is [Elika Etemad's article on filing good bug reports.](http://fantasai.inkedblade.net/style/talks/filing-good-bugs/).

As a general rule of thumb, please try to create bug reports that are:

- *Reproducible.* Include steps to reproduce the problem.
- *Specific.* Include as much detail as possible: which version, what environment, etc.
- *Unique.* Do not duplicate existing tickets.

# Contributing notes

## Pre-requisites

This project relies on the following dependencies:

- git, make, curl, go, nginx

## Local setup

1. Run `make -C api init` to install dependencies.

### To run nginx

1. Install latest nginx.
2. Change directory to `api`.
3. Run `make serve` to start nginx server.
4. Swagger UI will be available on http://127.0.0.1:8080/swagger-ui.html.

### To update API

1. Make changes in proto files.
2. Run `make gen` in `api` directory to generate go files and swagger.json.

## Run PMM-Server in Docker

1. Run `docker run -d -p 80:8080 -p 443:8443  --name pmm-server public.ecr.aws/e7j3v3n0/pmm-server:3-dev-latest`.
2. Open http://localhost/.

Please note, the use of port 80 is discouraged and should be avoided. For optimal security, use port 443 along with a valid SSL certificate.
