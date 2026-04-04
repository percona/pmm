default: help

help:                           ## Display this help message
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
	    awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

# `cut` is used to remove the leading `v` from the version
PMM_RELEASE_PATH ?= ../bin
PMM_RELEASE_VERSION ?= $(shell git describe --always --dirty | cut -b2-)
PMM_RELEASE_TIMESTAMP ?= $(shell date '+%s')
PMM_RELEASE_FULLCOMMIT ?= $(shell git rev-parse HEAD)
PMM_RELEASE_BRANCH ?= $(shell git describe --always --contains --all)

PMM_CONTAINER ?= pmm-server

release:                        ## Build qan-api2 release binary
	env CGO_ENABLED=0 go build -v -o $(PMM_RELEASE_PATH)/qan-api2 -ldflags " \
		-X 'github.com/percona/pmm/version.ProjectName=qan-api2' \
		-X 'github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
		-X 'github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
		-X 'github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
		-X 'github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
		-X 'github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)' \
		"

install:                        ## Install qan-api2 binary
	go install -v ./...

install-race:                   ## Install qan-api2 binary with race detector
	go install -v -race ./...

test-env-up:                    ## Start docker containers used for testing
	docker run -d \
		--platform=linux/amd64 \
		--name pmm-clickhouse-test \
		-p 19000:9000 \
		-p 18123:8123 \
		-e CLICKHOUSE_USER=default \
		-e CLICKHOUSE_PASSWORD=clickhouse \
		clickhouse/clickhouse-server:25.3.6.56
	$(MAKE) test-env

test-env:                       ## Create pmm_test DB and load test data
	sleep 10
	docker exec pmm-clickhouse-test clickhouse client --password=clickhouse --query="CREATE DATABASE IF NOT EXISTS pmm_test;"
	go run ./cmd/render-migrations | docker exec -i pmm-clickhouse-test clickhouse client --password=clickhouse -d pmm_test --multiline --multiquery
	cat fixture/metrics.part_*.json | docker exec -i pmm-clickhouse-test clickhouse client --password=clickhouse -d pmm_test --query="INSERT INTO metrics FORMAT JSONEachRow"

test-env-down:                  ## Stop and remove docker containers used for testing
	docker stop pmm-clickhouse-test
	docker rm pmm-clickhouse-test

test:                           ## Run tests
	go test ./...

test-race:                      ## Run tests with race detector
	go test -race ./...

test-cover:                     ## Run tests and collect coverage information
	go test -coverprofile=cover.out -covermode=count ./...

RUN_FLAGS = ## -todo-use-kingpin-for-flags

run: install _run               ## Run qan-api2

run-race: install-race _run     ## Run qan-api2 with race detector

run-race-cover: install-race    ## Run qan-api2 with race detector and collect coverage information
	go test -coverpkg="github.com/percona/pmm/qan-api2/..." \
			-tags maincover \
			-race -c -o bin/qan-api2.test
	bin/qan-api2.test -test.coverprofile=cover.out -test.run=TestMainCover $(RUN_FLAGS)

_run:
	qan-api2 $(RUN_FLAGS)

env-up:                         ## Run ClickHouse, MySQL Server and sysbench containers. Create pmm DB in ClickHouse
	mkdir -p logs
	docker compose up -d $(DCFLAGS) --force-recreate --renew-anon-volumes --remove-orphans ch sysbench-ps
	sleep 60
	docker exec ch-server clickhouse client -h 127.0.0.1 --password=clickhouse --query="CREATE DATABASE IF NOT EXISTS pmm;"

env-down:                       ## Remove docker containers
	docker compose down --volumes
	rm -rf logs

pmm-env-up:                     ## Run PMM server, MySQL Server and sysbench containers
	docker compose up -d --force-recreate --renew-anon-volumes --remove-orphans $(PMM_CONTAINER)
	docker exec $(PMM_CONTAINER) sed -i 's|<!-- <listen_host>0.0.0.0</listen_host> -->|<listen_host>0.0.0.0</listen_host>|g' /etc/clickhouse-server/config.xml
	docker exec $(PMM_CONTAINER) supervisorctl restart clickhouse
	docker exec $(PMM_CONTAINER) supervisorctl stop qan-api2
	docker exec -i $(PMM_CONTAINER) clickhouse client -d pmm --password=clickhouse --query="drop database pmm"
	docker exec -i $(PMM_CONTAINER) clickhouse client -d pmm --password=clickhouse --query="create database pmm"
	docker cp bin/qan-api2 $(PMM_CONTAINER):/usr/sbin/percona-qan-api2
	docker exec $(PMM_CONTAINER) supervisorctl start qan-api2
	cat fixture/metrics.part_*.json | docker exec -i pmm-clickhouse-test clickhouse client -d pmm_test --password=clickhouse --query="INSERT INTO metrics FORMAT JSONEachRow"

deploy:
	GOOS=linux GOARCH=amd64 make release
	docker exec $(PMM_CONTAINER) supervisorctl stop qan-api2
	docker cp $(PMM_RELEASE_PATH)/qan-api2 $(PMM_CONTAINER):/usr/sbin/percona-qan-api2
	docker exec $(PMM_CONTAINER) supervisorctl start qan-api2
	docker exec $(PMM_CONTAINER) supervisorctl status
	docker exec $(PMM_CONTAINER) supervisorctl tail -f qan-api2

## ClickHouse bernchmarking
##
## NOTE: All commands down below are intended to be run from the host system (outside of any container).
## Do not run these targets inside a Docker or PMM container.
##
## Usage:
## 1. Download the PMM Demo backup
##    Run:
##      make download-clickhouse-backup
##
##    The default backup is 2026-01-20.
##    All available backups can be found here:
##      https://github.com/Percona-Lab/pmm-demo-dump/releases/tag/pmm-demo
##
##    To download a different backup, set the Makefile variable BACKUP_NAME
##
## 2. Restore the PMM Demo backup
##    After downloading, the backup will be available in "dev/clickhouse-backups/"
##
##    Example:
##      dev/clickhouse-backups/20260120
##
##    Restore the backup by running:
##      make restore-clickhouse-backup
##
##    You must specify the latest migration applied in the backup.
##    For the default backup (20260120), this is migration 21 (default).
##
##    To restore a different backup, set the Makefile variable BACKUP_NAME and BACKUP_LAST_MIGRATION
##    For other backups, the latest applied migration can be found
##    in the release description:
##      https://github.com/Percona-Lab/pmm-demo-dump/releases/tag/pmm-demo
##
## 3. Run benchmarks
##    To run benchmarks for all main endpoints:
##      make bench
##
##    To run benchmarks for specific endpoints:
##      make bench-filters
##      make bench-report
##      make bench-metrics
##      make bench-example
##
##    Each benchmark runs 10 iterations and reports AVG, MIN, and MAX timings.
##
##    This allows you to compare performance before and after changes
##    and ensure there is no negative performance impact.
##
## 4. Summary
##
##    Run:
##      make download-clickhouse-backup
##      make restore-clickhouse-backup
##      make bench

# Since the ClickHouse backup can come from a different migration state (e.g., created when the latest migration was 21),
# and the current latest migration is 22, we need to ensure that the remaining migrations are applied
# to make the system fully functional after restore.
#
# We also drop the database because, in the case of mismatched migrations, it would not be possible
# to apply them correctly. Therefore, we need to provide the backup name and the last migration
# applied to the backup.

# Default backup name and backup last migration.
BACKUP_NAME ?= 20260120
BACKUP_LAST_MIGRATION ?= 21

# Download the ClickHouse backup, unzip it and prepare the directory to restore.
download-clickhouse-backup: 
	./download_clickhouse_backup.sh $(BACKUP_NAME)

# Default ClickHouse credentials.
CLICKHOUSE_USER ?= default
CLICKHOUSE_PASSWORD ?= clickhouse

# Restore the ClickHouse backup into the pmm-server container and apply differential migrations if needed.
restore-clickhouse-backup: 
	docker exec pmm-server clickhouse-client --user=$(CLICKHOUSE_USER) --password=$(CLICKHOUSE_PASSWORD) --query="DROP DATABASE IF EXISTS pmm";
	docker exec pmm-server clickhouse-client --user=$(CLICKHOUSE_USER) --password=$(CLICKHOUSE_PASSWORD) --query="RESTORE DATABASE pmm FROM Disk('backup', '$(BACKUP_NAME)')";
	docker exec -u root pmm-server go run /root/go/src/github.com/percona/pmm/qan-api2/clickhouse_migrate/main.go --last-migration $(BACKUP_LAST_MIGRATION) --user CLICKHOUSE_USER=$(CLICKHOUSE_USER) --password CLICKHOUSE_PASSWORD=$(CLICKHOUSE_PASSWORD)

#
# Default time periods for benchmarking.
# NOTE: These defaults are aligned with the default demo backup (BACKUP_NAME=20260120).
# PMM_DEMO_BENCH_PERIOD_TO may be in the future relative to the current date and is
# intended for benchmarking with demo data that includes timestamps up to 2026-01-21.
# When using a different backup or real data, override these values so they match
# the actual time range of your data to avoid empty benchmark results.
PMM_DEMO_BENCH_PERIOD_FROM ?= 2025-12-27T00:00:00+01:00
PMM_DEMO_BENCH_PERIOD_TO ?= 2026-01-21T23:59:59+01:00
#

# Run bench for all main endpoints.
bench: bench-filters bench-report bench-metrics bench-example

# Run bench for getFilters.
bench-filters:
	@echo "Running Filters benchmark"
	PMM_DEMO_BENCH_PERIOD_FROM="$(PMM_DEMO_BENCH_PERIOD_FROM)" PMM_DEMO_BENCH_PERIOD_TO="$(PMM_DEMO_BENCH_PERIOD_TO)" \
	go test -bench ^BenchmarkGetFilters$$ -run=^$$ -v -benchtime=10x pmm_demo_benchmark_test.go

# Run bench for getReport.
bench-report:
	@echo "Running Report benchmark"
	PMM_DEMO_BENCH_PERIOD_FROM="$(PMM_DEMO_BENCH_PERIOD_FROM)" PMM_DEMO_BENCH_PERIOD_TO="$(PMM_DEMO_BENCH_PERIOD_TO)" \
	go test -bench ^BenchmarkGetReport$$ -run=^$$ -v -benchtime=10x pmm_demo_benchmark_test.go

# Run bench for getMetrics.
bench-metrics:
	@echo "Running Metrics benchmark"
	PMM_DEMO_BENCH_PERIOD_FROM="$(PMM_DEMO_BENCH_PERIOD_FROM)" PMM_DEMO_BENCH_PERIOD_TO="$(PMM_DEMO_BENCH_PERIOD_TO)" \
	go test -bench ^BenchmarkGetMetrics$$ -run=^$$ -v -benchtime=10x pmm_demo_benchmark_test.go

# Run bench for getExample.
bench-example:
	@echo "Running Example benchmark"
	PMM_DEMO_BENCH_PERIOD_FROM="$(PMM_DEMO_BENCH_PERIOD_FROM)" PMM_DEMO_BENCH_PERIOD_TO="$(PMM_DEMO_BENCH_PERIOD_TO)" \
	go test -bench ^BenchmarkGetExample$$ -run=^$$ -v -benchtime=10x pmm_demo_benchmark_test.go
