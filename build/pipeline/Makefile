# Makefile for building PMM components using Docker

.PHONY: help build build-all build-dynamic build-arm64 clean volumes builder-image package-tarball build-client-docker push-client-docker

# Build script location
BUILD_SCRIPT := ./scripts/build-component
PACKAGE_SCRIPT := ./scripts/package-tarball
DOCKER_SCRIPT := ./scripts/build-client-docker
OUTPUT_DIR ?= $(CURDIR)/output
PACKAGE_DIR ?= $(CURDIR)/package

# Default target
help:
	@echo "PMM Component Build Targets:"
	@echo ""
	@echo "  make build COMPONENT=<name>  - Build a specific component"
	@echo "  make build-all                - Build all components"
	@echo "  make build-dynamic COMPONENT=<name> - Build with dynamic linking (GSSAPI)"
	@echo "  make build-arm64 COMPONENT=<name>   - Build for ARM64"
	@echo "  make builder-image            - Build the pmm-builder Docker image"
	@echo "  make volumes                  - Create Docker volumes for caching"
	@echo "  make package-tarball          - Generate pmm-client tarball from built components"
	@echo "  make build-client-docker      - Build PMM Client Docker image"
	@echo "  make push-client-docker       - Build and push PMM Client Docker image"
	@echo "  make clean                    - Remove output directory"
	@echo ""
	@echo "Workspace Components:"
	@echo "  pmm-admin, pmm-agent"
	@echo ""
	@echo "External Components:"
	@echo "  node_exporter, mysqld_exporter, mongodb_exporter, postgres_exporter,"
	@echo "  proxysql_exporter, rds_exporter, azure_metrics_exporter, redis_exporter,"
	@echo "  vmagent, nomad, percona-toolkit"
	@echo ""
	@echo "Environment Variables:"
	@echo "  PMM_VERSION    - Version to build (default: from VERSION file)"
	@echo "  BUILD_TYPE     - Build type: static or dynamic (default: static)"
	@echo "  GOARCH         - Target architecture: amd64 or arm64 (default: amd64)"
	@echo "  PLATFORM       - Docker platform: linux/amd64 or linux/arm64"
	@echo "  OUTPUT_DIR     - Output directory (default: ./output)"
	@echo ""
	@echo "Examples:"
	@echo "  make build COMPONENT=pmm-admin"
	@echo "  make build-dynamic COMPONENT=mongodb_exporter"
	@echo "  make build-arm64 COMPONENT=pmm-agent"
	@echo "  make build-all"
	@echo ""

# Configuration
PMM_VERSION ?= $(shell curl -fsSL https://raw.githubusercontent.com/Percona-Lab/pmm-submodules/refs/heads/v3/VERSION 2>/dev/null || echo "dev")
BUILD_TYPE ?= static
GOARCH ?= amd64
PLATFORM ?= linux/amd64
GO_VERSION ?= 1.25

export PMM_VERSION
export BUILD_TYPE
export GOARCH
export PLATFORM
export GO_VERSION
export OUTPUT_DIR
export PACKAGE_DIR

# Docker configuration
DOCKERFILE ?= Dockerfile.client
DOCKER_TAG ?= perconalab/pmm-client:$(PMM_VERSION)

export DOCKERFILE
export DOCKER_TAG

# Component lists
WORKSPACE_COMPONENTS := pmm-admin pmm-agent
EXTERNAL_COMPONENTS := node_exporter mysqld_exporter mongodb_exporter postgres_exporter \
                       proxysql_exporter rds_exporter azure_metrics_exporter redis_exporter \
                       vmagent nomad percona-toolkit
ALL_COMPONENTS := $(WORKSPACE_COMPONENTS) $(EXTERNAL_COMPONENTS)

# Build the pmm-builder Docker image
builder-image:
	@echo "Building pmm-builder image..."
	@docker build \
		--platform $(PLATFORM) \
		--build-arg GO_VERSION=$(GO_VERSION) \
		-t pmm-builder:latest \
		-f Dockerfile.builder \
		.
	@echo "pmm-builder image built successfully"

# Component validation
define check_component
	@if [ -z "$(COMPONENT)" ]; then \
		echo "Error: COMPONENT not specified"; \
		echo "Usage: make $(1) COMPONENT=<name>"; \
		echo ""; \
		echo "Available components:"; \
		echo "  $(ALL_COMPONENTS)"; \
		exit 1; \
	fi
endef

# Build a single component
build:
	$(call check_component,build)
	@$(BUILD_SCRIPT) $(COMPONENT)

# Build with dynamic linking
build-dynamic:
	$(call check_component,build-dynamic)
	@$(MAKE) build BUILD_TYPE=dynamic COMPONENT=$(COMPONENT)

# Build for ARM64
build-arm64:
	$(call check_component,build-arm64)
	@$(MAKE) build GOARCH=arm64 COMPONENT=$(COMPONENT)

# Build all components
build-all:
	@echo "Building all PMM components..."
	@mkdir -p $(OUTPUT_DIR)
	@for component in $(ALL_COMPONENTS); do \
		echo ""; \
		echo "================================================="; \
		echo "Building $$component..."; \
		echo "================================================="; \
		$(BUILD_SCRIPT) $$component || exit 1; \
	done
	@echo ""
	@echo "All components built successfully!"
	@echo "Artifacts available in: $(OUTPUT_DIR)"
	@ls -lh $(OUTPUT_DIR)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(OUTPUT_DIR)
	@echo "Clean complete"

# Clean volumes (use with caution - removes cache)
clean-volumes:
	@echo "Removing Docker volumes..."
	@docker volume rm pmm-mod pmm-build pmm-source 2>/dev/null || true
	@echo "Volumes removed"

# Generate tarball package
package-tarball:
	@echo "Creating pmm-client tarball package..."
	@$(PACKAGE_SCRIPT)
	@echo "Package created successfully!"

# Build PMM Client Docker image
build-client-docker:
	@echo "Building PMM Client Docker image..."
	@$(DOCKER_SCRIPT)
	@echo "Docker image built successfully!"

# Build and push PMM Client Docker image
push-client-docker:
	@echo "Building and pushing PMM Client Docker image..."
	@PUSH_DOCKER=1 $(DOCKER_SCRIPT)
	@echo "Docker image pushed successfully!"
