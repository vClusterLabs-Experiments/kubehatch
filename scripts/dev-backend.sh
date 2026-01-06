#!/bin/bash

# Local Backend Development Script
# This script helps you run the KubeHatch backend locally

set -e

echo "🚀 Starting KubeHatch Backend for Local Development"
echo "=================================================="

# Get the script directory and project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
BACKEND_DIR="$PROJECT_ROOT/backend"

cd "$BACKEND_DIR"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed. Please install Go 1.22 or later."
    exit 1
fi

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo "⚠️  Warning: kubectl is not installed. Backend may not work properly."
fi

# Check if vcluster CLI is installed
if ! command -v vcluster &> /dev/null; then
    echo "⚠️  Warning: vcluster CLI is not installed."
    echo "   Install it from: https://www.vcluster.com/docs/getting-started/setup"
    echo "   Or run: curl -L -o vcluster https://github.com/loft-sh/vcluster/releases/latest/download/vcluster-linux-amd64"
    echo "   Then: chmod +x vcluster && sudo mv vcluster /usr/local/bin/"
fi

# Check if kubeconfig is set
if [ -z "$KUBECONFIG" ]; then
    if [ -f "$HOME/.kube/config" ]; then
        echo "ℹ️  Using default kubeconfig: $HOME/.kube/config"
        export KUBECONFIG="$HOME/.kube/config"
    else
        echo "⚠️  Warning: No kubeconfig found. Backend will use in-cluster config if running in a pod."
        echo "   You can set KUBECONFIG environment variable or upload a kubeconfig via the UI."
    fi
else
    echo "ℹ️  Using kubeconfig: $KUBECONFIG"
fi

# Create requests directory if it doesn't exist
mkdir -p "$PROJECT_ROOT/requests"

# Install Go dependencies
echo ""
echo "📦 Installing Go dependencies..."
go mod download

# Run the backend
echo ""
echo "✅ Starting backend server on http://localhost:8081"
echo "   Press Ctrl+C to stop"
echo ""
go run main.go



