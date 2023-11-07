# pmm-agent

pmm-agent for PMM 2.x.

# Contributing notes

## Pre-requisites:
git, make, curl, go, gcc, docker, docker-compose, pmm-server

## Local setup
Install one or more exporters:
* node_exporter
* mysqld_exporter
* rds_exporter
* postgres_exporter
* mongodb_exporter
* proxysql_exporter

#### To run pmm-agent
- Run [pmm-server docker container](https://hub.docker.com/r/percona/pmm-server) or [pmm-managed](https://github.com/percona/pmm-managed).  
- Run `make setup-dev` to configure pmm-agent
- Run `make run` to run pmm-agent


## Testing
Run `make env-up` to set-up environment.    
Run `make test` to run tests.

## Code style
Before making PR, please run `make check-all` locally to run all checkers and linters.
