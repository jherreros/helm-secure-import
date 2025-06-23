#!/usr/bin/env bash

set -e

prerequisites=(cosign trivy copa)

# Check for prerequisites
for command in $"${prerequisites[@]}"
do
    if ! command -v $command >/dev/null 2>&1; then
        echo "Warning: $command is not installed. Please install $command if you want to get the most of this plugin."
    fi
done

# Create bin directory if it doesn't exist
mkdir -p "$HELM_PLUGIN_DIR/bin"

# Install the tool

version=$(yq '.version' < "$HELM_PLUGIN_DIR/plugin.yaml")

# Determine the OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
  x86_64)
    ARCH="amd64"
    ;;
  arm64)
    ARCH="arm64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

if [ "$OS" == "linux" ]; then
  BINARY_NAME="helm-secure-import-linux-$ARCH"
elif [ "$OS" == "darwin" ]; then
  BINARY_NAME="helm-secure-import-macos-$ARCH"
elif [ "$OS" == "windows" ]; then
  BINARY_NAME="helm-secure-import-windows-$ARCH.exe"
else
  echo "Unsupported OS: $OS"
  exit 1
fi

wget "https://github.com/jherreros/helm-secure-import/releases/download/v$version/$BINARY_NAME"
mv "$BINARY_NAME" "$HELM_PLUGIN_DIR/bin/helm-secure-import"
chmod +x "$HELM_PLUGIN_DIR/bin/helm-secure-import"

echo "Plugin installation completed successfully!"