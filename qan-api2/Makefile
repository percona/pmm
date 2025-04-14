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

test-env-up:					          ## Start docker containers used for testing
	docker run -d --platform=linux/amd64 --name pmm-clickhouse-test -p19000:9000 -p18123:8123 clickhouse/clickhouse-server:23.8.2.7
	make test-env

test-env-up-apple:				      ## Start docker containers used for testing on Apple chips
	docker run -d --platform=linux/arm64 --name pmm-clickhouse-test -p19000:9000 -p18123:8123 clickhouse/clickhouse-server:23.8.2.7
	make test-env

test-env:                       ## Create pmm_test DB and load test data
	sleep 10
	docker exec pmm-clickhouse-test clickhouse client --query="CREATE DATABASE IF NOT EXISTS pmm_test;"
	cat migrations/sql/*.up.sql | docker exec -i pmm-clickhouse-test clickhouse client -d pmm_test --multiline --multiquery
	cat fixture/metrics.part_*.json | docker exec -i pmm-clickhouse-test clickhouse client -d pmm_test --query="INSERT INTO metrics FORMAT JSONEachRow"

test-env-down:					        ## Stop and remove docker containers used for testing
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
	# docker compose up $(DCFLAGS) ch sysbench-pstpcc
	sleep 60
	docker exec ch-server clickhouse client -h 127.0.0.1 --query="CREATE DATABASE IF NOT EXISTS pmm;"

env-down:                       ## Remove docker containers
	docker compose down --volumes
	rm -rf logs

pmm-env-up:                     ## Run PMM server, MySQL Server and sysbench containers
	docker compose up -d --force-recreate --renew-anon-volumes --remove-orphans $(PMM_CONTAINER)
	docker exec $(PMM_CONTAINER) sed -i 's|<!-- <listen_host>0.0.0.0</listen_host> -->|<listen_host>0.0.0.0</listen_host>|g' /etc/clickhouse-server/config.xml
	docker exec $(PMM_CONTAINER) supervisorctl restart clickhouse
	docker exec $(PMM_CONTAINER) supervisorctl stop qan-api2
	docker exec -i $(PMM_CONTAINER) clickhouse client -d pmm --query="drop database pmm"
	docker exec -i $(PMM_CONTAINER) clickhouse client -d pmm --query="create database pmm"
	docker cp bin/qan-api2 $(PMM_CONTAINER):/usr/sbin/percona-qan-api2
	docker exec $(PMM_CONTAINER) supervisorctl start qan-api2
	cat fixture/metrics.part_*.json | docker exec -i pmm-clickhouse-test clickhouse client -d pmm_test --query="INSERT INTO metrics FORMAT JSONEachRow"

deploy:
	GOOS=linux GOARCH=amd64 make release
	docker exec $(PMM_CONTAINER) supervisorctl stop qan-api2
	docker cp $(PMM_RELEASE_PATH)/qan-api2 $(PMM_CONTAINER):/usr/sbin/percona-qan-api2
	docker exec $(PMM_CONTAINER) supervisorctl start qan-api2
	docker exec $(PMM_CONTAINER) supervisorctl status
	docker exec $(PMM_CONTAINER) supervisorctl tail -f qan-api2

