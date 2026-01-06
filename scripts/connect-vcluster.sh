#!/bin/bash

# Helper script to connect to a vcluster and set up port-forwarding
# Usage: ./connect-vcluster.sh <cluster-name> [namespace]

set -e

CLUSTER_NAME="${1}"
NAMESPACE="${2:-vcluster-${CLUSTER_NAME}}"

if [ -z "$CLUSTER_NAME" ]; then
    echo "❌ Error: Cluster name is required"
    echo ""
    echo "Usage: $0 <cluster-name> [namespace]"
    echo ""
    echo "Example:"
    echo "  $0 de"
    echo "  $0 de vcluster-de"
    echo ""
    echo "Available clusters:"
    vcluster list 2>/dev/null | tail -n +3 | awk '{print "  - " $1 " (namespace: " $2 ")"}' || echo "  (No clusters found or vcluster CLI not available)"
    exit 1
fi

echo "🔌 Connecting to vcluster: $CLUSTER_NAME"
echo "📦 Namespace: $NAMESPACE"
echo ""

# Check if cluster exists
if ! kubectl get namespace "$NAMESPACE" &>/dev/null; then
    echo "❌ Error: Namespace '$NAMESPACE' not found"
    echo ""
    echo "Available namespaces with vclusters:"
    kubectl get namespaces | grep vcluster- | awk '{print "  - " $1}' || echo "  (None found)"
    exit 1
fi

# Check if vcluster pod is running
if ! kubectl get pods -n "$NAMESPACE" | grep -q Running; then
    echo "⚠️  Warning: No running pods found in namespace '$NAMESPACE'"
    echo "   The vcluster might still be starting up..."
    echo ""
    kubectl get pods -n "$NAMESPACE"
    echo ""
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo "✅ Starting port-forwarding to localhost:8443..."
echo "   This will forward port 8443 to the vcluster service."
echo "   Keep this terminal running. Press Ctrl+C to stop."
echo ""

# Use kubectl port-forward directly - more reliable and stays running
# Forward local port 8443 to the vcluster service port 443
kubectl port-forward -n "$NAMESPACE" "svc/$CLUSTER_NAME" 8443:443

