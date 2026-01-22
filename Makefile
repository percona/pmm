# Host Makefile.

include Makefile.include
-include documentation/Makefile

ifeq ($(PROFILES),)
PROFILES := 'pmm'
endif

env-up: 							## Start devcontainer
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose up -d --wait --wait-timeout 100

env-up-rebuild: env-update-image	## Rebuild and start devcontainer. Useful for custom $PMM_SERVER_IMAGE
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose up --build -d

env-update-image:					## Pull latest dev image
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose pull

env-compose-up: env-update-image
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose up --detach --renew-anon-volumes --remove-orphans --wait --wait-timeout 100

env-devcontainer:
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm --user root pmm-server python .devcontainer/setup.py

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

env-root:								## Run `make TARGET` in devcontainer (`make env-root TARGET=help`); TARGET defaults to bash
	COMPOSE_PROFILES=$(PROFILES) \
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm --user root pmm-server make $(TARGET)

rotate-encryption: 							## Rotate encryption key
	go run ./encryption-rotation/main.go

restore-backup:  ## Copy ClickHouse backup into pmm-server container
# 	docker cp dev/clickhouse-backups/pmm_backup_20260120 pmm-server:/srv/clickhouse/
	docker exec pmm-server clickhouse-client --password=clickhouse --query="DROP DATABASE pmm";
	docker exec pmm-server clickhouse-client --password=clickhouse --query="RESTORE DATABASE pmm FROM Disk('backup', 'pmm_backup_20260120')";
