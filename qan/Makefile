.DEFAULT_GOAL := help
CURDIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
BIN_DIR := $(CURDIR)/../bin

.PHONY: help release install test

help: ## Display this help message
	@echo "Please use \`make <target>\`, where <target> is one of the following:"
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9-]+:.*## / {printf "  %-28s%s\n", $$1, $$2}' $(MAKEFILE_LIST)

PMM_RELEASE_PATH ?= $(BIN_DIR)
PMM_RELEASE_VERSION ?= $(shell git describe --always --dirty | cut -b2-)
PMM_RELEASE_TIMESTAMP ?= $(shell date '+%s')
PMM_RELEASE_FULLCOMMIT ?= $(shell git rev-parse HEAD)
PMM_RELEASE_BRANCH ?= $(shell git describe --always --contains --all)

release:                        ## Build qan release binary
	env CGO_ENABLED=0 go build -v -o $(PMM_RELEASE_PATH)/qan -ldflags " \
		-X 'github.com/percona/pmm/version.ProjectName=qan' \
		-X 'github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
		-X 'github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
		-X 'github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
		-X 'github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
		-X 'github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)' \
		"

install:                        ## Install qan binary
	go install -v ./...

test:                           ## Run tests (integration tests need ClickHouse; they skip if unavailable)
	go test -race -count=1 ./...
