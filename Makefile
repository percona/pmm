help:                           ## Display this help message.
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
	    awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

install:                        ## Install pmm-admin binary.
	go install -v ./...

install-race:                   ## Install pmm-admin binary with race detector.
	go install -v -race ./...

check-license:                  ## Check that all files have the same license header.
	go run .github/check-license.go

check: install check-license    ## Run checkers and linters.
	golangci-lint run

format:                         ## Run `goimports`.
	goimports -local github.com/Percona-Lab/pmm-admin -l -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")
