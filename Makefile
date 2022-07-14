# Host Makefile.

include Makefile.include

env-up: env-compose-up			## Start devcontainer.
	docker-compose up -d

env-up-rebuild: env-compose-up	## Start devcontainer with rebuild.
	docker-compose up --build -d

env-compose-up:					## Pull latest dev image
	docker pull "perconalab/pmm-server:dev-latest"

env-devcontainer:
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-managed-server .devcontainer/setup.py

env-down:						## Stop devcontainer.
	docker-compose down --remove-orphans

env-remove:
	docker-compose down --volumes --remove-orphans

TARGET ?= _bash

env:							## Run `make TARGET` in devcontainer (`make env TARGET=help`); TARGET defaults to bash.
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-managed-server make $(TARGET)
