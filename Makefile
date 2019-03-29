help:                           ## Display this help message.
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
	    awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

PMM_RELEASE_PATH ?= bin
PMM_RELEASE_VERSION ?= 2.0.0-dev
PMM_RELEASE_TIMESTAMP ?= $(shell date '+%s')
PMM_RELEASE_FULLCOMMIT ?= $(shell git rev-parse HEAD)
PMM_RELEASE_BRANCH ?= $(shell git describe --all --contains --dirty HEAD)

release:                        ## Build qan-api2 release binary.
	env CGO_ENABLED=0 go build -v -o $(PMM_RELEASE_PATH)/qan-api2 -ldflags " \
		-X 'github.com/percona/qan-api2/vendor/github.com/percona/pmm/version.ProjectName=qan-api2' \
		-X 'github.com/percona/qan-api2/vendor/github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
		-X 'github.com/percona/qan-api2/vendor/github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
		-X 'github.com/percona/qan-api2/vendor/github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
		-X 'github.com/percona/qan-api2/vendor/github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
		-X 'github.com/percona/qan-api2/vendor/github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)' \
		"

init:                           ## Installs tools to $GOPATH/bin (which is expected to be in $PATH).
	curl https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin

	go install -v ./vendor/github.com/kevinburke/go-bindata/go-bindata

gen:                            ## Generate files.
	go-bindata -nometadata -pkg migrations -o migrations/bindata.go -prefix migrations/sql migrations/sql

install:                        ## Install qan-api2 binary.
	go install -v ./...

install-race:                   ## Install qan-api2 binary with race detector.
	go install -v -race ./...

test-env-up:
	docker run -d --name pmm-clickhouse-test -p19000:9000 yandex/clickhouse-server:19.1.10
	sleep 10s
	docker exec pmm-clickhouse-test clickhouse client --query="CREATE DATABASE IF NOT EXISTS pmm_test;"
	cat migrations/sql/*.up.sql | docker exec -i pmm-clickhouse-test clickhouse client -d pmm_test --multiline --multiquery
	cat fixture/metrics.csv | docker exec -i pmm-clickhouse-test clickhouse client -d pmm_test --query="INSERT INTO metrics FORMAT CSV"

test:                           ## Run tests.
	go test -v ./...

test-race:                      ## Run tests with race detector.
	go test -v -race ./...

test-cover:                     ## Run tests and collect coverage information.
	go test -v -coverprofile=cover.out -covermode=count ./...

check-license:                  ## Check that all files have the same license header.
	go run .github/check-license.go

check: install check-license    ## Run checkers and linters.
	golangci-lint run

FILES = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

format:                         ## Format source code.
	gofmt -w -s $(FILES)
	goimports -local github.com/percona/qan-api2 -l -w $(FILES)

RUN_FLAGS = -todo-use-kingpin-for-flags

run: install _run               ## Run qan-api2.

run-race: install-race _run     ## Run qan-api2 with race detector.

run-race-cover: install-race    ## Run qan-api2 with race detector and collect coverage information.
	go test -coverpkg="github.com/percona/qan-api2/..." \
			-tags maincover \
			-race -c -o bin/qan-api2.test
	bin/qan-api2.test -test.coverprofile=cover.out -test.run=TestMainCover $(RUN_FLAGS)

_run:
	qan-api2 $(RUN_FLAGS)

env-up:                         ## Run ClickHouse, MySQL Server and sysbench containers. Create pmm DB in ClickHouse.
	mkdir -p logs
	docker-compose up $(DCFLAGS) ch sysbench-ps
	#docker-compose up $(DCFLAGS) ch sysbench-pstpcc
	sleep 60
	docker exec ch-server clickhouse client -h 127.0.0.1 --query="CREATE DATABASE IF NOT EXISTS pmm;"

env-down:                       ## Remove docker containers.
	docker-compose down --volumes
	rm -rf logs

pmm-env-up:                     ## Run PMM server, MySQL Server and sysbench containers.
	docker-compose up pmm-server sysbench-ps
