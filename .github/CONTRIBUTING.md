# Contributing notes

## Pre-requirements: 
git, make, curl, go, [dep](https://github.com/golang/dep), gcc, docker, docker-compose, pmm-server

## Local setup  
Install exporters: 
* node_exporter
* mysqld_exporter
* rds_exporter
* postgres_exporter
* mongodb_exporter
* proxysql_exporter

Run `make init` to install dependencies.

#### To run pmm-agent
Run [PMM-server](https://github.com/percona/pmm) docker container or [pmm-managed](https://github.com/percona/pmm-managed).  
Run `make setup-dev` to configure pmm-agent
Run `make run` to run pmm-agent
 

## Testing
Run `make env-up` to set-up environment.    
Run `make test` to run tests. 

## Vendoring

We use [dep](https://github.com/golang/dep) to vendor dependencies.
