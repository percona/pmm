include .env

init:                 ## Install prototool.
	# https://github.com/uber/prototool#installation
	curl -L https://github.com/uber/prototool/releases/download/v1.3.0/prototool-$(shell uname -s)-$(shell uname -m) -o ./prototool
	chmod +x ./prototool

# Run ClickHouse, MySQL Server and sysbench containers. Create pmm DB in ClickHouse.
env-up:
	mkdir -p logs
	docker-compose up $(DCFLAGS) ch sysbench-ps
	sleep 60
	docker exec ch-server clickhouse client -h 127.0.0.1 --query="CREATE DATABASE IF NOT EXISTS pmm;"

# Remove docker containers.
env-down:
	docker-compose down --volumes
	rm -rf logs

# Run PMM server, MySQL Server and sysbench containers.
pmm-env-up:
	docker-compose up pmm-server sysbench-ps

deploy:
	# docker exec pmm-server supervisorctl reload
	docker exec pmm-server supervisorctl stop qan-api2
	docker cp percona-qan-api2 pmm-server:/usr/sbin/percona-qan-api2
	docker exec pmm-server supervisorctl start qan-api2

# Connect to pmm DB.
ch-client:
	docker exec -ti ch-server clickhouse client -d pmm

ps-client:
	docker exec -ti ps-server mysql -uroot -psecret

# Run qan-api with envs.
# env $(cat .env | xargs) go run *.go
go-run:
	@echo "  > Runing with envs..."
	GRPC_VERBOSITY=DEBUG GRPC_TRACE=all go run *.go

# Pack ClickHouse migrations into go file.
go-generate:
	@echo "  >  Generating dependency files..."

	go install -v ./vendor/github.com/jteeuwen/go-bindata/go-bindata
	go-bindata -pkg migrations -o migrations/bindata.go -prefix migrations/sql migrations/sql

	go install -v ./vendor/github.com/golang/protobuf/protoc-gen-go \
					./vendor/github.com/mwitkow/go-proto-validators/protoc-gen-govalidators
	./prototool all

# Build binary.
linux-go-build: go-generate
	@echo "  >  Building binary..."
	GOOS=linux go build -o percona-qan-api2 *.go

# Build binary.
go-build: go-generate
	@echo "  >  Building binary..."
	go build -o percona-qan-api2 *.go

# Request API version.
api-version:
	./prototool grpc api/version --address 0.0.0.0:9911 --method version.Version/HandleVersion --data '{"name": "john"}'

# Lint project.
lint:
	golangci-lint run

# Run tests
test:
	go test -v -p 1 -race ./...

format:                         ## Run `goimports`.
	goimports -local github.com/Percona-Lab/qan-api -l -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")
