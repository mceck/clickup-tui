#!/usr/bin/env bash

set -e

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Convert architecture to Go format
case $ARCH in
  x86_64)
    ARCH="amd64"
    ;;
  aarch64|arm64)
    ARCH="arm64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Create temporary directory for download
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

echo "Downloading latest release for $OS $ARCH..."

# Get the latest release URL and download it
LATEST_URL="https://github.com/mceck/clickup-tui/releases/latest/download/clickup-tui-${OS}-${ARCH}.tar.gz"
curl -Lfo clickup-tui.tar.gz "$LATEST_URL"

# Extract the archive
tar xzf clickup-tui.tar.gz

# Install the binary
sudo mv clickup-tui /usr/local/bin
sudo chmod +x /usr/local/bin/clickup-tui

# Clean up
cd - > /dev/null
rm -rf "$TMP_DIR"

echo "clickup-tui has been installed successfully!"
