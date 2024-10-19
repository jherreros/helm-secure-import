#!/usr/bin/env bash

set -e

# Plugin installation directory
if [ -n "${HELM_PLUGIN_DIR}" ]; then
    PLUGIN_DIR="${HELM_PLUGIN_DIR}"
else
    echo "Error: HELM_PLUGIN_DIR is not set"
    exit 1
fi

# Remove the binary
if [ -f "${PLUGIN_DIR}/bin/secure-import" ]; then
    rm "${PLUGIN_DIR}/bin/secure-import"
    echo "Removed plugin binary"
fi

# Remove the bin directory if empty
if [ -d "${PLUGIN_DIR}/bin" ] && [ -z "$(ls -A ${PLUGIN_DIR}/bin)" ]; then
    rmdir "${PLUGIN_DIR}/bin"
    echo "Removed empty bin directory"
fi

CONTAINER_IMAGE="ghcr.io/jherreros/helm-secure-import:latest"
# Remove the Docker image if it exists
if docker image inspect "$CONTAINER_IMAGE" >/dev/null 2>&1; then
    echo "Removing Docker image $CONTAINER_IMAGE..."
    docker rmi "$CONTAINER_IMAGE" || true
fi

echo "Plugin uninstallation completed successfully!"