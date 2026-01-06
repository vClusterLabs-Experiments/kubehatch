# KubeHatch Production Deployment Guide

## Overview

This guide explains how to deploy KubeHatch as a production-ready Internal Developer Portal (IDP) on Kubernetes with proper authentication, multi-user support, and internet exposure.

## Architecture Flow

```
Internet Users
    ↓
[Ingress Controller] (with TLS + Authentication)
    ↓
[KubeHatch Frontend] (Nginx serving static files)
    ↓
[KubeHatch Backend] (Go API server)
    ↓
[Host Kubernetes Cluster] (where vClusters are created)
```

## Step-by-Step Deployment

### 1. Prerequisites

- Kubernetes cluster (can be the same cluster where vClusters will be created)
- kubectl configured
- Docker/container registry access
- Domain name (optional, for production)
- Ingress controller installed (nginx-ingress recommended)

### 2. Build and Push Images

```bash
# Build backend
docker build -t your-registry/kubehatch-backend:v1.0 \
  -f backend/Dockerfile.backend backend/
docker push your-registry/kubehatch-backend:v1.0

# Build frontend
docker build -t your-registry/kubehatch-frontend:v1.0 \
  -f frontend/Dockerfile.frontend frontend/
docker push your-registry/kubehatch-frontend:v1.0
```

### 3. Prepare Kubeconfig Secret

The backend needs access to create vClusters:

```bash
# Create secret with kubeconfig that has permissions to create vClusters
kubectl create secret generic vcluster-default-kubeconfig \
  --from-file=kubeconfig=/path/to/kubeconfig \
  -n default
```

**Important:** The kubeconfig must have cluster-admin or sufficient permissions to:
- Create namespaces
- Create StatefulSets, Services, Secrets
- Manage RBAC resources

### 4. Set Up Authentication

#### Option A: Basic Authentication (Simple, Quick Setup)

Create users with htpasswd:

```bash
# Install htpasswd if not available
# macOS: brew install httpd
# Linux: apt-get install apache2-utils

# Create auth file with multiple users
htpasswd -c auth admin
htpasswd auth developer1
htpasswd auth developer2
htpasswd auth team-lead
# ... add more users

# Create Kubernetes secret
kubectl create secret generic vcluster-basic-auth \
  --from-file=auth \
  -n default
```

**Limitations:**
- All users share the same portal
- No user-specific cluster isolation
- Password management is manual

#### Option B: OAuth2/OIDC (Recommended for Production)

Integrate with your identity provider (Google Workspace, Okta, Azure AD, etc.):

**1. Deploy OAuth2 Proxy:**

```bash
helm repo add oauth2-proxy https://oauth2-proxy.github.io/manifests
helm install oauth2-proxy oauth2-proxy/oauth2-proxy \
  --set config.clientID=YOUR_CLIENT_ID \
  --set config.clientSecret=YOUR_CLIENT_SECRET \
  --set config.oidcIssuerURL=https://your-idp.com \
  --set config.cookieSecret=$(openssl rand -base64 32 | head -c 32 | base64) \
  --set config.provider=oidc
```

**2. Update Ingress to use OAuth:**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: vcluster-ingress
  annotations:
    nginx.ingress.kubernetes.io/auth-url: "https://oauth2-proxy.default.svc.cluster.local/oauth2/auth"
    nginx.ingress.kubernetes.io/auth-signin: "https://oauth2-proxy.default.svc.cluster.local/oauth2/start?rd=$escaped_request_uri"
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - vcluster-portal.yourdomain.com
      secretName: vcluster-tls
  rules:
    - host: vcluster-portal.yourdomain.com
      http:
        paths:
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: vcluster-backend
                port:
                  number: 8081
          - path: /
            pathType: Prefix
            backend:
              service:
                name: vcluster-frontend
                port:
                  number: 80
```

#### Option C: Multi-User with User Isolation (Advanced)

For true multi-tenancy where each user/team has isolated clusters:

1. **Add user identification to backend:**
   - Extract user from OAuth token or session
   - Create clusters in user-specific namespaces
   - Filter clusters by user

2. **Update backend to track users:**
   ```go
   // Add user context to cluster creation
   type VclusterInfo struct {
       Name      string
       Namespace string
       Owner     string  // Add this
       // ...
   }
   ```

3. **Implement namespace per user:**
   - Create clusters in `vcluster-<user>-<cluster-name>`
   - Filter dashboard by current user

### 5. Deploy KubeHatch

Update the deployment manifests with your image names:

```bash
# Edit k8s/backenddeploy.yaml - update image
# Edit k8s/deploymentfrontend.yaml - update image

# Deploy everything
kubectl apply -f k8s/backenddeploy.yaml
kubectl apply -f k8s/deploymentfrontend.yaml
kubectl apply -f k8s/backendsvc.yaml
kubectl apply -f k8s/svcfrontend.yaml
kubectl apply -f k8s/role.yaml
```

### 6. Configure Ingress

**For Basic Auth (current setup):**

```bash
kubectl apply -f k8s/ingress.yaml
```

**For OAuth (update ingress.yaml first):**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: vcluster-ingress
  annotations:
    nginx.ingress.kubernetes.io/auth-url: "https://oauth2-proxy.default.svc.cluster.local/oauth2/auth"
    nginx.ingress.kubernetes.io/auth-signin: "https://oauth2-proxy.default.svc.cluster.local/oauth2/start?rd=$escaped_request_uri"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"  # For TLS
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - vcluster-portal.yourdomain.com
      secretName: vcluster-tls
  rules:
    - host: vcluster-portal.yourdomain.com
      http:
        paths:
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: vcluster-backend
                port:
                  number: 8081
          - path: /
            pathType: Prefix
            backend:
              service:
                name: vcluster-frontend
                port:
                  number: 80
```

### 7. Set Up TLS (HTTPS)

**Option A: cert-manager (Recommended)**

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Create ClusterIssuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: your-email@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

**Option B: Manual TLS Secret**

```bash
# Generate certificate (use Let's Encrypt, your CA, or self-signed)
kubectl create secret tls vcluster-tls \
  --cert=tls.crt \
  --key=tls.key \
  -n default
```

### 8. Expose Over Internet

**Option A: LoadBalancer (Cloud)**

If your cluster has LoadBalancer support:
- The ingress controller will get an external IP
- Point your domain's A record to that IP
- Access via: `https://vcluster-portal.yourdomain.com`

**Option B: NodePort (On-Prem)**

```bash
# Get node IPs
kubectl get nodes -o wide

# Access via: http://<node-ip>:<nodeport>
# Or set up a reverse proxy in front
```

**Option C: Port-Forward (Testing Only)**

```bash
kubectl port-forward svc/ingress-nginx-controller 8080:80 -n ingress-nginx
# Access via: http://localhost:8080
```

## User Access Flow

### For End Users (Developers)

1. **Access Portal:**
   - Navigate to `https://vcluster-portal.yourdomain.com`
   - Authenticate (Basic Auth or OAuth)

2. **Create Cluster:**
   - Fill in cluster name
   - Select options (HA, LoadBalancer)
   - Click "Create Cluster"
   - Wait 1-2 minutes

3. **Download Kubeconfig:**
   - Click "Download Kubeconfig"
   - Use with kubectl

4. **Use Cluster:**
   ```bash
   kubectl --kubeconfig kubeconfig.yaml get nodes
   ```

## Multi-User/Team Support

### Current Implementation
- All users see all clusters
- No user isolation
- Shared namespace

### Enhanced Multi-User Support (Future)

To add proper multi-user support:

1. **Backend Changes:**
   - Extract user from auth token/session
   - Store user info with each cluster
   - Filter clusters by user

2. **Namespace Isolation:**
   - Create clusters in `vcluster-<user-id>-<cluster-name>`
   - Implement RBAC per user

3. **Team Support:**
   - Add team concept
   - Share clusters within team
   - Team-level quotas

## Troubleshooting

### Ingress Not Accessible
- Check ingress controller: `kubectl get pods -n ingress-nginx`
- Check ingress status: `kubectl describe ingress vcluster-ingress`
- Verify DNS points to ingress IP

### Authentication Not Working
- Check auth secret exists: `kubectl get secret vcluster-basic-auth`
- Verify ingress annotations
- Check OAuth proxy logs if using OAuth

### Clusters Not Creating
- Check backend logs: `kubectl logs deploy/vcluster-backend`
- Verify kubeconfig secret exists
- Check RBAC permissions

## Security Best Practices

1. **Use HTTPS:** Always use TLS in production
2. **Strong Authentication:** Use OAuth/OIDC, not Basic Auth
3. **RBAC:** Limit backend permissions to minimum required
4. **Network Policies:** Restrict pod-to-pod communication
5. **Audit Logging:** Log all cluster operations
6. **Resource Quotas:** Set limits per user/team
7. **Rate Limiting:** Prevent abuse

## Next Steps

1. Deploy with Basic Auth for testing
2. Set up OAuth for production
3. Add user tracking (if needed)
4. Configure TLS
5. Set up monitoring and alerting
6. Implement quotas and limits


