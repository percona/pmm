# Contributing notes

## Pre-requirements: 
git, make, curl, go, gcc, nginx, mkcert

## Local setup
Run `make -C api init` to install dependencies.

#### To run nginx 
Install latest nginx https://www.linuxbabe.com/ubuntu/install-nginx-latest-version-ubuntu-18-04  
Install mkcert: https://github.com/FiloSottile/mkcert
Change directory to `api`    
Run `make cert` to generate certificate  
Run `make serve` to start nginx server    
Swagger UI will be available on http://127.0.0.1:8080/swagger-ui.html.
After this you can run [pmm-managed](http://github.com/percona/pmm-managed/) and [qan-api2](https://github.com/percona/qan-api2/)
and they will be available on https://localhost:8443/.

#### To update api
Make changes in proto files  
Run `make gen` in `api` directory to generate go files and swagger.json


## PMM-Server
PMM-server can be run in docker container or partially.

#### To run in docker
Run `docker run -d -p 80:80 -p 443:443  --name pmm-server perconalab/pmm-server:dev-latest`  
Open `http://localhost/`  

#### To run partially
Pre-requirements: git, make, curl, go, gcc, docker, docker-compose, 
Clone repositories
* this repo
* [pmm-managed](http://github.com/percona/pmm-managed/)
* [qan-api2](https://github.com/percona/qan-api2/)  

Start nginx  
Start pmm-managed and dependencies
Start qan-api2 and dependencies


## Vendoring

We use [dep](https://github.com/golang/dep) to vendor dependencies.
