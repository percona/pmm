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

ifdef NO_VCS
	NO_VCS = -buildvcs=false
endif
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

gen: clean                      ## Generate files
	go generate ./...

clean:                          ## Remove generated files
	find . -name *_reform.go -delete

release:                        ## Build pmm-admin release binary
	CGO_ENABLED=0 go build -v $(NO_VCS) $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/pmm-admin ./cmd/pmm-admin/
	CGO_ENABLED=0 go build -v $(NO_VCS) $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/pmm ./cmd/pmm/

install:                        ## Install pmm & pmm-admin binary
	go build -v $(LD_FLAGS) -o $(GOBIN)/pmm-admin ./cmd/pmm-admin/
	go build -v $(LD_FLAGS) -o $(GOBIN)/pmm ./cmd/pmm/

install-race:                   ## Install pmm & pmm-admin binary with race detector
	go build -v $(LD_FLAGS) -race -o $(GOBIN)/pmm-admin ./cmd/pmm-admin/
	go build -v $(LD_FLAGS) -race -o $(GOBIN)/pmm ./cmd/pmm/

TEST_FLAGS ?= -timeout=20s

test:                           ## Run tests
	go test $(TEST_FLAGS) -race ./...

test-cover:                     ## Run tests and collect cross-package coverage information
	go test $(TEST_FLAGS) -race -coverprofile=cover.out -covermode=atomic -coverpkg=./... ./...

env-up:                         ## Start development environment
	docker compose up --force-recreate --abort-on-container-exit --renew-anon-volumes --remove-orphans

env-down:                       ## Stop development environment
	docker compose down --volumes --remove-orphans

