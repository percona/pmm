# Host Makefile.

include Makefile.include

TARGET ?= _bash

env-up-ex:                    ## Start experimental devcontainer.
	docker-compose -f devcontainer.docker-compose.yml up -d
	TARGET="run" make env-ex

env-up-ex-rebuild:            ## Start experimental with rebuild devcontainer.
	docker-compose -f devcontainer.docker-compose.yml up --build -d
	TARGET="run" make env-ex

env-down-ex:                  ## Stop experimental devcontainer.
	docker-compose -f devcontainer.docker-compose.yml down

env-ex:  					  ## Enter devcontainer / or run TARGET
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-managed-server-experimental make $(TARGET)

							  ## Start devcontainer.
env-up: env-compose-up env-devcontainer

env-compose-up:
	docker-compose pull
	docker-compose up --detach --renew-anon-volumes --remove-orphans

env-devcontainer:
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-managed-server .devcontainer/setup.py

							  ## Stop devcontainer.
env-down:
	docker-compose down --remove-orphans

env-remove:
	docker-compose down --volumes --remove-orphans

env:                          ## Run `make TARGET` in devcontainer (`make env TARGET=help`); TARGET defaults to bash.
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-managed-server make $(TARGET)
