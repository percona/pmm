# Percona Monitoring and Management

[![CI](https://github.com/percona/pmm/actions/workflows/main.yml/badge.svg)](https://github.com/percona/pmm/actions/workflows/main.yml)
[![CLA assistant](https://cla-assistant.percona.com/readme/badge/percona/pmm)](https://cla-assistant.percona.com/percona/pmm)
[![Code coverage](https://codecov.io/gh/percona/pmm/branch/main/graph/badge.svg)](https://codecov.io/gh/percona/pmm)
[![Go Report Card](https://goreportcard.com/badge/github.com/percona/pmm)](https://goreportcard.com/report/github.com/percona/pmm)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/percona/pmm/badge)](https://scorecard.dev/viewer/?uri=github.com/percona/pmm)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/9702/badge)](https://www.bestpractices.dev/projects/9702)
[![Forum](https://img.shields.io/badge/Forum-join-brightgreen)](https://forums.percona.com/)

![PMM](img/pmm-logo.png)

A **single pane of glass** to easily view and monitor the performance of your MySQL, MongoDB, PostgreSQL, and MariaDB databases.

## Table of Contents

- [Introduction](#introduction)
- [Use Cases](#use-cases)
- [Architecture](#architecture)
- [Installation](#installation)
- [Need Help?](#need-help)
- [How to Get Involved](#how-to-get-involved)
- [Submitting Bug Reports](#submitting-bug-reports)
- [Licensing](#licensing)

## Introduction

[Percona Monitoring and Management (PMM)](https://www.percona.com/software/database-tools/percona-monitoring-and-management) is a best-of-breed open source database monitoring solution. It helps you reduce complexity, optimize performance, and improve the security of your business-critical database environments, no matter where they are located or deployed.

PMM helps users to:
* Reduce Complexity
* Optimize Database Performance
* Improve Data Security

See the [PMM Documentation](https://docs.percona.com/percona-monitoring-and-management/3/index.html) for more information.

## Use Cases

* Monitor your database performance with customizable dashboards and real-time alerting.
* Spot critical performance issues faster, understand the root cause of incidents better and troubleshoot them more efficiently.
* Zoom-in, drill-down database performance from node to single query levels. Perform in-depth troubleshooting and performance optimization.
* Built-in Advisors run regular checks of the databases connected to PMM. The checks identify and alert you of potential security threats, performance degradation, data loss and data corruption.
* Backup and restore databases up to a specific moment with Point-in-Time-Recovery.

## Architecture

Check [PMM documentation](documentation/docs/index.md) for the actual architecture.

![Overall Architecture](documentation/docs/images/C_S_Architecture.jpg "Client Server Architecture")

![PMM Server](https://docs.percona.com/percona-monitoring-and-management/images/PMM-Server-Component-Based-View.jpg 'PMM Server Architecture')

![PMM Client](documentation/docs/images/PMM-Client-Component-Based-View.jpg 'PMM Client Architecture')

## Installation

There are a number of installation methods, please check our [About PMM installation](https://docs.percona.com/percona-monitoring-and-management/3/install-pmm/index.html) documentation page.

In a nutshell:

1. **Download PMM server Docker image:**
   ```bash
   docker pull percona/pmm-server:3
   ```
2. **Create the data volume container:**
   ```bash
   docker volume create pmm-data
   ```
3. **Run PMM Server container:**
   ```bash
   docker run --detach --restart always \
   --publish 443:8443 \
   --volume pmm-data:/srv \
   --name pmm-server \
   percona/pmm-server:3
   ```
4. **Launch the PMM UI:**
   Start a web browser and in the address bar enter the server name or IP address of the PMM server host.
   
   **Note:** The PMM UI is exposed on port 443. You might need to use `https://<PMM_SERVER_IP>` to access it.

   **Default Credentials:**
   - **Username:** admin
   - **Password:** admin

## Need Help?

| **Commercial Support** | **Community Support** |
|:--|:--|
| **Enterprise-grade support** for mission-critical monitoring deployments with Percona Monitoring and Management. <br/><br/>Get expert guidance for complex monitoring scenarios across hybrid environments—from cloud providers to bare metal infrastructures. | Connect with our engineers and community members to troubleshoot issues, share best practices, and discuss monitoring strategies. |
| **[Get Percona Support](https://hubs.ly/Q02_Fs100)** | **[Visit our Forum](https://forums.percona.com/c/percona-monitoring-and-management-pmm)** |

## How to Get Involved

We encourage contributions and are always looking for new members that are as dedicated to serving the community as we are.

If you’re looking for information about how you can contribute, we have [contribution guidelines](CONTRIBUTING.md) across all our repositories in `CONTRIBUTING.md` files. Some of them may just link to the main project’s repository’s contribution guidelines.

We're looking forward to your contributions and hope to hear from you soon on our [Forums](https://forums.percona.com).

## Submitting Bug Reports

If you find a bug in Percona Monitoring and Management or one of the related projects, you should submit a report to that project's [JIRA](https://jira.percona.com) issue tracker. Some related projects also have GitHub Issues enabled, so you could also submit there.

Your first step should be [to search](https://jira.percona.com/issues/?jql=project=PMM) the existing set of open tickets for a similar report. If you find that someone else has already reported your problem, then you can upvote that report to increase its visibility.

If there is no existing report, submit a report following these steps:

1. [Sign in to Percona JIRA](https://jira.percona.com). You will need to create an account if you do not have one.
2. From the top navigation bar, anywhere in Jira, click **Create**. 
3. Select Percona Monitoring and Management (PMM) from the **Project** drop-down menu. 
4. Fill in the fields of **Summary**, **Description**, **Steps To Reproduce**, and **Affects Version** to the best you can. If the bug corresponds to a crash, attach the stack trace from the logs.

An excellent resource is [Elika Etemad's article on filing good bug reports](http://fantasai.inkedblade.net/style/talks/filing-good-bugs/).

As a general rule of thumb, please try to create bug reports that are:

- *Reproducible* - Include steps to reproduce the problem.
- *Specific* - Include as much detail as possible: which version, what environment, etc.
- *Unique* - Do not duplicate existing tickets.

## Licensing

Percona is dedicated to **keeping open source open**. Wherever possible, we strive to include permissive licensing for both our software and documentation. For this project, we are using the [GNU AGPLv3](./LICENSE) license.
