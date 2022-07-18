# Host Makefile.

include Makefile.include

env-up:										## Start devcontainer.
	docker-compose up -d

env-up-rebuild: env-update-image	## Rebuild and start devcontainer. Useful for custom $PMM_SERVER_IMAGE
	docker-compose up --build -d

env-update-image:					## Pull latest dev image
	docker-compose pull

env-compose-up: env-update-image
	docker-compose up --detach --renew-anon-volumes --remove-orphans

env-devcontainer:
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-managed-server .devcontainer/setup.py

env-down:									## Stop devcontainer.
	docker-compose down --remove-orphans

env-remove:
	docker-compose down --volumes --remove-orphans

TARGET ?= _bash

env:										## Run `make TARGET` in devcontainer (`make env TARGET=help`); TARGET defaults to bash.
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-managed-server make $(TARGET)

ext-pmm-env-up:								## Start devcontainer with pmm-managed running outside.
	docker-compose -f ./docker-compose.external-pmm.yml up -d

ext-pmm-env-up-rebuild: env-update-image	## Start devcontainer with pmm-managed running outside.
	docker-compose -f ./docker-compose.external-pmm.yml up -d --build

ext-pmm-env:								## Enter modular devcontainer.
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-managed-server-external-pmm make $(TARGET)

ext-pmm-env-down:							## Stop modular devcontainer.
	docker-compose -f ./docker-compose.external-pmm.yml down --remove-orphans
