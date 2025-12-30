# PMM Build Pipeline

Docker-based build system for PMM components using a custom `pmm-builder` image based on `golang:latest`.

## Quick Start

```bash
cd build/pipeline

# Build a single component
make build COMPONENT=pmm-admin

# Build all components
make build-all

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

| Variable | Description | Default |
|----------|-------------|---------|
| `PMM_VERSION` | Version to build | From `VERSION` file |
| `BUILD_TYPE` | Build type: `static` or `dynamic` | `static` |
| `GOARCH` | Target architecture: `amd64` or `arm64` | `amd64` |
| `PLATFORM` | Docker platform: `linux/amd64` or `linux/arm64` | Auto-detected |
| `GO_VERSION` | Go version for builder image | `latest` |
| `OUTPUT_DIR` | Output directory for artifacts | `./output` |


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

## Caching

Two Docker volumes are used for caching:

- **pmm-mod** - Go module cache (`/go/pkg/mod`)
- **pmm-build** - Build source cache (`/build`)

To clear caches:

```bash
make clean-volumes
```

## Directory Structure

```
build/
├── pipeline/
│   ├── Makefile          # Main build targets
│   ├── README.md         # This file
│   └── output/           # Build artifacts (created)
└── scripts/
    └── build-component   # Build script
```

## Examples

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
