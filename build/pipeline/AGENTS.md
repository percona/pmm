# PMM Build Pipeline - AI Agent Guidelines

This document provides comprehensive guidelines for AI agents working with the PMM build pipeline. The build pipeline is a Docker-based system for building PMM components.

## Architecture Overview

The build pipeline uses `docker run` for all server component builds, outputting artifacts directly to the host via Docker volumes. Client components are built with a custom `pmm-builder` image. Different images are used for different server components depending on their requirements.

### Core Components

1. **Dockerfile.builder** - Defines the `pmm-builder` image (golang-based) for Go component builds
2. **Dockerfile.server** - Assembly-only Dockerfile; copies pre-built artifacts from host paths
3. **scripts/build-component** - Main build orchestration script for client builds
4. **.env** - Single source of truth for all component URLs, refs, and PMM_VERSION
5. **Makefile** - Build targets and convenience commands
6. **README.md** - User-facing documentation

> **Note:** All server components are built via `docker run` (no component Dockerfiles). Each
> `build-artifact-*` target in the Makefile runs a container, clones the source from a local
> bare-repo cache, builds, and copies artifacts to `output/server/<component>/` on the host.
> Dockerfile.server then assembles the runtime image from those host-side paths.

### Design Principles

- **Volume Caching** - Use Docker volumes for Go modules and build artifacts
- **Single Source of Truth** - All component URLs, git refs, and `PMM_VERSION` live in `.env`. Run `scripts/migrate-from-submodules` once to populate from percona/pmm-submodules. `GO_VERSION` is defined once in Makefile and passed as `--build-arg` to all component Dockerfiles
- **Explicit REF args** - All `*_REF` build args in component Dockerfiles have no defaults; they must be passed via `--build-arg`. Omitting one causes an immediate build failure at `git checkout`
- **Split server builds** - All server components are built independently via `docker run`, writing artifacts to `output/server/<component>/` on the host:
  - **pmm-managed, pmm-dump, VictoriaMetrics**: `pmm-builder:latest` (pure Go, `CGO_ENABLED=0`), run at `HOST_ARCH` for native speed; Go cross-compiles for `GOARCH`
  - **grafana-go**: `golang:$(GO_VERSION)` (CGO enabled, has gcc for `go-sqlite3`), runs at `--platform linux/$(GOARCH)` so sqlite3's C code compiles natively for the target arch
  - **grafana-ui, pmm-dashboards, pmm-ui**: `node:22` (git included), run at `HOST_ARCH`; Yarn cache at `/usr/local/share/.cache/yarn` shared via `YARN_CACHE_VOL`
  - **BuildKit** (`docker buildx build`) is used **only** for the final `build-server-docker` step (OracleLinux runtime image assembly)
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
- `build_workspace_component()` - Builds pmm-admin/pmm-agent from the monorepo
- `build_external_component()` - Clones and builds external components using `*_URL` / `*_REF` from `.env`

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
PACKAGE_DIR="${PACKAGE_DIR:-./tarball}"  # Output location
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
PACKAGE_DIR ?= ./tarball
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

2. Add `NEW_EXPORTER_URL` and `NEW_EXPORTER_REF` to `.env` (and `.env.example`).

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
- **Don't add runtime network fetches** - all URLs and refs must come from `.env`, not fetched at build time
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

### Docker Volumes

Four Docker volumes provide caching:

1. **pmm-mod** (`/go/pkg/mod`) - Go modules, shared across all Go builds
2. **pmm-build** (`/root/.cache/go-build`) - Go build cache, shared across all Go builds
3. **pmm-yarn** (`/usr/local/share/.cache/yarn`) - Yarn package cache for Node.js server builds (grafana-ui, pmm-dashboards, pmm-ui)

### Bare Repo Cache

Both server and client builds require bare repos to be present in `REPO_CACHE_DIR` (`.cache/repos/`) — populated by `make populate-cache` and kept current by `make update-cache`. A missing bare repo is a hard failure in both cases; there is no internet-clone fallback like there was in the old build system.

| Bare repo | Used by |
|-----------|--------|
| `pmm.git` | server (pmm-managed, qan-api2, vmproxy, UI, dashboards) |
| `pmm-dump.git` | server |
| `grafana.git` | server (grafana-go, grafana-ui) |
| `VictoriaMetrics.git` | server + client (vmagent) |
| `node_exporter.git` | client |
| `mysqld_exporter.git` | client |
| `mongodb_exporter.git` | client |
| `postgres_exporter.git` | client |
| `proxysql_exporter.git` | client |
| `rds_exporter.git` | client |
| `azure_metrics_exporter.git` | client |
| `redis_exporter.git` | client |
| `nomad.git` | client |
| `percona-toolkit.git` | client |

Use `make populate-cache` to clone all missing repos from upstream, then `make update-cache` to fetch
the required refs. `make sync-cache` does a full fetch with pruning across all repos.

### Artifact Caching (Stamp Files)

Each non-monorepo component stores its resolved commit hash in `.cache/stamps/<component>.hash` after a successful build. On the next run, `scripts/check-build-cache` compares the current ref's commit hash against the stamp; if they match and the output directory is non-empty, the build is skipped.

- **Server**: pmm-dump, grafana-go, grafana-ui, victoriametrics, pmm-dashboards, pmm-ui use stamp-based caching. pmm-managed always rebuilds (monorepo — path-aware caching planned separately).
- **Client**: All external components (exporters, vmagent, nomad, percona-toolkit) use stamp-based caching. Workspace components (pmm-admin, pmm-agent) always rebuild (monorepo).

Cache is persistent across builds. Clear with:
```bash
make clean-volumes  # Warning: destroys all Go/Yarn caches!
make clean-cache    # Warning: removes all bare repos and stamp files!
```

## Component Metadata

All component URLs and git refs are stored in `.env` as `<PREFIX>_URL` and `<PREFIX>_REF` pairs
(e.g. `NODE_EXPORTER_URL`, `NODE_EXPORTER_REF`). `PMM_VERSION` is also stored there.

The `percona/pmm` monorepo contains three independently versioned sub-trees, each with its
own ref variable pointing at the same `pmm.git` bare repo:

| Variable | Sub-tree | Build target |
|---|---|---|
| `PMM_REF` | Backend (`managed/`, `qan-api2/`, `vmproxy/`) | `build-pmm-managed` |
| `PMM_UI_REF` | Frontend (`ui/`) | `build-pmm-ui` |
| `PMM_DASHBOARDS_REF` | Dashboards (`dashboards/`) | `build-pmm-dashboards` |

Most of the time all three are the same commit, but separate frontend or dashboards PRs can
set `PMM_UI_REF` / `PMM_DASHBOARDS_REF` to a different branch without touching the backend.

Run `scripts/migrate-from-submodules` once to populate empty values from the legacy
percona/pmm-submodules repository. After that the build has no network dependency on
pmm-submodules.

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

### "PMM_VERSION is not set"

Either run the migration script or set it directly in `.env`:
```bash
scripts/migrate-from-submodules   # fetches from pmm-submodules once
# — or —
echo "PMM_VERSION=3.2.0" >> ~/build/.env
```

### "NODE_EXPORTER_URL or NODE_EXPORTER_REF is not set in .env"

The component's URL or ref is missing from `.env`. Run `scripts/migrate-from-submodules`
or add the values manually.

### Permission denied in volumes

Should not happen with golang image (runs as root). If it does, check volume mount paths.

### Platform mismatch warning

Add `--platform "${PLATFORM}"` to docker command.

## Integration with CI/CD

Minimal example:
```yaml
- name: Bootstrap
  run: ./build/pipeline/scripts/bootstrap --skip-cache

- name: Build Components
  run: |
    cd ~/build
    make build-all
  env:
    PMM_VERSION: ${{ github.ref_name }}
```

For specific components:
```bash
cd ~/build
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
