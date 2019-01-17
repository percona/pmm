help:                           ## Display this help message.
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
	    awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

init:                           ## Installs tools to $GOPATH/bin (which is expected to be in $PATH).
	curl https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin

	go install -v ./vendor/gopkg.in/reform.v1/reform
	go install -v ./vendor/github.com/vektra/mockery/cmd/mockery
	go get -u github.com/prometheus/prometheus/cmd/promtool

	go install -v ./vendor/github.com/golang/protobuf/protoc-gen-go
	go install -v ./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	go install -v ./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
	go install -v ./vendor/github.com/go-swagger/go-swagger/cmd/swagger

	go test -v -i ./...
	go test -v -race -i ./...

gen:                            ## Generate files.
	rm -f models/*_reform.go

	go generate ./...

	rm -fr api/*.pb.* api/swagger/*.json api/swagger/client api/swagger/models

	protoc -Iapi -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		api/*.proto --go_out=plugins=grpc:api
	protoc -Iapi -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		api/*.proto --grpc-gateway_out=logtostderr=true,request_context=true,allow_delete_body=true:api
	# protoc -Iapi -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		# api/*.proto --swagger_out=logtostderr=true,allow_delete_body=true:api/swagger

	# swagger mixin api/swagger/*.swagger.json > api/swagger/swagger.json
	# swagger validate api/swagger/swagger.json
	# swagger generate client -f api/swagger/swagger.json -t api/swagger -A pmm-managed

	# go install -v github.com/percona/pmm-managed/api github.com/percona/pmm-managed/api/swagger/client

	cp ./vendor/github.com/percona/pmm/api/inventory/json/inventory.json api/swagger/swagger.json

install:                        ## Install pmm-managed binary.
	go install -v ./...

install-race:                   ## Install pmm-managed binary with race detector.
	go install -v -race ./...

test:                           ## Run tests.
	go test -v -p 1 ./...

test-race:                      ## Run tests with race detector.
	go test -v -p 1 -race ./...

test-cover:                     ## Run tests and collect coverage information.
	go test -v -p 1 -coverprofile=cover.out -covermode=count ./...

check-license:                  ## Check that all files have the same license header.
	go run .github/check-license.go

check: install check-license    ## Run checkers and linters.
	golangci-lint run

format:                         ## Run `goimports`.
	goimports -local github.com/percona/pmm-managed -l -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")

run: install _run               ## Run pmm-managed.

run-race: install-race _run     ## Run pmm-managed with race detector.

_run:
	pmm-managed -swagger=json -debug \
		-agent-mysqld-exporter=mysqld_exporter \
		-agent-postgres-exporter=postgres_exporter \
		-agent-rds-exporter=rds_exporter \
		-agent-rds-exporter-config=testdata/rds_exporter/rds_exporter.yml \
		-prometheus-config=testdata/prometheus/prometheus.yml \
		-db-name=pmm-managed-dev

env-up:                         ## Start development environment.
	docker-compose up --force-recreate --abort-on-container-exit --renew-anon-volumes --remove-orphans

env-down:                       ## Stop development environment.
	docker-compose down --volumes --remove-orphans
