# Makefile for building PMM components using Docker

.PHONY: help build build-all build-client build-client-tarball build-client-docker build-server client server all build-dynamic build-arm64 clean clean-cache clean-volumes clean-all builder-image build-server-docker push-client-docker gitmodules download-cache update-cache ensure-builder build-pmm-managed check-minio

# Determine directory paths regardless of where make is invoked from
PIPELINE_DIR := $(dir $(abspath $(firstword $(MAKEFILE_LIST))))
BUILD_ROOT := $(abspath $(PIPELINE_DIR)/..)

# Build script location
BUILD_SCRIPT := ./scripts/build-component
PACKAGE_SCRIPT := ./scripts/package-tarball
CLIENT_DOCKER_SCRIPT := ./scripts/build-client-docker
OUTPUT_DIR ?= $(CURDIR)/output
PACKAGE_DIR ?= $(CURDIR)/package

# Default target
help:
	@echo "PMM Build Targets:"
	@echo ""
	@echo "Main Targets:"
	@echo "  make build-client             - Build PMM Client components and Docker image"
	@echo "  make client-tarball           - Generate pmm-client tarball from built components"
	@echo "  make build-server             - Build PMM Server Docker image (downloads cache first)"
	@echo "  make build-all                - Build both client and server"
	@echo ""
	@echo "Component Build Targets:"
	@echo "  make build COMPONENT=<name>   - Build a specific component"
	@echo "  make build-dynamic COMPONENT=<name> - Build with dynamic linking (GSSAPI)"
	@echo "  make build-arm64 COMPONENT=<name>   - Build for ARM64"
	@echo ""
	@echo "Docker Image Targets:"
	@echo "  make build-client-docker      - Build PMM Client Docker image"
	@echo "  make build-server-docker      - Build PMM Server Docker image"
	@echo "  make push-client-docker       - Build and push PMM Client Docker image"
	@echo ""
	@echo "Cache Management:"
	@echo "  make download-cache           - Download repository cache from Minio"
	@echo "  make update-cache             - Fetch upstream changes and upload to Minio"
	@echo "  make clean-cache              - Remove local cache directory"
	@echo "  make clean-volumes            - Remove Docker cache volumes"
	@echo "  make clean-all                - Remove cache, volumes, and build artifacts"
	@echo ""
	@echo "Utility Targets:"
	@echo "  make builder-image            - Build the pmm-builder Docker image"
	@echo "  make gitmodules               - Build gitmodules parser binary"
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
	@echo "  PMM_VERSION       - Version to build (default: from VERSION file)"
	@echo "  BUILD_TYPE        - Build type: static or dynamic (default: static)"
	@echo "  GOARCH            - Target architecture: amd64 or arm64 (default: amd64)"
	@echo "  PLATFORM          - Docker platform: linux/amd64 or linux/arm64"
	@echo "  SERVER_PLATFORMS  - Server platforms: linux/amd64,linux/arm64 or linux/amd64 (default: linux/amd64)"
	@echo "  MINIO_ENDPOINT    - Minio endpoint (default: http://localhost:9000)"
	@echo "  MINIO_BUCKET      - Minio bucket name (default: cache)"
	@echo "  OUTPUT_DIR        - Output directory (default: ./output)"
	@echo ""
	@echo "Examples:"
	@echo "  make build COMPONENT=pmm-admin"
	@echo "  make build-dynamic COMPONENT=mongodb_exporter"
	@echo "  make build-arm64 COMPONENT=pmm-agent"
	@echo "  make server SERVER_PLATFORMS=linux/amd64"
	@echo "  make download-cache"
	@echo "  make build-all"
	@echo ""

# Configuration
include .env
export

PMM_VERSION ?= $(shell curl -fsSL https://raw.githubusercontent.com/Percona-Lab/pmm-submodules/refs/heads/v3/VERSION 2>/dev/null || echo "dev")
BUILD_TYPE ?= static
GOARCH ?= amd64
PLATFORM ?= linux/amd64
SERVER_PLATFORMS ?= linux/amd64
GO_VERSION ?= 1.25

# Minio S3 cache configuration
# On macOS/Docker Desktop use the magic hostname; on Linux query the bridge gateway.
_IS_DARWIN            := $(filter Darwin,$(shell uname -s))
_DOCKER_HOST_BRIDGE   := $(shell docker network inspect bridge --format='{{(index .IPAM.Config 0).Gateway}}' 2>/dev/null)
_DOCKER_HOST_INTERNAL := $(if $(_IS_DARWIN),host.docker.internal,$(if $(_DOCKER_HOST_BRIDGE),$(_DOCKER_HOST_BRIDGE),$(error Cannot determine Docker bridge gateway. Is Docker running?)))
MINIO_ENDPOINT ?= http://$(_DOCKER_HOST_INTERNAL):9000
MINIO_BUCKET ?= cache
MINIO_CACHE_PREFIX ?= repos
CACHE_DIR := $(CURDIR)/.cache
REPO_CACHE_DIR := $(CACHE_DIR)/repos

export PMM_VERSION
export BUILD_TYPE
export GOARCH
export PLATFORM
export SERVER_PLATFORMS
export GO_VERSION
export OUTPUT_DIR
export PACKAGE_DIR
export MINIO_ENDPOINT
export MINIO_BUCKET
export CACHE_DIR

# Docker configuration
CLIENT_DOCKERFILE ?= $(PIPELINE_DIR)/Dockerfile.client
CLIENT_DOCKER_TAG ?= perconalab/pmm-client:$(PMM_VERSION)
SERVER_DOCKERFILE ?= $(PIPELINE_DIR)/Dockerfile.server
SERVER_DOCKER_TAG ?= perconalab/pmm-server:$(PMM_VERSION)

export CLIENT_DOCKERFILE
export CLIENT_DOCKER_TAG
export SERVER_DOCKERFILE
export SERVER_DOCKER_TAG

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
		--progress plain \
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

# Build PMM Client (all client components + Docker image)
build-client:
	@echo "Building PMM Client..."
	@mkdir -p $(OUTPUT_DIR)
	@for component in $(ALL_COMPONENTS); do \
		echo ""; \
		echo "================================================="; \
		echo "Building $$component..."; \
		echo "================================================="; \
		$(BUILD_SCRIPT) $$component || exit 1; \
	done
	@echo "Artifacts available in: $(OUTPUT_DIR)"
	@ls -lh $(OUTPUT_DIR)
# Build PMM Client Docker image
build-client-docker:
	@echo "Building PMM Client Docker image..."
	@$(CLIENT_DOCKER_SCRIPT)
	@echo ""
	@echo "Docker image built successfully!"

# Generate PMM Client tarball package
build-client-tarball:
	@echo "Creating pmm-client tarball package..."
	@$(PACKAGE_SCRIPT)

# Build PMM Server (Docker image with all server components)
build-server: download-cache
	@echo "Building PMM Server..."
	@$(MAKE) build-server-docker
	@echo ""
	@echo "PMM Server built successfully!"

# Build all components (client + server)
build-all:
	@echo "Building all PMM components (client + server)..."
	@$(MAKE) build-client
	@echo ""
	@$(MAKE) build-server
	@echo ""
	@echo "All components built successfully!"

# Convenience shortcuts
client: build-client build-client-docker build-client-tarball

server: build-server

all: build-all

# Verify mc is installed and MinIO is reachable
check-minio:
	@if ! command -v mc >/dev/null 2>&1; then \
		echo "Error: mc (MinIO client) is not installed"; \
		echo "See: https://min.io/docs/minio/linux/reference/minio-mc.html"; \
		exit 1; \
	fi
	@echo "Checking MinIO connectivity at $(MINIO_ENDPOINT)..."
	@if ! mc ls pmm/$(MINIO_BUCKET) >/dev/null 2>&1; then \
		echo "Error: Cannot reach MinIO at $(MINIO_ENDPOINT)"; \
		echo "Make sure the MinIO container is running and the 'pmm' alias is configured:"; \
		echo "  mc alias set pmm $(MINIO_ENDPOINT) <access-key> <secret-key>"; \
		exit 1; \
	fi
	@echo "MinIO is reachable"

# Download repository cache from Minio (mandatory - fails if unavailable)
download-cache: check-minio
	@echo "Downloading repository cache from Minio..."
	@echo "Endpoint: $(MINIO_ENDPOINT)"
	@echo "Bucket: $(MINIO_BUCKET)/$(MINIO_CACHE_PREFIX)"
	@mkdir -p $(REPO_CACHE_DIR)
	@echo "Syncing cache from Minio..."
	@mc mirror --overwrite pmm/$(MINIO_BUCKET)/$(MINIO_CACHE_PREFIX) $(REPO_CACHE_DIR) || \
		(echo "Error: Failed to download cache from Minio" && exit 1)
	@echo "Fixing bare repository structure..."
	@for repo in $(REPO_CACHE_DIR)/*.git; do \
		mkdir -p "$$repo/refs/heads" "$$repo/refs/tags" "$$repo/refs/remotes"; \
	done
	@echo "Cache download complete"
	@echo "Cached repositories:"
	@ls -lh $(REPO_CACHE_DIR)

# Fetch upstream changes into local bare repos and upload the updated cache to Minio
update-cache: check-minio download-cache
	@echo "Fetching upstream changes for all cached repositories..."
	@for repo in $(REPO_CACHE_DIR)/*.git; do \
		echo "Updating $$repo..."; \
		git -C "$$repo" fetch --all --prune || { echo "Error: failed to fetch $$repo"; exit 1; }; \
	done
	@echo "Uploading updated cache to Minio..."
	@mc mirror --overwrite $(REPO_CACHE_DIR) pmm/$(MINIO_BUCKET)/$(MINIO_CACHE_PREFIX) || \
		(echo "Error: Failed to upload cache to Minio" && exit 1)
	@echo "Cache update complete"

# Clean local cache
clean-cache:
	@echo "Removing local cache..."
	@rm -rf $(CACHE_DIR)
	@echo "Local cache removed"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(OUTPUT_DIR)
	@echo "Clean complete"

# Clean Docker volumes
clean-volumes:
	@echo "Removing Docker volumes..."
	@docker volume rm pmm-mod pmm-build 2>/dev/null || true
	@echo "Volumes removed"

# Full clean (cache + volumes + output)
clean-all: clean-cache clean-volumes clean
	@echo "All caches and build artifacts cleaned"

# Build gitmodules parser binary
gitmodules:
	@echo "Building gitmodules parser..."
	@cd $(PIPELINE_DIR)/scripts && \
		go mod init github.com/percona/pmm/gitmodules 2>/dev/null || true && \
		go mod tidy && \
		go build -o gitmodules gitmodules.go && \
		rm -f go.mod go.sum
	@echo "gitmodules binary built successfully at scripts/gitmodules"

# Build PMM Server Docker image
build-server-docker:
	@echo "Building PMM Server Docker image..."
	@if [ -f "$(PACKAGE_DIR)/pmm-client-$(PMM_VERSION).tar.gz" ]; then \
		cp "$(PACKAGE_DIR)/pmm-client-$(PMM_VERSION).tar.gz" pmm-client.tar.gz; \
	else \
		echo "Warning: pmm-client tarball not found at $(PACKAGE_DIR)/pmm-client-$(PMM_VERSION).tar.gz"; \
		echo "Run 'make build-client-tarball' first to create the tarball"; \
	fi
	@git rev-parse HEAD > gitCommit
	@docker buildx build \
		--platform $(SERVER_PLATFORMS) \
		--progress plain \
		--build-arg BUILD_DATE="$$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
		--build-arg VERSION="$(PMM_VERSION)" \
		--build-arg PMM_REF="$(PMM_REF)" \
		--build-arg GRAFANA_REF="$(GRAFANA_REF)" \
		--build-arg VM_REF="$(VM_REF)" \
		--build-arg DASHBOARDS_REF="$(DASHBOARDS_REF)" \
		--build-arg PMM_DUMP_REF="$(PMM_DUMP_REF)" \
		-f $(SERVER_DOCKERFILE) \
		-t $(SERVER_DOCKER_TAG) \
		$(BUILD_ROOT)
	@rm -f pmm-client.tar.gz gitCommit
	@echo "Docker image built successfully!"

# Build and push PMM Client Docker image
push-client-docker:
	@echo "Building and pushing PMM Client Docker image..."
	@PUSH_DOCKER=1 $(CLIENT_DOCKER_SCRIPT)
	@echo "Docker image pushed successfully!"

BUILDX_BUILDER ?= pmm-s3-builder

# Ensure a buildx builder with the docker-container driver exists and is running.
# The default "docker" driver does not support remote cache backends (s3, registry, etc.).
ensure-builder:
	@docker buildx inspect $(BUILDX_BUILDER) >/dev/null 2>&1 || \
		docker buildx create --name $(BUILDX_BUILDER) --driver docker-container
	@docker buildx inspect $(BUILDX_BUILDER) --bootstrap >/dev/null

build-pmm-managed: ensure-builder check-minio update-cache download-cache
	@docker buildx build \
			--builder $(BUILDX_BUILDER) \
			--target pmm-managed-builder \
			--cache-from type=s3,region=us-east-1,bucket=$(MINIO_BUCKET),endpoint_url=$(MINIO_ENDPOINT),access_key_id=$(MINIO_ACCESS_KEY),secret_access_key=$(MINIO_SECRET_KEY),use_path_style=true \
			--cache-to   type=s3,region=us-east-1,bucket=$(MINIO_BUCKET),endpoint_url=$(MINIO_ENDPOINT),access_key_id=$(MINIO_ACCESS_KEY),secret_access_key=$(MINIO_SECRET_KEY),mode=max,use_path_style=true \
			--build-arg PMM_REF=$(PMM_REF) \
			--build-arg TARGETARCH=$(GOARCH) \
			--progress plain \
			-f $(SERVER_DOCKERFILE) \
			$(BUILD_ROOT)
	@docker buildx stop $(BUILDX_BUILDER)

check:
	@echo "Current directory: $(CURDIR)"
