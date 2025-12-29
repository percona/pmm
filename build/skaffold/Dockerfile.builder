ARG BASE_IMAGE=golang:latest
FROM ${BASE_IMAGE} AS base-builder

# Install additional build dependencies
RUN apt-get update && apt-get install -y \
    zip \
    && rm -rf /var/lib/apt/lists/*

# Set up Go environment
ENV CGO_ENABLED=0
ENV GO111MODULE=auto
ENV GOMODCACHE=/go/pkg/mod

WORKDIR /workspace

# Cache Go modules by copying go.mod/go.sum first
COPY go.mod go.sum ./

# Download dependencies for workspace (will be cached in Docker layer and host volume)
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the entire PMM repository
COPY . .

# Build arguments - these will be passed from Skaffold
ARG PMM_VERSION
ARG FULL_PMM_VERSION  
ARG BUILD_TYPE
ARG GOOS=linux
ARG GOARCH=amd64

# Set GOOS and GOARCH from args (Go needs these)
ENV GOOS=${GOOS}
ENV GOARCH=${GOARCH}

# Create build directories
RUN mkdir -p /build/source /build/binary /build/output

# Production builder stage
FROM base-builder AS prod-builder

# Declare ARGs again in this stage so they're available
ARG PMM_VERSION
ARG FULL_PMM_VERSION
ARG BUILD_TYPE

# Run the build script
COPY build/skaffold/scripts/build-all-components.sh /scripts/build-all-components.sh
COPY build/skaffold/scripts/gitmodules.go /scripts/gitmodules.go
RUN chmod +x /scripts/build-all-components.sh

# Run script with variables only if they're set (not empty or <no value>)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/build/source/pmm-client-cache \
    set -e; \
    [ "${PMM_VERSION:-}" = "<no value>" ] && unset PMM_VERSION || true; \
    [ "${FULL_PMM_VERSION:-}" = "<no value>" ] && unset FULL_PMM_VERSION || true; \
    [ "${BUILD_TYPE:-}" = "<no value>" ] && unset BUILD_TYPE || true; \
    [ "${GOOS:-}" = "<no value>" ] && unset GOOS || export GOOS="${GOOS}"; \
    [ "${GOARCH:-}" = "<no value>" ] && unset GOARCH || export GOARCH="${GOARCH}"; \
    /scripts/build-all-components.sh

# Development builder stage (with race detector)
FROM base-builder AS dev-builder

# Declare ARGs again in this stage
ARG PMM_VERSION
ARG FULL_PMM_VERSION
ARG BUILD_TYPE

# Build with race detector for development
COPY build/skaffold/scripts/build-all-components.sh /scripts/build-all-components.sh
COPY build/skaffold/scripts/gitmodules.go /scripts/gitmodules.go
RUN chmod +x /scripts/build-all-components.sh

# Run script with variables only if they're set (not empty or <no value>)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/build/source/pmm-client-cache \
    set -e; \
    [ "${PMM_VERSION:-}" = "<no value>" ] && unset PMM_VERSION || true; \
    [ "${FULL_PMM_VERSION:-}" = "<no value>" ] && unset FULL_PMM_VERSION || true; \
    [ "${BUILD_TYPE:-}" = "<no value>" ] && unset BUILD_TYPE || true; \
    [ "${GOOS:-}" = "<no value>" ] && unset GOOS || export GOOS="${GOOS}"; \
    [ "${GOARCH:-}" = "<no value>" ] && unset GOARCH || export GOARCH="${GOARCH}"; \
    export BUILD_MODE=dev; \
    /scripts/build-all-components.sh

# Final stage - minimal image with binaries
FROM scratch AS artifacts

COPY --from=prod-builder /build/binary /build/binary
COPY --from=prod-builder /build/output /build/output
