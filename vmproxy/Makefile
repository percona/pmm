default: help

help:                           ## Display this help message
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
		awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

# `cut` is used to remove first `v` from `git describe` output
PMM_RELEASE_PATH ?= ../bin
PMM_RELEASE_VERSION ?= $(shell git describe --always --dirty | cut -b2-)
PMM_RELEASE_TIMESTAMP ?= $(shell date '+%s')
PMM_RELEASE_FULLCOMMIT ?= $(shell git rev-parse HEAD)
PMM_RELEASE_BRANCH ?= $(shell git describe --always --contains --all)
ifeq ($(GOBIN),)
	GOBIN := $(shell go env GOPATH)/bin
endif

LD_FLAGS = -ldflags " \
			-X 'github.com/percona/pmm/version.ProjectName=vmproxy' \
			-X 'github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
			-X 'github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
			-X 'github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)' \
			"

release:                        ## Build vmproxy release binary
	env CGO_ENABLED=0 go build -v $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/vmproxy .

install:                        ## Install vmproxy binary
	go build -v $(LD_FLAGS) -o $(GOBIN)/vmproxy .

install-race:                   ## Install vmproxy binary with race detector
	go build -v $(LD_FLAGS) -race -o $(GOBIN)/vmproxy .

build:                          ## Build vmproxy binary to current directory
	go build -v $(LD_FLAGS) -o vmproxy .

TEST_FLAGS ?= -timeout=20s

test:                           ## Run tests
	go test $(TEST_FLAGS) -race ./...

test-cover:                     ## Run tests and collect cross-package coverage information
	go test $(TEST_FLAGS) -race -coverprofile=cover.out -covermode=atomic -coverpkg=./... ./...
