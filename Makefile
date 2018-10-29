all: test

# installs tools to $GOBIN (or $GOPATH/bin) which is expected to be in $PATH
init:
	go install -v ./vendor/gopkg.in/reform.v1/reform
	go install -v ./vendor/github.com/vektra/mockery/cmd/mockery
	go get -u github.com/prometheus/prometheus/cmd/promtool

	go install -v ./vendor/github.com/golang/protobuf/protoc-gen-go
	go install -v ./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	go install -v ./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
	go install -v ./vendor/github.com/go-swagger/go-swagger/cmd/swagger

	go get -u github.com/AlekSi/gocoverutil

	curl https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh

check-license:
	go run .github/check-license.go

install: check-license
	go install -v ./...
	go test -v -i ./...

install-race: check-license
	go install -v -race ./...
	go test -v -race -i ./...

test: install
	go test -v -p 1 ./...

test-race: install-race
	go test -v -p 1 -race ./...

cover: install
	gocoverutil -ignore=github.com/percona/pmm-managed/api/... test -v -p 1 ./...

check: install
	./bin/golangci-lint run

run: install _run

run-race: install-race _run

_run:
	pmm-managed -swagger=rest -debug \
		-agent-mysqld-exporter=mysqld_exporter \
		-agent-postgres-exporter=postgres_exporter \
		-agent-rds-exporter=rds_exporter \
		-agent-rds-exporter-config=testdata/rds_exporter/rds_exporter.yml \
		-prometheus-config=testdata/prometheus/prometheus.yml \
		-db-name=pmm-managed-dev

gen:
	rm -f models/*_reform.go

	go generate ./...

	rm -fr api/*.pb.* api/swagger/*.json api/swagger/client api/swagger/models

	protoc -Iapi -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		api/*.proto --go_out=plugins=grpc:api
	protoc -Iapi -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		api/*.proto --grpc-gateway_out=logtostderr=true,request_context=true,allow_delete_body=true:api
	protoc -Iapi -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		api/*.proto --swagger_out=logtostderr=true,allow_delete_body=true:api/swagger

	swagger mixin api/swagger/*.swagger.json > api/swagger/swagger.json
	swagger validate api/swagger/swagger.json
	swagger generate client -f api/swagger/swagger.json -t api/swagger -A pmm-managed

	go install -v github.com/percona/pmm-managed/api github.com/percona/pmm-managed/api/swagger/client

up:
	docker-compose up --force-recreate --abort-on-container-exit --renew-anon-volumes --remove-orphans

down:
	docker-compose down --volumes --remove-orphans
