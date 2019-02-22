help:                           ## Display this help message.
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
	    awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

PMM_RELEASE_VERSION ?= 2.0.0-dev
PMM_RELEASE_TIMESTAMP = $(shell date '+%s')
PMM_RELEASE_FULLCOMMIT = $(shell git rev-parse HEAD)
PMM_RELEASE_BRANCH = $(shell git describe --all --contains --dirty HEAD)

release:                        ## Build bin/pmm-agent release binary.
	go build -v -o bin/pmm-agent -ldflags " \
		-X 'github.com/percona/pmm-agent/vendor/github.com/percona/pmm/version.ProjectName=pmm-agent' \
		-X 'github.com/percona/pmm-agent/vendor/github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
		-X 'github.com/percona/pmm-agent/vendor/github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
		-X 'github.com/percona/pmm-agent/vendor/github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
		-X 'github.com/percona/pmm-agent/vendor/github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
		-X 'github.com/percona/pmm-agent/vendor/github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)' \
		"

install:                        ## Install pmm-agent binary.
	go install -v ./...

install-race:                   ## Install pmm-agent binary with race detector.
	go install -v -race ./...

test:                           ## Run tests.
	go test -v ./...

test-race:                      ## Run tests with race detector.
	go test -v -race ./...

test-cover:                     ## Run tests and collect coverage information.
	go test -v -coverprofile=cover.out -covermode=count ./...

check-license:                  ## Check that all files have the same license header.
	go run .github/check-license.go

check: install check-license    ## Run checkers and linters.
	golangci-lint run

format:                         ## Run `goimports`.
	goimports -local github.com/percona/pmm-agent -l -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")
