# PMM Client Skaffold Build - Quick Reference

## One-Command Builds

```bash
# Navigate to skaffold directory
cd /Users/alex/Projects/pmm/pmm5/build/skaffold

# Build (default: static, amd64)
make build

# Build and extract artifacts
make test-build

# Build for production (dynamic with GSSAPI)
make build-dynamic

# Build for ARM64
make build-arm64

# Development build
make build-dev
```

## Manual Skaffold Commands

```bash
# Basic build
skaffold build

# Build with profile
skaffold build --profile=dynamic

# Build for ARM64
GOARCH=arm64 skaffold build --profile=arm64

# Development mode (watch and rebuild)
skaffold dev
```

## Extract Artifacts

```bash
# Using Make
make extract

# Manual extraction
IMAGE_ID=$(docker images pmm-client-builder --format "{{.ID}}" | head -1)
docker create --name pmm-extract $IMAGE_ID
docker cp pmm-extract:/build/output/. ./results/
docker rm pmm-extract
```

## Build Profiles

| Profile | Description | Command |
|---------|-------------|---------|
| default | Static, AMD64 | `skaffold build` |
| static | Static linking | `skaffold build --profile=static` |
| dynamic | Dynamic (GSSAPI) | `BUILD_TYPE=dynamic skaffold build --profile=dynamic` |
| arm64 | ARM64 arch | `GOARCH=arm64 skaffold build --profile=arm64` |
| amd64 | AMD64 arch | `GOARCH=amd64 skaffold build --profile=amd64` |
| dev | Race detector | `BUILD_MODE=dev skaffold build --profile=dev` |

## Common Workflows

### Daily Development
```bash
# Make changes to pmm-admin or pmm-agent
cd /Users/alex/Projects/pmm/pmm5

# Build and test
cd build/skaffold
make test-build

# Check results
ls -lh ../../results/skaffold/
tar -tzf ../../results/skaffold/pmm-client-*.tar.gz | grep bin/
```

### Release Build
```bash
cd /Users/alex/Projects/pmm/pmm5/build/skaffold

# Build all variants
make build-all

# Extract artifacts
make extract
```

### CI/CD Integration
```bash
# In CI pipeline
cd build/skaffold
skaffold build --push=false
make extract RESULTS_DIR=/workspace/artifacts
```

## Troubleshooting

### No image found
```bash
# List images
docker images pmm-client-builder

# If missing, rebuild
make build
```

### Clean start
```bash
make clean
make build
```

### Check build logs
```bash
# Run with verbose output
skaffold build -v debug
```

## Directory Structure

```
build/skaffold/
├── Makefile                    # Convenience targets
├── README.md                   # Full documentation
├── QUICKSTART.md              # This file
├── skaffold.yaml              # Skaffold config
├── Dockerfile.builder         # Build container
└── scripts/
    └── build-all-components.sh # Build script
```

## Output Structure

```
results/skaffold/
└── pmm-client-{version}.tar.gz
    └── pmm-client-{version}/
        ├── bin/                # All binaries
        │   ├── pmm-admin
        │   ├── pmm-agent
        │   ├── node_exporter
        │   ├── mysqld_exporter
        │   └── ...
        ├── config/             # Configuration
        ├── rpm/                # RPM packaging
        ├── debian/             # DEB packaging
        ├── install_tarball     # Installation script
        └── VERSION             # Version file
```

## Next Steps

- Read [README.md](README.md) for detailed documentation
- Check [skaffold.yaml](skaffold.yaml) for configuration
- Review [build script](scripts/build-all-components.sh) for build process
- Consult https://skaffold.dev/docs/ for Skaffold features
