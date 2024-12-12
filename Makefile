# Host Makefile.

include Makefile.include
-include documentation/Makefile

ifeq ($(PROFILES),)
PROFILES := 'pmm'
endif

env-up: 							## Start devcontainer
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose up -d

env-up-rebuild: env-update-image	## Rebuild and start devcontainer. Useful for custom $PMM_SERVER_IMAGE
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose up --build -d

env-update-image:					## Pull latest dev image
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose pull

env-compose-up: env-update-image
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose up --detach --renew-anon-volumes --remove-orphans

env-devcontainer:
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-server .devcontainer/setup.py

env-down:							## Stop devcontainer
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose down --remove-orphans

env-remove:
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose down --volumes --remove-orphans

TARGET ?= _bash

env:								## Run `make TARGET` in devcontainer (`make env TARGET=help`); TARGET defaults to bash
	COMPOSE_PROFILES=$(PROFILES) \
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-server make $(TARGET)

update-dbaas-catalog: 				## Update the DBaaS catalog from the latest production branch (percona-platform).
	wget https://raw.githubusercontent.com/percona/dbaas-catalog/percona-platform/percona-dbaas-catalog.yaml -O managed/data/crds/olm/percona-dbaas-catalog.yaml
