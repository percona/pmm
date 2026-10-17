# Contributing notes

**pmm-managed** is a core component of PMM Server. As such, its development and testing are best done inside a specially crafted "devcontainer", which allows the developer to provision dev tools into a regular PMM Server container. For details, see [PMM architecture](https://docs.percona.com/percona-monitoring-and-management/3/reference/index.html).

# Devcontainer setup

1. Install Docker and Docker Compose.

2. Clone this repo and check out the `main` branch.

3. Run `make` to see a list of targets that can be run on the host:

```
$ make
Please use `make <target>` where <target> is one of:
  env-up                    Start devcontainer.
  env-down                  Stop devcontainer.
  env                       Run `make TARGET` in devcontainer (`make env TARGET=help`); TARGET defaults to bash.
  release                   Build pmm-managed release binaries.
  help                      Display this help message.
```

`make env-up` starts a devcontainer with all tools and mounts the source code from the host. You can write code using your IDE of choice as usual, or run an editor inside the devcontainer (see below how you can leverage VSCode). To run make targets inside the devcontainer, use `make env TARGET=target-name`. For example:

```
$ make env TARGET=help
docker exec -it --workdir=/root/go/src/github.com/percona/pmm/managed pmm-server make help
Please use `make <target>` where <target> is one of:
  gen                       Generate files.
  install                   Install pmm-managed binary.
  install-race              Install pmm-managed binary with race detector.
  test                      Run tests.
...
```

To run tests, use `make env TARGET=test`, etc.

Alternatively, it is possible to run `make env` to get inside the devcontainer and run make targets as usual:

```
$ make env
docker exec -it --workdir=/root/go/src/github.com/percona/pmm/managed pmm-server make _bash
/bin/bash
[root@pmm-server pmm-managed]# make test
make[1]: Entering directory `/root/go/src/github.com/percona/pmm/managed'
go test -timeout=30s -p 1 ./...
...
```

`run-managed` target replaces `/usr/sbin/pmm-managed` and restarts pmm-managed with `supervisorctl`. As a result, it will use regular filesystem locations (`/etc/victoriametrics-promscrape.yml`, `/etc/supervisord.d`, etc.) and `pmm-managed` PostgreSQL database. Other locations (inside `testdata`) and `pmm-managed-dev` database are used for unit tests.

# Advanced setup

## Available test environment variables

| Variable                                 | Description                                                                                                         | Default                                  |
|------------------------------------------|---------------------------------------------------------------------------------------------------------------------|------------------------------------------|
| PMM_DEV_ADVISOR_STARLARK_ALLOW_RECURSION | Allows recursive functions in checks scripts                                                                        | false                                    |
| PMM_ADVISORS_CHECKS_DISABLE_START_DELAY  | Disables checks service startup delay                                                                               | false                                    |
| PMM_DEV_TELEMETRY_INTERVAL               | Sets telemetry reporting interval                                                                                   | 24h                                      |
| PMM_DEV_TELEMETRY_DISABLE_SEND           | Disables sending of telemetry data to SaaS. This param doesn't affect telemetry data gathering from the datasources | false                                    |
| PMM_DEV_TELEMETRY_FILE                   | Sets path for telemetry config file                                                                                 |                                          |
| PMM_DEV_TELEMETRY_DISABLE_START_DELAY    | Disable the default telemetry execution start delay, so that telemetry gathering is run immediately upon system     | false                                    |
| PMM_DEV_TELEMETRY_RETRY_BACKOFF          | Sets telemetry reporting retry backoff time                                                                         | 1h                                       |   
| PMM_PERCONA_PLATFORM_ADDRESS             | Sets Percona Platform address                                                                                       | https://check-dev.percona.com            |

## Add instances for monitoring

The `make env-up` command starts PMM Server but doesn't configure any database instances for monitoring. To create a complete development environment, you'll need to set up database instances and connect them using PMM Client components:


1. Clone the pmm-admin [repo](https://github.com/percona/pmm-admin/) and install it by running `make install`.
2. Clone the pmm-agent [repo](https://github.com/percona/pmm-agent).
3.  Run database instances to be monitored. You can either run your own or use the [`docker-compose.yml`](https://github.com/percona/pmm-agent/blob/master/docker-compose.yml) file provided by pmm-agent to run MySQL, PostgreSQL, and MongoDB containers using `make env-up` in the pmm-agent repo (make sure to comment out the `pmm-server` service in the docker-compose file since we are already running pmm-managed in devcontainer).
4. Open another shell session and `cd` into the pmm-agent repo, run `make setup-dev` and `make run` to set up and run pmm-agent and connect it to pmm-managed
5. In another shell, use pmm-admin to add agents to the database instances and start monitoring them using `pmm-admin add mysql --username=root --password=root-password`, `pmm-admin add postgresql --username=pmm-agent --password=pmm-agent-password`, and `pmm-admin add mongodb --username=root --password=root-password`.
6. Once pmm-managed has started monitoring the databases. Log in to the web client in your browser to verify. The number of monitored instances will have increased.

## Working with Advisors

Advisors are automated checks in PMM that analyze monitored environments and provide insights or recommendations. As a contributor, you may need to test, extend, or troubleshoot Advisors while developing inside the PMM Server devcontainer.

To get started:

1. Set up the devcontainer using `make env-up`.
2. Enter the container with `make env`, then run your changes with `make run`.
3. [Add instances for monitoring](#add-instances-for-monitoring) so Advisors have databases to check. 
4. Verify results in PMM under **Advisors > Insights**. Any failed Advisor checks will appear there.
5. [Develop or update Advisors](https://docs.percona.com/percona-monitoring-and-management/3/advisors/develop-advisor-checks.html) as needed.


## Contributing to Advisors

Built-in Advisor checks are located in the `data/checks` folder, one check per YAML file. The `category` field of a check sets the group it is listed under in **Advisors > Catalog**. To add or change a built-in check:

1. Add or edit the check's file in `data/checks`. The file must start with `---`, hold exactly one check, and be named after the check (`<name>.yml`). Don't start the name with `custom_`: that prefix is reserved for checks that users create in the PMM UI.
2. Run `make pi-validate` in the `managed` directory to validate the checks.
3. Submit a pull request.

You can read more about the [check format and the script API in our documentation](https://docs.percona.com/percona-monitoring-and-management/3/advisors/develop-advisor-checks.html). To try out a check before you add its file, create it as a custom check under **Advisors > Catalog** in the PMM UI and test it on a service there.

Changes to built-in checks are visible in **Advisors > Catalog** once the PMM Server runs your build.



## Working with Percona Alerting

Go through the [Percona Alerting documentation](https://docs.percona.com/percona-monitoring-and-management/3/alert/index.html).

### Contributing to Percona Alerting Templates

Alert Templates are located in the `data/templates` folder. If you want to contribute to this section, make changes to the files in this folder and submit a pull request. For details, see [Alert Templates format](https://docs.percona.com/percona-monitoring-and-management/3/alert/alert_rules.html#add-an-alert-rule-based-on-a-template).


# Internals

There are three makefiles: `Makefile` (host), `.devcontainer/Makefile`, and `Makefile.include`. `.devcontainer/Makefile` is mounted on top of `Makefile` inside the devcontainer (see `docker-compose.dev.yml`) to enable `make env TARGET=target-name` usage.

Devcontainer initialization code is located in `.devcontainer/setup.sh`. It provisions several binaries required for code development.

## Code structure

```
.
├── bin - binaries
├── cmd - code for any scripts run by managed
├── data - built-in advisor checks, alerting templates and generated code
├── models - database helpers and types, the database schema can be found in models/database.go file
├── services - contains all the APIs for interacting with services like checks service, victoriametrics, etc
├── testdata - dummy data files used in unit tests
├── utils - utilities
```

# How to make a pull request (PR)

- If the changes require multiple PRs spanning multiple repos, make sure to keep the branch names the same.
- If the PR requires any API changes, make sure to contribute to the API docs (/documentation/api).
- If the PR changes any of `deps.go` files, make sure to run `make gen` to generate mock clients.

Before making a PR, please run these commands locally:
- `make env TARGET=check-all` to run all checkers and linters.
- `make env TARGET=test-race` to run tests.
- For help, post on the [PMM 3.x Forums](https://forums.percona.com/c/percona-monitoring-and-management-pmm/pmm-3/)

## VSCode

VSCode provides first-class support for devcontainers. See:

- https://code.visualstudio.com/docs/remote/remote-overview
- https://code.visualstudio.com/docs/remote/containers
