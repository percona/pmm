# PMM Client Skaffold Build - Quick Reference

## One-Command Builds

```bash
# Navigate to skaffold directory
cd /Users/alex/Projects/pmm/pmm5/build/skaffold

# Build all components (default: static, amd64)
make build

# Build and extract binaries
make test-build

# Build for production (dynamic with GSSAPI)
make build-dynamic

# Build for ARM64
make build-arm64
```

## Build Individual Components

```bash
# Using make (recommended)
make build-component COMPONENT=pmm-admin
make build-component COMPONENT=node-exporter

# Using skaffold directly (must set env vars)
PMM_VERSION=$(cat ../../VERSION) BASE_IMAGE=golang:latest skaffold build -b pmm-admin
PMM_VERSION=$(cat ../../VERSION) BASE_IMAGE=golang:latest skaffold build -b node-exporter

# Build multiple specific components
PMM_VERSION=$(cat ../../VERSION) BASE_IMAGE=golang:latest skaffold build -b pmm-admin -b pmm-agent -b node-exporter
```

## All Components

**Workspace:** pmm-admin, pmm-agent

**Exporters:** node-exporter, mysqld-exporter, mongodb-exporter, postgres-exporter, proxysql-exporter, rds-exporter, azure-metrics-exporter, redis-exporter

**Tools:** vmagent, nomad, percona-toolkit

## Manual Skaffold Commands

```bash
# Build all components
skaffold build

# Build all with dynamic linking
BUILD_TYPE=dynamic skaffold build --profile=dynamic

# Build all for ARM64
GOARCH=arm64 skaffold build --profile=arm64
```

## Extract Binaries

```bash
# Extract from all component images
make extract
# Binaries will be in ../bin/

# Extract from specific component
docker create --name temp-extract node-exporter:3.6.0
docker cp temp-extract:/output/node_exporter ./
docker rm temp-extract
```

## Build Profiles

| Profile | Affects | Description | Command |
|---------|---------|-------------|---------|
| default | all | Static, AMD64 | `make build` |
| dynamic | pmm-agent, mongodb-exporter | Dynamic linking (GSSAPI) | `make build-dynamic` |
| arm64 | all | ARM64 architecture | `make build-arm64` |

## Common Workflows

### Daily Development - Single Component
```bash
# Work on pmm-admin
cd /Users/alex/Projects/pmm/pmm5/admin
# ... make changes ...

# Build and test just pmm-admin
cd ../build/skaffold
skaffold build -b pmm-admin

# Extract binary
docker create --name temp pmm-admin:3.6.0
docker cp temp:/output/pmm-admin ./
docker rm temp

# Test
./pmm-admin --version
```

### Daily Development - All Components
```bash
# Make changes to any component
cd /Users/alex/Projects/pmm/pmm5

# Build all and extract
cd build/skaffold
make test-build

# Check results
ls -lh ../bin/
```

### Rebuild After Failure
```bash
# If node-exporter fails, just rebuild it
skaffold build -b node-exporter

# If multiple components failed
skaffold build -b node-exporter -b mysqld-exporter
```

### Release Build
```bash
cd /Users/alex/Projects/pmm/pmm5/build/skaffold

# Build all components
make build

# Extract all binaries
make extract

# Binaries are in ../bin/
ls -lh ../bin/
```

### CI/CD Integration
```bash
# Build all components (parallel, up to 4 at a time)
cd build/skaffold
make build

# Extract to specific directory
make extract RESULTS_DIR=/workspace/artifacts

# Or build subset for faster CI
skaffold build -b pmm-admin -b pmm-agent
```

## Troubleshooting

### Build Failure - One Component
```bash
# Check which components were built successfully
docker images | grep "3.6.0"

# Rebuild just the failed component
skaffold build -b <component-name>
```

### No images found
```bash
# List all component images
docker images | grep -E "pmm-admin|pmm-agent|exporter|vmagent|nomad|toolkit"

# If missing, rebuild specific ones
skaffold build -b pmm-admin -b node-exporter
```

### Clean start
```bash
make clean
make build
```

### Check build logs for specific component
```bash
# Build with verbose output
skaffold build -b node-exporter -v debug
```

### PMM_VERSION not set
```bash
# Check VERSION file
cat ../../VERSION

# Or set manually
PMM_VERSION=3.6.0 make build
```

## Directory Structure

```
build/skaffold/
├── Makefile                    # Convenience targets
├── README.md                   # Full documentation
├── QUICKSTART.md              # This file
├── skaffold.yaml              # Skaffold config (13 artifacts)
├── Dockerfile.component       # Workspace components
├── Dockerfile.external        # External components
└── scripts/
    ├── gitmodules.go          # .gitmodules parser
    └── component-helpers.sh   # Shared build logic
```

## Output Structure

After `make extract`:

```
../bin/
├── pmm-admin
├── pmm-agent
├── node_exporter
├── mysqld_exporter
├── postgres_exporter
├── mongodb_exporter
├── proxysql_exporter
├── rds_exporter
├── azure_exporter
├── redis_exporter
├── valkey_exporter
├── vmagent
├── nomad
├── pt-summary
├── pt-mysql-summary
├── pt-mongodb-summary
└── pt-pg-summary
```
        ├── install_tarball     # Installation script
        └── VERSION             # Version file
```

## Next Steps

- Read [README.md](README.md) for detailed documentation
- Check [skaffold.yaml](skaffold.yaml) for configuration
- Review [build script](scripts/build-all-components.sh) for build process
- Consult https://skaffold.dev/docs/ for Skaffold features
