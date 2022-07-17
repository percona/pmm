# Host Makefile.

include Makefile.include

env-up: 							## Start devcontainer.
	docker-compose up -d

env-up-rebuild:		## Rebuild and start devcontainer
	docker-compose up --build -d

env-update-image:					## Pull latest dev image
	docker pull "perconalab/pmm-server:dev-latest"

env-compose-up: env-update-image
	docker-compose up --detach --renew-anon-volumes --remove-orphans

env-devcontainer:
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-managed-server .devcontainer/setup.py

env-down:							## Stop devcontainer.
	docker-compose down --remove-orphans

env-remove:
	docker-compose down --volumes --remove-orphans

TARGET ?= _bash

env:								## Run `make TARGET` in devcontainer (`make env TARGET=help`); TARGET defaults to bash.
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-managed-server make $(TARGET)
