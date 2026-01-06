#!/bin/bash

# Script to update vcluster to v0.30.4

set -e

VERSION="v0.30.4"
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

echo "🔄 Updating vcluster to $VERSION"
echo "=================================="

# Determine architecture
if [ "$ARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then
    ARCH_NAME="arm64"
elif [ "$ARCH" = "x86_64" ] || [ "$ARCH" = "amd64" ]; then
    ARCH_NAME="amd64"
else
    echo "❌ Unsupported architecture: $ARCH"
    exit 1
fi

# Determine OS
if [ "$OS" = "darwin" ]; then
    OS_NAME="darwin"
elif [ "$OS" = "linux" ]; then
    OS_NAME="linux"
else
    echo "❌ Unsupported OS: $OS"
    exit 1
fi

BINARY_NAME="vcluster-${OS_NAME}-${ARCH_NAME}"
DOWNLOAD_URL="https://github.com/loft-sh/vcluster/releases/download/${VERSION}/${BINARY_NAME}"

echo "📥 Downloading vcluster $VERSION for $OS_NAME/$ARCH_NAME..."
curl -LO "$DOWNLOAD_URL"

echo "🔧 Installing vcluster..."
chmod +x "$BINARY_NAME"

# Backup existing vcluster if it exists
if command -v vcluster &> /dev/null; then
    CURRENT_VERSION=$(vcluster version 2>&1 | head -1 || echo "unknown")
    echo "📦 Current version: $CURRENT_VERSION"
    echo "💾 Backing up existing vcluster..."
    sudo cp /usr/local/bin/vcluster /usr/local/bin/vcluster.backup.$(date +%Y%m%d) 2>/dev/null || true
fi

# Install new version
echo "⬆️  Installing new version..."
sudo mv "$BINARY_NAME" /usr/local/bin/vcluster

echo "✅ Verifying installation..."
NEW_VERSION=$(vcluster version 2>&1 | head -1 || echo "unknown")
echo "🎉 vcluster updated successfully!"
echo "   New version: $NEW_VERSION"

# Cleanup
rm -f "$BINARY_NAME"



