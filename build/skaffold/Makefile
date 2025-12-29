# Makefile for PMM Client Skaffold builds
# Provides convenient targets for common build scenarios

.PHONY: help build build-static build-dynamic build-arm64 build-amd64 build-dev extract clean

# Default target
help:
	@echo "PMM Client Skaffold Build Targets:"
	@echo ""
	@echo "  make build          - Build PMM Client (static, amd64)"
	@echo "  make build-static   - Build static binaries (default)"
	@echo "  make build-dynamic  - Build dynamic binaries (with GSSAPI)"
	@echo "  make build-arm64    - Build for ARM64 architecture"
	@echo "  make build-amd64    - Build for AMD64 architecture"
	@echo "  make build-dev      - Build with race detector (development)"
	@echo "  make extract        - Extract build artifacts to $(RESULTS_DIR)"
	@echo "  make clean          - Clean build artifacts and images"
	@echo "  make test-build     - Test build and extract artifacts"
	@echo ""
	@echo "Environment variables:"
	@echo "  PMM_VERSION         - Override PMM version"
	@echo "  RESULTS_DIR         - Output directory (default: ../bin)"
	@echo ""

# Results directory
RESULTS_DIR ?= ../bin

VERSION_URL := https://raw.githubusercontent.com/Percona-Lab/pmm-submodules/refs/heads/v3/VERSION
# Fetch version from pmm-submodules repository
PMM_VERSION ?= $(shell curl -fsSL "${VERSION_URL}")
# Read version from VERSION file if not set
# PMM_VERSION ?= $(shell cat ../../VERSION 2>/dev/null || echo "dev")
export PMM_VERSION

# Base image selection based on CI environment
# Both local and CI builds use golang:latest by default
BASE_IMAGE ?= golang:latest
export BASE_IMAGE

# Default build (static, amd64)
build: build-static

# Static build
build-static:
	@echo "Building PMM Client (static, amd64)..."
	skaffold build

# Dynamic build (with GSSAPI support)
build-dynamic:
	@echo "Building PMM Client (dynamic, amd64)..."
	BUILD_TYPE=dynamic skaffold build --profile=dynamic

# ARM64 build
build-arm64:
	@echo "Building PMM Client (static, arm64)..."
	GOARCH=arm64 skaffold build --profile=arm64

# AMD64 build (explicit)
build-amd64:
	@echo "Building PMM Client (static, amd64)..."
	GOARCH=amd64 skaffold build --profile=amd64

# Development build with race detector
build-dev:
	@echo "Building PMM Client (development mode with race detector)..."
	BUILD_MODE=dev skaffold build --profile=dev

# Extract build artifacts from container
extract:
	@echo "Extracting build artifacts..."
	@mkdir -p $(RESULTS_DIR)
	@IMAGE_ID=$$(docker images pmm-client-builder --format "{{.ID}}" | head -1); \
	if [ -z "$$IMAGE_ID" ]; then \
		echo "Error: pmm-client-builder image not found. Run 'make build' first."; \
		exit 1; \
	fi; \
	echo "Extracting from image: $$IMAGE_ID"; \
	docker create --name pmm-extract $$IMAGE_ID; \
	docker cp pmm-extract:/build/output/. $(RESULTS_DIR)/; \
	docker rm pmm-extract
	@echo "Artifacts extracted to: $(RESULTS_DIR)"
	@ls -lh $(RESULTS_DIR)/

# Clean build artifacts and Docker images
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(RESULTS_DIR)
	@echo "Removing Docker images..."
	@docker images pmm-client-builder -q | xargs -r docker rmi -f || true
	@echo "Clean complete"

# Test build - build and extract in one command
test-build: build extract
	@echo "Build and extract complete!"
	@echo "Results in: $(RESULTS_DIR)"

# Build all variants (for CI/CD)
build-all: build-static build-dynamic build-arm64
	@echo "All build variants complete!"

# Development workflow - build, extract, and show contents
dev: build-dev extract
	@echo "Development build complete!"
	@echo "Binary contents:"
	@tar -tzf $(RESULTS_DIR)/pmm-client-*.tar.gz | grep -E "bin/(pmm-admin|pmm-agent)" || true
