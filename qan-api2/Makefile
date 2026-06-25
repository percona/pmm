.DEFAULT_GOAL := help
CURDIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
BIN_DIR := $(CURDIR)/../bin

# PMM Demo benchmark
include Makefile.mk

.PHONY: help release install install-race test-env-up test-env-down _start-clickhouse _stop-clickhouse
.PHONY: test run-race-cover _run env-up env-down pmm-env-up deploy _import-test-data

help: ## Display this help message
	@echo "Please use \`make <target>\`, where <target> is one of the following:"
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9-]+:.*## / {printf "  %-28s%s\n", $$1, $$2}' $(MAKEFILE_LIST)

# `cut` is used to remove the leading `v` from the version
PMM_RELEASE_PATH ?= $(BIN_DIR)
PMM_RELEASE_VERSION ?= $(shell git describe --always --dirty | cut -b2-)
PMM_RELEASE_TIMESTAMP ?= $(shell date '+%s')
PMM_RELEASE_FULLCOMMIT ?= $(shell git rev-parse HEAD)
PMM_RELEASE_BRANCH ?= $(shell git describe --always --contains --all)
QAN_LD_FLAGS = -ldflags " \
			-X 'github.com/percona/pmm/version.ProjectName=qan-api2' \
			-X 'github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
			-X 'github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
			-X 'github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)' \
			"
PMM_CONTAINER ?= pmm-server

release:                        ## Build qan-api2 release binary
	env CGO_ENABLED=0 go build -v $(QAN_LD_FLAGS) -o $(PMM_RELEASE_PATH)/qan-api2

install:                        ## Install qan-api2 binary
	go install -v ./...

install-race:                   ## Install qan-api2 binary with race detector
	go install -v -race ./...

TEST_CONTAINER_NAME := pmm-clickhouse-test
TEST_IMAGE_NAME := clickhouse/clickhouse-server:25.3.6.56

start-clickhouse:
	@if [ "$$(uname)" = "Darwin" ]; then \
		if ! command -v timeout > /dev/null 2>&1; then \
			echo "Error: missing timeout (run 'brew install coreutils')"; exit 1; \
		fi; \
	fi
	@echo "-> Starting Clickhouse container $(TEST_CONTAINER_NAME)"
	@docker run -d \
		--name $(TEST_CONTAINER_NAME) \
		-p 19000:9000 \
		-p 18123:8123 \
		-e CLICKHOUSE_USER=default \
		-e CLICKHOUSE_PASSWORD=clickhouse \
		$(TEST_IMAGE_NAME)
	@echo "Waiting for Clickhouse container to become healthy..."
	@timeout 60 sh -c 'until curl -fsS --max-time 1 http://127.0.0.1:18123/ping > /dev/null 2>&1; do sleep 1; done' || \
	   (echo "Timed out waiting for ClickHouse to become ready" && exit 1)
	@echo "✅ Clickhouse container is ready for use"

stop-clickhouse:
	@echo "-> Terminating Clickhouse container $(TEST_CONTAINER_NAME)"
	docker stop $(TEST_CONTAINER_NAME)
	docker rm $(TEST_CONTAINER_NAME)

import-test-data:				## Create pmm_test DB and load test data
	@echo "-> Importing test data..."
	docker exec $(TEST_CONTAINER_NAME) clickhouse client --password=clickhouse --query="CREATE DATABASE IF NOT EXISTS pmm_test;"
	go run $(CURDIR)/cmd/render-migrations | docker exec -i $(TEST_CONTAINER_NAME) clickhouse client --password=clickhouse -d pmm_test --multiline --multiquery
	cat $(CURDIR)/fixture/metrics.part_*.json | docker exec -i $(TEST_CONTAINER_NAME) clickhouse client --password=clickhouse -d pmm_test --query="INSERT INTO metrics FORMAT JSONEachRow"
	@echo "✅ Test data is ready for use"

test-env-up: start-clickhouse import-test-data	## Start docker containers used for testing and import test data
	@echo "✅ Clickhouse is ready for use on address 127.0.0.1 ports 18123,19000"

test-env-down: stop-clickhouse	## Stop and remove docker containers used for testing
	@echo "✅ Clickhouse container has been cleaned up"

test:                           ## Run tests (run `make test-env-up` beforehand)
	go test -race -count=1 ./...

test-cover:                     ## Run tests and collect coverage information (run `make test-env-up` beforehand)
	go test -count=1 -coverprofile=cover.out -covermode=count ./...

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
	# docker compose up $(DCFLAGS) ch sysbench-pstpcc
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
	cat fixture/metrics.part_*.json | docker exec -i $(TEST_CONTAINER_NAME) clickhouse client -d pmm_test --password=clickhouse --query="INSERT INTO metrics FORMAT JSONEachRow"

deploy:
	GOOS=linux GOARCH=amd64 make release
	docker exec $(PMM_CONTAINER) supervisorctl stop qan-api2
	docker cp $(PMM_RELEASE_PATH)/qan-api2 $(PMM_CONTAINER):/usr/sbin/percona-qan-api2
	docker exec $(PMM_CONTAINER) supervisorctl start qan-api2
	docker exec $(PMM_CONTAINER) supervisorctl status
	docker exec $(PMM_CONTAINER) supervisorctl tail -f qan-api2
