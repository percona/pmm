#!/usr/bin/env bash
# Build the PMM headless Docker image.
#
# Usage:
#   ./scripts/build-image.sh                      # builds with tag "latest"
#   ./scripts/build-image.sh v3.5.0-do.1          # builds with specific tag
#   REGISTRY=my.registry.io/team ./scripts/build-image.sh v3.5.0-do.1
#
# Environment:
#   REGISTRY   Container registry (default: registry.digitalocean.com/microserv-testing)
#   PLATFORM   Docker platform    (default: linux/amd64)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

REGISTRY="${REGISTRY:-registry.digitalocean.com/microserv-testing}"
IMAGE_NAME="pmm-headless"
IMAGE_TAG="${1:-latest}"
PLATFORM="${PLATFORM:-linux/amd64}"
FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"

log() { echo "[build] $*"; }

log "Building ${FULL_IMAGE}"
log "  Source:     ${REPO_ROOT}"
log "  Dockerfile: build/docker/server/Dockerfile"
log "  Platform:   ${PLATFORM}"
echo ""

docker build \
    --platform "${PLATFORM}" \
    --build-arg "VERSION=${IMAGE_TAG}" \
    -f "${REPO_ROOT}/build/docker/server/Dockerfile" \
    -t "${FULL_IMAGE}" \
    "${REPO_ROOT}"

echo ""
log "Image built: ${FULL_IMAGE}"
docker images "${FULL_IMAGE}" --format "  Size: {{.Size}}"

echo ""
log "To push:"
log "  docker push ${FULL_IMAGE}"
