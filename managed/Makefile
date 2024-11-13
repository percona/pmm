default: help

help:                           ## Display this help message
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep -h '^[a-zA-Z]' $(MAKEFILE_LIST) | \
		awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'
	@echo
	@echo "To test DBaaS components with minikube:"
	@echo "Start minikube: minikube start --cpus=2 --nodes=3 --kubernetes-version=v1.20.0"
	@echo "ENABLE_DBAAS=1 NETWORK=minikube make env-up"

# `cut` is used to remove first `v` from `git describe` output
PMM_RELEASE_PATH ?= ../bin
PMM_RELEASE_VERSION ?= $(shell git describe --always --dirty | cut -b2-)
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

gen: clean                      ## Generate files
	go generate ./...

clean:                          ## Remove generated files
	find . -name *_reform.go -delete

release:                        ## Build pmm-managed release binaries
	env CGO_ENABLED=0 go build -v $(PMM_LD_FLAGS) -o $(PMM_RELEASE_PATH)/ ./cmd/...
	$(PMM_RELEASE_PATH)/pmm-managed --version

release-starlark:
	env CGO_ENABLED=0 go build -v $(PMM_LD_FLAGS) -o $(PMM_RELEASE_PATH)/ ./cmd/pmm-managed-starlark/...
	$(PMM_RELEASE_PATH)/pmm-managed-starlark --version

ARCH=$(shell uname -m)
release-dev:										## Build pmm-managed binaries for development
	if [ $(ARCH) = "aarch64" ]; then \
		go build -gcflags="all=-N -l" -v $(PMM_LD_FLAGS) -o $(PMM_RELEASE_PATH)/ ./cmd/... ; \
	else \
		go build -race -gcflags="all=-N -l" -v $(PMM_LD_FLAGS) -o $(PMM_RELEASE_PATH)/ ./cmd/... ; \
    fi

PMM_TEST_FLAGS ?= -timeout=180s
PMM_TEST_RUN_UPDATE ?= 0
PMM_TEST_FILES ?= ./...

test:                           ## Run tests
	go test $(PMM_TEST_FLAGS) -p 1 -race $(PMM_TEST_FILES)

test-cover:                     ## Run tests and collect per-package coverage information
	go test $(PMM_TEST_FLAGS) -p 1 -race -coverprofile=cover.out -covermode=atomic -coverpkg=$(PMM_TEST_FILES) $(PMM_TEST_FILES)

test-update:                   ## Run pmm update test
	PMM_TEST_RUN_UPDATE=1 go test -timeout=600s -v -p 1 -race ./services/supervisord -run ^TestDevContainer/Update$$

fuzz-grafana:                   ## Run fuzzer for services/grafana package
	# go get -u github.com/dvyukov/go-fuzz/go-fuzz github.com/dvyukov/go-fuzz/go-fuzz-build
	mkdir -p managed/services/grafana/fuzzdata/corpus
	cd managed/services/grafana && go-fuzz-build
	cd managed/services/grafana && go-fuzz -workdir=fuzzdata
