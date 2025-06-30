# AI Chat Backend Makefile

# Variables
APP_NAME := aichat-backend
GO_VERSION := 1.23
BINARY := $(APP_NAME)
SOURCE_DIR := $(shell pwd)
BUILD_DIR := $(SOURCE_DIR)/bin

# PMM Release variables (following PMM managed pattern)
PMM_RELEASE_PATH ?= ../bin
PMM_RELEASE_VERSION ?= $(shell git describe --always | cut -b2-)
PMM_RELEASE_TIMESTAMP ?= $(shell date '+%s')
PMM_RELEASE_FULLCOMMIT ?= $(shell git rev-parse HEAD)
PMM_RELEASE_BRANCH ?= $(shell git describe --always --contains --all)

# PMM LD flags (following PMM managed pattern)
PMM_LD_FLAGS = -ldflags " \
			-X 'github.com/percona/pmm/version.ProjectName=aichat-backend' \
			-X 'github.com/percona/pmm/version.Version=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm/version.PMMVersion=$(PMM_RELEASE_VERSION)' \
			-X 'github.com/percona/pmm/version.Timestamp=$(PMM_RELEASE_TIMESTAMP)' \
			-X 'github.com/percona/pmm/version.FullCommit=$(PMM_RELEASE_FULLCOMMIT)' \
			-X 'github.com/percona/pmm/version.Branch=$(PMM_RELEASE_BRANCH)' \
			"

# Race flag detection (following PMM managed pattern)
IS_ARM        := $(filter arm64,$(shell uname -m))
PMM_RACE_FLAG ?= $(if $(IS_ARM),,-race)

# Test flags (following PMM managed pattern)
PMM_TEST_FLAGS ?= -timeout=180s
PMM_TEST_FILES ?= ./...

# Go build flags
GOOS := linux
GOARCH := amd64
CGO_ENABLED := 0

# Default target
.PHONY: all
all: build

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY)
	go clean -cache

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading Go dependencies..."
	go mod download
	go mod verify
	go mod tidy

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v $(PMM_TEST_FLAGS) $(PMM_TEST_FILES)

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run

# Build binary
.PHONY: build
build: deps
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -a -installsuffix cgo $(PMM_LD_FLAGS) \
		-o $(BUILD_DIR)/$(BINARY) main.go

# Build binary for development
.PHONY: build-dev
build-dev:
	@echo "Building $(BINARY) for development..."
	@mkdir -p $(BUILD_DIR)
	go build $(PMM_RACE_FLAG) -gcflags="all=-N -l" $(PMM_LD_FLAGS) -o $(BUILD_DIR)/$(BINARY) main.go

# Build release binaries (following PMM managed pattern)
.PHONY: release
release:
	@echo "Building release binaries..."
	env CGO_ENABLED=0 go build -v $(PMM_LD_FLAGS) -o $(PMM_RELEASE_PATH)/$(BINARY) main.go

# Build release binaries for development (following PMM managed pattern)
.PHONY: release-dev
release-dev:
	@echo "Building development release binaries..."
	go build $(PMM_RACE_FLAG) -gcflags="all=-N -l" -v $(PMM_LD_FLAGS) -o $(PMM_RELEASE_PATH)/$(BINARY) main.go

# Run the application
.PHONY: run
run: build-dev
	@echo "Running $(BINARY)..."
	$(BUILD_DIR)/$(BINARY)

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Generate Go code (if needed)
.PHONY: generate
generate:
	@echo "Generating Go code..."
	go generate ./...

# Cross-compile for multiple platforms
.PHONY: build-all
build-all:
	@echo "Cross-compiling for multiple platforms..."
	
	# Linux amd64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build $(PMM_LD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 main.go
	
	# Linux arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		go build $(PMM_LD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 main.go
	
	# macOS amd64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 \
		go build $(PMM_LD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 main.go
	
	# macOS arm64
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
		go build $(PMM_LD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 main.go

# Docker build
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(PMM_RELEASE_VERSION) .
	docker tag $(APP_NAME):$(PMM_RELEASE_VERSION) $(APP_NAME):latest

# Docker run
.PHONY: docker-run
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -p 3001:3001 \
		-e OPENAI_API_KEY=${OPENAI_API_KEY} \
		-v $(SOURCE_DIR)/config.yaml:/app/config.yaml \
		-v $(SOURCE_DIR)/mcp-servers.json:/app/mcp-servers.json \
		$(APP_NAME):$(PMM_RELEASE_VERSION)

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  build-dev      - Build for development"
	@echo "  build-all      - Cross-compile for multiple platforms"
	@echo "  release        - Build release binaries"
	@echo "  release-dev    - Build development release binaries"
	@echo "  run            - Build and run the application"
	@echo "  test           - Run tests"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  deps           - Download dependencies"
	@echo "  clean          - Clean build artifacts"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Build and run Docker container"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(PMM_RELEASE_VERSION)"

# Check if required tools are installed
.PHONY: check-tools
check-tools:
	@echo "Checking required tools..."
	@command -v go >/dev/null 2>&1 || { echo "Go is not installed"; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo "Warning: docker not found (needed for Docker builds)"; }
	@echo "Go version: $$(go version)" 