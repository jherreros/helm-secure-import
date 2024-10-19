#!/usr/bin/env bash

set -e

# Check for Docker
if ! command -v docker >/dev/null 2>&1; then
    echo "Warning: Docker is not installed. Please install Docker before using this plugin."
    echo "Installation instructions: https://docs.docker.com/engine/install/"
    exit 1
fi

# Create bin directory if it doesn't exist
mkdir -p "$HELM_PLUGIN_DIR/bin"

# Copy the wrapper script
cp "$HELM_PLUGIN_DIR/scripts/secure-import-wrapper.sh" "$HELM_PLUGIN_DIR/bin/"
chmod +x "$HELM_PLUGIN_DIR/bin/secure-import-wrapper.sh"

# Pull the container image
docker pull ghcr.io/jherreros/helm-secure-import:latest

echo "Plugin installation completed successfully!"