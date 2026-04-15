# Host Makefile.

include go.mk
include Makefile.include
-include documentation/Makefile

ifeq ($(PROFILES),)
PROFILES := 'pmm'
endif

env-up:               ## Start devcontainer
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose -f docker-compose.dev.yml up -d --wait --wait-timeout 100

env-down:             ## Stop devcontainer
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose -f docker-compose.dev.yml down --volumes --remove-orphans

env-pull:     ## Pull latest images
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose -f docker-compose.dev.yml pull

TARGET ?= _bash

env:								## Run `make TARGET` in devcontainer (`make env TARGET=help`); TARGET defaults to bash
	COMPOSE_PROFILES=$(PROFILES) \
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-server make $(TARGET)

env-root:								## Run `make TARGET` in devcontainer (`make env-root TARGET=help`); TARGET defaults to bash
	COMPOSE_PROFILES=$(PROFILES) \
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm --user root pmm-server make $(TARGET)

rotate-encryption: 							## Rotate encryption key
	go run ./encryption-rotation/main.go
