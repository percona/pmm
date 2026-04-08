# Development Makefile
.PHONY: default help init release gen test clean all
.DEFAULT_GOAL := help

-include documentation/Makefile
-include .devcontainer/Makefile

default: help

help:                 ## Display this help message
	@echo "Please use \`make <target>\`, where <target> is one of the following:"
	@grep -h '^[a-zA-Z]' $(MAKEFILE_LIST) | awk -F ':.*## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}' | sort
	@echo
	@echo For developers: check docker-compose.dev.yml to see which environment variables are available.

init:                 ## Install tools
	rm -rf bin/*
	cd tools && go generate -x -tags=tools

	# Install golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./bin v2.6.2 # Version should match specified in CI


PROFILES ?= pmm
COMPOSE_FILE ?= docker-compose.dev.yml

env-up:               ## Start devcontainer
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose -f $(COMPOSE_FILE) up -d --wait --wait-timeout 100

env-up-rebuild: env-update-image	## Rebuild and start devcontainer. Useful for custom $PMM_SERVER_IMAGE
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose -f $(COMPOSE_FILE) up --build -d

env-update-image:     ## Pull latest dev image
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose -f $(COMPOSE_FILE) pull

env-compose-up: env-update-image  ## Pull the image, then start devcontainer waiting for it to be ready
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose -f $(COMPOSE_FILE) up -d --renew-anon-volumes --remove-orphans --wait --wait-timeout 100

env-devcontainer:     ## Provision devcontainer (run this after `make env-up` or `make env-compose-up`)
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm --user root pmm-server python .devcontainer/setup.py

env-down:             ## Stop devcontainer
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose -f $(COMPOSE_FILE) down --remove-orphans

env-remove:           ## Stop devcontainer and remove volumes
	COMPOSE_PROFILES=$(PROFILES) \
	docker compose -f $(COMPOSE_FILE) down --volumes --remove-orphans

TARGET ?= _bash

env:									## Run `make TARGET` in devcontainer (`make env TARGET=help`); TARGET defaults to bash
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm pmm-server make $(TARGET)

env-root:             ## Run `make TARGET` in devcontainer (`make env-root TARGET=help`); TARGET defaults to bash
	docker exec -it --workdir=/root/go/src/github.com/percona/pmm --user root pmm-server make $(TARGET)

rotate-encryption:    ## Rotate encryption key
	go run ./encryption-rotation/main.go

release:              ## Build release versions of all components
	$(MAKE) -C agent release
	$(MAKE) -C admin release
	$(MAKE) -C managed release
	$(MAKE) -C qan-api2 release

gen: clean            ## Generate files
	make -C api gen
	make -C api clean-swagger

	make -C agent gen
	make -C admin gen
	make -C managed gen

	make gen-mocks      ## Generate mocks

	make format
	make format ## TODO: One formatting run is not enough, figure out why.
	go install -v ./...

clean:                ## Remove generated files
	make -C api clean

gen-mocks:						## Generate mocks for API
	find . -name mock_*.go -delete
	./bin/mockery --config .mockery.yaml

test-common:          ## Run tests from API (and other shared) packages only (i.e it ignores directories that are explicitly listed)
	go test $(shell go list ./... | grep -v -e admin -e agent -e managed -e api-tests -e qan-api2 -e update)

api-test:             ## Run API tests on dev env.
	go test -count=1 -race -p 1 -v ./api-tests/... -pmm.server-insecure-tls

GOLANG_CI_LINT_RUN_OPTS ?=
check:                ## Run required checks and linters
	bin/buf lint -v api
	LOG_LEVEL=error bin/golangci-lint run -c=.golangci.yml --new-from-rev=$(shell git merge-base v3 HEAD) --new $(GOLANG_CI_LINT_RUN_OPTS)
	bin/go-sumtype ./...

check-license:        ## Run license header checks against source files
	bin/license-eye -c .licenserc.yaml header check

check-all: check-license check    ## Run linter and license checks

check-new:           ## Run linters only against new code since v3 branch point
	bin/golangci-lint run -c=.golangci.yml --new-from-rev=origin/v3 --new

FILES = $(shell find . -type f -name '*.go')

format:               ## Format source code
	make -C api format
	bin/gofumpt -l -w $(FILES)
	bin/goimports -local github.com/percona/pmm -l -w $(FILES)
	bin/gci write --section Standard --section Default --section "Prefix(github.com/percona/pmm)" $(FILES)

serve:                ## Serve API documentation with nginx
	nginx -p . -c api/nginx/nginx.conf

GOLANG_CI_LINT_RUN_OPTS=--fix
prepare-pr: gen check-all  ## Run all checks and generate files
	go mod tidy
