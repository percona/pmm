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
			-X 'github.com/percona/pmm-managed/vendor/github.com/percona/pmm/version.ProjectName=pmm-managed' \
			-X 'github.com/percona/pmm-managed/vendor/github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm-managed/vendor/github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm-managed/vendor/github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
			-X 'github.com/percona/pmm-managed/vendor/github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
			-X 'github.com/percona/pmm-managed/vendor/github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)' \
			"

release:                        ## Build pmm-managed release binary.
	env CGO_ENABLED=0 go build -v $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/pmm-managed

init:                           ## Installs tools to $GOPATH/bin (which is expected to be in $PATH).
	curl https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin

	go install ./vendor/github.com/BurntSushi/go-sumtype \
				./vendor/github.com/vektra/mockery/cmd/mockery \
				./vendor/golang.org/x/tools/cmd/goimports \
				./vendor/gopkg.in/reform.v1/reform

	go get -u github.com/prometheus/prometheus/cmd/promtool

	go test -i ./...
	go test -race -i ./...

gen:                            ## Generate files.
	rm -f models/*_reform.go
	go generate ./...

install:                        ## Install pmm-managed binary.
	go install $(LD_FLAGS) ./...

install-race:                   ## Install pmm-managed binary with race detector.
	go install $(LD_FLAGS) -race ./...

TEST_FLAGS ?= -timeout=20s

test:                           ## Run tests.
	go test $(TEST_FLAGS) -p 1 ./...

test-race:                      ## Run tests with race detector.
	go test $(TEST_FLAGS) -p 1 -race ./...

test-cover:                     ## Run tests and collect per-package coverage information.
	go test $(TEST_FLAGS) -p 1 -coverprofile=cover.out -covermode=count ./...

test-crosscover:                ## Run tests and collect cross-package coverage information.
	go test $(TEST_FLAGS) -p 1 -coverprofile=crosscover.out -covermode=count -coverpkg=./... ./...

check:                          ## Run required checkers and linters.
	go run .github/check-license.go
	go-sumtype ./vendor/... ./...

FILES = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

format:                         ## Format source code.
	gofmt -w -s $(FILES)
	goimports -local github.com/percona/pmm-managed -l -w $(FILES)

RUN_FLAGS = -debug \
			-agent-mysqld-exporter=mysqld_exporter \
			-agent-postgres-exporter=postgres_exporter \
			-agent-rds-exporter=rds_exporter \
			-agent-rds-exporter-config=testdata/rds_exporter/rds_exporter.yml \
			-prometheus-config=testdata/prometheus/prometheus.yml \
			-db-name=pmm-managed-dev \
			-postgres-name=pmm-managed-dev

run: install _run               ## Run pmm-managed.

run-race: install-race _run     ## Run pmm-managed with race detector.

run-race-cover: install-race    ## Run pmm-managed with race detector and collect coverage information.
	go test -coverpkg="github.com/percona/pmm-managed/..." \
			-tags maincover \
			$(LD_FLAGS) \
			-race -c -o bin/pmm-managed.test
	bin/pmm-managed.test -test.coverprofile=cover.out -test.run=TestMainCover $(RUN_FLAGS)

_run:
	pmm-managed $(RUN_FLAGS)

env-up:                         ## Start development environment.
	docker-compose up --force-recreate --abort-on-container-exit --renew-anon-volumes --remove-orphans

env-down:                       ## Stop development environment.
	docker-compose down --volumes --remove-orphans

clean:                          ## Removes generated artifacts.
	rm -Rf ./bin
