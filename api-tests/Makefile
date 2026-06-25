.DEFAULT_GOAL := help
CURDIR := $(shell dirname $(realpath $(MAKEFILE_LIST)))
BIN_DIR := $(CURDIR)/bin

# --- Go-related variables ----------------------------------------------------------------
GO_VERSION := $(shell grep '^go ' $(CURDIR)/../go.mod | awk '{print $$2}')

# --- Git variables ------------------------------------------------------------------------
GIT_VERSION := $(shell git describe --tags)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# --- Test-related variables --------------------------------------------------------------
TEST_DIRS := $(shell find $(CURDIR) -type f -name '*_test.go' -not -path '*/.*' -exec dirname {} + | sort -u)
TEST_DIRS := $(subst $(CURDIR)/,,$(TEST_DIRS))
TEST_BINARY_TARGETS := $(subst /,-,$(subst ./,,$(TEST_DIRS)))
RUN_TEST_TARGETS := $(addprefix test-,$(TEST_BINARY_TARGETS))
# Exclude server-settings tests from this list, as they require special handling (Grafana restart).
RUN_TEST_TARGETS := $(filter-out test-server-settings%,$(RUN_TEST_TARGETS))
REPORT_OUT := $(CURDIR)/pmm-api-tests-junit-report.xml
TEST_OUTPUT_PREFIX := pmm-api-tests-output
TEST_FLAGS := -test.count=1 -test.v

# --- Docker related variables -----------------------------------------------------------
DOCKER_IMAGE := local/pmm-api-tests
PMM_SERVER_INSECURE_TLS ?= 1
PMM_RUN_UPDATE_TEST ?= 0
PMM_RUN_ADVISOR_TESTS ?= 0
# ------------------------------------------------------------------------------------------

.PHONY: help format guard-% init $(TEST_BINARY_TARGETS) $(RUN_TEST_TARGETS)
.PHONY: test-server-settings test test-report clean docker-build-image docker-run-tests

help: 	## Display this help message
	@echo "Please use \`make <target>\`, where <target> is one of the following:"
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9-]+:.*## / {printf "  %-23s%s\n", $$1, $$2}' $(MAKEFILE_LIST)

format: FILES = $(shell find . -type f -name '*.go')
format: 	## Format source code
	go tool gofumpt -w $(FILES)
	go tool goimports -local github.com/percona/pmm -w $(FILES)
	go tool gci write --section Standard --section Default --section "Prefix(github.com/percona/pmm)" $(FILES)

guard-%:
	@[ -n "${$*}" ] || (echo "ERROR: $* is not set"; exit 1)

$(TEST_BINARY_TARGETS): ## Build test binaries
	@dir="$(subst -,/,$@)"; \
	echo "-> Building test binary for dir: $$dir"; \
	go test -c -v -race -o $(BIN_DIR)/$@ $(CURDIR)/$$dir

init: clean $(TEST_BINARY_TARGETS) ## Build tests

$(RUN_TEST_TARGETS):
	@binary="$(subst test-,,$@)"; \
	echo "🚀 Run $$binary tests"; \
	PMM_SERVER_INSECURE_TLS=$(PMM_SERVER_INSECURE_TLS) $(BIN_DIR)/$$binary $(TEST_FLAGS) 2>&1 | tee $(TEST_OUTPUT_PREFIX)-$@.txt

# This target shall be running separately from the rest of tests,
# as it triggers Grafana restart that may cause other tests to fail.
test-server-settings: guard-PMM_SERVER_URL ## Run server-settings tests (requires Grafana restart)
	@binary="$(subst test-,,$@)"; \
	echo "🚀 Run $$binary tests"; \
	PMM_SERVER_INSECURE_TLS=$(PMM_SERVER_INSECURE_TLS) $(BIN_DIR)/$$binary $(TEST_FLAGS) 2>&1 | tee $(CURDIR)/$(TEST_OUTPUT_PREFIX)-$@.txt

test: guard-PMM_SERVER_URL $(RUN_TEST_TARGETS) ## Run tests
	$(MAKE) test-server-settings
	@echo "✅ All tests completed."

test-report: test ## Run tests and generate JUnit report
	cat $(CURDIR)/$(TEST_OUTPUT_PREFIX)-*.txt | go tool go-junit-report -out $(REPORT_OUT)
	@echo "✅ Report generated: $(REPORT_OUT)."

clean:	## Cleanup reports and test binaries
	@echo "🧹 Cleaning up test binaries and reports..."
	rm -f $(BIN_DIR)/test-* || true
	rm -f $(CURDIR)/pmm-api-tests-*.txt $(CURDIR)/pmm-api-tests-*.xml  || true
	rm -f $(CURDIR)/$(REPORT_OUT) || true
	@echo "✅ Cleanup completed."

## Docker-related targets

docker-build-image: ## Build Docker image with tests
	DOCKER_BUILDKIT=1 docker build \
	-f $(CURDIR)/Dockerfile \
	--build-arg GO_VERSION=$(GO_VERSION) \
	--build-arg VERSION=$(GIT_VERSION) \
	--build-arg GIT_COMMIT=$(GIT_COMMIT) \
	--build-arg BUILD_TIME=$(BUILD_TIME) \
	-t $(DOCKER_IMAGE) $(CURDIR)/..
	@echo "✅ Docker image '$(DOCKER_IMAGE)' built successfully."

docker-run-tests: guard-PMM_SERVER_URL ## Run tests inside Docker container
	docker run \
	-e PMM_SERVER_URL=$(PMM_SERVER_URL) \
	-e PMM_SERVER_INSECURE_TLS=$(PMM_SERVER_INSECURE_TLS) \
	-e PMM_RUN_UPDATE_TEST=$(PMM_RUN_UPDATE_TEST) \
	-e PMM_RUN_ADVISOR_TESTS=$(PMM_RUN_ADVISOR_TESTS) \
	--name api-tests \
	--network host \
	$(DOCKER_IMAGE)
	@echo "✅ Tests executed successfully inside Docker container."
