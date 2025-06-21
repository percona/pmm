# AI Chat Backend Makefile

# Variables
APP_NAME := aichat-backend
VERSION := 1.0.0
RELEASE := 1
ARCH := x86_64
GO_VERSION := 1.23
BINARY := $(APP_NAME)
SOURCE_DIR := $(shell pwd)
BUILD_DIR := $(SOURCE_DIR)/build
RPM_BUILD_DIR := $(BUILD_DIR)/rpm
TARBALL := $(APP_NAME)-$(VERSION).tar.gz

# Go build flags
GOOS := linux
GOARCH := amd64
CGO_ENABLED := 0
LDFLAGS := -w -s -X main.version=$(VERSION)

# Default target
.PHONY: all
all: build

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY)
	rm -f $(TARBALL)
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
	go test -v ./...

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run

# Build binary
.PHONY: build
build: deps
	@echo "Building $(BINARY)..."
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -a -installsuffix cgo -ldflags="$(LDFLAGS)" \
		-o $(BINARY) cmd/main.go

# Build binary for development
.PHONY: build-dev
build-dev:
	@echo "Building $(BINARY) for development..."
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) cmd/main.go

# Run the application
.PHONY: run
run: build-dev
	@echo "Running $(BINARY)..."
	./$(BINARY)

# Create source tarball for RPM
.PHONY: tarball
tarball: clean
	@echo "Creating source tarball $(TARBALL)..."
	mkdir -p $(BUILD_DIR)/$(APP_NAME)-$(VERSION)
	cp -r . $(BUILD_DIR)/$(APP_NAME)-$(VERSION)/
	cd $(BUILD_DIR)/$(APP_NAME)-$(VERSION) && \
		rm -rf .git .gitignore build && \
		find . -name "*.log" -delete && \
		find . -name "*~" -delete
	cd $(BUILD_DIR) && tar czf $(TARBALL) $(APP_NAME)-$(VERSION)
	mv $(BUILD_DIR)/$(TARBALL) ./
	@echo "Source tarball created: $(TARBALL)"

# Setup RPM build environment
.PHONY: rpm-setup
rpm-setup:
	@echo "Setting up RPM build environment..."
	mkdir -p $(RPM_BUILD_DIR)/{BUILD,BUILDROOT,RPMS,SOURCES,SPECS,SRPMS}
	cp $(TARBALL) $(RPM_BUILD_DIR)/SOURCES/
	cp $(APP_NAME).spec $(RPM_BUILD_DIR)/SPECS/

# Build RPM package
.PHONY: rpm
rpm: tarball rpm-setup
	@echo "Building RPM package..."
	rpmbuild --define "_topdir $(RPM_BUILD_DIR)" \
		--define "_version $(VERSION)" \
		--define "_release $(RELEASE)" \
		-ba $(RPM_BUILD_DIR)/SPECS/$(APP_NAME).spec
	@echo "RPM packages built:"
	@find $(RPM_BUILD_DIR)/RPMS -name "*.rpm" -type f
	@find $(RPM_BUILD_DIR)/SRPMS -name "*.rpm" -type f

# Build SRPM (source RPM) only
.PHONY: srpm
srpm: tarball rpm-setup
	@echo "Building source RPM..."
	rpmbuild --define "_topdir $(RPM_BUILD_DIR)" \
		--define "_version $(VERSION)" \
		--define "_release $(RELEASE)" \
		-bs $(RPM_BUILD_DIR)/SPECS/$(APP_NAME).spec
	@echo "Source RPM built:"
	@find $(RPM_BUILD_DIR)/SRPMS -name "*.rpm" -type f

# Install RPM package (requires sudo)
.PHONY: rpm-install
rpm-install: rpm
	@echo "Installing RPM package..."
	sudo rpm -ivh $(RPM_BUILD_DIR)/RPMS/$(ARCH)/$(APP_NAME)-$(VERSION)-$(RELEASE).*.$(ARCH).rpm

# Uninstall RPM package (requires sudo)
.PHONY: rpm-uninstall
rpm-uninstall:
	@echo "Uninstalling RPM package..."
	sudo rpm -e $(APP_NAME)

# Check RPM package contents
.PHONY: rpm-check
rpm-check: rpm
	@echo "Checking RPM package contents..."
	rpm -qpl $(RPM_BUILD_DIR)/RPMS/$(ARCH)/$(APP_NAME)-$(VERSION)-$(RELEASE).*.$(ARCH).rpm

# Install dependencies for development
.PHONY: dev-deps
dev-deps:
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

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
	mkdir -p $(BUILD_DIR)/binaries
	
	# Linux amd64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/binaries/$(APP_NAME)-linux-amd64 cmd/main.go
	
	# Linux arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/binaries/$(APP_NAME)-linux-arm64 cmd/main.go
	
	# macOS amd64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 \
		go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/binaries/$(APP_NAME)-darwin-amd64 cmd/main.go
	
	# macOS arm64
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
		go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/binaries/$(APP_NAME)-darwin-arm64 cmd/main.go

# Docker build
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

# Docker run
.PHONY: docker-run
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -p 3001:3001 \
		-e OPENAI_API_KEY=${OPENAI_API_KEY} \
		-v $(SOURCE_DIR)/config.yaml:/app/config.yaml \
		-v $(SOURCE_DIR)/mcp-servers.json:/app/mcp-servers.json \
		$(APP_NAME):$(VERSION)

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  build-dev      - Build for development"
	@echo "  build-all      - Cross-compile for multiple platforms"
	@echo "  run            - Build and run the application"
	@echo "  test           - Run tests"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  deps           - Download dependencies"
	@echo "  dev-deps       - Install development dependencies"
	@echo "  clean          - Clean build artifacts"
	@echo ""
	@echo "RPM targets:"
	@echo "  tarball        - Create source tarball"
	@echo "  rpm            - Build RPM package"
	@echo "  srpm           - Build source RPM"
	@echo "  rpm-setup      - Setup RPM build environment"
	@echo "  rpm-install    - Install RPM package (requires sudo)"
	@echo "  rpm-uninstall  - Uninstall RPM package (requires sudo)"
	@echo "  rpm-check      - Check RPM package contents"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Build and run Docker container"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  RELEASE=$(RELEASE)"
	@echo "  ARCH=$(ARCH)"

# Check if required tools are installed
.PHONY: check-tools
check-tools:
	@echo "Checking required tools..."
	@command -v go >/dev/null 2>&1 || { echo "Go is not installed"; exit 1; }
	@command -v rpmbuild >/dev/null 2>&1 || { echo "Warning: rpmbuild not found (needed for RPM building)"; }
	@command -v docker >/dev/null 2>&1 || { echo "Warning: docker not found (needed for Docker builds)"; }
	@echo "Go version: $$(go version)" 