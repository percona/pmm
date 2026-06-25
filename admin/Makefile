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
	-X 'github.com/percona/pmm/version.ProjectName=pmm-admin' \
	-X 'github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
	-X 'github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
	-X 'github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
	-X 'github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
	-X 'github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)' \
	"
# --- Tests-related variables ------------------------------------------------------------
TEST_FLAGS ?= -timeout=20s
# ------------------------------------------------------------------------------------------

.PHONY: help format release install install-race test test-cover env-up env-down

$(PMM_RELEASE_PATH):
	mkdir -p $(PMM_RELEASE_PATH)

help: 	## Display this help message
	@echo "Please use \`make <target>\`, where <target> is one of the following:"
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9-]+:.*## / {printf "  %-23s%s\n", $$1, $$2}' $(MAKEFILE_LIST)

format: FILES = $(shell find . -type f -name '*.go')
format: 	## Format source code
	go tool gofumpt -w $(FILES)
	go tool goimports -local github.com/percona/pmm -w $(FILES)
	go tool gci write --section Standard --section Default --section "Prefix(github.com/percona/pmm)" $(FILES)

release: $(PMM_RELEASE_PATH) 	## Build pmm-admin release binary
	CGO_ENABLED=0 go build -v $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/pmm-admin ./cmd/pmm-admin/

install:                        ## Install pmm & pmm-admin binary
	go build -v $(LD_FLAGS) -o $(GOBIN)/pmm-admin ./cmd/pmm-admin/

install-race:                   ## Install pmm & pmm-admin binary with race detector
	go build -v $(LD_FLAGS) -race -o $(GOBIN)/pmm-admin ./cmd/pmm-admin/

test:                           ## Run tests
	go test $(TEST_FLAGS) -race ./...

test-cover:                     ## Run tests and collect per-package coverage information
	go test $(TEST_FLAGS) -race -coverprofile=cover.out -covermode=atomic ./...

env-up:                         ## Start development environment
	docker compose up --force-recreate --abort-on-container-exit --renew-anon-volumes --remove-orphans

env-down:                       ## Stop development environment
	docker compose down --volumes --remove-orphans

