# Makefile for building PMM components using Docker

.PHONY: $(MAKECMDGOALS)

SHELL := bash
.SHELLFLAGS := -o pipefail -c

# Determine directory paths regardless of where make is invoked from
PIPELINE_DIR := $(dir $(abspath $(firstword $(MAKEFILE_LIST))))
BUILD_ROOT := $(abspath $(PIPELINE_DIR)..)
# Workspace root is auto-detected for local dev (full repo checkout); left empty
# for CI/bootstrap builds where workspace components clone from the bare cache.
ifneq ($(wildcard $(BUILD_ROOT)/go.mod),)
WORKSPACE_DIR ?= $(BUILD_ROOT)
else ifneq ($(wildcard $(abspath $(BUILD_ROOT)/..)/go.mod),)
WORKSPACE_DIR ?= $(abspath $(BUILD_ROOT)/..)
else
WORKSPACE_DIR ?=
endif

# Build script location
BUILD_SCRIPT := $(PIPELINE_DIR)/scripts/build-component
PACKAGE_SCRIPT := $(PIPELINE_DIR)/scripts/package-tarball
CLIENT_DOCKER_SCRIPT := $(PIPELINE_DIR)/scripts/build-client-docker
OUTPUT_DIR ?= $(PIPELINE_DIR)/output
PACKAGE_DIR ?= $(PIPELINE_DIR)/package

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
	@echo "  make update-cache             - Fetch required refs and upload to Minio (fast, per-build)"
	@echo "  make sync-cache               - Full fetch + prune stale branches + upload to Minio (slow, periodic)"
	@echo "  make populate-cache           - Clone any missing bare repos from upstream (first-time setup)"
	@echo "  make clean-cache              - Remove local cache directory"
	@echo "  make clean-volumes            - Remove Docker cache volumes"
	@echo "  make clean-all                - Remove cache, volumes, and build artifacts"
	@echo ""
	@echo "Utility Targets:"
	@echo "  make builder-image            - Build the pmm-builder Docker image"
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
	@echo "  GOARCH            - Target architecture: amd64 or arm64 (default: host arch)"
	@echo "  PLATFORM          - Docker platform: linux/amd64 or linux/arm64 (default: host platform)"
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

ifeq ($(PMM_VERSION),)
$(error Fatal: PMM_VERSION is not set, exiting...)
endif
BUILD_TYPE ?= static
# Native platform of the build host — containers run natively for speed;
# Go cross-compiles for GOARCH independently.
HOST_ARCH := $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
GOARCH ?= $(HOST_ARCH)
PLATFORM ?= linux/$(HOST_ARCH)
SERVER_PLATFORMS ?= linux/amd64
GO_VERSION ?= 1.26
LOG_FILE ?= $(PIPELINE_DIR)/build.log

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

# Bare repository URLs used by populate-cache and update-cache.
# Keys are the bare directory names stored in REPO_CACHE_DIR; values are the upstream clone URLs.
# Server-side repos:
SERVER_REPO_URLS := \
	pmm.git=https://github.com/percona/pmm.git \
	pmm-dump.git=https://github.com/percona/pmm-dump.git \
	grafana.git=https://github.com/percona/grafana.git \
	VictoriaMetrics.git=https://github.com/VictoriaMetrics/VictoriaMetrics.git
# Client external component repos (vmagent re-uses VictoriaMetrics.git above):
CLIENT_REPO_URLS := \
	node_exporter.git=https://github.com/percona/node_exporter.git \
	mysqld_exporter.git=https://github.com/percona/mysqld_exporter.git \
	mongodb_exporter.git=https://github.com/percona/mongodb_exporter.git \
	postgres_exporter.git=https://github.com/percona/postgres_exporter.git \
	proxysql_exporter.git=https://github.com/percona/proxysql_exporter.git \
	rds_exporter.git=https://github.com/percona/rds_exporter.git \
	azure_metrics_exporter.git=https://github.com/percona/azure_metrics_exporter.git \
	redis_exporter.git=https://github.com/oliver006/redis_exporter.git \
	nomad.git=https://github.com/hashicorp/nomad.git \
	percona-toolkit.git=https://github.com/percona/percona-toolkit.git
ALL_REPO_URLS := $(SERVER_REPO_URLS) $(CLIENT_REPO_URLS)

# repo.git=REF pairs for targeted fetching in update-cache.
# VictoriaMetrics.git and pmm.git each appear more than once (multiple components share the same repo).
REPO_FETCH_REFS := \
	pmm.git=$(PMM_REF) \
	pmm.git=$(PMM_UI_REF) \
	pmm.git=$(PMM_DASHBOARDS_REF) \
	grafana.git=$(GRAFANA_REF) \
	VictoriaMetrics.git=$(VM_REF) \
	VictoriaMetrics.git=$(VMAGENT_REF) \
	pmm-dump.git=$(PMM_DUMP_REF) \
	node_exporter.git=$(NODE_EXPORTER_REF) \
	mysqld_exporter.git=$(MYSQLD_EXPORTER_REF) \
	mongodb_exporter.git=$(MONGODB_EXPORTER_REF) \
	postgres_exporter.git=$(POSTGRES_EXPORTER_REF) \
	proxysql_exporter.git=$(PROXYSQL_EXPORTER_REF) \
	rds_exporter.git=$(RDS_EXPORTER_REF) \
	azure_metrics_exporter.git=$(AZURE_METRICS_EXPORTER_REF) \
	redis_exporter.git=$(REDIS_EXPORTER_REF) \
	nomad.git=$(NOMAD_REF) \
	percona-toolkit.git=$(PERCONA_TOOLKIT_REF)

# S3 cache flags (reused across all artifact builds)
S3_CACHE_FROM := type=s3,region=us-east-1,bucket=$(MINIO_BUCKET),endpoint_url=$(MINIO_ENDPOINT),access_key_id=$(MINIO_ACCESS_KEY),secret_access_key=$(MINIO_SECRET_KEY),use_path_style=true
S3_CACHE_TO   := type=s3,region=us-east-1,bucket=$(MINIO_BUCKET),endpoint_url=$(MINIO_ENDPOINT),access_key_id=$(MINIO_ACCESS_KEY),secret_access_key=$(MINIO_SECRET_KEY),mode=max,use_path_style=true
# Docker configuration
CLIENT_DOCKERFILE ?= $(PIPELINE_DIR)/Dockerfile.client
CLIENT_DOCKER_TAG ?= perconalab/pmm-client:$(PMM_VERSION)
SERVER_DOCKERFILE ?= $(PIPELINE_DIR)/Dockerfile.server
SERVER_DOCKER_TAG ?= perconalab/pmm-server:$(PMM_VERSION)
BUILDX_BUILDER    ?= pmm-s3-builder
GOMOD_CACHE_VOL   ?= pmm-mod
BUILD_CACHE_VOL   ?= pmm-build
YARN_CACHE_VOL    ?= pmm-yarn
SERVER_OUTPUT_DIR := $(PIPELINE_DIR)output/server

export LOG_FILE
export PMM_VERSION
export BUILD_TYPE
export GOARCH
export PLATFORM
export SERVER_PLATFORMS
export GO_VERSION
export OUTPUT_DIR
export PACKAGE_DIR
export WORKSPACE_DIR
export MINIO_ENDPOINT
export MINIO_BUCKET
export CACHE_DIR
export REPO_CACHE_DIR
export CLIENT_DOCKERFILE
export CLIENT_DOCKER_TAG
export SERVER_DOCKERFILE
export SERVER_DOCKER_TAG
export PMM_URL
export PMM_REF
export PMM_UI_REF
export PMM_DASHBOARDS_REF
export GRAFANA_URL
export GRAFANA_REF
export VM_URL
export VM_REF
export PMM_DUMP_URL
export PMM_DUMP_REF
export NODE_EXPORTER_URL
export NODE_EXPORTER_REF
export MYSQLD_EXPORTER_URL
export MYSQLD_EXPORTER_REF
export MONGODB_EXPORTER_URL
export MONGODB_EXPORTER_REF
export POSTGRES_EXPORTER_URL
export POSTGRES_EXPORTER_REF
export PROXYSQL_EXPORTER_URL
export PROXYSQL_EXPORTER_REF
export RDS_EXPORTER_URL
export RDS_EXPORTER_REF
export AZURE_METRICS_EXPORTER_URL
export AZURE_METRICS_EXPORTER_REF
export REDIS_EXPORTER_URL
export REDIS_EXPORTER_REF
export VMAGENT_URL
export VMAGENT_REF
export NOMAD_URL
export NOMAD_REF
export PERCONA_TOOLKIT_URL
export PERCONA_TOOLKIT_REF

# Component lists
WORKSPACE_COMPONENTS := pmm-admin pmm-agent
EXTERNAL_COMPONENTS := node_exporter mysqld_exporter mongodb_exporter postgres_exporter \
                       proxysql_exporter rds_exporter azure_metrics_exporter redis_exporter \
                       vmagent nomad percona-toolkit
ALL_COMPONENTS := $(WORKSPACE_COMPONENTS) $(EXTERNAL_COMPONENTS)

# Build the pmm-builder Docker image
builder-image:
	@echo "Building pmm-builder image (linux/$(HOST_ARCH))..."
	@docker buildx build \
		--platform linux/$(HOST_ARCH) \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--progress plain \
		--load \
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

build-client:
ifeq ($(LOGGING),)
	@$(MAKE) LOGGING=1 build-client 2>&1 | tee -a "$(LOG_FILE)"
else
	@echo "Building PMM Client..."
	@echo ""
	@$(MAKE) build-client-components
	@echo ""
	@$(MAKE) build-client-tarball
	@echo ""
	@$(MAKE) build-client-docker
endif

# Build PMM Client components (all client components)
build-client-components: ensure-pmm-builder ensure-build-volumes
	@echo "Building PMM Client components..."
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
build-server:
ifeq ($(LOGGING),)
	@$(MAKE) LOGGING=1 build-server 2>&1 | tee -a "$(LOG_FILE)"
else
	@$(MAKE) update-cache
	@$(MAKE) build-server-components
	@$(MAKE) generate-version-json
	@echo "Assembling PMM Server Docker image..."
	@$(MAKE) build-server-docker
	@echo ""
	@echo "PMM Server built successfully!"
endif

# Build all components (client + server)
build-all:
ifeq ($(LOGGING),)
	@$(MAKE) LOGGING=1 build-all 2>&1 | tee -a "$(LOG_FILE)"
else
	@echo "Building all PMM components (client + server)..."
	@$(MAKE) build-client
	@echo ""
	@$(MAKE) build-server
	@echo ""
	@echo "All components built successfully!"
endif

# Convenience shortcuts
client: build-client

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

# Upload local bare-repo cache to Minio (no download, no fetch — just push what's on disk).
# Used by bootstrap after populate-cache on a fresh setup.
upload-cache: check-minio
	@echo "Uploading local cache to Minio..."
	@mc mirror --overwrite $(REPO_CACHE_DIR) pmm/$(MINIO_BUCKET)/$(MINIO_CACHE_PREFIX) || \
		(echo "Error: Failed to upload cache to Minio" && exit 1)
	@echo "Cache upload complete"

# Fetch upstream changes into local bare repos and upload the updated cache to Minio
update-cache: check-minio download-cache
	@echo "Fetching required refs from upstream..."
	@for item in $(REPO_FETCH_REFS); do \
		repo=$${item%%=*}; \
		ref=$${item##*=}; \
		dest="$(REPO_CACHE_DIR)/$$repo"; \
		[ -d "$$dest" ] || continue; \
		git -C "$$dest" config remote.origin.fetch '+refs/heads/*:refs/heads/*' 2>/dev/null || true; \
		echo "  $$repo: fetching $$ref..."; \
		git -C "$$dest" fetch origin "$$ref" || { echo "Error: failed to fetch $$ref from $$repo"; exit 1; }; \
	done
	@echo "Uploading updated cache to Minio..."
	@mc mirror --overwrite $(REPO_CACHE_DIR) pmm/$(MINIO_BUCKET)/$(MINIO_CACHE_PREFIX) || \
		(echo "Error: Failed to upload cache to Minio" && exit 1)
	@echo "Cache update complete"

# Full fetch + prune stale branches across all bare repos, then upload to Minio.
# Run periodically (e.g. on a CI schedule) to remove branches deleted from upstream.
# Slower than update-cache because it fetches all refs, not just the ones in .env.
sync-cache: check-minio download-cache
	@echo "Syncing all bare repos from upstream (fetch --prune)..."
	@for item in $(ALL_REPO_URLS); do \
		name=$${item%%=*}; \
		dest="$(REPO_CACHE_DIR)/$$name"; \
		[ -d "$$dest" ] || continue; \
		git -C "$$dest" config remote.origin.fetch '+refs/heads/*:refs/heads/*' 2>/dev/null || true; \
		echo "  $$name: fetching all branches, pruning stale..."; \
		git -C "$$dest" fetch --prune origin '+refs/heads/*:refs/heads/*' || { echo "Error: failed to sync $$name"; exit 1; }; \
	done
	@echo "Uploading pruned cache to Minio..."
	@mc mirror --overwrite $(REPO_CACHE_DIR) pmm/$(MINIO_BUCKET)/$(MINIO_CACHE_PREFIX) || \
		(echo "Error: Failed to upload cache to Minio" && exit 1)
	@echo "Sync complete"

# Clone any bare repos that are missing from the local cache directory.
# Skips repos that already exist. Useful for first-time setup or when adding new components.
# Run 'make update-cache' afterwards to fetch latest commits and push to Minio.
populate-cache:
	@echo "Populating bare-repo cache from upstream (skipping repos that already exist)..."
	@mkdir -p $(REPO_CACHE_DIR)
	@for item in $(ALL_REPO_URLS); do \
		name=$${item%%=*}; \
		url=$${item##*=}; \
		dest="$(REPO_CACHE_DIR)/$$name"; \
		if [ -d "$$dest" ]; then \
			echo "  $$name — already exists, skipping"; \
		else \
			echo "Cloning $$name from $$url ..."; \
			git clone --bare "$$url" "$$dest"; \
			git -C "$$dest" config remote.origin.fetch '+refs/heads/*:refs/heads/*'; \
			mkdir -p "$$dest/refs/heads" "$$dest/refs/tags" "$$dest/refs/remotes"; \
		fi; \
	done
	@echo "Cache populated. Run 'make update-cache' to fetch latest commits and push to Minio."

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
	@docker volume rm pmm-mod pmm-build pmm-yarn 2>/dev/null || true
	@echo "Volumes removed"

# Full clean (cache + volumes + output)
clean-all: clean-cache clean-volumes clean
	@echo "All caches and build artifacts cleaned"

# Build PMM Server Docker image (assembly stage only — references pre-built artifact images)
build-server-docker: ensure-builder
	@echo "Building PMM Server Docker image..."
	@if [ ! -f "$(PACKAGE_DIR)/pmm-client.tar.gz" ]; then \
		echo "Error: pmm-client tarball not found at $(PACKAGE_DIR)/pmm-client.tar.gz"; \
		echo "Run 'make build-client-tarball' first to create the tarball"; \
		exit 1; \
	fi
	@docker buildx build \
		--load \
		--builder $(BUILDX_BUILDER) \
		--platform $(SERVER_PLATFORMS) \
		--progress plain \
		--cache-from $(S3_CACHE_FROM) --cache-to $(S3_CACHE_TO) \
		--build-arg BUILD_DATE="$$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
		--build-arg VERSION="$(PMM_VERSION)" \
		-f $(SERVER_DOCKERFILE) \
		-t $(SERVER_DOCKER_TAG) \
		$(BUILD_ROOT)
	@docker buildx stop $(BUILDX_BUILDER) 2>/dev/null || true
	@echo "Docker image built successfully!"

# Build and push PMM Client Docker image
push-client-docker:
	@echo "Building and pushing PMM Client Docker image..."
	@PUSH_DOCKER=1 $(CLIENT_DOCKER_SCRIPT)
	@echo "Docker image pushed successfully!"

# Generate VERSION.json with all component git refs.
generate-version-json:
	@echo "Generating VERSION.json..."
	@$(PIPELINE_DIR)/scripts/generate-version-json

# Ensure a buildx builder with the docker-container driver exists and is running.
# The default "docker" driver does not support remote cache backends (s3, registry, etc.).
ensure-builder:
	@docker buildx inspect $(BUILDX_BUILDER) >/dev/null 2>&1 || \
		docker buildx create --name $(BUILDX_BUILDER) --driver docker-container
	@docker buildx inspect $(BUILDX_BUILDER) --bootstrap >/dev/null

# Ensure pmm-builder image exists (used for docker run-based component builds)
ensure-pmm-builder:
	@docker image inspect pmm-builder:latest >/dev/null 2>&1 || $(MAKE) builder-image

# Ensure Docker volumes exist for Go module, build, and Yarn caches
ensure-build-volumes:
	@docker volume inspect $(GOMOD_CACHE_VOL)  >/dev/null 2>&1 || docker volume create $(GOMOD_CACHE_VOL)
	@docker volume inspect $(BUILD_CACHE_VOL)  >/dev/null 2>&1 || docker volume create $(BUILD_CACHE_VOL)
	@docker volume inspect $(YARN_CACHE_VOL)   >/dev/null 2>&1 || docker volume create $(YARN_CACHE_VOL)

# ---------------------------------------------------------------------------
# Individual server component builds (each produces a local Docker image)
# ---------------------------------------------------------------------------

build-pmm-managed: ensure-pmm-builder ensure-build-volumes
	@echo "Building pmm-managed binaries..."
	@mkdir -p \
		$(SERVER_OUTPUT_DIR)/pmm-managed/bin \
		$(SERVER_OUTPUT_DIR)/pmm-managed/data/swagger \
		$(SERVER_OUTPUT_DIR)/pmm-managed/data/advisors \
		$(SERVER_OUTPUT_DIR)/pmm-managed/data/checks \
		$(SERVER_OUTPUT_DIR)/pmm-managed/data/alerting-templates
	@docker run --rm \
		--platform linux/$(HOST_ARCH) \
		-v $(REPO_CACHE_DIR)/pmm.git:/repos/pmm.git:ro \
		-v $(SERVER_OUTPUT_DIR)/pmm-managed:/output \
		-v $(GOMOD_CACHE_VOL):/go/pkg/mod \
		-v $(BUILD_CACHE_VOL):/root/.cache/go-build \
		-e GOCACHE=/root/.cache/go-build \
		-e GOARCH=$(GOARCH) \
		-e PMM_RELEASE_VERSION=$(PMM_VERSION) \
		pmm-builder:latest bash -c "\
			set -e; git clone /repos/pmm.git /build/pmm; \
			cd /build/pmm; git checkout $(PMM_REF); \
			make -C managed release GOARCH=$(GOARCH); \
			cd /build/pmm/qan-api2 && make release GOARCH=$(GOARCH); \
			cd /build/pmm/vmproxy && make release GOARCH=$(GOARCH); \
			cp /build/pmm/bin/pmm-managed /build/pmm/bin/pmm-encryption-rotation \
			   /build/pmm/bin/pmm-managed-init /build/pmm/bin/pmm-managed-starlark \
			   /build/pmm/bin/qan-api2 /build/pmm/bin/vmproxy /output/bin/; \
			cp -r /build/pmm/api/swagger/. /output/data/swagger/; \
			cp /build/pmm/managed/data/advisors/*.yml /output/data/advisors/; \
			cp /build/pmm/managed/data/checks/*.yml /output/data/checks/; \
			cp /build/pmm/managed/data/alerting-templates/*.yml /output/data/alerting-templates/"

build-pmm-dump: ensure-pmm-builder ensure-build-volumes
	@echo "Building pmm-dump binary..."
	@mkdir -p $(SERVER_OUTPUT_DIR)/pmm-dump
	@docker run --rm \
		--platform linux/$(HOST_ARCH) \
		-v $(REPO_CACHE_DIR)/pmm-dump.git:/repos/pmm-dump.git:ro \
		-v $(SERVER_OUTPUT_DIR)/pmm-dump:/output \
		-v $(GOMOD_CACHE_VOL):/go/pkg/mod \
		-v $(BUILD_CACHE_VOL):/root/.cache/go-build \
		-e GOCACHE=/root/.cache/go-build \
		-e GOARCH=$(GOARCH) \
		pmm-builder:latest bash -c "\
			set -e; git clone /repos/pmm-dump.git /build/pmm-dump; \
			cd /build/pmm-dump; git checkout $(PMM_DUMP_REF); \
			CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) make build; \
			cp pmm-dump /output/"

build-grafana-go: ensure-build-volumes
	@echo "Building Grafana Go binaries..."
	@mkdir -p $(SERVER_OUTPUT_DIR)/grafana-go
	@docker run --rm \
		--platform linux/$(GOARCH) \
		-v $(REPO_CACHE_DIR)/grafana.git:/repos/grafana.git:ro \
		-v $(SERVER_OUTPUT_DIR)/grafana-go:/output \
		-v $(GOMOD_CACHE_VOL):/go/pkg/mod \
		-v $(BUILD_CACHE_VOL):/root/.cache/go-build \
		-e GOCACHE=/root/.cache/go-build \
		-e GOARCH=$(GOARCH) \
		golang:$(GO_VERSION) bash -c "\
			set -e; git clone /repos/grafana.git /build/grafana; \
			cd /build/grafana; git checkout $(GRAFANA_REF); \
			sed -i 's/unknown-dev/11.6.4/' pkg/build/git.go; \
			make build-go; \
			cp bin/linux-$(GOARCH)/grafana-server bin/linux-$(GOARCH)/grafana bin/linux-$(GOARCH)/grafana-cli /output/"

build-grafana-ui: ensure-build-volumes
	@echo "Building Grafana UI assets..."
	@mkdir -p $(SERVER_OUTPUT_DIR)/grafana-ui
	@docker run --rm \
		--platform linux/$(HOST_ARCH) \
		-v $(REPO_CACHE_DIR)/grafana.git:/repos/grafana.git:ro \
		-v $(SERVER_OUTPUT_DIR)/grafana-ui:/output \
		-v $(YARN_CACHE_VOL):/yarn-cache \
		-e YARN_CACHE_FOLDER=/yarn-cache \
		-e YARN_GLOBAL_FOLDER=/yarn-cache \
		-e NODE_OPTIONS="--max-old-space-size=4096" \
		node:22 bash -c "\
			set -e; npm install -g grunt-cli; \
			git clone /repos/grafana.git /build/grafana; \
			cd /build/grafana; git checkout $(GRAFANA_REF); \
			make deps-js && make build-js; \
			cp -r public conf tools /output/"

build-victoriametrics: ensure-pmm-builder ensure-build-volumes
	@echo "Building VictoriaMetrics binaries..."
	@mkdir -p $(SERVER_OUTPUT_DIR)/victoriametrics
	@docker run --rm \
		--platform linux/$(HOST_ARCH) \
		-v $(REPO_CACHE_DIR)/VictoriaMetrics.git:/repos/VictoriaMetrics.git:ro \
		-v $(SERVER_OUTPUT_DIR)/victoriametrics:/output \
		-v $(GOMOD_CACHE_VOL):/go/pkg/mod \
		-v $(BUILD_CACHE_VOL):/root/.cache/go-build \
		-e GOCACHE=/root/.cache/go-build \
		-e GOARCH=$(GOARCH) \
		pmm-builder:latest bash -c "\
			set -e; git clone /repos/VictoriaMetrics.git /build/VictoriaMetrics; \
			cd /build/VictoriaMetrics; git checkout $(VM_REF); \
			make victoria-metrics-pure vmalert-pure PKG_TAG=$(VM_REF) BUILDINFO_TAG=$(VM_REF); \
			cp bin/victoria-metrics-pure bin/vmalert-pure /output/"

build-pmm-dashboards: ensure-build-volumes
	@echo "Building PMM dashboards assets..."
	@mkdir -p $(SERVER_OUTPUT_DIR)/pmm-dashboards
	@docker run --rm \
		--platform linux/$(HOST_ARCH) \
		-v $(REPO_CACHE_DIR)/pmm.git:/repos/pmm.git:ro \
		-v $(SERVER_OUTPUT_DIR)/pmm-dashboards:/output \
		-v $(YARN_CACHE_VOL):/yarn-cache \
		-e YARN_CACHE_FOLDER=/yarn-cache \
		-e YARN_GLOBAL_FOLDER=/yarn-cache \
		-e NODE_OPTIONS="--max-old-space-size=4096" \
		node:22 bash -c "\
			set -e; git clone /repos/pmm.git /build/pmm; \
			cd /build/pmm; git checkout $(PMM_DASHBOARDS_REF); \
			cd dashboards && make release; \
			cp -r panels /output/panels; \
			cp -r pmm-app/dist /output/pmm-app-dist"

build-pmm-ui: ensure-build-volumes
	@echo "Building PMM UI assets..."
	@mkdir -p $(SERVER_OUTPUT_DIR)/pmm-ui
	@docker run --rm \
		--platform linux/$(HOST_ARCH) \
		-v $(REPO_CACHE_DIR)/pmm.git:/repos/pmm.git:ro \
		-v $(SERVER_OUTPUT_DIR)/pmm-ui:/output \
		-v $(YARN_CACHE_VOL):/yarn-cache \
		-e YARN_CACHE_FOLDER=/yarn-cache \
		-e YARN_GLOBAL_FOLDER=/yarn-cache \
		-e NODE_OPTIONS="--max-old-space-size=4096" \
		node:22 bash -c "\
			set -e; git clone /repos/pmm.git /build/pmm-ui; \
			cd /build/pmm-ui; git checkout $(PMM_UI_REF); \
			cd ui && make release; \
			cp -r /build/pmm-ui/ui/apps/pmm/dist /output/pmm-dist; \
			cp -r /build/pmm-ui/ui/apps/pmm-compat/dist /output/pmm-compat-dist"

# ---------------------------------------------------------------------------
# Orchestration: Go builds in parallel, then Node builds sequentially
# ---------------------------------------------------------------------------

build-server-components:
	@echo "Building server component images (Go stages in parallel)..."
	@$(MAKE) -j4 build-pmm-managed build-pmm-dump build-victoriametrics build-grafana-go
	@echo ""
	@echo "Building server component images sequentially..."
	@$(PIPELINE_DIR)/scripts/check-ui-cache \
	    "grafana" "$(GRAFANA_REF)" \
	    "$(REPO_CACHE_DIR)/grafana.git" \
	    "$(SERVER_OUTPUT_DIR)/grafana-ui" \
	    "$(SERVER_OUTPUT_DIR)/version.json" \
	  && $(MAKE) build-grafana-ui || true
	@$(PIPELINE_DIR)/scripts/check-ui-cache \
	    "pmm-dashboards" "$(PMM_DASHBOARDS_REF)" \
	    "$(REPO_CACHE_DIR)/pmm.git" \
	    "$(SERVER_OUTPUT_DIR)/pmm-dashboards" \
	    "$(SERVER_OUTPUT_DIR)/version.json" \
	  && $(MAKE) build-pmm-dashboards || true
	@$(PIPELINE_DIR)/scripts/check-ui-cache \
	    "pmm" "$(PMM_UI_REF)" \
	    "$(REPO_CACHE_DIR)/pmm.git" \
	    "$(SERVER_OUTPUT_DIR)/pmm-ui" \
	    "$(SERVER_OUTPUT_DIR)/version.json" \
	  && $(MAKE) build-pmm-ui || true
	@echo "All server component images built successfully"

# Remove intermediate server artifact images
clean-server-artifacts:
	@echo "Removing server artifact images..."
	@docker rmi \
		pmm-build/pmm-dashboards:$(PMM_BUILD_TAG) \
		pmm-build/pmm-ui:$(PMM_BUILD_TAG) \
		2>/dev/null || true
	@rm -rf $(SERVER_OUTPUT_DIR)/{victoriametrics,pmm-managed,pmm-dump,grafana-go,grafana-ui}
	@echo "Server artifact images removed"
