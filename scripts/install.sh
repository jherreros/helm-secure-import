#!/usr/bin/env bash

set -e

prerequisites=(cosign trivy copa yq docker)

# Check for Docker
for command in $prerequisites
do
    if ! command -v $command >/dev/null 2>&1; then
        echo "Warning: $command is not installed. Please install $command before using this plugin."
    fi
done

# Create bin directory if it doesn't exist
mkdir -p "$HELM_PLUGIN_DIR/bin"

# Install the tool

version=$(yq '.version' < "$HELM_PLUGIN_DIR/plugin.yaml")
wget "https://github.com/jherreros/helm-secure-import/releases/download/v$version/helm-secure-import"
mv helm-secure-import "$HELM_PLUGIN_DIR/bin/"
chmod +x "$HELM_PLUGIN_DIR/bin/helm-secure-import"

echo "Plugin installation completed successfully!"