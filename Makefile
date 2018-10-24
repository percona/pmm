include .env

# Run docker container with ClickHouse and create pmm DB.
db-server:
	docker run -d --name ch-server -p 127.0.0.1:9000:9000 --ulimit nofile=262144:262144 yandex/clickhouse-server
	docker exec ch-server clickhouse client --query="CREATE DATABASE IF NOT EXISTS pmm;"

# Connect to pmm DB.
db-client:
	docker exec -ti ch-server clickhouse client --database=pmm

# Run qan-api with envs.
# env $(cat .env | xargs) go run *.go
go-run:
	@echo "  > Runing with envs..." 
	go run *.go

# Pack ClickHouse migrations into go file.
go-generate:
	@echo "  >  Generating dependency files..."
	go-bindata -pkg migrations -o migrations/bindata.go -prefix migrations/sql migrations/sql
	protoc api/version/version.proto --go_out=plugins=grpc:.

# Build binary.
go-build: go-generate
	@echo "  >  Building binary..."
	go build -o qan-api *.go

# Request API version.
# require https://github.com/uber/prototool
api-version:
	prototool grpc api/version --address 127.0.0.1:9001 --method version.Version/HandleVersion --data '{"name": "me"}'
