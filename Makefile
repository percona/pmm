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
			-X 'github.com/percona/pmm-agent/vendor/github.com/percona/pmm/version.ProjectName=pmm-agent' \
			-X 'github.com/percona/pmm-agent/vendor/github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm-agent/vendor/github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm-agent/vendor/github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
			-X 'github.com/percona/pmm-agent/vendor/github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
			-X 'github.com/percona/pmm-agent/vendor/github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)' \
			"

release:                        ## Build pmm-agent release binary.
	env CGO_ENABLED=1 go build -v $(LD_FLAGS) -o $(PMM_RELEASE_PATH)/pmm-agent

init:                           ## Installs tools to $GOPATH/bin (which is expected to be in $PATH).
	curl https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin

	go install ./vendor/github.com/BurntSushi/go-sumtype \
				./vendor/github.com/vektra/mockery/cmd/mockery \
				./vendor/golang.org/x/perf/cmd/benchstat \
				./vendor/golang.org/x/tools/cmd/goimports \
				./vendor/gopkg.in/reform.v1/reform

	go test -i ./...
	go test -race -i ./...

gen:                            ## Generate files.
	go generate ./...

gen-init:
	go install ./vendor/gopkg.in/reform.v1/reform-db
	mkdir tmp-mysql
	reform-db -db-driver=mysql -db-source='root:root-password@tcp(127.0.0.1:3306)/performance_schema' init tmp-mysql

install:                        ## Install pmm-agent binary.
	go install $(LD_FLAGS) ./...

install-race:                   ## Install pmm-agent binary with race detector.
	go install $(LD_FLAGS) -race ./...

TEST_FLAGS ?= -timeout=20s

test:                           ## Run tests.
	go test $(TEST_FLAGS) ./...

test-race:                      ## Run tests with race detector.
	go test $(TEST_FLAGS) -race ./...

test-cover:                     ## Run tests and collect per-package coverage information.
	go test $(TEST_FLAGS) -coverprofile=cover.out -covermode=count ./...

test-crosscover:                ## Run tests and collect cross-package coverage information.
	go test $(TEST_FLAGS) -coverprofile=crosscover.out -covermode=count -coverpkg=./... ./...

bench:                          ## Run benchmarks.
	go test -bench=. -benchtime=1s -count=3 -cpu=1 -failfast github.com/percona/pmm-agent/agents/mysql/slowlog/parser | tee slowlog_parser_new.bench
	benchstat slowlog_parser_old.bench slowlog_parser_new.bench

check:                          ## Run required checkers and linters.
	go run .github/check-license.go
	go-sumtype ./vendor/... ./...

FILES = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

format:                         ## Format source code.
	gofmt -w -s $(FILES)
	goimports -local github.com/percona/pmm-agent -l -w $(FILES)

RUN_FLAGS = --config-file=pmm-agent-dev.yaml

run: install _run               ## Run pmm-agent.

run-race: install-race _run     ## Run pmm-agent with race detector.

run-race-cover: install-race    ## Run pmm-agent with race detector and collect coverage information.
	go test -coverpkg="github.com/percona/pmm-agent/..." \
			-tags maincover \
			$(LD_FLAGS) \
			-race -c -o bin/pmm-agent.test
	bin/pmm-agent.test -test.coverprofile=cover.out -test.run=TestMainCover -- $(RUN_FLAGS)

_run:
	pmm-agent $(RUN_FLAGS)

env-up:                         ## Start development environment.
	docker-compose up --force-recreate --renew-anon-volumes --remove-orphans

env-down:                       ## Stop development environment.
	docker-compose down --volumes --remove-orphans
