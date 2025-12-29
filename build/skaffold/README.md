# PMM Client Skaffold Build Pipeline

This directory contains the Skaffold-based build pipeline for PMM Client, providing a containerized, reproducible build environment.

## Overview

The Skaffold build pipeline replaces the traditional shell-script-based build process with a modern, container-native approach that:

- **Containerizes the entire build process** - All builds happen in isolated Docker containers
- **Provides reproducible builds** - Same inputs always produce the same outputs
- **Supports multiple build variants** - Static/dynamic builds, different architectures
- **Enables local and CI/CD builds** - Works identically in development and production
- **Simplifies dependencies** - All build tools are containerized

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

### Build PMM Client (default - static, amd64)

```bash
cd /Users/alex/Projects/pmm/pmm5/build/skaffold
skaffold build
```

This will:
1. Build all PMM Client components (pmm-admin, pmm-agent)
2. Build all required exporters (node, mysqld, postgres, mongodb, etc.)
3. Build supporting tools (vmagent, nomad, percona-toolkit)
4. Create a tarball with all binaries and configuration files

### Extract Build Artifacts

After building, extract the artifacts from the container:

```bash
# Get the image ID
docker images pmm-client-builder --format "{{.ID}}" | head -1

# Create a container and copy artifacts
docker create --name pmm-extract <IMAGE_ID>
docker cp pmm-extract:/build/output/. ./results/
docker rm pmm-extract
```

## Build Variants

### Static Build (Default)

Produces statically-linked binaries without external dependencies:

```bash
skaffold build
```

### Dynamic Build (with GSSAPI support)

Produces dynamically-linked binaries with Kerberos/GSSAPI support:

```bash
BUILD_TYPE=dynamic skaffold build --profile=dynamic
```

### ARM64 Architecture

Build for ARM64 (aarch64) architecture:

```bash
GOARCH=arm64 skaffold build --profile=arm64
```

### AMD64 Architecture (Default)

```bash
GOARCH=amd64 skaffold build --profile=amd64
```

### Development Build (with race detector)

Build with Go race detector for development/testing:

```bash
BUILD_MODE=dev skaffold build --profile=dev
```

### Combined Profiles

You can combine profiles:

```bash
# Dynamic build for ARM64
BUILD_TYPE=dynamic GOARCH=arm64 skaffold build --profile=dynamic --profile=arm64
```

## Architecture

### Directory Structure

```
build/skaffold/
├── skaffold.yaml              # Main Skaffold configuration
├── Dockerfile.builder         # Multi-stage Dockerfile for builds
├── scripts/
│   └── build-all-components.sh # Build orchestration script
└── README.md                   # This file
```

### Build Process Flow

1. **Base Builder Stage**
   - Uses rpmbuild:3 image with Go toolchain
   - Copies entire PMM repository into container
   - Sets up build environment variables

2. **Component Build**
   - Builds PMM components (pmm-admin, pmm-agent) from workspace
   - Clones and builds external dependencies (exporters, tools)
   - Each component built with appropriate flags and configuration

3. **Artifact Collection**
   - All binaries collected in `/build/binary/pmm-client-<version>/bin/`
   - Configuration files copied from workspace
   - VERSION file created
   - Final tarball created in `/build/output/`

4. **Multi-stage Build**
   - Production builder: optimized, static builds
   - Development builder: race detector enabled
   - Artifacts stage: minimal scratch image with only build outputs

### Components Built

#### PMM Core Components
- **pmm-admin** - CLI tool for managing PMM client
- **pmm-agent** - Agent that runs exporters and collects metrics

#### Exporters
- **node_exporter** - System metrics exporter
- **mysqld_exporter** - MySQL metrics exporter
- **postgres_exporter** - PostgreSQL metrics exporter
- **mongodb_exporter** - MongoDB metrics exporter
- **proxysql_exporter** - ProxySQL metrics exporter
- **rds_exporter** - AWS RDS metrics exporter
- **azure_exporter** - Azure metrics exporter
- **redis_exporter** - Redis metrics exporter (also copied as valkey_exporter)

#### Supporting Tools
- **vmagent** - VictoriaMetrics agent for metrics collection
- **nomad** - HashiCorp Nomad for workload orchestration
- **pt-summary, pt-mysql-summary** - Percona Toolkit utilities (Perl)
- **pt-mongodb-summary, pt-pg-summary** - Percona Toolkit utilities (Go)

#### Configuration Files
- RPM packaging files
- DEB packaging files
- Installation scripts
- Exporter configuration files (queries, examples)

## Environment Variables

### Build Configuration

- `PMM_VERSION` - Version number (auto-detected from VERSION file)
- `FULL_PMM_VERSION` - Full version including git metadata
- `BUILD_TYPE` - `static` (default) or `dynamic`
- `GOOS` - Target OS (default: `linux`)
- `GOARCH` - Target architecture (`amd64` or `arm64`)
- `BUILD_MODE` - `prod` (default) or `dev`

### Advanced Options

You can pass these as build arguments:

```bash
skaffold build \
  --build-env PMM_VERSION=3.0.0 \
  --build-env BUILD_TYPE=dynamic
```

## Skaffold Features Used

### Build Artifacts
- Docker build with BuildKit
- Multi-stage builds for optimization
- Build argument templating

### Profiles
- Conditional activation based on environment variables
- Different build configurations per profile
- Composable profiles for complex scenarios

### File Sync (for development)
- Automatic sync of Go source files
- Enables faster iteration during development
- Configure with `skaffold dev`

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
  run: skaffold build --push=false

- name: Extract Artifacts
  run: |
    IMAGE_ID=$(docker images pmm-client-builder --format "{{.ID}}" | head -1)
    docker create --name pmm-extract $IMAGE_ID
    docker cp pmm-extract:/build/output/. ./artifacts/
    docker rm pmm-extract

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
            skaffold build --push=false
            
            IMAGE_ID=$(docker images pmm-client-builder --format "{{.ID}}" | head -1)
            docker create --name pmm-extract $IMAGE_ID
            docker cp pmm-extract:/build/output/. ${WORKSPACE}/results/
            docker rm pmm-extract
        '''
    }
}
```

## Troubleshooting

### Build Fails - Git Not Found

The build script requires git to fetch component metadata. Ensure the base image has git installed (it should be in the Dockerfile).

### External Component Build Failures

If an external component fails to build:
1. Check if the git tag/branch exists
2. Verify the build command is correct for that version
3. Check component-specific dependencies

### Tarball Not Created

Ensure the `/build/output` directory is writable and has sufficient space.

### Architecture Mismatch

If building for a different architecture than your host, you may need to enable Docker BuildKit multi-platform support:

```bash
docker buildx create --use
docker buildx inspect --bootstrap
```

## Comparison with Legacy Build

### Legacy Build (`build/scripts/build-client-binary`)
- Shell script run directly on host or in manually managed container
- Requires pre-setup of build environment
- Complex volume mounting for caching
- Manual extraction of source tarballs
- Hard to reproduce across different environments

### Skaffold Build
- Fully containerized, self-contained
- Declarative configuration
- Reproducible builds via Docker layers
- Integrated caching through BuildKit
- Works identically in dev and CI/CD
- Easy to extend and modify

## Development Workflow

### Rapid Iteration with Skaffold Dev

For active development, use Skaffold's dev mode:

```bash
cd build/skaffold
skaffold dev
```

This will:
- Build the initial image
- Watch for file changes
- Automatically sync changed files and rebuild
- Stream logs from the build process

### Testing Local Changes

To test local changes to pmm-admin or pmm-agent:

1. Make changes in `admin/` or `agent/` directories
2. Run `skaffold build` from `build/skaffold/`
3. Extract and test the built binaries

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
