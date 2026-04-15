PROJECT_NAME := pmm-admin
BUILD_SOURCE := ./cmd/pmm-admin/
TEST_FLAGS   ?= -timeout=20s

include ../go.mk

release:                        ## Build pmm-admin release binary
	$(go-release)

install:                        ## Install pmm-admin binary
	$(go-install)

install-race:                   ## Install pmm-admin binary with race detector
	$(go-install-race)

test:                           ## Run tests
	$(go-test)

test-cover:                     ## Run tests and collect coverage information
	$(go-test-cover)

gen: clean                      ## Generate files
	go generate ./...

clean:                          ## Remove generated files
	find . -name *_reform.go -delete
