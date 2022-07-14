# Host Makefile.

include Makefile.include

env-up:						## Start devcontainer.
	docker-compose up -d

env-up-rebuild:				## Start devcontainer with rebuild.
	docker pull "perconalab/pmm-server:dev-latest"
	docker-compose up --build -d

env-compose-up:
	docker-compose up --detach --renew-anon-volumes --remove-orphans

env-devcontainer:
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-managed-server .devcontainer/setup.py

env-down:					## Stop devcontainer.
	docker-compose down

env-remove:
	docker-compose down --volumes --remove-orphans

TARGET ?= _bash

env:						## Run `make TARGET` in devcontainer (`make env TARGET=help`); TARGET defaults to bash.
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-managed-server make $(TARGET)
