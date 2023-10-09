# Contributing notes

## Pre-requirements: 
git, make, curl, go, [dep](https://github.com/golang/dep)

## Local setup  
Run `make init` to install dependencies.

#### To run qan-api2 
Run `make env-up` to set-up environment.
Run `make run` to start qan-api2

## Run as part of pmm-server docker container
Start PMM-server docker container as it mentioned in [pmm](https://github.com/percona/pmm) repository  
Run `PMM_CONTAINER=pmm-server make release deploy` to deploy local qan-api2 into pmm-server container
where PMM_CONTAINER is a name of PMM-Server container.

## Testing
Run `make test-env-up` to set-up environment for tests
Run `make test` to run tests. 

## Vendoring

We use [dep](https://github.com/golang/dep) to vendor dependencies.
