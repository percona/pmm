.DEFAULT_GOAL := help

# --- Go-related variables ----------------------------------------------------------------
# `cut` is used to remove first `v` from `git describe` output
PMM_RELEASE_PATH ?= $(CURDIR)/bin
PMM_RELEASE_VERSION ?= $(shell git describe --always | cut -b2-)
PMM_RELEASE_TIMESTAMP ?= $(shell date '+%s')
PMM_RELEASE_FULLCOMMIT ?= $(shell git rev-parse HEAD)
PMM_RELEASE_BRANCH ?= $(shell git describe --always --contains --all)
PMM_LD_FLAGS = -ldflags " \
	-X 'github.com/percona/pmm/version.ProjectName=pmm-managed' \
	-X 'github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
	-X 'github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
	-X 'github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
	-X 'github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
	-X 'github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)' \
	"
# --- Tests-related variables -------------------------------------------------------------
IS_ARM        := $(filter arm64,$(shell uname -m))
PMM_RACE_FLAG ?= $(if $(IS_ARM),,-race)

PMM_TEST_FLAGS ?= -timeout=180s
PMM_TEST_RUN_UPDATE ?= 0
PMM_TEST_FILES ?= ./...
# ------------------------------------------------------------------------------------------

.PHONY: help gen _gen _gen-mocks format pi-validate release release-pi-validator
.PHONY: release-encryption-rotation release-starlark release-dev test test-cover
.PHONY: fuzz-grafana

help: 	## Display this help message
	@echo "Please use \`make <target>\`, where <target> is one of the following:"
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9-]+:.*## / {printf "  %-23s%s\n", $$1, $$2}' $(MAKEFILE_LIST)

gen: _gen _gen-mocks format 	## Generate files (includes formatting)

_gen:
	find . -name *_reform.go -delete
	go generate ./...

_gen-mocks:
	find . -name mock_*.go -delete
	go tool mockery --config .mockery.yaml

format: FILES = $(shell find . -type f -name '*.go')
format: 	## Format source code
	go tool gofumpt -w $(FILES)
	go tool goimports -local github.com/percona/pmm -w $(FILES)
	go tool gci write --section Standard --section Default --section "Prefix(github.com/percona/pmm)" $(FILES)

pi-validate:              ## Validate Percona Intelligence advisors and checks
	go run ./cmd/pi-validator/main.go advisors --advisors.dir ./data/advisors/ --checks.dir ./data/checks/

release:                        ## Build pmm-managed release binaries
	env CGO_ENABLED=0 go build -v $(PMM_LD_FLAGS) -o $(PMM_RELEASE_PATH)/ ./cmd/...

release-pi-validator: 		## Build PMM Percona Intelligence validator tool
	env CGO_ENABLED=0 go build -v $(PMM_LD_FLAGS) -o $(PMM_RELEASE_PATH)/ ./cmd/pi-validator/...

release-encryption-rotation:                        ## Build PMM encryption rotation tool
	env CGO_ENABLED=0 go build -v $(PMM_LD_FLAGS) -o $(PMM_RELEASE_PATH)/ ./cmd/pmm-encryption-rotation/...

release-starlark:
	env CGO_ENABLED=0 go build -v $(PMM_LD_FLAGS) -o $(PMM_RELEASE_PATH)/ ./cmd/pmm-managed-starlark/...
	$(PMM_RELEASE_PATH)/pmm-managed-starlark --version

release-dev:										## Build pmm-managed binaries for development
	go build $(PMM_RACE_FLAG) -gcflags="all=-N -l" -v $(PMM_LD_FLAGS) -o $(PMM_RELEASE_PATH)/ ./cmd/...

test:                           ## Run tests
	go test $(PMM_TEST_FLAGS) -p 1 -race $(PMM_TEST_FILES)

test-cover:                     ## Run tests and collect per-package coverage information
	go test $(PMM_TEST_FLAGS) -p 1 -race -coverprofile=cover.out -covermode=atomic $(PMM_TEST_FILES)

fuzz-grafana:                   ## Run fuzzer for services/grafana package
	# go get -u github.com/dvyukov/go-fuzz/go-fuzz github.com/dvyukov/go-fuzz/go-fuzz-build
	mkdir -p managed/services/grafana/fuzzdata/corpus
	cd managed/services/grafana && go-fuzz-build
	cd managed/services/grafana && go-fuzz -workdir=fuzzdata
