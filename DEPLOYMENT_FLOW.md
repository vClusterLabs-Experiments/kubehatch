# Complete Deployment Flow: From Code to Production IDP

## Overview

This document explains the complete flow of deploying KubeHatch as a production Internal Developer Portal.

## Current State vs Production State

### Current State (What You Have)
- ✅ Basic authentication (single password for all)
- ✅ Deployed on Kubernetes
- ✅ Exposed via Ingress with Basic Auth
- ✅ Works for single user/team

### Production State (What You Need)
- 🔐 Multi-user authentication (OAuth/OIDC)
- 🌐 Internet exposure with TLS
- 👥 User/team isolation
- 📊 Usage tracking and quotas
- 🔒 Enhanced security

## Complete Deployment Flow

### Phase 1: Basic Deployment (Current)

```
1. Build Images
   ↓
2. Push to Registry
   ↓
3. Deploy to Kubernetes
   ↓
4. Configure Ingress with Basic Auth
   ↓
5. Access via Ingress URL
```

**Result:** Single password, all users see all clusters

### Phase 2: Production Deployment (Recommended)

```
1. Build Images
   ↓
2. Push to Registry
   ↓
3. Deploy OAuth2 Proxy
   ↓
4. Deploy KubeHatch
   ↓
5. Configure Ingress with OAuth
   ↓
6. Set up TLS (HTTPS)
   ↓
7. Configure DNS
   ↓
8. Users access via domain
```

**Result:** Each user authenticates with their own credentials

### Phase 3: Multi-Tenancy (Advanced)

```
1. All of Phase 2
   ↓
2. Add user tracking to backend
   ↓
3. Implement namespace per user
   ↓
4. Add team support
   ↓
5. Implement quotas
```

**Result:** Each user/team has isolated clusters

## Step-by-Step: Production Deployment

### Step 1: Prepare Your Environment

```bash
# 1. Ensure you have a Kubernetes cluster
kubectl cluster-info

# 2. Install Ingress Controller (if not installed)
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.9.4/deploy/static/provider/cloud/deploy.yaml

# 3. Install cert-manager (for TLS)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

### Step 2: Build and Push Images

```bash
# Set your registry
export REGISTRY=your-registry.com
export VERSION=v1.0

# Build backend
docker build -t $REGISTRY/kubehatch-backend:$VERSION \
  -f backend/Dockerfile.backend backend/
docker push $REGISTRY/kubehatch-backend:$VERSION

# Build frontend
docker build -t $REGISTRY/kubehatch-frontend:$VERSION \
  -f frontend/Dockerfile.frontend frontend/
docker push $REGISTRY/kubehatch-frontend:$VERSION
```

### Step 3: Set Up Authentication

#### Option A: OAuth2 Proxy (Recommended)

```bash
# 1. Create OAuth application in your IdP (Google, Okta, etc.)
#    Get: Client ID, Client Secret, OIDC Issuer URL

# 2. Create OAuth secret
kubectl create secret generic oauth2-proxy \
  --from-literal=client-id=YOUR_CLIENT_ID \
  --from-literal=client-secret=YOUR_CLIENT_SECRET \
  --from-literal=cookie-secret=$(openssl rand -base64 32 | head -c 32 | base64)

# 3. Deploy OAuth2 Proxy
helm repo add oauth2-proxy https://oauth2-proxy.github.io/manifests
helm install oauth2-proxy oauth2-proxy/oauth2-proxy \
  --set config.clientID=YOUR_CLIENT_ID \
  --set config.clientSecret=YOUR_CLIENT_SECRET \
  --set config.oidcIssuerURL=https://your-idp.com \
  --set config.provider=oidc \
  --set config.redirectURL=https://vcluster-portal.yourdomain.com/oauth2/callback
```

#### Option B: Basic Auth (Quick Start)

```bash
# Create users
htpasswd -c auth admin
htpasswd auth user1
htpasswd auth user2

# Create secret
kubectl create secret generic vcluster-basic-auth \
  --from-file=auth \
  -n default
```

### Step 4: Prepare Kubeconfig

```bash
# Create secret with kubeconfig that has permissions
kubectl create secret generic vcluster-default-kubeconfig \
  --from-file=kubeconfig=/path/to/kubeconfig \
  -n default
```

### Step 5: Deploy KubeHatch

```bash
# Update image names in manifests
sed -i '' "s|ttl.sh/kubehatch-backend:v27|$REGISTRY/kubehatch-backend:$VERSION|g" k8s/backenddeploy.yaml
sed -i '' "s|ttl.sh/kubehatch-frontend:v7|$REGISTRY/kubehatch-frontend:$VERSION|g" k8s/deploymentfrontend.yaml

# Deploy
kubectl apply -f k8s/backenddeploy.yaml
kubectl apply -f k8s/deploymentfrontend.yaml
kubectl apply -f k8s/backendsvc.yaml
kubectl apply -f k8s/svcfrontend.yaml
kubectl apply -f k8s/role.yaml
```

### Step 6: Configure Ingress

**For OAuth (update k8s/ingress.yaml):**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: vcluster-ingress
  annotations:
    nginx.ingress.kubernetes.io/auth-url: "https://oauth2-proxy.default.svc.cluster.local/oauth2/auth"
    nginx.ingress.kubernetes.io/auth-signin: "https://oauth2-proxy.default.svc.cluster.local/oauth2/start?rd=$escaped_request_uri"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
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

**For Basic Auth (current k8s/ingress.yaml):**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: vcluster-ingress
  annotations:
    nginx.ingress.kubernetes.io/auth-type: "basic"
    nginx.ingress.kubernetes.io/auth-secret: "vcluster-basic-auth"
    nginx.ingress.kubernetes.io/auth-realm: "KubeHatch Portal - Authentication Required"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
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

### Step 7: Set Up TLS

```bash
# Create ClusterIssuer for Let's Encrypt
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

# Apply ingress (cert-manager will automatically create certificate)
kubectl apply -f k8s/ingress.yaml
```

### Step 8: Configure DNS

```bash
# Get ingress external IP
kubectl get ingress vcluster-ingress

# Point your domain's A record to the ingress IP
# vcluster-portal.yourdomain.com -> <ingress-ip>
```

### Step 9: Access

Users can now access:
```
https://vcluster-portal.yourdomain.com
```

They will:
1. Be redirected to authenticate (OAuth or Basic Auth)
2. See the dashboard
3. Create and manage clusters

## User Experience Flow

### For End Users

1. **Access Portal**
   - Navigate to `https://vcluster-portal.yourdomain.com`
   - Authenticate (OAuth redirect or Basic Auth prompt)

2. **Create Cluster**
   - Click "Create Cluster"
   - Enter name, select options
   - Wait for creation (1-2 minutes)

3. **Download Kubeconfig**
   - Click "Download Kubeconfig"
   - Use with kubectl

4. **Use Cluster**
   ```bash
   kubectl --kubeconfig kubeconfig.yaml get nodes
   ```

## Multi-User Support

### Current: Shared Access
- All authenticated users see all clusters
- No isolation
- Works for small teams

### Enhanced: User Isolation (Future)

To add user-specific clusters:

1. **Backend Enhancement:**
   - Extract user from OAuth token
   - Store user with each cluster
   - Filter by user

2. **Namespace Per User:**
   - Create in `vcluster-<user-id>-<cluster-name>`
   - Implement RBAC

3. **Team Support:**
   - Add team concept
   - Share within team
   - Team quotas

## Troubleshooting

### Can't Access Portal
- Check ingress: `kubectl get ingress`
- Check DNS: `dig vcluster-portal.yourdomain.com`
- Check ingress controller: `kubectl get pods -n ingress-nginx`

### Authentication Fails
- Check OAuth proxy: `kubectl logs deploy/oauth2-proxy`
- Verify secrets: `kubectl get secret`
- Check ingress annotations

### Clusters Not Creating
- Check backend logs: `kubectl logs deploy/vcluster-backend`
- Verify kubeconfig secret
- Check RBAC permissions

## Security Checklist

- [ ] HTTPS enabled (TLS)
- [ ] Strong authentication (OAuth, not Basic Auth)
- [ ] RBAC properly configured
- [ ] Network policies in place
- [ ] Resource quotas set
- [ ] Audit logging enabled
- [ ] Regular security updates


