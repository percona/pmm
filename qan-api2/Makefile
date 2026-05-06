# PMM Demo benchmark
PROJECT_NAME  := qan-api2
COVERAGE_MODE := count

include Makefile.mk
include ../go.mk

PMM_CONTAINER ?= pmm-server

release:                        ## Build qan-api2 release binary
	env CGO_ENABLED=0 go build -v $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/qan-api2 .

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
	make test-env

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

RUN_FLAGS =

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
	docker compose up -d $(DCFLAGS) $(ENV_UP_FLAGS) ch sysbench-ps
	# docker compose up $(DCFLAGS) ch sysbench-pstpcc
	sleep 60
	docker exec ch-server clickhouse client -h 127.0.0.1 --password=clickhouse --query="CREATE DATABASE IF NOT EXISTS pmm;"

env-down:                       ## Remove docker containers
	docker compose down $(ENV_DOWN_FLAGS)
	rm -rf logs

# TODO: move to .devcontainer/Makefile
pmm-env-up:                     ## Run PMM server, MySQL Server and sysbench containers
	docker compose up -d $(ENV_UP_FLAGS) $(PMM_CONTAINER)
	docker exec $(PMM_CONTAINER) sed -i 's|<!-- <listen_host>0.0.0.0</listen_host> -->|<listen_host>0.0.0.0</listen_host>|g' /etc/clickhouse-server/config.xml
	docker exec $(PMM_CONTAINER) supervisorctl restart clickhouse
	docker exec $(PMM_CONTAINER) supervisorctl stop qan-api2
	docker exec -i $(PMM_CONTAINER) clickhouse client -d pmm --password=clickhouse --query="drop database pmm"
	docker exec -i $(PMM_CONTAINER) clickhouse client -d pmm --password=clickhouse --query="create database pmm"
	docker cp bin/qan-api2 $(PMM_CONTAINER):/usr/sbin/percona-qan-api2
	docker exec $(PMM_CONTAINER) supervisorctl start qan-api2
	cat fixture/metrics.part_*.json | docker exec -i pmm-clickhouse-test clickhouse client -d pmm_test --password=clickhouse --query="INSERT INTO metrics FORMAT JSONEachRow"

# TODO: move to .devcontainer/Makefile
deploy:
	GOOS=linux GOARCH=amd64 make release
	docker exec $(PMM_CONTAINER) supervisorctl stop qan-api2
	docker cp $(PMM_RELEASE_PATH)/qan-api2 $(PMM_CONTAINER):/usr/sbin/percona-qan-api2
	docker exec $(PMM_CONTAINER) supervisorctl start qan-api2
	docker exec $(PMM_CONTAINER) supervisorctl status
	docker exec $(PMM_CONTAINER) supervisorctl tail -f qan-api2
