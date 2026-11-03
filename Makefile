# Host Makefile.

include Makefile.include
-include documentation/Makefile
-include Makefile.local

ifeq ($(PROFILES),)
PROFILES := 'pmm'
endif

env-up: 							## Start devcontainer
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose -f ./docker-compose.dev.yml up -d --wait --wait-timeout 100
	$(MAKE) env-pg-config

env-update-image:			## Pull latest dev image
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose -f ./docker-compose.dev.yml pull

env-compose-up: env-update-image
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose up --detach --renew-anon-volumes --remove-orphans --wait --wait-timeout 100

env-devcontainer:     ## Run `make TARGET` in devcontainer (`make env-devcontainer TARGET=help`); TARGET defaults to bash
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm --user root pmm-server bash .devcontainer/setup.sh

# /srv is initialized by the entrypoint on the first container start, with a stock PostgreSQL config.
# `make env-up` runs this target; run it by hand after re-creating the pmm-data volume or when the
# container was started without `make env-up`.
env-pg-config:        ## Tweak the embedded PostgreSQL for development
	docker exec --workdir=/root/go/src/github.com/percona/pmm --user root pmm-server bash .devcontainer/setup.sh --postgres-only

env-down:             ## Stop devcontainer
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose -f ./docker-compose.dev.yml down --remove-orphans

env-remove:           ## Remove devcontainer and its volumes
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose -f ./docker-compose.dev.yml down --volumes --remove-orphans

TARGET ?= _bash

env:                  ## Run `make TARGET` in devcontainer (`make env TARGET=help`); TARGET defaults to bash
	COMPOSE_PROFILES=$(PROFILES) \
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-server make $(TARGET)

env-root:             ## Run `make TARGET` in devcontainer (`make env-root TARGET=help`); TARGET defaults to bash
	COMPOSE_PROFILES=$(PROFILES) \
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm --user root pmm-server make $(TARGET)

rotate-encryption:    ## Rotate encryption key
	go run ./encryption-rotation/main.go
