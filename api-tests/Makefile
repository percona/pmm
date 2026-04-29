.DEFAULT_GOAL := help
BIN_DIR := $(CURDIR)/bin

# --- Go-related variables ----------------------------------------------------------------
GO_VERSION := $(shell grep '^go ' $(CURDIR)/../go.mod | awk '{print $$2}')

# --- Git variables ------------------------------------------------------------------------
GIT_VERSION := $(shell git describe --tags)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# --- Test-related variables --------------------------------------------------------------
TEST_DIRS := $(shell find . -type f -name '*_test.go' -not -path '*/.*' -exec dirname {} + | sort -u)
TEST_BINARY_TARGETS := $(subst /,-,$(subst ./,,$(TEST_DIRS)))
RUN_TEST_TARGETS := $(addprefix test-,$(TEST_BINARY_TARGETS))
# Exclude server-settings tests from this list, as they require special handling (Grafana restart).
RUN_TEST_TARGETS := $(filter-out test-server-settings%,$(RUN_TEST_TARGETS))
REPORT_OUT := pmm-api-tests-junit-report.xml
TEST_OUTPUT_PREFIX := pmm-api-tests-output
TEST_FLAGS := -test.count=1 -test.v

# --- Docker related variables -----------------------------------------------------------
DOCKER_IMAGE := local/pmm-api-tests
PMM_SERVER_INSECURE_TLS ?= 1
PMM_RUN_UPDATE_TEST ?= 0
PMM_RUN_ADVISOR_TESTS ?= 0

.PHONY: help guard-% init $(TEST_BINARY_TARGETS) $(RUN_TEST_TARGETS)
.PHONY: test-server-settings test test-report clean docker-build-image docker-run-tests

help: ## Show available targets
	@echo "Available targets: $(TARGETS)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

guard-%:
	@[ -n "${$*}" ] || (echo "ERROR: $* is not set"; exit 1)

$(TEST_BINARY_TARGETS): ## Build test binaries
	@dir="$(subst -,/,$@)"; \
	echo "-> Building test binary for dir: $$dir"; \
	go test -c -v -race -o $(BIN_DIR)/$@ $(CURDIR)/$$dir

init: $(TEST_BINARY_TARGETS) ## Installs development tools
	@echo "-> Installing tools"
	cd tools && go generate -x -tags=tools

$(RUN_TEST_TARGETS):
	@binary="$(subst test-,,$@)"; \
	echo "🚀 Run $$binary tests"; \
	$(BIN_DIR)/$$binary $(TEST_FLAGS) 2>&1 | tee $(TEST_OUTPUT_PREFIX)-$@.txt

# This target shall be running separately from the rest of tests,
# as it triggers Grafana restart that may cause other tests to fail.
test-server-settings: guard-PMM_SERVER_URL ## Run server-settings tests (requires Grafana restart)
	@binary="$(subst test-,,$@)"; \
	echo "🚀 Run $$binary tests"; \
	$(BIN_DIR)/$$binary $(TEST_FLAGS) 2>&1 | tee $(TEST_OUTPUT_PREFIX)-$@.txt

test: guard-PMM_SERVER_URL $(RUN_TEST_TARGETS) ## Run tests
	$(MAKE) test-server-settings
	@echo "✅ All tests completed."

test-report: test ## Run tests and generate JUnit report
	cat $(CURDIR)/$(TEST_OUTPUT_PREFIX)-*.txt | $(BIN_DIR)/go-junit-report > $(REPORT_OUT)
	@echo "✅ Report is generated: $(REPORT_OUT)."

clean:	## Cleanup reports
	rm -f $(CURDIR)/pmm-api-tests-*.txt
	rm -f $(CURDIR)/$(REPORT_OUT)
	@echo "✅ Cleanup completed."

## Docker-related targets

docker-build-image: ## Build Docker image with tests
	DOCKER_BUILDKIT=1 docker build \
	-f Dockerfile \
	--build-arg GO_VERSION=$(GO_VERSION) \
	--build-arg VERSION=$(GIT_VERSION) \
	--build-arg GIT_COMMIT=$(GIT_COMMIT) \
	--build-arg BUILD_TIME=$(BUILD_TIME) \
	-t $(DOCKER_IMAGE) ../
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
