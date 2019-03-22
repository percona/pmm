help:                           ## Display this help message.
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
		awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

PMM_RELEASE_PATH ?= bin
PMM_RELEASE_VERSION ?= 2.0.0-dev
PMM_RELEASE_TIMESTAMP ?= $(shell date '+%s')
PMM_RELEASE_FULLCOMMIT ?= $(shell git rev-parse HEAD)
PMM_RELEASE_BRANCH ?= $(shell git describe --all --contains --dirty HEAD)

LD_FLAGS = -ldflags " \
			-X 'github.com/percona/pmm-admin/vendor/github.com/percona/pmm/version.ProjectName=pmm-admin' \
			-X 'github.com/percona/pmm-admin/vendor/github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm-admin/vendor/github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm-admin/vendor/github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
			-X 'github.com/percona/pmm-admin/vendor/github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
			-X 'github.com/percona/pmm-admin/vendor/github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)' \
			"

release:                        ## Build pmm-admin release binary.
	env CGO_ENABLED=0 go build -v $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/pmm-admin

install:                        ## Install pmm-admin binary.
	go install -v $(LD_FLAGS) ./...

install-race:                   ## Install pmm-admin binary with race detector.
	go install -v $(LD_FLAGS) -race ./...

check-license:                  ## Check that all files have the same license header.
	go run .github/check-license.go

check: install check-license    ## Run checkers and linters.
	golangci-lint run

format:                         ## Run `goimports`.
	goimports -local github.com/percona/pmm-admin -l -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")
