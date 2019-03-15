help:                           ## Display this help message.
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
	    awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

PMM_RELEASE_PATH ?= bin
PMM_RELEASE_VERSION ?= 2.0.0-dev
PMM_RELEASE_TIMESTAMP ?= $(shell date '+%s')
PMM_RELEASE_FULLCOMMIT ?= $(shell git rev-parse HEAD)
PMM_RELEASE_BRANCH ?= $(shell git describe --all --contains --dirty HEAD)

release:                        ## Build qan-api release binary.
	env CGO_ENABLED=0 go build -v -o $(PMM_RELEASE_PATH)/qan-api -ldflags " \
		-X 'github.com/Percona-Lab/qan-api/vendor/github.com/percona/pmm/version.ProjectName=qan-api' \
		-X 'github.com/Percona-Lab/qan-api/vendor/github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
		-X 'github.com/Percona-Lab/qan-api/vendor/github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
		-X 'github.com/Percona-Lab/qan-api/vendor/github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
		-X 'github.com/Percona-Lab/qan-api/vendor/github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
		-X 'github.com/Percona-Lab/qan-api/vendor/github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)' \
		"

init:                           ## Installs tools to $GOPATH/bin (which is expected to be in $PATH).
	curl https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin

	go install -v ./vendor/github.com/kevinburke/go-bindata/go-bindata

gen:                            ## Generate files.
	go-bindata -nometadata -pkg migrations -o migrations/bindata.go -prefix migrations/sql migrations/sql

install:                        ## Install qan-api binary.
	go install -v ./...

install-race:                   ## Install qan-api binary with race detector.
	go install -v -race ./...

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
	goimports -local github.com/Percona-Lab/qan-api -l -w $(FILES)

RUN_FLAGS = -todo-use-kingpin-for-flags

run: install _run               ## Run qan-api.

run-race: install-race _run     ## Run qan-api with race detector.

run-race-cover: install-race    ## Run qan-api with race detector and collect coverage information.
	go test -coverpkg="github.com/Percona-Lab/qan-api/..." \
			-tags maincover \
			-race -c -o bin/qan-api.test
	bin/qan-api.test -test.coverprofile=cover.out -test.run=TestMainCover $(RUN_FLAGS)

_run:
	qan-api $(RUN_FLAGS)

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

deploy:
	# docker exec pmm-server supervisorctl reload
	docker exec pmm-server supervisorctl stop qan-api2
	docker cp percona-qan-api2 pmm-server:/usr/sbin/percona-qan-api2
	docker exec pmm-server supervisorctl start qan-api2

ch-client:                      ## Connect to pmm DB.
	docker exec -ti ch-server clickhouse client -d pmm

ch-dump:                        ## Connect to pmm DB.
	docker exec -ti ch-server clickhouse client -d pmm --query="SELECT * FROM metrics FORMAT Native" > queries.native
	#docker exec -ti ch-server clickhouse client -d pmm --query="INSERT INTO queries FORMAT Native" < queries.native

ps-client:
	docker exec -ti ps-server mysql -uroot -psecret
