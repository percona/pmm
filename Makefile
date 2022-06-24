# Host Makefile.

include Makefile.include

env-up: 							## Start devcontainer.
	docker-compose up -d

env-up-rebuild: env-update-image	## Rebuild and start devcontainer. Useful for custom $PMM_SERVER_IMAGE
	docker-compose up --build -d

env-update-image:					## Pull latest dev image
	docker-compose pull

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

# Modular PMM tasks

env-up-modular: env-compose-up-modular env-devcontainer-modular     ## Start modular devcontainer.

env-compose-up-modular:
	docker-compose -f docker-compose.modular.yml pull
	docker-compose -f docker-compose.modular.yml up --detach --renew-anon-volumes --remove-orphans

env-devcontainer-modular:
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-managed-server-modular .devcontainer/setup.py

env-modular:                                ## Enter modular devcontainer.
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-managed-server-modular make $(TARGET)

env-down-modular:                           ## Stop modular devcontainer.
	docker-compose -f docker-compose.modular.yml down --remove-orphans

env-remove-modular:
	docker-compose -f docker-compose.modular.yml down --volumes --remove-orphans
