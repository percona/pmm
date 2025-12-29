# PMM Client Skaffold Build Pipeline

This directory contains the Skaffold-based build pipeline for PMM Client, providing a containerized, reproducible build environment with per-component granularity.

## Overview

The Skaffold build pipeline replaces the traditional shell-script-based build process with a modern, container-native approach that:

- **Builds components independently** - Each component is built as a separate Docker artifact
- **Provides fault isolation** - One component failure doesn't invalidate the entire build
- **Enables parallel builds** - Multiple components build concurrently (up to 4 at a time)
- **Supports incremental rebuilds** - Only changed components are rebuilt
- **Provides reproducible builds** - Same inputs always produce the same outputs
- **Supports multiple build variants** - Static/dynamic builds, different architectures
- **Enables local and CI/CD builds** - Works identically in development and production
- **Simplifies dependencies** - All build tools are containerized

## Components Built

The pipeline builds 13 separate artifacts:

**Workspace Components:**
- `pmm-admin` - PMM Admin CLI tool
- `pmm-agent` - PMM Agent for client-side monitoring

**Exporters:**
- `node-exporter` - System metrics exporter
- `mysqld-exporter` - MySQL metrics exporter
- `mongodb-exporter` - MongoDB metrics exporter
- `postgres-exporter` - PostgreSQL metrics exporter
- `proxysql-exporter` - ProxySQL metrics exporter
- `rds-exporter` - AWS RDS metrics exporter
- `azure-metrics-exporter` - Azure metrics exporter
- `redis-exporter` - Redis/Valkey metrics exporter

**Supporting Tools:**
- `vmagent` - VictoriaMetrics agent
- `nomad` - HashiCorp Nomad orchestrator
- `percona-toolkit` - Percona database utilities

## Prerequisites

1. **Skaffold v2.17.0 or later** - Install from https://skaffold.dev/docs/install/
   ```bash
   # macOS (recommended)
   brew install skaffold
   
   # Alternative: Direct download
   # macOS
   curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-darwin-amd64
   chmod +x skaffold
   sudo mv skaffold /usr/local/bin
   
   # Linux
   curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64
   chmod +x skaffold
   sudo mv skaffold /usr/local/bin
   
   # Verify installation
   skaffold version
   ```

2. **Docker** - Ensure Docker is installed and running
   ```bash
   docker --version
   ```

## Quick Start

### Build All Components

```bash
cd /Users/alex/Projects/pmm/pmm5/build/skaffold
make build
```

This will:
1. Build all 13 components in parallel (4 at a time)
2. Create individual Docker images for each component
3. Tag all images with the PMM version

### Build Individual Components

Build just one component using make:

```bash
make build-component COMPONENT=pmm-admin
make build-component COMPONENT=node-exporter
make build-component COMPONENT=vmagent
```

Or use skaffold directly with environment variables:

```bash
PMM_VERSION=$(cat ../../VERSION) BASE_IMAGE=golang:latest skaffold build -b pmm-admin
PMM_VERSION=$(cat ../../VERSION) BASE_IMAGE=golang:latest skaffold build -b node-exporter
```

Build a subset of components:

```bash
PMM_VERSION=$(cat ../../VERSION) BASE_IMAGE=golang:latest skaffold build -b pmm-admin -b pmm-agent
```

### Extract Binaries

After building, extract binaries from all component images:

```bash
make extract
# Binaries will be in ../bin/
```

Or extract from a specific component:

```bash
docker create --name temp-extract pmm-admin:3.6.0
docker cp temp-extract:/output/. ./results/
docker rm temp-extract
```

## Build Variants

### Static Build (Default)

Produces statically-linked binaries without external dependencies:

```bash
make build
# or
skaffold build
```

### Dynamic Build (with GSSAPI support)

Produces dynamically-linked binaries with Kerberos/GSSAPI support for pmm-agent and mongodb-exporter:

```bash
make build-dynamic
# or
BUILD_TYPE=dynamic skaffold build --profile=dynamic
```

### ARM64 Architecture

Build for ARM64 (aarch64) architecture:

```bash
make build-arm64
# or
GOARCH=arm64 skaffold build --profile=arm64
```

## Architecture

### Directory Structure

```
build/skaffold/
├── skaffold.yaml               # Main Skaffold configuration
├── Dockerfile.component        # Dockerfile for workspace components
├── Dockerfile.external         # Dockerfile for external components
├── Makefile                    # Convenience targets
├── scripts/
│   ├── gitmodules.go          # .gitmodules parser
│   └── component-helpers.sh   # Shared component build logic
└── README.md                   # This file
```

### Build Process Flow

1. **Parallel Component Builds**
   - Each component builds in its own Docker context
   - Skaffold builds up to 4 components concurrently
   - Build cache is shared across all components

2. **Workspace Components (pmm-admin, pmm-agent)**
   - Built from local workspace source
   - Use Dockerfile.component
   - Support both static and dynamic linking

3. **External Components (exporters, tools)**
   - Source fetched from .gitmodules or hardcoded refs
   - Use Dockerfile.external with per-component targets
   - Git repositories cached in `/build/source`
   - Go modules cached in `/go/pkg/mod`

4. **Artifact Collection**
   - Each component produces binaries in `/output/`
   - Extract script collects from all component images
   - Binaries placed in `../bin/` directory

### Components and Versions

Component versions are managed through:
- `.gitmodules` from pmm-submodules repository
- Hardcoded fallback commit hashes for VictoriaMetrics, redis_exporter, and nomad
- Git tags/branches specified in .gitmodules

### Build Caching

The build uses two levels of caching:

1. **Docker BuildKit Layer Cache**
   - Each component has independent cache layers
   - Go module downloads cached per component
   - Source code changes only invalidate affected layers

2. **Named Cache Mounts**
   - `/go/pkg/mod` - Go module cache (shared across components)
   - `/build/source` - Git repository cache (shared across components)
   - Persisted across builds on the host

## Environment Variables

### Build Configuration

- `PMM_VERSION` - Version number (auto-detected from VERSION file, **required**)
- `BUILD_TYPE` - `static` (default) or `dynamic`
- `GOARCH` - Target architecture (`amd64` or `arm64`)
- `BASE_IMAGE` - Base Docker image (default: `golang:latest`)

### CI/CD Variables

- `CI` - When set, uses `public.ecr.aws/e7j3v3n0/rpmbuild:3` as base image
- `RESULTS_DIR` - Output directory for extracted binaries (default: `../bin`)

### Advanced Options

You can pass these as environment variables:

```bash
PMM_VERSION=3.0.0 BUILD_TYPE=dynamic make build
```

## Skaffold Features Used

### Build Artifacts
- Multi-artifact builds (13 separate components)
- Docker build with BuildKit
- Multi-stage builds for optimization
- Build argument templating
- Platform specification (linux/amd64 or linux/arm64)

### Profiles
- Conditional activation based on environment variables
- Different build configurations per profile
- Profile-based build argument overrides

### Build Configuration
- Concurrent builds (up to 4 components in parallel)
- Local builds without pushing to registry
- Custom tag policy using environment template

## Integration with CI/CD

### GitHub Actions Example

```yaml
- name: Install Skaffold
  run: |
    curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64
    chmod +x skaffold
    sudo mv skaffold /usr/local/bin

- name: Build PMM Client
  working-directory: build/skaffold
  run: make build

- name: Extract Artifacts
  working-directory: build/skaffold
  run: make extract RESULTS_DIR=../../artifacts

- name: Upload Artifacts
  uses: actions/upload-artifact@v3
  with:
    name: pmm-client-binaries
    path: ./artifacts/
```

### Jenkins Example

```groovy
stage('Build PMM Client') {
    steps {
        sh '''
            cd build/skaffold
            make build
            make extract RESULTS_DIR=${WORKSPACE}/results/
        '''
    }
}
```

### Build Individual Components in CI

```yaml
- name: Build Core Components Only
  working-directory: build/skaffold
  run: skaffold build -b pmm-admin -b pmm-agent

- name: Build Exporters Only
  working-directory: build/skaffold
  run: |
    skaffold build -b node-exporter \
                   -b mysqld-exporter \
                   -b postgres-exporter \
                   -b mongodb-exporter
```

## Troubleshooting

### Build Fails - PMM_VERSION Not Set

Error: `ERROR: PMM_VERSION environment variable is not set`

**Solution:** Ensure VERSION file exists in workspace root or set PMM_VERSION explicitly:
```bash
PMM_VERSION=3.6.0 make build
```

### Individual Component Build Failure

If one component fails, other components continue building. To rebuild just the failed component:

```bash
skaffold build -b <component-name>
```

### External Component Build Failures

If an external component fails to build:
1. Check if the git tag/branch exists in the source repository
2. Verify the build command is correct for that version
3. Check component-specific dependencies
4. Review logs for the specific component build

### Cache Not Working

If builds seem slower than expected:
1. Ensure Docker BuildKit is enabled: `export DOCKER_BUILDKIT=1`
2. Check disk space for Docker cache
3. Verify cache mounts are working: `docker system df -v`

### Architecture Mismatch

If building for a different architecture than your host:

```bash
# Enable buildx
docker buildx create --use
docker buildx inspect --bootstrap

# Build for ARM64
make build-arm64
```

### Extracting Binaries Fails

If `make extract` fails:
1. Ensure all component images were built successfully
2. Check image names match the expected format: `<component>:<PMM_VERSION>`
3. Verify Docker permissions for container creation

## Comparison with Legacy Build

### Legacy Build (`build/scripts/build-client-binary`)
- Single monolithic build process
- All-or-nothing - one failure stops everything
- Sequential builds (slower)
- Single tarball output
- Shell script run directly on host or in manually managed container
- Requires pre-setup of build environment
- Manual extraction of source tarballs

### Skaffold Build
- Per-component granular builds
- Fault isolation - component failures are isolated
- Parallel builds (4 concurrent by default)
- Individual component artifacts
- Fully containerized, self-contained
- Declarative configuration
- Reproducible builds via Docker layers
- Integrated caching through BuildKit
- Works identically in dev and CI/CD
- Easy to extend and modify
- Selective rebuilds of changed components

## Development Workflow

### Build and Test Individual Components

```bash
# Develop and test just pmm-admin
skaffold build -b pmm-admin
docker create --name temp pmm-admin:3.6.0
docker cp temp:/output/pmm-admin ./
docker rm temp
./pmm-admin --version

# Test an exporter
skaffold build -b node-exporter
docker create --name temp node-exporter:3.6.0
docker cp temp:/output/node_exporter ./
docker rm temp
./node_exporter --version
```

### Rebuilding After Changes

Skaffold automatically detects changes and rebuilds only affected components:

```bash
# Make changes to pmm-agent
vim ../../agent/runner/runner.go

# Only pmm-agent will rebuild
skaffold build -b pmm-agent
```

### Testing Local Changes

To test local changes to pmm-admin or pmm-agent:

1. Make changes in `admin/` or `agent/` directories
2. Run `skaffold build -b pmm-admin` or `skaffold build -b pmm-agent`
3. Extract and test the built binary
4. Iterate quickly on individual components

### Adding New Components

To add a new component to the build:

1. Edit `build-all-components.sh`
2. Add a new `build_external_component` call with:
   - Component name
   - Git repository URL
   - Version/tag
   - Build command
   - Binary path
3. Test the build

## Future Enhancements

Potential improvements to consider:

- **Parallel builds** - Build independent components in parallel
- **Build caching** - More aggressive layer caching for faster builds
- **Multi-architecture** - Single command to build for multiple architectures
- **Registry push** - Push built images to container registry
- **Helm integration** - Deploy built client for testing
- **Build verification** - Automated testing of built binaries

## Support

For issues or questions:
- Check existing GitHub issues
- Review Skaffold documentation: https://skaffold.dev/docs/
- Consult PMM build documentation in `/build/docs/`

## License

Same as PMM - see LICENSE file in repository root.
