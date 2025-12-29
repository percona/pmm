# Makefile for PMM Client Skaffold builds
# Provides convenient targets for common build scenarios

.PHONY: help build build-dynamic build-arm64 build-component extract clean test-build

# Default target
help:
	@echo "PMM Client Skaffold Build Targets:"
	@echo ""
	@echo "  make build          - Build all components (static, amd64)"
	@echo "  make build-dynamic  - Build with dynamic linking (GSSAPI support)"
	@echo "  make build-arm64    - Build for ARM64 architecture"
	@echo "  make build-component COMPONENT=<name> - Build single component"
	@echo "  make extract        - Extract binaries from all component images"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make test-build     - Build and extract (for testing)"
	@echo ""
	@echo "Examples:"
	@echo "  make build-component COMPONENT=pmm-admin"
	@echo "  make build-component COMPONENT=node-exporter"
	@echo ""
	@echo "Direct Skaffold (requires env vars):"
	@echo "  PMM_VERSION=\$$(cat ../../VERSION) BASE_IMAGE=golang:latest skaffold build -b pmm-admin"
	@echo ""
	@echo "Environment Variables:"
	@echo "  PMM_VERSION         - Version to build (auto-detected from VERSION file)"
	@echo "  BASE_IMAGE          - Base Docker image (default: golang:latest)"
	@echo "  RESULTS_DIR         - Output directory (default: ../bin)"
	@echo ""

# Results directory
RESULTS_DIR ?= ../bin

# Read version from pmm-submodules repository
PMM_VERSION ?= $(shell curl -fsSL https://raw.githubusercontent.com/Percona-Lab/pmm-submodules/refs/heads/v3/VERSION 2>/dev/null || echo "dev")
export PMM_VERSION

# Base image selection based on CI environment
ifdef CI
	BASE_IMAGE ?= public.ecr.aws/e7j3v3n0/rpmbuild:3
else
	BASE_IMAGE ?= golang:latest
endif
export BASE_IMAGE

BUILD_TYPE ?= static
export BUILD_TYPE

GOARCH ?= amd64
export GOARCH

# Component list
COMPONENTS := pmm-admin pmm-agent node-exporter mysqld-exporter mongodb-exporter \
              postgres-exporter proxysql-exporter rds-exporter azure-metrics-exporter \
              redis-exporter vmagent nomad percona-toolkit

# Build all components
build:
	@echo "Building PMM Client components (version: $(PMM_VERSION))..."
	@echo "Using base image: $(BASE_IMAGE)"
	@export PMM_VERSION=$(PMM_VERSION); \
	export BASE_IMAGE=$(BASE_IMAGE); \
	export BUILD_TYPE=$(BUILD_TYPE); \
	export GOARCH=$(GOARCH); \
	skaffold build

# Build with dynamic linking (GSSAPI support)
build-dynamic:
	$(MAKE) build BUILD_TYPE=dynamic

# Build for ARM64
build-arm64:
	$(MAKE) build GOARCH=arm64

# Build a single component
build-component:
	@if [ -z "$(COMPONENT)" ]; then \
		echo "Error: COMPONENT not specified. Usage: make build-component COMPONENT=<name>"; \
		echo "Available components: $(COMPONENTS)"; \
		exit 1; \
	fi
	@echo "Building component: $(COMPONENT) (version: $(PMM_VERSION))..."
	@echo "Using base image: $(BASE_IMAGE)"
	@export PMM_VERSION=$(PMM_VERSION); \
	export BASE_IMAGE=$(BASE_IMAGE); \
	export BUILD_TYPE=$(BUILD_TYPE); \
	export GOARCH=$(GOARCH); \
	skaffold build -b $(COMPONENT)

# Extract binaries from all component images
extract:
	@echo "Extracting binaries from images..."
	@mkdir -p $(RESULTS_DIR)
	@for component in $(COMPONENTS); do \
		image="$$component:$(PMM_VERSION)"; \
		echo "Extracting from $$image..."; \
		docker create --name pmm-extract-$$component $$image 2>/dev/null || true; \
		docker cp pmm-extract-$$component:/output/. $(RESULTS_DIR)/ 2>/dev/null || true; \
		docker rm pmm-extract-$$component 2>/dev/null || true; \
	done
	@echo "Binaries extracted to $(RESULTS_DIR)"
	@ls -lh $(RESULTS_DIR)

# Clean build artifacts
clean:
	rm -rf $(RESULTS_DIR)
	skaffold delete || true

# Build and extract for testing
test-build: build extract
	@echo "Build complete! Binaries available in $(RESULTS_DIR)"

# Build all variants (for CI/CD)
build-all: build-static build-dynamic build-arm64
	@echo "All build variants complete!"

# Development workflow - build, extract, and show contents
dev: build-dev extract
	@echo "Development build complete!"
	@echo "Binary contents:"
	@tar -tzf $(RESULTS_DIR)/pmm-client-*.tar.gz | grep -E "bin/(pmm-admin|pmm-agent)" || true
