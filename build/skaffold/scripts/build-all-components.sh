#!/bin/bash

set -o errexit
set -o xtrace

# This script builds all PMM Client components within the Skaffold container
# It replaces the functionality from build-client-binary script

# Set default values
# PMM_VERSION must be passed from the Makefile/Skaffold build args
if [ -z "${PMM_VERSION}" ]; then
    echo "ERROR: PMM_VERSION environment variable is not set" >&2
    echo "PMM_VERSION must be passed from the Makefile/Skaffold build args" >&2
    exit 1
fi

FULL_PMM_VERSION=${FULL_PMM_VERSION:-${PMM_VERSION}}
BUILD_TYPE=${BUILD_TYPE:-static}
GOOS=${GOOS:-linux}
GOARCH=${GOARCH:-amd64}
BUILD_MODE=${BUILD_MODE:-prod}

# Directories
SOURCE_DIR="/build/source/pmm-client-cache"  # Shared cache for all versions
BINARY_DIR="/build/binary/pmm-client-${PMM_VERSION}"
OUTPUT_DIR="/build/output"
WORKSPACE_DIR="/workspace"

# Create necessary directories
mkdir -p "${SOURCE_DIR}"
mkdir -p "${BINARY_DIR}/bin"
mkdir -p "${OUTPUT_DIR}"

# Set build timestamp and git info
export PMM_RELEASE_VERSION=${FULL_PMM_VERSION}
PMM_RELEASE_TIMESTAMP=$(date '+%s')
export PMM_RELEASE_TIMESTAMP
export PMM_RELEASE_PATH="${BINARY_DIR}/bin"

# Fetch component versions from pmm-submodules
GITMODULES_URL="https://raw.githubusercontent.com/Percona-Lab/pmm-submodules/refs/heads/v3/.gitmodules"
GITMODULES_FILE="${SOURCE_DIR}/.gitmodules"

echo "Fetching component versions from ${GITMODULES_URL}..."
curl -fsSL "${GITMODULES_URL}" -o "${GITMODULES_FILE}"

# Build the .gitmodules parser
GITMODULES="/tmp/gitmodules"
GITMODULES_SOURCE="$(dirname "$0")/gitmodules.go"

echo "Building .gitmodules parser..."
go build -o "${GITMODULES}" "${GITMODULES_SOURCE}"

# Function to extract git reference (branch/tag/commit) from .gitmodules
get_component_ref() {
    local component=$1
    local ref
    
    # Try branch first, then tag
    ref=$("${GITMODULES}" "${GITMODULES_FILE}" "${component}" "branch" 2>/dev/null || \
          "${GITMODULES}" "${GITMODULES_FILE}" "${component}" "tag" 2>/dev/null || true)
    
    # If not found in .gitmodules, check for fallback commit hash
    if [ -z "$ref" ]; then
        case "${component}" in
            VictoriaMetrics)
                # https://github.com/VictoriaMetrics/VictoriaMetrics/releases/tag/pmm-6401-v1.114.0
                ref="a5e3c6d4492db765800363dfae48a04b4d7888be"
                ;;
            redis_exporter)
                # https://github.com/oliver006/redis_exporter/releases/tag/v1.72.1
                ref="8d5f9dea4a8863ce7cad62352957ae5e75047346"
                ;;
            nomad)
                # https://github.com/hashicorp/nomad/releases/tag/v1.11.0
                ref="9103d938133311b2da905858801f0e111a2df0a1"
                ;;
            *)
                echo "ERROR: Could not find branch/tag for component '${component}' in .gitmodules or fallback" >&2
                return 1
                ;;
        esac
        echo "Using fallback commit hash for ${component}: ${ref}" >&2
    fi
    
    echo "$ref"
}

# Function to extract git URL from .gitmodules
get_component_url() {
    local component=$1
    local url
    
    url=$("${GITMODULES}" "${GITMODULES_FILE}" "${component}" "url" 2>/dev/null || true)
    
    # If not found in .gitmodules, check for fallback URLs
    if [ -z "$url" ]; then
        case "${component}" in
            VictoriaMetrics)
                url="https://github.com/VictoriaMetrics/VictoriaMetrics.git"
                ;;
            redis_exporter)
                url="https://github.com/oliver006/redis_exporter.git"
                ;;
            nomad)
                url="https://github.com/hashicorp/nomad.git"
                ;;
            *)
                echo "ERROR: Could not find URL for component '${component}' in .gitmodules or fallback" >&2
                return 1
                ;;
        esac
        echo "Using fallback URL for ${component}: ${url}" >&2
    fi
    
    echo "$url"
}

echo "Component versions loaded from pmm-submodules"

# Function to build a Go component from the workspace
build_workspace_component() {
    local component=$1
    local component_dir=$2
    local output_name=${3:-$component}
    local extra_flags=${4:-}
    
    echo "Building ${component} from ${component_dir}..."
    
    pushd "${WORKSPACE_DIR}/${component_dir}"
    
    # Get git info for the component
    export PMM_RELEASE_FULLCOMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    export PMM_RELEASE_BRANCH=$(git describe --always --contains --all 2>/dev/null || echo "unknown")
    export COMPONENT_VERSION=$(git describe --abbrev=0 --always 2>/dev/null || echo "${PMM_VERSION}")
    
    # Build based on component and build type
    case "${component}" in
        pmm-admin)
            if [ "${BUILD_MODE}" = "dev" ]; then
                CGO_ENABLED=0 go build -race -v -o "${PMM_RELEASE_PATH}/${output_name}" ./cmd/pmm-admin/
            else
                make release PMM_RELEASE_PATH="${PMM_RELEASE_PATH}"
            fi
            ;;
        pmm-agent)
            if [ "${BUILD_TYPE}" = "dynamic" ]; then
                make release-gssapi PMM_RELEASE_PATH="${PMM_RELEASE_PATH}"
            elif [ "${BUILD_MODE}" = "dev" ]; then
                make release-dev PMM_RELEASE_PATH="${PMM_RELEASE_PATH}"
            else
                make release PMM_RELEASE_PATH="${PMM_RELEASE_PATH}"
            fi
            ;;
        *)
            echo "Unknown component: ${component}"
            return 1
            ;;
    esac
    
    popd
}

# Function to build external component from source
build_external_component() {
    local component=$1
    local git_url=$2
    local git_ref=$3
    local make_target=$4
    local binary_path=$5
    local extra_env=$6
    
    echo "Building external component: ${component}"
    
    local component_src="${SOURCE_DIR}/${component}"
    
    if [ ! -d "${component_src}" ]; then
        echo "Cloning ${component} from ${git_url}..."
        
        # Check if git_ref looks like a commit hash (40 hex chars or shorter hash)
        if [[ "${git_ref}" =~ ^[0-9a-f]{7,40}$ ]]; then
            # For commit hashes, clone without --branch and checkout the hash
            git clone "${git_url}" "${component_src}"
            pushd "${component_src}"
            git checkout "${git_ref}"
            popd
        else
            # For branches/tags, use --branch for shallow clone
            git clone --depth 1 --branch "${git_ref}" "${git_url}" "${component_src}"
        fi
    else
        echo "Updating existing ${component} repository..."
        pushd "${component_src}"
        
        # Check if git_ref looks like a commit hash
        if [[ "${git_ref}" =~ ^[0-9a-f]{7,40}$ ]]; then
            # For commit hashes, fetch and checkout
            git fetch origin "${git_ref}" || git fetch origin
            git checkout "${git_ref}"
        else
            # For branches/tags, fetch the specific ref
            git fetch --depth 1 origin "${git_ref}"
            git reset --hard FETCH_HEAD
        fi
        
        # Clean any untracked files
        git clean -fdx
        
        popd
    fi
    
    pushd "${component_src}"
    
    PMM_RELEASE_FULLCOMMIT=$(git rev-parse HEAD)
    export PMM_RELEASE_FULLCOMMIT
    COMPONENT_VERSION=$(git describe --abbrev=0 --always)
    export COMPONENT_VERSION
    
    # Execute make command with optional environment variables
    # shellcheck disable=SC2086
    if [ -n "${extra_env}" ]; then
        env ${extra_env} make ${make_target}
    else
        make ${make_target}
    fi
    
    local target_path
    # Copy built binary only if it wasn't already placed by make target
    target_path="${BINARY_DIR}/bin/$(basename "${binary_path}")"
    if [ ! -f "${target_path}" ]; then
        cp "${binary_path}" "${BINARY_DIR}/bin/"
    else
        echo "Binary already in place: ${target_path}"
    fi
    
    popd
}

# Build PMM components
echo "=== Building PMM Admin ==="
build_workspace_component "pmm-admin" "admin" "pmm-admin"

echo "=== Building PMM Agent ==="
build_workspace_component "pmm-agent" "agent" "pmm-agent"

# Build external exporters and tools
echo "=== Building Node Exporter ==="
NODE_EXPORTER_URL=$(get_component_url "node_exporter")
NODE_EXPORTER_REF=$(get_component_ref "node_exporter")
build_external_component \
    "node_exporter" \
    "${NODE_EXPORTER_URL}" \
    "${NODE_EXPORTER_REF}" \
    "release" \
    "node_exporter"

echo "=== Building MySQL Exporter ==="
MYSQLD_EXPORTER_URL=$(get_component_url "mysqld_exporter")
MYSQLD_EXPORTER_REF=$(get_component_ref "mysqld_exporter")
build_external_component \
    "mysqld_exporter" \
    "${MYSQLD_EXPORTER_URL}" \
    "${MYSQLD_EXPORTER_REF}" \
    "release" \
    "mysqld_exporter"

echo "=== Building PostgreSQL Exporter ==="
POSTGRES_EXPORTER_URL=$(get_component_url "postgres_exporter")
POSTGRES_EXPORTER_REF=$(get_component_ref "postgres_exporter")
build_external_component \
    "postgres_exporter" \
    "${POSTGRES_EXPORTER_URL}" \
    "${POSTGRES_EXPORTER_REF}" \
    "release" \
    "postgres_exporter"

echo "=== Building MongoDB Exporter ==="
MONGODB_EXPORTER_URL=$(get_component_url "mongodb_exporter")
MONGODB_EXPORTER_REF=$(get_component_ref "mongodb_exporter")
if [ "${BUILD_TYPE}" = "dynamic" ]; then
    MONGODB_MAKE_TARGET="build-gssapi"
else
    MONGODB_MAKE_TARGET="build"
fi
build_external_component \
    "mongodb_exporter" \
    "${MONGODB_EXPORTER_URL}" \
    "${MONGODB_EXPORTER_REF}" \
    "${MONGODB_MAKE_TARGET}" \
    "mongodb_exporter"

echo "=== Building ProxySQL Exporter ==="
PROXYSQL_EXPORTER_URL=$(get_component_url "proxysql_exporter")
PROXYSQL_EXPORTER_REF=$(get_component_ref "proxysql_exporter")
build_external_component \
    "proxysql_exporter" \
    "${PROXYSQL_EXPORTER_URL}" \
    "${PROXYSQL_EXPORTER_REF}" \
    "release" \
    "proxysql_exporter"

echo "=== Building RDS Exporter ==="
RDS_EXPORTER_URL=$(get_component_url "rds_exporter")
RDS_EXPORTER_REF=$(get_component_ref "rds_exporter")
build_external_component \
    "rds_exporter" \
    "${RDS_EXPORTER_URL}" \
    "${RDS_EXPORTER_REF}" \
    "release" \
    "rds_exporter"

echo "=== Building Azure Metrics Exporter ==="
AZURE_EXPORTER_URL=$(get_component_url "azure_metrics_exporter")
AZURE_EXPORTER_REF=$(get_component_ref "azure_metrics_exporter")
build_external_component \
    "azure_metrics_exporter" \
    "${AZURE_EXPORTER_URL}" \
    "${AZURE_EXPORTER_REF}" \
    "release" \
    "azure_exporter"

echo "=== Building VictoriaMetrics Agent ==="
VMAGENT_URL=$(get_component_url "VictoriaMetrics")
VMAGENT_REF=$(get_component_ref "VictoriaMetrics")
build_external_component \
    "vmagent" \
    "${VMAGENT_URL}" \
    "${VMAGENT_REF}" \
    "vmagent" \
    "bin/vmagent"

echo "=== Building Redis Exporter ==="
REDIS_EXPORTER_URL=$(get_component_url "redis_exporter")
REDIS_EXPORTER_REF=$(get_component_ref "redis_exporter")
build_external_component \
    "redis_exporter" \
    "${REDIS_EXPORTER_URL}" \
    "${REDIS_EXPORTER_REF}" \
    "build" \
    "redis_exporter"

# Copy as valkey_exporter
cp "${BINARY_DIR}/bin/redis_exporter" "${BINARY_DIR}/bin/valkey_exporter"

echo "=== Building Nomad ==="
# Determine target based on architecture
case "${GOARCH}" in
    amd64)
        NOMAD_TARGET="linux_amd64"
        ;;
    arm64)
        NOMAD_TARGET="linux_arm64"
        ;;
    *)
        echo "Unsupported architecture: ${GOARCH}"
        exit 1
        ;;
esac

NOMAD_URL=$(get_component_url "nomad")
NOMAD_REF=$(get_component_ref "nomad")
build_external_component \
    "nomad" \
    "${NOMAD_URL}" \
    "${NOMAD_REF}" \
    "deps release" \
    "pkg/${NOMAD_TARGET}/nomad" \
    "TARGETS=${NOMAD_TARGET}"

# Copy configuration and auxiliary files from workspace
echo "=== Copying configuration files ==="
cp -r "${WORKSPACE_DIR}/build/packages/rpm/client" "${BINARY_DIR}/rpm"
cp -r "${WORKSPACE_DIR}/build/packages/config" "${BINARY_DIR}/config"
cp -r "${WORKSPACE_DIR}/build/packages/deb" "${BINARY_DIR}/debian"
cp -r "${WORKSPACE_DIR}/build/scripts/install_tarball" "${BINARY_DIR}/install_tarball"

# Copy exporter configuration files
if [ -d "${SOURCE_DIR}/node_exporter" ]; then
    cp "${SOURCE_DIR}/node_exporter/example.prom" "${BINARY_DIR}/"
fi

if [ -d "${SOURCE_DIR}/mysqld_exporter" ]; then
    cp "${SOURCE_DIR}/mysqld_exporter/queries-mysqld.yml" "${BINARY_DIR}/"
    [ -f "${SOURCE_DIR}/mysqld_exporter/queries-mysqld-group-replication.yml" ] && \
        cp "${SOURCE_DIR}/mysqld_exporter/queries-mysqld-group-replication.yml" "${BINARY_DIR}/"
fi

if [ -d "${SOURCE_DIR}/postgres_exporter" ]; then
    for file in example-queries-postgres.yml queries-postgres-uptime.yml queries-hr.yml queries-mr.yaml queries-lr.yaml; do
        [ -f "${SOURCE_DIR}/postgres_exporter/${file}" ] && \
            cp "${SOURCE_DIR}/postgres_exporter/${file}" "${BINARY_DIR}/"
    done
fi

# Build Percona Toolkit components
echo "=== Building Percona Toolkit ==="
PT_SRC="${SOURCE_DIR}/percona-toolkit"
if [ ! -d "${PT_SRC}" ]; then
    echo "Cloning percona-toolkit..."
    git clone --depth 1 https://github.com/percona/percona-toolkit.git "${PT_SRC}"
else
    echo "Updating existing percona-toolkit repository..."
    pushd "${PT_SRC}"
    git fetch --depth 1 origin HEAD
    git reset --hard FETCH_HEAD
    git clean -fdx
    popd
fi

# Copy Perl scripts
cp "${PT_SRC}/bin/pt-summary" "${BINARY_DIR}/bin/"
cp "${PT_SRC}/bin/pt-mysql-summary" "${BINARY_DIR}/bin/"

# Build Go-based toolkit components
pushd "${PT_SRC}/src/go/pt-mongodb-summary"
go build -o "${BINARY_DIR}/bin/pt-mongodb-summary" .
popd

pushd "${PT_SRC}/src/go/pt-pg-summary"
go build -o "${BINARY_DIR}/bin/pt-pg-summary" .
popd

# Write version file
echo "${PMM_VERSION}" > "${BINARY_DIR}/VERSION"

# Create tarball
echo "=== Creating tarball ==="
if [ "${BUILD_TYPE}" = "dynamic" ]; then
    TARBALL_NAME="pmm-client-${PMM_VERSION}-dynamic.tar.gz"
else
    TARBALL_NAME="pmm-client-${PMM_VERSION}.tar.gz"
fi

tar -C "$(dirname "${BINARY_DIR}")" -zcpf "${OUTPUT_DIR}/${TARBALL_NAME}" "$(basename "${BINARY_DIR}")"

echo "=== Build completed successfully ==="
echo "Build type: ${BUILD_TYPE}"
echo "Architecture: ${GOARCH}"
echo "Version: ${PMM_VERSION}"
echo "Output tarball: ${OUTPUT_DIR}/${TARBALL_NAME}"

ls -lh "${OUTPUT_DIR}/"
ls -lh "${BINARY_DIR}/bin/"
