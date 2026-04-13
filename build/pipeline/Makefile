# Makefile for building PMM components using Docker

.PHONY: $(MAKECMDGOALS)

SHELL := bash
.SHELLFLAGS := -o pipefail -c

# Determine directory paths regardless of where make is invoked from
PIPELINE_DIR := $(patsubst %/,%,$(dir $(abspath $(firstword $(MAKEFILE_LIST)))))
# Workspace root — set manually when building workspace components from a local
# repo checkout: make build COMPONENT=pmm-admin WORKSPACE_DIR=/path/to/pmm
# Left empty for CI/bootstrap builds (source is cloned from the bare cache).
WORKSPACE_DIR ?=

# Build script location
BUILD_SCRIPT := $(PIPELINE_DIR)/scripts/build-component
PACKAGE_SCRIPT := $(PIPELINE_DIR)/scripts/package-tarball
CLIENT_DOCKER_SCRIPT := $(PIPELINE_DIR)/scripts/build-client-docker
OUTPUT_DIR ?= $(PIPELINE_DIR)/output
PACKAGE_DIR ?= $(PIPELINE_DIR)/tarball

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
	@echo "  make populate-cache           - Clone any missing bare repos from upstream (first-time setup)"
	@echo "  make update-cache             - Fetch required refs from upstream into local bare repos"
	@echo "  make sync-cache               - Prune stale branches + fetch required refs"
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
	@echo "  OUTPUT_DIR        - Output directory (default: ./output)"
	@echo ""
	@echo "Examples:"
	@echo "  make build COMPONENT=pmm-admin"
	@echo "  make build-dynamic COMPONENT=mongodb_exporter"
	@echo "  make build-arm64 COMPONENT=pmm-agent"
	@echo "  make server SERVER_PLATFORMS=linux/amd64"
	@echo "  make populate-cache"
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

# Grafana plugin versions — sourced from the dashboards RPM spec.
DASHBOARDS_SPEC := $(PIPELINE_DIR)/packages/rpm/server/SPECS/percona-dashboards.spec
CLICKHOUSE_DATASOURCE_VERSION := $(shell sed -n 's/^%define clickhouse_datasource_version //p' $(DASHBOARDS_SPEC))
POLYSTAT_PANEL_VERSION        := $(shell sed -n 's/^%define polystat_panel_version *//p' $(DASHBOARDS_SPEC))

CACHE_DIR := $(CURDIR)/.cache
REPO_CACHE_DIR := $(CACHE_DIR)/repos
STAMP_DIR := $(CACHE_DIR)/stamps

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

# Docker configuration
CLIENT_DOCKERFILE ?= $(PIPELINE_DIR)/Dockerfile.client
CLIENT_DOCKER_TAG ?= perconalab/pmm-client:$(PMM_VERSION)
SERVER_DOCKERFILE ?= $(PIPELINE_DIR)/Dockerfile.server
SERVER_DOCKER_TAG ?= perconalab/pmm-server:$(PMM_VERSION)
BUILDX_BUILDER    ?= pmm-builder
GOMOD_CACHE_VOL   ?= pmm-mod
BUILD_CACHE_VOL   ?= pmm-build
YARN_CACHE_VOL    ?= pmm-yarn
SERVER_OUTPUT_DIR := $(PIPELINE_DIR)/output/server

export ALL_REPO_URLS
export REPO_FETCH_REFS
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
export CACHE_DIR
export REPO_CACHE_DIR
export STAMP_DIR
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
	@: > "$(LOG_FILE)"
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
	@: > "$(LOG_FILE)"
	@$(MAKE) LOGGING=1 build-server 2>&1 | tee -a "$(LOG_FILE)"
else
	@$(MAKE) update-cache
	@$(MAKE) build-server-components
	@$(MAKE) build-client-components
	@$(MAKE) build-client-tarball
	@$(MAKE) generate-version-json
	@echo "Assembling PMM Server Docker image..."
	@$(MAKE) build-server-docker
	@echo ""
	@echo "PMM Server built successfully!"
endif

# Build all components (client + server)
build-all:
ifeq ($(LOGGING),)
	@: > "$(LOG_FILE)"
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

# Fetch required refs from upstream into local bare repos.
# Runs populate-cache first if the cache directory is missing.
update-cache:
	@$(PIPELINE_DIR)/scripts/manage-cache update

# Prune stale branches from all bare repos, then fetch the refs listed in .env.
# Run periodically to remove branches deleted from upstream without fetching every branch.
sync-cache:
	@$(PIPELINE_DIR)/scripts/manage-cache sync

# Clone any bare repos that are missing from the local cache directory.
# Skips repos that already exist. Useful for first-time setup or when adding new components.
# Run 'make update-cache' afterwards to fetch latest commits.
populate-cache:
	@$(PIPELINE_DIR)/scripts/manage-cache populate

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
		--build-arg BUILD_DATE="$$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
		--build-arg VERSION="$(PMM_VERSION)" \
		-f $(SERVER_DOCKERFILE) \
		-t $(SERVER_DOCKER_TAG) \
		$(PIPELINE_DIR)
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
# The docker-container driver is required for multi-platform builds.
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
	@if $(PIPELINE_DIR)/scripts/check-build-cache \
	    "pmm-managed" "$(PMM_REF)" \
	    "$(REPO_CACHE_DIR)/pmm.git" \
	    "$(SERVER_OUTPUT_DIR)/pmm-managed" \
	    "$(STAMP_DIR)"; then \
	  echo "Building pmm-managed binaries..."; \
	  mkdir -p \
		$(SERVER_OUTPUT_DIR)/pmm-managed/bin \
		$(SERVER_OUTPUT_DIR)/pmm-managed/data/swagger \
		$(SERVER_OUTPUT_DIR)/pmm-managed/data/advisors \
		$(SERVER_OUTPUT_DIR)/pmm-managed/data/checks \
		$(SERVER_OUTPUT_DIR)/pmm-managed/data/alerting-templates; \
	  docker run --rm \
		--platform linux/$(HOST_ARCH) \
		-v $(REPO_CACHE_DIR)/pmm.git:/repos/pmm.git:ro \
		-v $(SERVER_OUTPUT_DIR)/pmm-managed:/output \
		-v $(GOMOD_CACHE_VOL):/go/pkg/mod \
		-v $(BUILD_CACHE_VOL):/root/.cache/go-build \
		-e GOCACHE=/root/.cache/go-build \
		-e GOARCH=$(GOARCH) \
		-e PMM_RELEASE_VERSION=$(PMM_VERSION) \
		pmm-builder:latest bash -c "\
			set -e; git config --global --add safe.directory /repos/pmm.git; \
			git clone /repos/pmm.git /build/pmm; \
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
			cp /build/pmm/managed/data/alerting-templates/*.yml /output/data/alerting-templates/"; \
	  hash=$$(git -C "$(REPO_CACHE_DIR)/pmm.git" rev-parse "$(PMM_REF)^{commit}" 2>/dev/null || echo "$(PMM_REF)"); \
	  mkdir -p "$(STAMP_DIR)"; echo "$$hash" > "$(STAMP_DIR)/pmm-managed.hash"; \
	fi

build-pmm-dump: ensure-pmm-builder ensure-build-volumes
	@if $(PIPELINE_DIR)/scripts/check-build-cache \
	    "pmm-dump" "$(PMM_DUMP_REF)" \
	    "$(REPO_CACHE_DIR)/pmm-dump.git" \
	    "$(SERVER_OUTPUT_DIR)/pmm-dump" \
	    "$(STAMP_DIR)"; then \
	  echo "Building pmm-dump binary..."; \
	  mkdir -p $(SERVER_OUTPUT_DIR)/pmm-dump; \
	  docker run --rm \
	    --platform linux/$(HOST_ARCH) \
	    -v $(REPO_CACHE_DIR)/pmm-dump.git:/repos/pmm-dump.git:ro \
	    -v $(SERVER_OUTPUT_DIR)/pmm-dump:/output \
	    -v $(GOMOD_CACHE_VOL):/go/pkg/mod \
	    -v $(BUILD_CACHE_VOL):/root/.cache/go-build \
	    -e GOCACHE=/root/.cache/go-build \
	    -e GOARCH=$(GOARCH) \
	    pmm-builder:latest bash -c "\
	      set -e; git config --global --add safe.directory /repos/pmm-dump.git; \
	      git clone /repos/pmm-dump.git /build/pmm-dump; \
	      cd /build/pmm-dump; git checkout $(PMM_DUMP_REF); \
	      CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) make build; \
	      cp pmm-dump /output/"; \
	  hash=$$(git -C "$(REPO_CACHE_DIR)/pmm-dump.git" rev-parse "$(PMM_DUMP_REF)^{commit}" 2>/dev/null || echo "$(PMM_DUMP_REF)"); \
	  mkdir -p "$(STAMP_DIR)"; echo "$$hash" > "$(STAMP_DIR)/pmm-dump.hash"; \
	fi

build-grafana-go: ensure-build-volumes
	@if $(PIPELINE_DIR)/scripts/check-build-cache \
	    "grafana-go" "$(GRAFANA_REF)" \
	    "$(REPO_CACHE_DIR)/grafana.git" \
	    "$(SERVER_OUTPUT_DIR)/grafana-go" \
	    "$(STAMP_DIR)"; then \
	  echo "Building Grafana Go binaries..."; \
	  mkdir -p $(SERVER_OUTPUT_DIR)/grafana-go; \
	  docker run --rm \
	    --platform linux/$(GOARCH) \
	    -v $(REPO_CACHE_DIR)/grafana.git:/repos/grafana.git:ro \
	    -v $(SERVER_OUTPUT_DIR)/grafana-go:/output \
	    -v $(GOMOD_CACHE_VOL):/go/pkg/mod \
	    -v $(BUILD_CACHE_VOL):/root/.cache/go-build \
	    -e GOCACHE=/root/.cache/go-build \
	    -e GOARCH=$(GOARCH) \
	    golang:$(GO_VERSION) bash -c "\
	      set -e; git config --global --add safe.directory /repos/grafana.git; \
	      git clone /repos/grafana.git /build/grafana; \
	      cd /build/grafana; git checkout $(GRAFANA_REF); \
	      sed -i 's/unknown-dev/11.6.4/' pkg/build/git.go; \
	      make build-go; \
	      cp bin/linux-$(GOARCH)/grafana-server bin/linux-$(GOARCH)/grafana bin/linux-$(GOARCH)/grafana-cli /output/"; \
	  hash=$$(git -C "$(REPO_CACHE_DIR)/grafana.git" rev-parse "$(GRAFANA_REF)^{commit}" 2>/dev/null || echo "$(GRAFANA_REF)"); \
	  mkdir -p "$(STAMP_DIR)"; echo "$$hash" > "$(STAMP_DIR)/grafana-go.hash"; \
	fi

build-grafana-ui: ensure-build-volumes
	@if $(PIPELINE_DIR)/scripts/check-build-cache \
	    "grafana-ui" "$(GRAFANA_REF)" \
	    "$(REPO_CACHE_DIR)/grafana.git" \
	    "$(SERVER_OUTPUT_DIR)/grafana-ui" \
	    "$(STAMP_DIR)"; then \
	  echo "Building Grafana UI assets..."; \
	  mkdir -p $(SERVER_OUTPUT_DIR)/grafana-ui; \
	  docker run --rm \
	    --platform linux/$(HOST_ARCH) \
	    -v $(REPO_CACHE_DIR)/grafana.git:/repos/grafana.git:ro \
	    -v $(SERVER_OUTPUT_DIR)/grafana-ui:/output \
	    -v $(YARN_CACHE_VOL):/yarn-cache \
	    -v $(PIPELINE_DIR)/scripts/entrypoints/build-grafana-ui:/entrypoint.sh:ro \
	    -e YARN_CACHE_FOLDER=/yarn-cache \
	    -e YARN_GLOBAL_FOLDER=/yarn-cache \
	    -e NODE_OPTIONS="--max-old-space-size=8192" \
	    -e REF=$(GRAFANA_REF) \
	    node:22 bash /entrypoint.sh; \
	  hash=$$(git -C "$(REPO_CACHE_DIR)/grafana.git" rev-parse "$(GRAFANA_REF)^{commit}" 2>/dev/null || echo "$(GRAFANA_REF)"); \
	  mkdir -p "$(STAMP_DIR)"; echo "$$hash" > "$(STAMP_DIR)/grafana-ui.hash"; \
	fi

build-victoriametrics: ensure-pmm-builder ensure-build-volumes
	@if $(PIPELINE_DIR)/scripts/check-build-cache \
	    "victoriametrics" "$(VM_REF)" \
	    "$(REPO_CACHE_DIR)/VictoriaMetrics.git" \
	    "$(SERVER_OUTPUT_DIR)/victoriametrics" \
	    "$(STAMP_DIR)"; then \
	  echo "Building VictoriaMetrics binaries..."; \
	  mkdir -p $(SERVER_OUTPUT_DIR)/victoriametrics; \
	  docker run --rm \
	    --platform linux/$(HOST_ARCH) \
	    -v $(REPO_CACHE_DIR)/VictoriaMetrics.git:/repos/VictoriaMetrics.git:ro \
	    -v $(SERVER_OUTPUT_DIR)/victoriametrics:/output \
	    -v $(GOMOD_CACHE_VOL):/go/pkg/mod \
	    -v $(BUILD_CACHE_VOL):/root/.cache/go-build \
	    -e GOCACHE=/root/.cache/go-build \
	    -e GOARCH=$(GOARCH) \
	    pmm-builder:latest bash -c "\
	      set -e; git config --global --add safe.directory /repos/VictoriaMetrics.git; \
	      git clone /repos/VictoriaMetrics.git /build/VictoriaMetrics; \
	      cd /build/VictoriaMetrics; git checkout $(VM_REF); \
	      make victoria-metrics-pure vmalert-pure PKG_TAG=$(VM_REF) BUILDINFO_TAG=$(VM_REF); \
	      cp bin/victoria-metrics-pure bin/vmalert-pure /output/"; \
	  hash=$$(git -C "$(REPO_CACHE_DIR)/VictoriaMetrics.git" rev-parse "$(VM_REF)^{commit}" 2>/dev/null || echo "$(VM_REF)"); \
	  mkdir -p "$(STAMP_DIR)"; echo "$$hash" > "$(STAMP_DIR)/victoriametrics.hash"; \
	fi

build-pmm-dashboards: ensure-build-volumes
	@if $(PIPELINE_DIR)/scripts/check-build-cache \
	    "pmm-dashboards" "$(PMM_DASHBOARDS_REF)" \
	    "$(REPO_CACHE_DIR)/pmm.git" \
	    "$(SERVER_OUTPUT_DIR)/pmm-dashboards" \
	    "$(STAMP_DIR)"; then \
	  echo "Building PMM dashboards assets..."; \
	  mkdir -p $(SERVER_OUTPUT_DIR)/pmm-dashboards; \
	  docker run --rm \
	    --platform linux/$(HOST_ARCH) \
	    -v $(REPO_CACHE_DIR)/pmm.git:/repos/pmm.git:ro \
	    -v $(SERVER_OUTPUT_DIR)/pmm-dashboards:/output \
	    -v $(PIPELINE_DIR)/scripts/entrypoints/build-dashboards:/entrypoint.sh:ro \
	    -v $(YARN_CACHE_VOL):/yarn-cache \
	    -e YARN_CACHE_FOLDER=/yarn-cache \
	    -e YARN_GLOBAL_FOLDER=/yarn-cache \
	    -e NODE_OPTIONS="--max-old-space-size=8192" \
	    -e REF=$(PMM_DASHBOARDS_REF) \
	    -e CLICKHOUSE_DATASOURCE_VERSION=$(CLICKHOUSE_DATASOURCE_VERSION) \
	    -e POLYSTAT_PANEL_VERSION=$(POLYSTAT_PANEL_VERSION) \
	    -e HOST_ARCH=$(HOST_ARCH) \
	    node:22 bash /entrypoint.sh; \
	  hash=$$(git -C "$(REPO_CACHE_DIR)/pmm.git" rev-parse "$(PMM_DASHBOARDS_REF)^{commit}" 2>/dev/null || echo "$(PMM_DASHBOARDS_REF)"); \
	  mkdir -p "$(STAMP_DIR)"; echo "$$hash" > "$(STAMP_DIR)/pmm-dashboards.hash"; \
	fi

build-pmm-ui: ensure-build-volumes
	@if $(PIPELINE_DIR)/scripts/check-build-cache \
	    "pmm-ui" "$(PMM_UI_REF)" \
	    "$(REPO_CACHE_DIR)/pmm.git" \
	    "$(SERVER_OUTPUT_DIR)/pmm-ui" \
	    "$(STAMP_DIR)"; then \
	  echo "Building PMM UI assets..."; \
	  mkdir -p $(SERVER_OUTPUT_DIR)/pmm-ui; \
	  docker run --rm \
	    --platform linux/$(HOST_ARCH) \
	    -v $(REPO_CACHE_DIR)/pmm.git:/repos/pmm.git:ro \
	    -v $(SERVER_OUTPUT_DIR)/pmm-ui:/output \
	    -v $(YARN_CACHE_VOL):/yarn-cache \
	    -e YARN_CACHE_FOLDER=/yarn-cache \
	    -e YARN_GLOBAL_FOLDER=/yarn-cache \
	    -v $(PIPELINE_DIR)/scripts/entrypoints/build-ui:/entrypoint.sh:ro \
	    -e NODE_OPTIONS="--max-old-space-size=8192" \
	    -e REF=$(PMM_UI_REF) \
	    node:22 bash /entrypoint.sh; \
	  hash=$$(git -C "$(REPO_CACHE_DIR)/pmm.git" rev-parse "$(PMM_UI_REF)^{commit}" 2>/dev/null || echo "$(PMM_UI_REF)"); \
	  mkdir -p "$(STAMP_DIR)"; echo "$$hash" > "$(STAMP_DIR)/pmm-ui.hash"; \
	fi

# ---------------------------------------------------------------------------
# Orchestration: Go builds in parallel, then Node builds sequentially
# ---------------------------------------------------------------------------

build-server-components:
	@echo "Building server component images (Go stages in parallel)..."
	@$(MAKE) -j4 build-pmm-managed build-pmm-dump build-victoriametrics build-grafana-go
	@echo ""
	@echo "Building server component images (Node stages sequentially)..."
	@$(MAKE) build-grafana-ui
	@$(MAKE) build-pmm-dashboards
	@$(MAKE) build-pmm-ui
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
