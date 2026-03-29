# PMM Build Pipeline - AI Agent Guidelines

This document provides comprehensive guidelines for AI agents working with the PMM build pipeline. The build pipeline is a Docker-based system for building PMM components.

## Architecture Overview

The build pipeline uses `docker run` for all server component builds, outputting artifacts directly to the host via Docker volumes. Client components are built with a custom `pmm-builder` image. Different images are used for different server components depending on their requirements.

### Core Components

1. **Dockerfile.builder** - Defines the `pmm-builder` image (golang-based) for Go component builds
2. **Dockerfile.server** - Assembly-only Dockerfile; copies pre-built artifacts from host paths
3. **scripts/build-component** - Main build orchestration script for client builds
4. **scripts/gitmodules.go** - Parser for .gitmodules configuration
5. **Makefile** - Build targets and convenience commands
6. **README.md** - User-facing documentation

> **Note:** All server components are built via `docker run` (no component Dockerfiles). Each
> `build-artifact-*` target in the Makefile runs a container, clones the source from a local
> bare-repo cache, builds, and copies artifacts to `output/server/<component>/` on the host.
> Dockerfile.server then assembles the runtime image from those host-side paths.

### Design Principles

- **Volume Caching** - Use Docker volumes for Go modules and build artifacts
- **Single Source of Truth** - Component metadata from pmm-submodules/.gitmodules; `GO_VERSION` defined once in Makefile and passed as `--build-arg` to all component Dockerfiles
- **Explicit REF args** - All `*_REF` build args in component Dockerfiles have no defaults; they must be passed via `--build-arg`. Omitting one causes an immediate build failure at `git checkout`
- **Split server builds** - All server components are built independently via `docker run`, writing artifacts to `output/server/<component>/` on the host:
  - **pmm-managed, pmm-dump, VictoriaMetrics**: `pmm-builder:latest` (pure Go, `CGO_ENABLED=0`), run at `HOST_ARCH` for native speed; Go cross-compiles for `GOARCH`
  - **grafana-go**: `golang:$(GO_VERSION)` (CGO enabled, has gcc for `go-sqlite3`), runs at `--platform linux/$(GOARCH)` so sqlite3's C code compiles natively for the target arch
  - **grafana-ui, pmm-dashboards, pmm-ui**: `node:22` (git included), run at `HOST_ARCH`; Yarn cache at `/usr/local/share/.cache/yarn` shared via `YARN_CACHE_VOL`
  - **BuildKit** (`docker buildx build`) is used **only** for the final `build-server-docker` step (OracleLinux runtime image assembly with S3 layer cache)
  - Node components build sequentially to avoid Yarn network saturation
- **Platform Awareness** - Explicit --platform flags to avoid warnings
- **Minimal Containers** - Run as root in golang image, no permission issues

## Key Files and Their Roles

### Dockerfile.builder

```dockerfile
ARG GO_VERSION=latest
FROM golang:${GO_VERSION}
```

**Purpose**: Define the build environment  
**Key Points**:
- Based on official golang image (currently uses `latest` by default)
- Installs build dependencies: zip (for nomad), and some other tools (like gssapi-dev for dynamic builds)
- Sets default Go environment variables
- Runs as root (no user permission issues)

**When to modify**:
- Adding new build tools needed by components
- Changing base Go version (via GO_VERSION build arg)

### scripts/build-component

**Purpose**: Main build orchestration  
**Key Functions**:
- `build_builder_image()` - Ensures pmm-builder image exists
- `create_volumes()` - Creates Docker volumes for caching
- `setup_gitmodules()` - Downloads .gitmodules and builds parser
- `build_workspace_component()` - Builds pmm-admin/pmm-agent
- `build_external_component()` - Clones and builds external components
- `get_component_info()` - Fetches metadata from .gitmodules

**Key Variables**:
```bash
BUILDER_IMAGE="pmm-builder:latest"
GOMOD_CACHE_VOL="pmm-mod"        # Go module cache
BUILD_CACHE_VOL="pmm-build"      # Build artifacts cache
PLATFORM="${PLATFORM:-linux/amd64}"
```

**When to modify**:
- Adding new workspace components (update case statements)
- Adding new external components (update component lists and build commands)
- Changing Docker volume paths
- Modifying build environment variables

### scripts/gitmodules.go

**Purpose**: Parse .gitmodules INI file  
**Usage**: `./gitmodules <file> <component> <field>`  
**Returns**: URL or branch/tag for a component

**When to modify**:
- Changing .gitmodules location or format
- Adding new fields to parse

### scripts/package-tarball

**Purpose**: Generate pmm-client distribution tarball  
**Output**: `pmm-client-${VERSION}.tar.gz` in `PACKAGE_DIR`  
**Dependencies**: Requires built components in `OUTPUT_DIR`

**Archive Structure**:
```
pmm-client-${VERSION}/
├── bin/               # All built binaries
├── config/            # systemd service files
├── debian/            # Debian packaging files
├── rpm/               # RPM spec files
├── queries-*.yml      # Query examples (if present)
├── install_tarball    # Installation script
└── VERSION            # Version identifier
```

**Key Variables**:
```bash
OUTPUT_DIR="${OUTPUT_DIR:-./output}"     # Source binaries
PACKAGE_DIR="${PACKAGE_DIR:-./package}"  # Output location
PMM_VERSION="${PMM_VERSION}"             # Version string
```

**When to modify**:
- Adding new files to distribution
- Changing directory structure
- Modifying VERSION file format
- Adjusting query file locations

### Makefile

**Purpose**: User-facing build targets  
**Key Targets**:
- `build` - Build single component (requires COMPONENT=)
- `build-all` - Build all components sequentially
- `build-dynamic` - Build with dynamic linking (GSSAPI)
- `build-arm64` - Build for ARM64 architecture
- `builder-image` - Build pmm-builder Docker image
- `package-tarball` - Generate pmm-client distribution archive
- `clean` - Remove output directory
- `clean-volumes` - Remove Docker volumes (cache)

**Key Variables**:
```makefile
WORKSPACE_COMPONENTS := pmm-admin pmm-agent
EXTERNAL_COMPONENTS := node_exporter mysqld_exporter ...
GO_VERSION ?= 1.26
HOST_ARCH  := $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')  # native host arch
GOARCH     ?= amd64          # target architecture for compiled binaries
PLATFORM   ?= linux/amd64   # used by client builds
GOMOD_CACHE_VOL ?= pmm-mod  # Docker volume: Go module cache
BUILD_CACHE_VOL ?= pmm-build # Docker volume: Go build cache
YARN_CACHE_VOL  ?= pmm-yarn  # Docker volume: Yarn package cache
SERVER_OUTPUT_DIR := $(PIPELINE_DIR)output/server
OUTPUT_DIR  ?= ./output
PACKAGE_DIR ?= ./package
```

**When to modify**:
- Adding new components (update component lists)
- Adding new build targets
- Changing default values

## Common Development Patterns

### Adding a New Workspace Component

1. Add to `WORKSPACE_COMPONENTS` in Makefile:
```makefile
WORKSPACE_COMPONENTS := pmm-admin pmm-agent new-component
```

2. Add to component list in `build-component`:
```bash
WORKSPACE_COMPONENTS="pmm-admin pmm-agent new-component"
```

3. Add subdirectory mapping in `build_workspace_component()`:
```bash
case "${component}" in
    pmm-admin) subdir="admin" ;;
    pmm-agent) subdir="agent" ;;
    new-component) subdir="newcomponent" ;;
```

4. Add build command in `build_workspace_component()`:
```bash
case "${component}" in
    new-component)
        build_cmd="make -C /workspace/${subdir} release PMM_RELEASE_PATH=/output"
        ;;
```

### Adding a New External Component

1. Add to `EXTERNAL_COMPONENTS` in both Makefile and `build-component`

2. Add to `.gitmodules` in pmm-submodules repository (preferred), OR add fallback in `get_component_info()`:
```bash
case "${component}_${field}" in
    new_exporter_url) echo "https://github.com/..." ;;
    new_exporter_branch) echo "main" ;;
```

3. Add build command in `build_external_component()`:
```bash
case "${component}" in
    new_exporter)
        build_cmd="make build && cp new_exporter /output/"
        ;;
```

### Modifying Docker Volumes

Volume paths are critical for caching:

**Both workspace and external components**:
```bash
-v "${GOMOD_CACHE_VOL}:/go/pkg/mod" \
-v "${BUILD_CACHE_VOL}:/root/.cache/go-build" \
-e GOCACHE=/root/.cache/go-build \
```

**External components also mount source cache**:
```bash
-v "${SOURCE_CACHE_VOL}:/build/source" \
```

**Important**: External components clone into `/build/source/${component}` (component-specific subdirectories to avoid conflicts). The volume persists repositories between builds, using `git clean -fdx` to ensure clean state.

## Critical Conventions

### Do's

- **Always use --platform flag** in docker run/build commands
- **Export variables** in Makefile that need to be passed to build-component
- **Keep component lists in sync** between Makefile and build-component script
- **Use realpath** for path resolution (portable across systems)
- **Check for component existence** before building
- **Use case statements** instead of associative arrays (broader shell compatibility)
- **Log informative messages** during build steps
- **Update the README.md** when adding/modifying components or build steps

### Don'ts

- **Don't hardcode version numbers** - use `latest` for flexibility
- **Don't use --user flag** - golang image runs as root, no permission issues
- **Don't modify .gitmodules locally** - it's fetched from pmm-submodules
- **Don't assume volumes are writable** - they're created as root, but golang image handles this
- **Don't use complex shell features** - keep it POSIX-compatible when possible
- **Don't nest Make calls** unnecessarily - use $(call) for reusable functions

### Error Handling

```bash
# Always check for required variables
PMM_VERSION="${PMM_VERSION:?No PMM_VERSION specified}"

# Provide clear error messages
if [ -z "$url" ] || [ -z "$ref" ]; then
    echo "Error: Could not determine URL or ref for ${component}" >&2
    return 1
fi

# Exit on errors
set -o errexit
set -o pipefail
set -o nounset
```

### Build Commands

**Workspace components** use their native Makefiles:
```bash
make -C /workspace/admin release PMM_RELEASE_PATH=/output
```

**External components** follow their own build systems:
```bash
make release && cp binary /output/
```

## Docker Platform Handling

Always specify platform to avoid warnings on Apple Silicon:

```bash
docker run --rm \
    --platform "${PLATFORM}" \
    ...

docker build \
    --platform "${PLATFORM}" \
    ...
```

Platform is auto-detected in Makefile but can be overridden:
```bash
PLATFORM=linux/arm64 make build COMPONENT=pmm-admin
```

## Caching Strategy

Four Docker volumes provide caching:

1. **pmm-mod** (`/go/pkg/mod`) - Go modules, shared across all Go builds
2. **pmm-build** (`/root/.cache/go-build`) - Go build cache, shared across all Go builds
3. **pmm-yarn** (`/usr/local/share/.cache/yarn`) - Yarn package cache for Node.js server builds
   (grafana-ui, pmm-dashboards, pmm-ui)
4. **pmm-source** (`/build/source`) - Git repository clones for **client** component builds
   (managed by the `build-component` script, not used by server `build-artifact-*` targets)

Client external components use smart clone/update logic:
- First build: Clone the repository
- Subsequent builds: Run `git clean -fdx`, fetch latest, and checkout
- Each component gets its own subdirectory: `/build/source/${component}`

Server builds clone sources at build time from read-only bare-repo mounts
(`$(REPO_CACHE_DIR)/<repo>.git`) and do not persist the working tree between runs.

Cache is persistent across builds. Clear with:
```bash
make clean-volumes  # Warning: destroys all caches!
```

## Component Metadata

Component URLs and refs come from pmm-submodules/.gitmodules:
```ini
[submodule "sources/mysqld_exporter"]
    path = sources/mysqld_exporter
    url = https://github.com/percona/mysqld_exporter
    branch = main
```

Fallback values in `get_component_info()` for components not in .gitmodules.

## Make Target Patterns

### Reusable Functions

Use Make's `define` for DRY:
```makefile
define check_component
    @if [ -z "$(COMPONENT)" ]; then \
        echo "Error: COMPONENT not specified"; \
        exit 1; \
    fi
endef

build:
    $(call check_component)
    @$(BUILD_SCRIPT) $(COMPONENT)
```

### Target Dependencies

```makefile
build: volumes  # Ensure volumes exist before building
```

## Testing Changes

1. **Build the builder image**:
```bash
make builder-image
```

2. **Test single component**:
```bash
make build COMPONENT=pmm-admin
```

3. **Verify artifacts**:
```bash
ls -lh output/
```

4. **Test cache persistence**:
```bash
make build COMPONENT=pmm-admin  # Should be fast on second run
```

## Troubleshooting

### "No PMM_VERSION specified"

Set in Makefile or environment:
```bash
PMM_VERSION=3.0.0 make build COMPONENT=pmm-admin
```

Default fetches from pmm-submodules VERSION file.

### "Could not determine URL or ref"

Component not in .gitmodules. Add fallback in `get_component_info()` or update pmm-submodules.

### Permission denied in volumes

Should not happen with golang image (runs as root). If it does, check volume mount paths.

### Platform mismatch warning

Add `--platform "${PLATFORM}"` to docker command.

## Integration with CI/CD

Minimal example:
```yaml
- name: Build Components
  run: |
    cd build/pipeline
    make build-all
  env:
    PMM_VERSION: ${{ github.ref_name }}
```

For specific components:
```bash
make build COMPONENT=pmm-admin
make build COMPONENT=mysqld_exporter
```

## Future Enhancements

When extending the build pipeline:

1. **Build matrix** - Multiple architectures/build types in one command
2. **Artifact signing** - Add GPG signing step
3. **Image scanning** - Security scan of pmm-builder image
4. **Build metrics** - Timing and size tracking

## References

- Main docs: [README.md](README.md)
- Project guidelines: [../../.github/copilot-instructions.md](../../.github/copilot-instructions.md)
- pmm-submodules: https://github.com/Percona-Lab/pmm-submodules
