# Host Makefile.

include Makefile.include

env-up: env-compose-up env-devcontainer     ## Start devcontainer.

env-compose-up:
	docker-compose pull
	docker-compose up --detach --renew-anon-volumes --remove-orphans

env-devcontainer:
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm-managed pmm-managed-server .devcontainer/setup.py

env-down:                                   ## Stop devcontainer.
	docker-compose down --remove-orphans

env-remove:
	docker-compose down --volumes --remove-orphans


TARGET ?= _bash

env:                                        ## Run `make TARGET` in devcontainer (`make env TARGET=help`); TARGET defaults to bash.
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm-managed pmm-managed-server make $(TARGET)

env-ci:                                     ## Run `make TARGET` in devcontainer (`make env TARGET=help`); TARGET defaults to bash.
	docker exec -i --workdir=/root/go/src/github.com/percona/pmm-managed pmm-managed-server make $(TARGET)
