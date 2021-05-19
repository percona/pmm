# Contributing notes

## Pre-requirements

git, make, curl, go, nginx

## Local setup

1. Install [dep](https://github.com/golang/dep).
2. Run `make -C api init` to install dependencies.

### To run nginx

1. Install latest nginx.
2. Change directory to `api`.
3. Run `make serve` to start nginx server.
4. Swagger UI will be available on http://127.0.0.1:8080/swagger-ui.html.

### To update api

1. Make changes in proto files.
2. Run `make gen` in `api` directory to generate go files and swagger.json.


## To run PMM-Server in Docker

1. Run `docker run -d -p 80:80 -p 443:443  --name pmm-server perconalab/pmm-server:dev-latest`.
2. Open http://localhost/.
