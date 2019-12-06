# Contributing notes

## Pre-requirements: 
git, make, curl, go, gcc, docker, docker-compose

Run dependencies `make env-up`
To test
Run `make test`
To run
Run `make run`

## Local setup
Run `make init` to install dependencies.  
Run `make env-up` to set-up environment

Start pmm-managed with

```sh
make run
```

PMM-managed API GRPC server will be available on http://localhost:7771  
PMM-managed API JSON server will be available on http://localhost:7772

## Vendoring

We use [dep](https://github.com/golang/dep) to vendor dependencies.
