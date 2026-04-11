PROJECT_NAME := vmproxy
TEST_FLAGS   ?= -timeout=20s

include ../go.mk

release:                        ## Build vmproxy release binary
	$(go-release)

install:                        ## Install vmproxy binary
	$(go-install)

install-race:                   ## Install vmproxy binary with race detector
	$(go-install-race)

test:                           ## Run tests
	$(go-test)

test-cover:                     ## Run tests and collect coverage information
	$(go-test-cover)

build:                          ## Build vmproxy binary to current directory
	go build -v $(LD_FLAGS) -o vmproxy .
