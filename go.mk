# Common Go component variables and canned recipes.
#
# Include from component Makefiles after setting required variables:
#
#   PROJECT_NAME  (required) — version.ProjectName value, e.g. pmm-admin
#   BINARY_NAME   — output binary name        (default: PROJECT_NAME)
#   BUILD_SOURCE  — go build source path       (default: .)
#   TEST_FLAGS    — go test flags              (default: -timeout=30s)
#   TEST_PKGS     — packages to test           (default: ./...)
#   TEST_PARALLEL — parallelism flag, e.g. -p 1 (default: empty)
#   COVERAGE_MODE — -covermode value           (default: atomic)
#
# Canned recipes for use in component targets:
#   $(go-release)      — CGO_ENABLED=0 go build to PMM_RELEASE_PATH
#   $(go-install)      — go build to GOBIN
#   $(go-install-race) — go build -race to GOBIN
#   $(go-test)         — go test -race
#   $(go-test-cover)   — go test -race with coverage

BINARY_NAME   ?= $(PROJECT_NAME)
BUILD_SOURCE  ?= .
TEST_FLAGS    ?= -timeout=30s
TEST_PKGS     ?= ./...
COVERAGE_MODE ?= atomic

# Release metadata. `cut -b2-` strips the leading `v` from git describe.
PMM_RELEASE_PATH      ?= ../bin
PMM_RELEASE_VERSION   ?= $(shell git describe --always --dirty | cut -b2-)
PMM_RELEASE_TIMESTAMP ?= $(shell date '+%s')
PMM_RELEASE_FULLCOMMIT ?= $(shell git rev-parse HEAD)
PMM_RELEASE_BRANCH    ?= $(shell git describe --always --contains --all)

# Go binary install path.
ifeq ($(GOBIN),)
	GOBIN := $(shell go env GOPATH)/bin
endif

# Version linker flags (raw, without -ldflags wrapper).
# Components that need custom linking (e.g. static builds) can use this directly.
VERSION_FLAGS = -X 'github.com/percona/pmm/version.ProjectName=$(PROJECT_NAME)' \
	-X 'github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
	-X 'github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
	-X 'github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
	-X 'github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
	-X 'github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)'

# Wrapped linker flags for simple go build calls.
LD_FLAGS = -ldflags "$(VERSION_FLAGS)"

## Canned recipes

define go-release
env CGO_ENABLED=0 go build -v $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/$(BINARY_NAME) $(BUILD_SOURCE)
endef

define go-install
go build -v $(LD_FLAGS) -o $(GOBIN)/$(BINARY_NAME) $(BUILD_SOURCE)
endef

define go-install-race
go build -v $(LD_FLAGS) -race -o $(GOBIN)/$(BINARY_NAME) $(BUILD_SOURCE)
endef

define go-test
go test $(TEST_FLAGS) $(TEST_PARALLEL) -race $(TEST_PKGS)
endef

define go-test-cover
go test $(TEST_FLAGS) $(TEST_PARALLEL) -race -coverprofile=cover.out -covermode=$(COVERAGE_MODE) -coverpkg=$(TEST_PKGS) $(TEST_PKGS)
endef

## Default target and help

default: help

help:                           ## Display this help message
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep -h '^[a-zA-Z]' $(MAKEFILE_LIST) | \
		awk -F ':.*?## ' 'NF==2 {a[$$1] = sprintf("  %-26s%s", $$1, $$2)} END {for (k in a) print a[k]}' | sort
