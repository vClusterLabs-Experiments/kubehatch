# KubeHatch IDP Architecture & Deployment Guide

## Overview

KubeHatch is designed as an **Internal Developer Portal (IDP)** that allows platform teams to provide self-service virtual cluster provisioning to developers. This document explains how it works and how to deploy it in production.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Developer Access                          │
│  (Web Browser) → Ingress → Frontend (Nginx)                 │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                    Platform Team Hosts                       │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Authentication Layer (Ingress Basic Auth / OAuth)  │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Frontend (Static UI)                                │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Backend API (Go)                                    │   │
│  │  - Creates vClusters                                 │   │
│  │  - Manages lifecycle                                 │   │
│  │  - Returns kubeconfigs                               │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│              Host Kubernetes Cluster                        │
│  (Where vClusters are created)                              │
│  - vcluster-deployment-1                                    │
│  - vcluster-deployment-2                                    │
│  - vcluster-deployment-N                                    │
└─────────────────────────────────────────────────────────────┘
```

## How It Works

### 1. **Platform Team Deployment**
   - Platform team deploys KubeHatch on a Kubernetes cluster
   - Configures authentication (Basic Auth via Ingress, or OAuth/OIDC)
   - Sets up RBAC so backend can create vClusters
   - Exposes the portal via Ingress

### 2. **Developer Access**
   - Developers access the portal URL (e.g., `https://vcluster-portal.company.com`)
   - Authenticate via configured method (Basic Auth, OAuth, etc.)
   - See dashboard with their clusters
   - Create new virtual clusters on-demand

### 3. **Cluster Creation Flow**
   ```
   Developer clicks "Create Cluster"
        ↓
   Frontend → Backend API (POST /api/vcluster)
        ↓
   Backend creates vCluster using vcluster CLI
        ↓
   Backend waits for cluster to be ready
        ↓
   Backend fetches kubeconfig from Kubernetes secret
        ↓
   Backend returns kubeconfig to Frontend
        ↓
   Developer downloads kubeconfig and uses it
   ```

## Authentication Options

### Option 1: Basic Authentication (Current Setup)
The ingress is configured with Basic Auth. You can create users:

```bash
# Create auth file
htpasswd -c auth admin
htpasswd auth developer1
htpasswd auth developer2

# Create Kubernetes secret
kubectl create secret generic vcluster-basic-auth \
  --from-file=auth \
  --dry-run=client -o yaml | kubectl apply -f -
```

**Pros:** Simple, quick to set up
**Cons:** Not ideal for many users, password management

### Option 2: OAuth/OIDC (Recommended for Production)
Integrate with your identity provider (Google, Okta, Azure AD, etc.):

1. **Update Ingress** to use OAuth proxy:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: vcluster-ingress
  annotations:
    nginx.ingress.kubernetes.io/auth-url: "https://oauth-proxy/oauth2/auth"
    nginx.ingress.kubernetes.io/auth-signin: "https://oauth-proxy/oauth2/start"
spec:
  # ... rest of config
```

2. **Deploy OAuth2 Proxy:**
```bash
helm repo add oauth2-proxy https://oauth2-proxy.github.io/manifests
helm install oauth2-proxy oauth2-proxy/oauth2-proxy \
  --set config.clientID=YOUR_CLIENT_ID \
  --set config.clientSecret=YOUR_CLIENT_SECRET \
  --set config.oidcIssuerURL=https://your-idp.com
```

### Option 3: Service Mesh Authentication
If using Istio/Linkerd, use their authentication mechanisms.

## Deployment Steps

### 1. Build and Push Images

```bash
# Build backend
docker build -t your-registry/kubehatch-backend:v1.0 \
  -f backend/Dockerfile.backend backend/

# Build frontend
docker build -t your-registry/kubehatch-frontend:v1.0 \
  -f frontend/Dockerfile.frontend frontend/

# Push to registry
docker push your-registry/kubehatch-backend:v1.0
docker push your-registry/kubehatch-frontend:v1.0
```

### 2. Prepare Kubeconfig Secret

The backend needs access to the host cluster to create vClusters:

```bash
# Create secret with kubeconfig
kubectl create secret generic vcluster-default-kubeconfig \
  --from-file=kubeconfig=$HOME/.kube/config \
  -n default
```

**Important:** This kubeconfig should have permissions to:
- Create namespaces
- Create StatefulSets, Services, Secrets
- Manage RBAC resources

### 3. Update Deployment Manifests

Update `k8s/backenddeploy.yaml` and `k8s/deploymentfrontend.yaml` with your image names.

### 4. Deploy

```bash
kubectl apply -f k8s/
```

### 5. Configure Ingress

Update `k8s/ingress.yaml` with your domain and authentication method.

### 6. Access Portal

Navigate to your ingress URL and authenticate.

## User Experience Flow

1. **Login** → Developer authenticates
2. **Dashboard** → See all their virtual clusters
3. **Create Cluster** → Fill form, click create
4. **Wait** → Cluster provisioning (1-2 minutes)
5. **Download** → Get kubeconfig file
6. **Use** → `kubectl --kubeconfig kubeconfig.yaml get nodes`

## Multi-Tenancy Considerations

### Current Implementation
- All clusters are created in the same host cluster
- Namespace isolation: Each vCluster gets its own namespace (`vcluster-<name>`)
- RBAC: Backend service account has cluster-wide permissions

### Future Enhancements
- **User Namespace Isolation**: Create clusters in user-specific namespaces
- **Quota Management**: Limit resources per user/team
- **Cost Tracking**: Track cluster usage and costs
- **Auto-cleanup**: Automatically delete idle clusters

## Security Best Practices

1. **RBAC**: Limit backend permissions to minimum required
2. **Network Policies**: Restrict network access
3. **Secret Management**: Use proper secret management (Vault, Sealed Secrets)
4. **Audit Logging**: Log all cluster creation/deletion events
5. **Rate Limiting**: Prevent abuse with rate limits
6. **Resource Quotas**: Set limits on cluster resources

## Monitoring & Observability

Add monitoring for:
- Cluster creation success/failure rates
- Active cluster count
- Resource usage
- API response times
- Error rates

## Troubleshooting

### Clusters not showing in dashboard
- Check backend logs: `kubectl logs deploy/vcluster-backend`
- Verify kubeconfig secret exists
- Check RBAC permissions

### Cluster creation fails
- Check vcluster CLI version
- Verify host cluster has resources
- Check backend logs for errors

### Authentication issues
- Verify ingress annotations
- Check auth secret exists
- Test with curl: `curl -u user:pass https://portal-url`

## Next Steps

1. **Add User Management**: Track which user created which cluster
2. **Add Quotas**: Limit clusters per user
3. **Add Notifications**: Email/Slack when cluster is ready
4. **Add Cost Tracking**: Track resource usage
5. **Add Auto-cleanup**: Delete idle clusters after X days



