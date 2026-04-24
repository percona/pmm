# Common Go component variables and canned recipes.
#
# Include from component Makefiles after setting required variables:
#
#   PROJECT_NAME     — version.ProjectName value (required), e.g. pmm-admin
#   BINARY_NAME      — output binary name                      (default: PROJECT_NAME)
#   BUILD_SOURCE     — go build source path                    (default: .)
#   TEST_FLAGS       — go test flags                           (default: -timeout=30s)
#   TEST_PKGS        — packages to test                        (default: ./...)
#   TEST_PARALLEL    — parallelism flag, e.g. -p 1             (default: empty)
#   COVERAGE_MODE    — -covermode value                        (default: atomic)
#   COMPOSE_PROFILES - comma-separated Docker Compose profiles (default: pmm)
#   ENV_UP_FLAGS     — docker compose up flags for env-up      (default: --force-recreate --renew-anon-volumes --remove-orphans)
#   ENV_DOWN_FLAGS   — docker compose down flags for env-down  (default: --volumes --remove-orphans)
#
# Canned recipes for use in component targets:
#   $(go-release)      — CGO_ENABLED=0 go build to PMM_RELEASE_PATH
#   $(go-install)      — go build to GOBIN
#   $(go-install-race) — go build -race to GOBIN
#   $(go-test)         — go test -race
#   $(go-test-cover)   — go test -race with coverage

REPO_ROOT     := $(patsubst %/,%,$(dir $(lastword $(MAKEFILE_LIST))))
BINARY_NAME   ?= $(PROJECT_NAME)
BUILD_SOURCE  ?= .
TEST_FLAGS    ?= -timeout=30s
TEST_PKGS     ?= ./...
COVERAGE_MODE ?= atomic
COMPOSE_PROFILES ?= pmm

# Shared docker compose flags for env-up / env-down targets.
ENV_UP_FLAGS   ?= --force-recreate --renew-anon-volumes --remove-orphans
ENV_DOWN_FLAGS ?= --volumes --remove-orphans

# Release metadata. `cut -b2-` strips the leading `v` from git describe.
PMM_RELEASE_PATH       ?= ../bin
PMM_RELEASE_VERSION    ?= $(shell git describe --always --dirty | cut -b2-)
PMM_RELEASE_TIMESTAMP  ?= $(shell date '+%s')
PMM_RELEASE_FULLCOMMIT ?= $(shell git rev-parse HEAD)
PMM_RELEASE_BRANCH     ?= $(or $(GITHUB_HEAD_REF),$(GITHUB_REF_NAME),$(shell git rev-parse --abbrev-ref HEAD))

# Go binary install path.
GOBIN := $(or $(GOBIN),$(shell go env GOPATH)/bin)

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
go test $(TEST_FLAGS) $(TEST_PARALLEL) -race -count 1 $(TEST_PKGS)
endef

define go-test-cover
go test $(TEST_FLAGS) $(TEST_PARALLEL) -race -coverprofile=cover.out -covermode=$(COVERAGE_MODE) $(TEST_PKGS)
endef

define go-mod-download
go mod download -x
endef

.PHONY: $(MAKECMDGOALS)

## Default target and help
default: help

help:                           ## Display this help message
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep -h '^[a-zA-Z]' $(MAKEFILE_LIST) | \
		awk -F ':.*?## ' 'NF==2 {a[$$1] = sprintf("  %-26s%s", $$1, $$2)} END {for (k in a) print a[k]}' | sort
	@echo
	@echo Check .env.dev.example to see which environment variables are available.


# PLAN:
# 1. Move all env-up, env-down target to the top-level Makefile
# 2. Move all services located in docker-compose.yml files to the top-level docker-compose.yml, 
# and use profiles to control which services are started in each environment (devcontainer, CI, etc.)
# 3. Remove all docker-compose.yml files except the top-level one
# 	 Out of scope: docker-compose-pg-load.yml
