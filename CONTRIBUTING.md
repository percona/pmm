# Welcome to Percona Monitoring and Management (PMM)!

We'd be glad to welcome you to Percona community which tries to keep the open source open. [Percona Monitoring and Management (PMM)](https://www.percona.com/software/database-tools/percona-monitoring-and-management) is an open source database monitoring solution. It allows you to monitor your databases, different services (HAProxy, ProxySQL and etc) as well as Nodes, Kubernetes clusters and containers. Please check our [Documentation](https://docs.percona.com/percona-monitoring-and-management/details/architecture.html) for the actual architecture.

## Table of contents
1. [Project repos structure](#Project-repos-structure)
2. [API documentation](#API-Reference-Documentation)
3. [Prerequisites](#Prerequisites)
4. [Submitting a bug](#Submitting-a-Bug)
5. [Setup your local development environment](#Setup-your-local-development-environment)
6. [Tests](#Tests)
7. [Feature Build](#Feature-Build)
8. [Code Reviews](#Code-Reviews)

## Project repos structure
This project is built from several repositories:

### APIs

* [percona/pmm](https://github.com/percona/pmm/tree/main/api)
* [percona-platform/saas](https://github.com/percona-platform/saas)

### PMM Server

#### Backends

* [percona/pmm-managed](https://github.com/percona/pmm/tree/main/managed) manages configuration of PMM server components (VictoriaMetrics, Grafana, etc.) and exposes API for that. APIs are used by [pmm-admin](https://github.com/percona/pmm/tree/main/admin)
* [percona/qan-api](https://github.com/percona/pmm/tree/main/qan-api2) query analytics API
* [percona/pmm-update](https://github.com/percona/pmm/tree/main/update) is a tool for updating packages and OS configuration for PMM

#### Frontends

* [percona/grafana-dashboards](https://github.com/percona/grafana-dashboards) PMM dashboards for database monitoring
* [percona-platform/grafana](https://github.com/percona-platform/grafana) user interface for PMM

### PMM Client

* [percona/pmm-agent](https://github.com/percona/pmm/tree/main/agent) monitoring agent for PMM. Runs exporters, and VMAgent that collects data from exporters and send to VictoriaMetrics
* [percona/pmm-admin](https://github.com/percona/pmm/tree/main/admin) admin tool for PMM to manage service that should be monitored by PMM
* [percona/node_exporter](https://github.com/percona/node_exporter) exports machine's metrics
* [percona/mysqld_exporter](https://github.com/percona/mysqld_exporter) exports MySQL server's metrics
* [percona/mongodb_exporter](https://github.com/percona/mongodb_exporter) exports MongoDB server's metrics
* [percona/postgres_exporter](https://github.com/percona/postgres_exporter) exports PostgreSQL server's metrics
* [percona/proxysql_exporter](https://github.com/percona/proxysql_exporter) exports ProxySQL server's metrics
* [percona/rds_exporter](https://github.com/percona/rds_exporter) exports metrics from RDS
* [percona/azure_exporter](https://github.com/percona/azure_metrics_exporter) exports metrics from Azure
* [Percona-Lab/clickhouse_exporter](https://github.com/Percona-Lab/clickhouse_exporter) exports metrics from ClickHouse
* [percona/percona-toolkit](https://github.com/percona/percona-toolkit) is a collection of advanced command-line tools to perform a variety of MySQL and system tasks that are too difficult or complex to perform manually


### Building and Packaging

* [Percona-Lab/pmm-submodules](https://github.com/Percona-Lab/pmm-submodules)
* [Percona-Lab/jenkins-pipelines](https://github.com/Percona-Lab/jenkins-pipelines)

### QA, Testing and Documentation
* [percona/pmm-ui-tests](https://github.com/percona/pmm-ui-tests)
* [percona/pmm-qa](https://github.com/percona/pmm-qa)
* [percona/pmm-doc](https://github.com/percona/pmm-doc)

## API Reference Documentation

You can review the PMM API definition [here](https://percona-pmm.readme.io/).

It is generated from our `.proto` [files](./api/) using a special [OpenAPI v2 tool](https://github.com/grpc-ecosystem/grpc-gateway/tree/master/protoc-gen-openapiv2) and additional API
documentation source files which are located in the `docs/api/` directory. The
content and structure of these is formatted using [Markdown markup
language](https://www.markdownguide.org/) and published on the
[ReadMe.com](https://readme.com/) service.

You can edit the content using your favorite editor (ideally one that supports
previewing MarkDown content, e.g. Microsoft Visual Studio Code).

If you need to create a new file, copy one of the existing `*.md` documents in
the folder to maintain the overall structure and format.

When choosing a file name, make sure that it reflects the topic or the theme you
are talking about and follow the format of `my-topic.md` (no spaces, only
letters and dashes).

Make sure to create a unique `slug` for your file, for example: `slug:
authentication`.

**Header rules**: in Markdown, the level of a header line is defined by the
number of hash signs, example: `###` would be equivalent to an H3 header. Please
avoid using H1 headers. Your first-level header must be H2. The rest of the
headers can by anything between H3 and H6.

Once you're done, please submit your proposed changes via a GitHub pull request
as outlined below.

After the PR has been merged, make sure you can see your contribution live at
https://percona-pmm.readme.io/

## Prerequisites

Before submitting code or documentation contributions, you should first complete the following prerequisites.


### 1. Sign the CLA

Before you can contribute, we kindly ask you to sign our [Contributor License Agreement](https://cla-assistant.percona.com/percona/pmm) (CLA). You can do this using your GitHub account and one click.

### 2. Code of Conduct

Please make sure to read and agree to our [Code of Conduct](https://github.com/percona/community/blob/main/content/contribute/coc.md).

## Submitting a Bug

See [Submitting Bug Reports](README.md#Submitting-Bug-Reports) in [README.md](README.md).


## Setup your local development environment

Since PMM has a lot of components, we will mention only three big parts of it.

### PMM Server

* Clone [pmm repository](https://github.com/percona/pmm)
* Run `make env-up` to start development container. This will be slow on first run, all consequent calls will be order of magnitude faster, because development container will be reused. From time to time it is recommended to perform container rebuild to pull the latest changes, for that run `make env-up-rebuild`.
* To run pmm-managed with a new changes just run `make env TARGET="run-managed"`, it updates `pmm-managed` running in container.

### PMM Client

* Clone [pmm repository](https://github.com/percona/pmm).
* Navigate to the `/agent` folder in the root of the repository.
* Run `make setup-dev` to connect pmm-agent to PMM Server.
  * This command will register local pmm-agent to PMM Server and generate config file `pmm-agent-dev.yaml`
* Once it's connected just use `make run` to run pmm-agent.
* To work correctly pmm-agent needs vmagent and exporters installed on the system.
  * First option is just install pmm-client using this instrucion https://docs.percona.com/percona-monitoring-and-management/setting-up/client/index.html#install. It will install all exporters as well.
  * Another option is to do it manually
    * vmagent and exporters can be installed by building each of them or by downloading the pmm-client tarball from [percona.com](https://www.percona.com/downloads/pmm2/) and copying binaries to the exporters_base directory configured in a `pmm-agent-dev.yaml` file.
    * All paths to exporters binaries are configured in `pmm-agent-dev.yaml`, so they can be changed manually

### Exporters

Exporters by themselves are independent applications, so each of them contains its own README files explaining how to set up a local environment [see PMM Client](#PMM-Client).

### UI

See [Grafana Dashboards Contribution Guide](https://github.com/percona/grafana-dashboards/blob/main/CONTRIBUTING.md).

## Tests

In a PMM we have 3 kind of tests.

### Unit tests

The first one is a Unit testing, so we have unit tests in each repository mentioned above. each of repositories has it's own instruction how to run unit tests.

### API tests

API tests are included into pmm repository and located in [api-tests directory](https://github.com/percona/pmm/tree/main/api-tests). API tests runs against running PMM Server container.

### End to End (E2E) tests

End to End tests are located in [pmm-qa repository](https://github.com/percona/pmm-qa). They includes UI tests and CLI tests.
Please see [readme](https://github.com/percona/pmm-qa#readme) for details on how to run these.

## Submitting a Pull Request

Before proceeding with your first pull request, we highly recommend you to read the following documents:
- [Working with Git and GitHub](docs/process/GIT_AND_GITHUB.md)
- [Tech stack](docs/process/tech_stack.md)
- [Best practices](docs/process/best_practices.md)

As a PR created you are responsible to:
* make sure PR is ready (linted, tested and etc)
* make sure it is reviewed (ask for review, ping reviewers and etc)
* make sure it is merged
  * merge when it is reviewed and tested
  * ask code owners/admins to merge it if merging is blocked for some reason

## Feature Build

PMM is quite complex project, it consists from many different repos descibed above. Feature Build (FB) is a way to get changes all together, build them all together, run tests and get client and server containers.

Please see: [How to create a feature build](https://github.com/Percona-Lab/pmm-submodules/blob/PMM-2.0/README.md#how-to-create-a-feature-build)

### The Goals of the Feature Builds

1. Provide an easy way to test/accept functionality for PO/PM and QA
2. Inform the Developer about Automation Tests results before the code is merged
3. (Future) Let the Developers add/modify e2e tests when they change functionality

### The Rules

1. Start Feature Build for every feature/improvement you are working on.
2. Start PullRequest to percona-lab/pmm-submodules as DRAFT.
3. Change the status of Pull Request from Draft to Open ONLY if your changes must be merged to pmm-submodules.
4. Include a short explanation in the Long Description field of the Feature in PR for feature build and checkboxes to all related Pull Requests. Check other [PRs](https://github.com/Percona-Lab/pmm-submodules/pulls) as examples.
5. After all related PRs in feature build are merged you should:
   a. either close the PR and delete the branch (this is the default option) or
   b. merge the PR to pmm-submodules repository (please note, this rarely needs to be merged, for example infrastructure changes do)

## Code Reviews

There are number of approaches for the code review and ownership: Code Ownership (CODEOWNERS), [github auto review](https://docs.github.com/en/github/setting-up-and-managing-organizations-and-teams/managing-code-review-assignment-for-your-team), PR owner assign ppl that are better fit for the particular code/job.

For more efficient review process we use a mixed approach:
* repos that have CODEOWNERS
  * github will assign reviewers automatically
* repos that don't have CODEOWNERS
  * add reviewers as follows:
      * add `pmm-review-fe` for UI/UX reviews
      * add `pmm-review-exporters` for exporter reviews [see PMM Client](#PMM-Client)
      * add `pmm-review-be` for backend reviews
* if you know exactly who should review your code
  * add them to the review


| Team                 | Description                                                    | Members |
| -------------------- | -------------------------------------------------------------- | ------- |
| pmm-review-fe        | UI reviewers of PRs to [FrontEnd repos](#FrontEnd)             | [FE team](https://github.com/orgs/percona/teams/pmm-review-fe/members)        |
| pmm-review-exporters | exporter reviewers of PRs to [PMM Client](#PMM-Client)         | [Exporters team](https://github.com/orgs/percona/teams/pmm-review-exporters/members) |
| pmm-review-be        | reviewers of backend (Go) PRs                                  | [BE team](https://github.com/orgs/percona/teams/pmm-review-be/members)        |
| PMM Admins           | reviewers that could use admins rights to force merge or change repo settings | [PMM Admin team](https://github.com/orgs/percona/teams/pmm-admins/members)           |


## After your Pull Request is merged

Once your pull request is merged, you are an official Percona Community Contributor. Welcome to the community!

We're looking forward to your contributions and hope to hear from you soon on our [Forums](https://forums.percona.com).
