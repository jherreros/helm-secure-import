#!/usr/bin/env bash

set -e

# Plugin configuration
PLUGIN_NAME="secure-import"
CONTAINER_IMAGE="ghcr.io/jherreros/helm-secure-import:latest"

# Ensure Docker is available
if ! command -v docker >/dev/null 2>&1; then
    echo "Error: Docker is required but not installed. Please install Docker first."
    exit 1
fi

# Parse arguments
ARGS=()
DOCKER_ARGS=()

while [[ $# -gt 0 ]]; do
    case $1 in
        --sign-key)
            # Mount the signing key into the container
            DOCKER_ARGS+=("-v" "$2:/cosign.key")
            ARGS+=("$1" "/cosign.key")
            shift 2
            ;;
        --values)
            # Mount the values file into the container
            VALUES_FILE="$2"
            VALUES_DIR=$(dirname "$VALUES_FILE")
            VALUES_FILENAME=$(basename "$VALUES_FILE")
            DOCKER_ARGS+=("-v" "$VALUES_DIR:/values")
            ARGS+=("$1" "/values/$VALUES_FILENAME")
            shift 2
            ;;
        *)
            ARGS+=("$1")
            shift
            ;;
    esac
done

# Mount Docker socket for Docker-in-Docker operations
DOCKER_ARGS+=("-v" "/var/run/docker.sock:/var/run/docker.sock")

# Mount Docker config directory for registry credentials
if [ -d "$HOME/.docker" ]; then
    DOCKER_ARGS+=("-v" "$HOME/.docker:/root/.docker:ro")
fi

# Run the container
docker run --rm \
    "${DOCKER_ARGS[@]}" \
    "$CONTAINER_IMAGE" \
    "${ARGS[@]}"