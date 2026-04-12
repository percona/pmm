# PMM Build Pipeline

Docker-based build system for PMM components. Server components are each built with `docker run`, outputting artifacts directly to the host via Docker volumes. The final server image is assembled in a single-stage `FROM oraclelinux:9-slim` Dockerfile using BuildKit.

## Setup

Use the bootstrap script to set up the build environment. It downloads all
required files, populates the bare-repo cache, and creates `.env`:

```bash
# From a fresh machine (curl | bash):
curl -sSL https://raw.githubusercontent.com/percona/pmm/v3/build/pipeline/scripts/bootstrap | bash

# Or from a repo checkout:
./build/pipeline/scripts/bootstrap
```

The build environment is created in `~/build/`. Customize `.env` to change
component versions (git refs for Grafana, VictoriaMetrics, exporters, etc.).

### Repository Cache

PMM builds require a local bare-repo cache. The bootstrap script runs `make populate-cache` automatically. Before subsequent builds, run `make update-cache` to fetch the latest refs. See [Cache Management](#cache-management) for details.

## Quick Start

```bash
cd ~/build

# Build all PMM Client components (binaries only)
make build-client

# Build PMM Client Docker image
make build-client-docker

# Build PMM Client tarball
make build-client-tarball

# Build everything for client (components + Docker image + tarball)
make client

# Build PMM Server (Docker image) - defaults to linux/amd64
make server

# Build PMM Server for ARM64
make server SERVER_PLATFORMS=linux/arm64

# Build everything (client + server)
make all

# Build a single component
make build COMPONENT=pmm-admin

# Build with dynamic linking (GSSAPI support)
make build-dynamic COMPONENT=mongodb_exporter

# Build for ARM64
make build-arm64 COMPONENT=pmm-agent

# Build the pmm-builder Docker image (optional - auto-built on first use)
make builder-image
```

## Components

### Workspace Components
Built from the PMM repository workspace:
- **pmm-admin** - PMM client CLI tool
- **pmm-agent** - PMM agent daemon

### External Components
Built from external Git repositories:
- **node_exporter** - Prometheus Node Exporter (Percona fork)
- **mysqld_exporter** - MySQL Server Exporter (Percona fork)
- **mongodb_exporter** - MongoDB Exporter (Percona fork)
- **postgres_exporter** - PostgreSQL Exporter (Percona fork)
- **proxysql_exporter** - ProxySQL Exporter (Percona fork)
- **rds_exporter** - AWS RDS Exporter (Percona fork)
- **azure_metrics_exporter** - Azure Metrics Exporter (Percona fork)
- **redis_exporter** - Redis Exporter
- **vmagent** - VictoriaMetrics Agent
- **nomad** - HashiCorp Nomad
- **percona-toolkit** - Percona Toolkit utilities

## Environment Variables

### Build Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PMM_VERSION` | Version to build | From `VERSION` file |
| `BUILD_TYPE` | Build type: `static` or `dynamic` | `static` |
| `GOARCH` | Target architecture: `amd64` or `arm64` | Host arch (auto-detected) |
| `PLATFORM` | Docker platform: `linux/amd64` or `linux/arm64` | `linux/$HOST_ARCH` |
| `SERVER_PLATFORMS` | Server build platforms (comma-separated) | `linux/amd64` |
| `GO_VERSION` | Go version for builder image | `1.26` |
| `HOST_ARCH` | Native arch of build host (pure-Go/Node containers run natively) | auto-detected |
| `GOMOD_CACHE_VOL` | Docker volume for Go module cache | `pmm-mod` |
| `BUILD_CACHE_VOL` | Docker volume for Go build cache | `pmm-build` |
| `YARN_CACHE_VOL` | Docker volume for Yarn package cache | `pmm-yarn` |
| `OUTPUT_DIR` | Output directory for artifacts | `./output` |
| `PACKAGE_DIR` | Output directory for tarball packages | `./package` |

### Component Versions (.env file)

Component versions are managed via the `.env` file. Copy `.env.example` to `.env` and customize as needed:

**Server Components:**
- `PMM_REF` - PMM (pmm-managed, qan-api2, vmproxy, UI) git ref (branch/tag/commit)
- `GRAFANA_REF` - Grafana git ref (branch/tag/commit)
- `VM_REF` - VictoriaMetrics git ref (branch/tag/commit)
- `DASHBOARDS_REF` - percona-dashboards git ref (branch/tag/commit)
- `PMM_DUMP_REF` - pmm-dump git ref (branch/tag/commit)

**Client Components:**
- `REDIS_EXPORTER_REF` - redis_exporter git ref (branch/tag/commit)
- `VMAGENT_REF` - vmagent git ref (branch/tag/commit)
- `NOMAD_REF` - nomad git ref (branch/tag/commit)


## Build Process

The build system:

1. **Builds the pmm-builder Docker image** (if not already present) based on `golang:latest`
2. **Creates Docker volumes** for caching Go modules, build cache, and Yarn packages
3. **Fetches component metadata** from `.gitmodules` in pmm-submodules repository
4. **Runs server component builds** using `docker run` — each component writes artifacts to
   `output/server/<component>/` on the host via a volume mount:
   - **pmm-managed, pmm-dump, VictoriaMetrics**: `pmm-builder:latest` (pure Go, CGO disabled)
   - **grafana-go**: `golang:$(GO_VERSION)` (CGO enabled for SQLite; runs at target `GOARCH`)
   - **grafana-ui, pmm-dashboards, pmm-ui**: `node:22` with shared Yarn cache
5. **Assembles the server runtime image** from the host-side artifacts using
   `docker buildx build` (BuildKit)
6. **Outputs artifacts** to the configured output directory

### Workspace Components

Workspace components (pmm-admin, pmm-agent) are built directly from the PMM repository using their respective Makefiles:

```bash
make build COMPONENT=pmm-admin
```

### External Components

External components are cloned from their Git repositories and built:

```bash
make build COMPONENT=mysqld_exporter
```

## Cache Management

PMM uses a **local bare-repo cache** to avoid cloning from upstream on every build.

### Cache Strategy

- **Local disk only**: Bare Git repositories live in `.cache/repos/` and are mounted read-only into build containers
- **Mandatory**: Both server and client builds fail hard if the bare repo cache is missing — no internet fallback at build time
- **First-time setup**: Run `make populate-cache` to clone all repos from upstream
- **Per-build refresh**: `make update-cache` fetches only the refs listed in `.env`
- **Artifact caching**: Each component's resolved commit hash is stored in `.cache/stamps/<component>.hash` after a successful build. On the next run, if the hash matches and output artifacts exist, the build is skipped. Monorepo components (pmm-managed, pmm-admin, pmm-agent) always rebuild — path-aware caching for those is planned separately

### Cache Targets

```bash
# Clone any missing bare repos from upstream (first-time / new component setup)
make populate-cache

# Fetch required refs from upstream into local bare repos
make update-cache

# Full fetch + prune stale branches (slow, run periodically)
make sync-cache

# Build server (runs update-cache automatically)
make server

# Clean local cache only
make clean-cache

# Clean Docker volumes only
make clean-volumes

# Clean everything (cache + volumes + output)
make clean-all
```

### Cache Structure

The `.cache/repos/` directory contains bare Git repositories for all components:

```
.cache/
├── repos/
│   ├── azure_metrics_exporter.git/    # client
│   ├── grafana.git/                   # server
│   ├── mongodb_exporter.git/          # client
│   ├── mysqld_exporter.git/           # client
│   ├── node_exporter.git/             # client
│   ├── nomad.git/                     # client
│   ├── percona-toolkit.git/           # client
│   ├── pmm-dump.git/                  # server
│   ├── pmm.git/                       # server + UI
│   ├── postgres_exporter.git/         # client
│   ├── proxysql_exporter.git/         # client
│   ├── rds_exporter.git/              # client
│   ├── redis_exporter.git/            # client
│   └── VictoriaMetrics.git/           # server + vmagent
└── stamps/                            # commit-hash stamps for artifact caching
    ├── pmm-dump.hash
    ├── grafana-go.hash
    ├── grafana-ui.hash
    ├── victoriametrics.hash
    └── ...
```

Server builds mount these repos read-only into each `docker run` build container.
Client builds (`build-component`) also require them — a missing bare repo is a hard failure,
consistent with server build behaviour.

### Troubleshooting

**Error: bare repo cache not found**
```bash
make populate-cache   # clones all repos from upstream
make update-cache     # fetches required refs
```

## Directory Structure

After bootstrap, `~/build/` (or your chosen directory) contains everything:

```
~/build/                          # PIPELINE_DIR (all-in-one)
├── Makefile                      # Main build targets
├── Dockerfile.builder            # pmm-builder image for Go component builds
├── Dockerfile.client             # PMM Client Docker image
├── Dockerfile.server             # PMM Server Docker image
├── Dockerfile.server.dockerignore
├── .env.example                  # Template for .env
├── .env                          # Component URLs, refs, PMM_VERSION
├── scripts/
│   ├── build-component           # Component build script
│   ├── build-client-docker       # Client Docker build script
│   ├── check-build-cache         # Stamp-based build cache check
│   ├── generate-version-json     # Writes version metadata JSON
│   ├── migrate-from-submodules   # Populates .env from pmm-submodules
│   ├── package-tarball           # Tarball packaging script
│   └── install_tarball           # Client installation script (packaged into tarball)
├── ansible/                      # Ansible playbooks (for server image)
├── docker/                       # Docker entrypoints and support files
├── packages/                     # Packaging configs (systemd, deb, rpm)
├── package/                      # Tarball staging + final archive (created by build)
│   ├── pmm-client/               # Staging directory (binaries, configs, query files)
│   └── pmm-client.tar.gz         # Final client tarball
├── output/                       # Build artifacts (created by build)
│   ├── <client binaries>         # pmm-admin, pmm-agent, exporters, etc.
│   ├── queries/                  # Exporter query files (YAML, .prom)
│   └── server/                   # Per-component server artifact directories
│       ├── pmm-managed/          # 6 Go binaries + swagger + YAML data dirs
│       ├── pmm-dump/             # pmm-dump binary
│       ├── grafana-go/           # grafana-server, grafana, grafana-cli
│       ├── grafana-ui/           # public/, conf/, tools/
│       ├── victoriametrics/      # victoria-metrics-pure, vmalert-pure
│       ├── pmm-dashboards/       # panels/, pmm-app-dist/
│       └── pmm-ui/               # pmm-dist/, pmm-compat-dist/
└── .cache/
    ├── repos/                    # Bare Git repo cache
    └── stamps/                   # Commit-hash stamps for artifact caching
```

## Build Targets

### Main Targets

- **`make client`** - Builds all client components, Docker image, and tarball
- **`make server`** - Builds the PMM Server Docker image
- **`make all`** - Builds both client and server

### Client Build Targets

- **`make build-client`** - Build all client components (binaries only)
- **`make build-client-docker`** - Build PMM Client Docker image
- **`make build-client-tarball`** - Build PMM Client tarball package

### Server Build Targets

- **`make build-server`** - Build PMM Server Docker image (multi-architecture)
- **`make build-server-docker`** - Same as build-server

### Component Targets

- **`make build COMPONENT=<name>`** - Build a specific component
- **`make build-dynamic COMPONENT=<name>`** - Build with dynamic linking (GSSAPI)
- **`make build-arm64 COMPONENT=<name>`** - Build for ARM64

### Utility Targets

- **`make builder-image`** - Build the pmm-builder Docker image
- **`make gitmodules`** - Build gitmodules parser binary
- **`make clean`** - Remove output directory
- **`make clean-volumes`** - Remove Docker cache volumes

## Examples

### Build PMM Client

```bash
cd ~/build
make client
```

This will:
1. Build all client components (pmm-admin, pmm-agent, exporters)
2. Create the PMM Client Docker image
3. Generate the PMM Client tarball package
4. Output artifacts to `./output/` and `./package/`

### Build PMM Server

```bash
cd ~/build
make server
```

This builds the PMM Server Docker image via a two-phase process:

**Phase 1 — component builds** (all `docker run`, no BuildKit):
1. Go components built in parallel (`-j4`): pmm-managed/qan-api2/vmproxy, pmm-dump, Grafana backend, VictoriaMetrics
2. Node.js assets built sequentially (to avoid Yarn network saturation): Grafana UI, PMM UI, percona-dashboards

Each component writes its artifacts to `output/server/<component>/` on the host. Pure-Go and Node containers run natively at `HOST_ARCH` (auto-detected from `uname -m`); Go cross-compiles for `GOARCH` independently. grafana-go (needs CGO/SQLite) runs at `--platform linux/$(GOARCH)` instead.

**Phase 2 — image assembly** (`docker buildx build` with BuildKit):
- Copies host-side artifacts into a single `FROM oraclelinux:9-slim` image

**Default Architecture:** `linux/amd64`

Set `GOARCH=arm64` to cross-compile binaries for ARM64:

```bash
# Build for ARM64
make server GOARCH=arm64 SERVER_PLATFORMS=linux/arm64
```

### Build Everything

```bash
cd ~/build
make all
```

### Build pmm-admin

```bash
cd ~/build
make build COMPONENT=pmm-admin
```

### Custom build with specific version

```bash
PMM_VERSION=3.0.0-rc1 make build COMPONENT=pmm-agent
```

### Build for ARM64 with dynamic linking

```bash
BUILD_TYPE=dynamic GOARCH=arm64 make build COMPONENT=mongodb_exporter
```

### Create distribution tarball

First build all components, then package them:

```bash
make build-client
make build-client-tarball
```

The tarball will be created at `./package/pmm-client-${VERSION}.tar.gz` with the following structure:

```
pmm-client-${VERSION}/
├── bin/               # All built binaries (pmm-admin, pmm-agent, exporters, etc.)
├── config/            # Configuration files (systemd services)
├── debian/            # Debian packaging files
├── rpm/               # RPM spec files
├── queries-*.yml      # Query examples (if present)
├── install_tarball    # Installation script
└── VERSION            # Version identifier
```

## Troubleshooting

### Build fails with "No URL/ref found"

The component might not be in `.gitmodules`. Check the fallback values in `build-component.sh`.

### Permission denied errors

Ensure Docker is running and you have permissions to create volumes.

### Build artifacts not found

Check `OUTPUT_DIR` (default: `./output`). Verify the build completed successfully.

## CI/CD Integration

```yaml
- name: Bootstrap build environment
  run: ./build/pipeline/scripts/bootstrap --skip-cache

- name: Build PMM Components
  run: |
    cd ~/build
    make build-all
  env:
    PMM_VERSION: ${{ github.ref_name }}
```

## License
This build pipeline is licensed under the [Apache License 2.0](LICENSE).
