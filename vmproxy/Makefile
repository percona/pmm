.DEFAULT_GOAL := help

# --- Go-related variables ----------------------------------------------------------------
# `cut` is used to remove first `v` from `git describe` output
PMM_RELEASE_PATH ?= $(CURDIR)/../bin
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
# --- Tests-related variables ------------------------------------------------------------
TEST_FLAGS ?= -timeout=20s
# ------------------------------------------------------------------------------------------

.PHONY: help format release install install-race build test test-cover

help: 	## Display this help message
	@echo "Please use \`make <target>\`, where <target> is one of the following:"
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9-]+:.*## / {printf "  %-23s%s\n", $$1, $$2}' $(MAKEFILE_LIST)

format: FILES = $(shell find . -type f -name '*.go')
format: 	## Format source code
	go tool gofumpt -w $(FILES)
	go tool goimports -local github.com/percona/pmm -w $(FILES)
	go tool gci write --section Standard --section Default --section "Prefix(github.com/percona/pmm)" $(FILES)

release:                        ## Build vmproxy release binary
	env CGO_ENABLED=0 go build -v $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/vmproxy .

install:                        ## Install vmproxy binary
	go build -v $(LD_FLAGS) -o $(GOBIN)/vmproxy .

install-race:                   ## Install vmproxy binary with race detector
	go build -v $(LD_FLAGS) -race -o $(GOBIN)/vmproxy .

build:                          ## Build vmproxy binary to current directory
	go build -v $(LD_FLAGS) -o vmproxy .

test:                           ## Run tests
	go test $(TEST_FLAGS) -race ./...

test-cover:                     ## Run tests and collect per-package coverage information
	go test $(TEST_FLAGS) -race -coverprofile=cover.out -covermode=atomic ./...
