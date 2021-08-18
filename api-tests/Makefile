BASE_PATH = $(shell pwd)
BIN_PATH := $(BASE_PATH)/bin

export PATH := $(BIN_PATH):$(PATH)

all: build

init:           ## Installs development tools
	go build -modfile=tools/go.mod -o $(BIN_PATH)/goimports golang.org/x/tools/cmd/goimports
	go build -modfile=tools/go.mod -o $(BIN_PATH)/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint
	go build -modfile=tools/go.mod -o $(BIN_PATH)/go-junit-report github.com/jstemmer/go-junit-report
	go build -modfile=tools/go.mod -o $(BIN_PATH)/reviewdog github.com/reviewdog/reviewdog/cmd/reviewdog

build:
	go install -v ./...
	go test -c -v ./inventory
	go test -c -v ./management
	go test -c -v ./server

dev-test:						## Run test on dev env. Use `PMM_KUBECONFIG=/path/to/kubeconfig.yaml make dev-test` to run tests for DBaaS.
	go test -count=1 -p 1 -v ./... -pmm.server-insecure-tls

run:
	go test -count=1 -p 1 -v ./... 2>&1 | tee pmm-api-tests-output.txt
	cat pmm-api-tests-output.txt | $(BIN_PATH)/go-junit-report > pmm-api-tests-junit-report.xml

run-race:
	go test -count=1 -p 1 -v -race ./... 2>&1 | tee pmm-api-tests-output.txt
	cat pmm-api-tests-output.txt | $(BIN_PATH)/go-junit-report > pmm-api-tests-junit-report.xml

FILES = $(shell find . -type f -name '*.go')

format:                         ## Format source code.
	gofmt -w -s $(FILES)
	$(BIN_PATH)/goimports -local github.com/Percona-Lab/pmm-api-tests -l -w $(FILES)

clean:
	rm -f ./pmm-api-tests-output.txt
	rm -f ./pmm-api-tests-junit-report.xml

check-all:                      ## Run golang ci linter to check new changes from master.
	$(BIN_PATH)/golangci-lint run -c=.golangci.yml --new-from-rev=master

ci-reviewdog:                   ## Runs reviewdog checks.
	$(BIN_PATH)/golangci-lint run -c=.golangci-required.yml --out-format=line-number | $(BIN_PATH)/reviewdog -f=golangci-lint -level=error -reporter=github-pr-check
	$(BIN_PATH)/golangci-lint run -c=.golangci.yml --out-format=line-number | $(BIN_PATH)/reviewdog -f=golangci-lint -level=error -reporter=github-pr-review
