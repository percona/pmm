# include .env

help:                           ## Display this help message.
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
	    awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

init:                           ## Install prototool.
	# https://github.com/uber/prototool#installation
	curl -L https://github.com/uber/prototool/releases/download/v1.3.0/prototool-$(shell uname -s)-$(shell uname -m) -o ./prototool
	chmod +x ./prototool

install:                        ## Install qan-api binary.
	go install -v ./...

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

ch-dump:                      ## Connect to pmm DB.
	docker exec -ti ch-server clickhouse client -d pmm --query="SELECT * FROM queries FORMAT Native" > queries.native
	#docker exec -ti ch-server clickhouse client -d pmm --query="INSERT INTO queries FORMAT Native" < queries.native

ps-client:
	docker exec -ti ps-server mysql -uroot -psecret

go-run:                         ## Run qan-api with envs.
	@echo "  > Runing with envs..."
	GRPC_VERBOSITY=DEBUG GRPC_TRACE=all go run *.go


go-generate:                    ## Pack ClickHouse migrations into go file.
	@echo "  >  Generating dependency files..."

	go install -v ./vendor/github.com/kevinburke/go-bindata/go-bindata
	go-bindata -pkg migrations -o migrations/bindata.go -prefix migrations/sql migrations/sql

linux-go-build: go-generate
	@echo "  >  Building binary..."
	GOOS=linux go build -o percona-qan-api2 *.go

go-build:
	@echo "  >  Building binary..."
	go build -o percona-qan-api2 *.go

api-version:                    ## Request API version.
	./prototool grpc api/version --address 0.0.0.0:9911 --method version.Version/HandleVersion --data '{"name": "john"}'

test: install                   ## Run tests
	go test -v -p 1 -race ./...

check-license:                  ## Check that all files have the same license header.
	go run .github/check-license.go

check: install check-license    ## Run checkers and linters.
	golangci-lint run

format:                         ## Run `goimports`.
	goimports -local github.com/Percona-Lab/qan-api -l -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")
