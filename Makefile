help:                           ## Display this help message.
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
		awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

# `cut` is used to remove first `v` from `git describe` output
PMM_RELEASE_PATH ?= bin
PMM_RELEASE_VERSION ?= $(shell git describe --always --dirty | cut -b2-)
PMM_RELEASE_TIMESTAMP ?= $(shell date '+%s')
PMM_RELEASE_FULLCOMMIT ?= $(shell git rev-parse HEAD)
PMM_RELEASE_BRANCH ?= $(shell git describe --always --contains --all)

LD_FLAGS = -ldflags " \
			-X 'github.com/percona/pmm-update/vendor/github.com/percona/pmm/version.ProjectName=pmm-update' \
			-X 'github.com/percona/pmm-update/vendor/github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm-update/vendor/github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm-update/vendor/github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
			-X 'github.com/percona/pmm-update/vendor/github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
			-X 'github.com/percona/pmm-update/vendor/github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)' \
			"

release:                        ## Build pmm-update release binary.
	env CGO_ENABLED=0 go build -v $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/pmm-update

init:                           ## Installs tools to $GOPATH/bin (which is expected to be in $PATH).
	curl https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin

	go install ./vendor/golang.org/x/tools/cmd/goimports

	go test -i ./...
	go test -race -i ./...

install:                        ## Install pmm-update binary.
	go install $(LD_FLAGS) ./...

install-race:                   ## Install pmm-update binary with race detector.
	go install $(LD_FLAGS) -race ./...

TEST_FLAGS ?= -timeout=60s -count=1 -v -p 1

test:                           ## Run tests.
	go test $(TEST_FLAGS) ./...

test-race:                      ## Run tests with race detector.
	go test $(TEST_FLAGS) -race ./...

test-cover:                     ## Run tests and collect per-package coverage information.
	go test $(TEST_FLAGS) -coverprofile=cover.out -covermode=count ./...

check:                          ## Run required checkers and linters.
	go run .github/check-license.go

	ansible-playbook --syntax-check ansible/playbook/tasks/update.yml
	ansible-playbook --check ansible/playbook/tasks/update.yml
	ansible-lint ansible/playbook/tasks/update.yml

FILES = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

format:                         ## Format source code.
	gofmt -w -s $(FILES)
	goimports -local github.com/percona/pmm-update -l -w $(FILES)

RUN_FLAGS ?=

run: install _run               ## Run pmm-update.

run-race: install-race _run     ## Run pmm-update with race detector.

run-race-cover: install-race    ## Run pmm-update with race detector and collect coverage information.
	go test -coverpkg="github.com/percona/pmm-update/..." \
			-tags maincover \
			$(LD_FLAGS) \
			-race -c -o pmm-update.test
	rm -f *.runcover.out
	./pmm-update.test -test.count=1 -test.v -test.coverprofile=$(shell mktemp --tmpdir=. XXX.runcover.out) -test.run=TestMainCover $(RUN_FLAGS)

_run:
	pmm-update $(RUN_FLAGS)

env-up:                         ## Start development environment.
	docker-compose up --force-recreate --abort-on-container-exit --renew-anon-volumes --remove-orphans

env-down:                       ## Stop development environment.
	docker-compose down --volumes --remove-orphans

check-all: check        ## Run golang ci linter to check new changes from master.
	golangci-lint run -c=.golangci.yml --new-from-rev=master

ci-reviewdog:           ## Runs reviewdog checks.
	golangci-lint run -c=.golangci-required.yml --out-format=line-number | bin/reviewdog -f=golangci-lint -level=error -reporter=github-pr-check
	golangci-lint run -c=.golangci.yml --out-format=line-number | bin/reviewdog -f=golangci-lint -level=error -reporter=github-pr-review
