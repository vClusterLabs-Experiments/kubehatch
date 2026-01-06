# Simple Multi-User Deployment Guide

## Overview

KubeHatch is designed to be **minimalistic, cost-free, and easy to deploy**. This guide shows you how to deploy it for your team with multi-user support.

## Quick Answer: How It Works

### Deployment
1. **Deploy on Kubernetes** (or run locally)
2. **Set up Basic Auth** with multiple users (one per team/user)
3. **Expose via Ingress** (with password protection)
4. **Done!** Teams can now access and create clusters

### Multi-User Support
- Each user/team has their own username/password
- Each user sees **only their own clusters**
- No external dependencies (no OAuth, no databases)
- Everything stored in Kubernetes (namespaces, annotations)

## Step 1: Create Users

```bash
# Create auth file with multiple users (one per team)
htpasswd -c auth team1
htpasswd auth team2
htpasswd auth team3
htpasswd auth developer1
# ... add as many as needed

# Create Kubernetes secret
kubectl create secret generic vcluster-basic-auth \
  --from-file=auth \
  -n default
```

## Step 2: Deploy KubeHatch

```bash
# Build and push images (or use existing)
# Update k8s/*.yaml with your image names

# Deploy
kubectl apply -f k8s/
```

## Step 3: Configure Ingress

The ingress is already configured with Basic Auth. Just update the host:

```yaml
# k8s/ingress.yaml
spec:
  rules:
  - host: vcluster-portal.yourdomain.com  # Change this
```

Apply:
```bash
kubectl apply -f k8s/ingress.yaml
```

## Step 4: Access

1. Users navigate to: `https://vcluster-portal.yourdomain.com`
2. Enter their username/password
3. See **only their clusters**
4. Create new clusters (owned by them)

## How Multi-User Works

### Backend Implementation
- Extracts username from Basic Auth header
- Stores owner in namespace annotation: `kubehatch.io/owner=<username>`
- Filters clusters by owner when listing
- Each user only sees their own clusters

### No External Dependencies
- No database needed (uses Kubernetes annotations)
- No OAuth provider needed (Basic Auth is built-in)
- No additional services
- Just Kubernetes + KubeHatch

## For Local Development

```bash
# Run locally
./scripts/dev-backend.sh
./scripts/dev-frontend.sh

# Access: http://localhost:8080
# No auth needed locally (for testing)
```

## For Production

1. Deploy on Kubernetes
2. Set up Basic Auth with users
3. Expose via Ingress
4. Add TLS (HTTPS) - optional but recommended
5. Point DNS to ingress

## Cost

**Zero additional cost!** Only uses:
- Kubernetes cluster resources (CPU/memory)
- Storage for clusters
- No external services
- No databases
- No SaaS subscriptions

## Kind Cluster Kubeconfig Issue

For local kind clusters, the kubeconfig requires port-forwarding:

```bash
# Run in background terminal:
vcluster connect <cluster-name> --namespace vcluster-<cluster-name>

# Then use kubeconfig
kubectl --kubeconfig kubeconfig.yaml get nodes
```

**Why:** Kind clusters don't have external LoadBalancers. Port-forwarding is required.

**Solution in Production:** Use LoadBalancer option when creating clusters, or deploy on a cloud cluster with LoadBalancer support.

## Summary

✅ **Deploy:** `kubectl apply -f k8s/`  
✅ **Users:** Create with htpasswd  
✅ **Access:** Via Ingress URL  
✅ **Isolation:** Each user sees only their clusters  
✅ **Cost:** Zero (just Kubernetes resources)  
✅ **Simple:** No external dependencies


