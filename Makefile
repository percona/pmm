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

# Since a backup can come from a different migration state (e.g., created when the latest migration was 21),
# and the current latest migration is 22, we need to ensure that the remaining migrations are applied
# to make the system fully functional after restore.
#
# We also drop the database because, in the case of mismatched migrations, it would not be possible
# to apply them correctly. Therefore, we need to provide the backup name and the last migration
# applied to the backup.
#
# Default backup name and backup last migration.
BACKUP_NAME ?= 20260120
BACKUP_LAST_MIGRATION ?= 21
#
# Copy the ClickHouse backup into the pmm-server container and apply differential migrations if needed.
restore-backup: 
	docker exec pmm-server clickhouse-client --password=clickhouse --query="DROP DATABASE IF EXISTS pmm";
	docker exec pmm-server clickhouse-client --password=clickhouse --query="RESTORE DATABASE pmm FROM Disk('backup', '$(BACKUP_NAME)')";
	./run_clickhouse_migrations.sh $(BACKUP_LAST_MIGRATION)
