# Host Makefile.

.DEFAULT_GOAL := help

-include documentation/Makefile
-include dev/Makefile

PROFILES ?= pmm

env-up: 							## Start devcontainer
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose up -d --wait --wait-timeout 100

env-up-rebuild: env-update-image	## Rebuild and start devcontainer. Useful for custom $PMM_SERVER_IMAGE
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose up --build -d

env-update-image:					## Pull latest dev image
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose pull

env-compose-up: env-update-image  ## Pull the image, then start devcontainer waiting for it to be ready
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose up -d --renew-anon-volumes --remove-orphans --wait --wait-timeout 100

env-devcontainer:     ## Provision devcontainer (run this after `make env-up` or `make env-compose-up`)
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm --user root pmm-server python .devcontainer/setup.py

env-down:							## Stop devcontainer
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose down --remove-orphans

env-remove:           ## Stop devcontainer and remove volumes
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose down --volumes --remove-orphans

TARGET ?= _bash

env:								    ## Run `make TARGET` in devcontainer (`make env TARGET=help`); TARGET defaults to bash
	COMPOSE_PROFILES=$(PROFILES) \
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-server make $(TARGET)

env-root:								## Run `make TARGET` in devcontainer (`make env-root TARGET=help`); TARGET defaults to bash
	COMPOSE_PROFILES=$(PROFILES) \
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm --user root pmm-server make $(TARGET)

rotate-encryption:      ## Rotate encryption key
	go run ./encryption-rotation/main.go
