#!/usr/bin/env bash

set -e

prerequisites=(cosign trivy copa)

# Check for prerequisites
for command in "${prerequisites[@]}"
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
    echo "Error: Unsupported architecture: $ARCH"
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
  echo "Error: Unsupported OS: $OS"
  exit 1
fi

DOWNLOAD_URL="https://github.com/jherreros/helm-secure-import/releases/download/v$version/$BINARY_NAME"
CHECKSUM_URL="$DOWNLOAD_URL.sha256"

echo "Downloading $BINARY_NAME..."

# Download the binary
if ! wget -q "$DOWNLOAD_URL"; then
    echo "Error: Failed to download $BINARY_NAME from $DOWNLOAD_URL"
    echo "Please check your internet connection and verify the release exists."
    exit 1
fi

# Download and verify checksum if available
if wget -q "$CHECKSUM_URL" 2>/dev/null; then
    echo "Verifying checksum..."
    if command -v sha256sum >/dev/null 2>&1; then
        echo "$(cat "$BINARY_NAME.sha256")  $BINARY_NAME" | sha256sum -c -
    elif command -v shasum >/dev/null 2>&1; then
        echo "$(cat "$BINARY_NAME.sha256")  $BINARY_NAME" | shasum -a 256 -c -
    else
        echo "Warning: No checksum verification tool found (sha256sum or shasum). Skipping verification."
    fi
    rm -f "$BINARY_NAME.sha256"
else
    echo "Warning: Checksum file not found. Skipping verification."
fi

# Move and set permissions
mv "$BINARY_NAME" "$HELM_PLUGIN_DIR/bin/helm-secure-import"
chmod +x "$HELM_PLUGIN_DIR/bin/helm-secure-import"

echo "Plugin installation completed successfully!"