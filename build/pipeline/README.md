# PMM Build Pipeline

Docker-based build system for PMM components using a custom `pmm-builder` image based on `golang:latest`.

## Setup

Before building, copy the example environment file and customize if needed:

```bash
cd build/pipeline
cp .env.example .env
```

The `.env` file contains git refs for all external dependencies (Grafana, VictoriaMetrics, exporters, etc.). You can modify these to build with different versions.

### Minio Requirement for Server Builds

PMM Server builds require a local Minio instance for repository cache. See [Cache Management](#cache-management) section for setup instructions.

Quick Minio setup:

```bash
# Start Minio container
docker run -d --name minio -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"

# Install and configure mc (Minio client)
brew install minio/stable/mc  # macOS
mc alias set pmm http://localhost:9000 minioadmin minioadmin
mc mb pmm/cache

# Populate cache (see Cache Maintenance section for details)
```

## Quick Start

```bash
cd build/pipeline

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
| `GOARCH` | Target architecture: `amd64` or `arm64` | `amd64` |
| `PLATFORM` | Docker platform: `linux/amd64` or `linux/arm64` | `linux/amd64` |
| `SERVER_PLATFORMS` | Server build platforms (comma-separated) | `linux/amd64` |
| `GO_VERSION` | Go version for builder image | `1.26` |
| `OUTPUT_DIR` | Output directory for artifacts | `./output` |
| `PACKAGE_DIR` | Output directory for tarball packages | `./package` |
| `MINIO_ENDPOINT` | Minio S3 endpoint URL | `http://localhost:9000` |
| `MINIO_BUCKET` | Minio bucket name | `cache` |
| `MINIO_CACHE_PREFIX` | Cache prefix in bucket | `repos` |

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
2. **Creates Docker volumes** for caching Go modules and build artifacts
3. **Fetches component metadata** from `.gitmodules` in pmm-submodules repository
4. **Runs builds in containers** using the pmm-builder image
5. **Outputs artifacts** to the configured output directory

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

PMM uses **Minio S3** for persistent build cache storage across ephemeral build agents.

### Cache Strategy

- **One-way sync**: Cache is downloaded from Minio to local disk before builds
- **No upload**: Local cache is never synced back to avoid conflicts between parallel builds  
- **Mandatory**: Builds fail if Minio cache is unavailable (no graceful fallback)
- **Cache maintenance**: A separate process/job maintains the Minio cache (see Cache Maintenance below)

### Local Minio Setup

Start a local Minio container for development:

```bash
docker run -d \
  --name minio \
  -p 9000:9000 \
  -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  -v minio-data:/data \
  minio/minio server /data --console-address ":9001"
```

Configure Minio client (mc):

```bash
# Install mc
brew install minio/stable/mc  # macOS
# Or for Linux: https://min.io/docs/minio/linux/reference/minio-mc.html

# Configure alias
mc alias set pmm http://127.0.0.1:9000 minioadmin minioadmin

# Create bucket
mc mb pmm/cache
```

### Minio Configuration

Configure Minio access via environment variables or `.env` file:

```bash
MINIO_ENDPOINT=http://127.0.0.1:9000
MINIO_BUCKET=cache
MINIO_CACHE_PREFIX=repos
```

### Cache Targets

```bash
# Download repository cache from Minio (mandatory for server builds)
make download-cache

# Build server (downloads cache first, fails if Minio unavailable)
make server

# Clean local cache only
make clean-cache

# Clean Docker volumes only
make clean-volumes

# Clean everything (cache + volumes + output)
make clean-all
```

### Cache Structure

The `.cache/repos/` directory contains bare Git repositories:

```
.cache/
└── repos/
    ├── grafana-dashboards.git/
    ├── grafana.git/
    ├── pmm-dump.git/
    ├── pmm.git/
    └── VictoriaMetrics.git/
```

These are mounted read-only into Docker build stages to speed up builds.

**Note:** The `download-cache` Make target automatically fixes bare repository structure by creating empty `refs` subdirectories. This is necessary because some sync tools (like `mc mirror`) don't preserve empty directories.

### Cache Maintenance (for build administrators)

To populate or update the Minio cache (run from a dedicated maintenance job, not build agents):

```bash
# Create cache directory
mkdir -p /tmp/pmm-cache-update && cd /tmp/pmm-cache-update

# Clone repositories as bare for efficiency
git clone --bare https://github.com/percona/pmm.git  pmm.git
git clone --bare https://github.com/percona/pmm-dump.git pmm-dump.git
git clone --bare https://github.com/percona/grafana.git grafana.git
git clone --bare https://github.com/VictoriaMetrics/VictoriaMetrics.git VictoriaMetrics.git
git clone --bare https://github.com/percona/grafana-dashboards.git grafana-dashboards.git

# Fix bare repository structure (git requires refs directories to exist)
for repo in *.git; do
    mkdir -p "$repo/refs/heads" "$repo/refs/tags" "$repo/refs/remotes"
done

# Update existing bare repositories (if refreshing cache)
for repo in *.git; do
    cd "$repo" && git fetch --all --prune && cd ..
done

# Upload to Minio
mc mirror --overwrite . pmm/cache/repos/
```

**Recommended**: Set up a scheduled job (e.g., daily cron) to keep the cache fresh.

### Troubleshooting

**Error: mc is not installed**
```bash
brew install minio/stable/mc  # macOS
# Or see: https://min.io/docs/minio/linux/reference/minio-mc.html
```

**Error: Failed to download cache from Minio**
- Ensure Minio container is running: `docker ps | grep minio`
- Check Minio endpoint: `mc ls pmm/cache/`
- Verify bucket exists: `mc mb pmm/cache`
- Populate cache: See Cache Maintenance section above

## Directory Structure

```
build/
├── pipeline/
│   ├── Makefile                  # Main build targets
│   ├── README.md                 # This file
│   ├── Dockerfile.client         # PMM Client Docker image
│   ├── Dockerfile.server         # PMM Server assembly (references pre-built images)
│   ├── Dockerfile.builder        # pmm-builder image for client component builds
│   ├── Dockerfile.pmm-managed    # Builds pmm-managed, qan-api2, vmproxy
│   ├── Dockerfile.pmm-dump       # Builds pmm-dump
│   ├── Dockerfile.grafana-go     # Builds Grafana backend binaries
│   ├── Dockerfile.grafana-ui     # Builds Grafana UI assets
│   ├── Dockerfile.victoriametrics # Builds victoria-metrics and vmalert
│   ├── Dockerfile.pmm-dashboards # Builds percona-dashboards panels
│   ├── Dockerfile.pmm-ui         # Builds PMM UI assets
│   ├── output/                   # Build artifacts (created)
│   └── package/                  # Tarballs (created)
└── scripts/
    ├── build-component           # Component build script
    ├── package-tarball           # Tarball packaging script
    └── build-client-docker       # Client Docker build script
```

## Build Targets

### Main Targets

- **`make client`** - Builds all client components, Docker image, and tarball
- **`make server`** - Builds the PMM Server Docker image using multi-stage build
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
cd build/pipeline
make client
```

This will:
1. Build all client components (pmm-admin, pmm-agent, exporters)
2. Create the PMM Client Docker image
3. Generate the PMM Client tarball package
4. Output artifacts to `./output/` and `./package/`

### Build PMM Server

```bash
cd build/pipeline
make server
```

This builds the PMM Server Docker image using a split build that:
1. Builds Go components in parallel: pmm-managed/qan-api2/vmproxy, pmm-dump, Grafana backend, VictoriaMetrics
2. Builds Node.js assets sequentially (to avoid network saturation): Grafana UI, PMM UI, percona-dashboards
3. Assembles a runtime image from the individually built component images

**Default Architecture:** `linux/amd64`

The server builds for `linux/amd64` by default, regardless of your host platform. To build for a different architecture:

```bash
# Build for ARM64
make server SERVER_PLATFORMS=linux/arm64

# Build for multiple architectures (multi-arch image)
make server SERVER_PLATFORMS=linux/amd64,linux/arm64
```

**Note:** The Dockerfile uses `--platform=$BUILDPLATFORM` for build stages, allowing fast native builds even when cross-compiling the final image.

### Build Everything

```bash
cd build/pipeline
make all
```

### Build pmm-admin

```bash
cd build/pipeline
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
- name: Build PMM Components
  run: |
    cd build/pipeline
    make volumes
    make build-all
  env:
    PMM_VERSION: ${{ github.ref_name }}
```

## License
This build pipeline is licensed under the [Apache License 2.0](LICENSE).
