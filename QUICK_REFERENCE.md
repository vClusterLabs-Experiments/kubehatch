# KubeHatch Quick Reference

## 🚀 How It Works: Complete Flow

### Current Setup (What You Have)
```
Kubernetes Cluster
    ↓
[KubeHatch Backend + Frontend] (Deployed as pods)
    ↓
[Ingress Controller] (nginx-ingress)
    ↓
[Basic Auth] (Single password for all)
    ↓
Internet Users → https://vcluster-portal.yourdomain.com
```

### Production Setup (What You Need)
```
Kubernetes Cluster
    ↓
[KubeHatch Backend + Frontend]
    ↓
[OAuth2 Proxy] (For multi-user auth)
    ↓
[Ingress Controller] (with TLS)
    ↓
Internet Users → https://vcluster-portal.yourdomain.com
    ↓
Each user authenticates with their own credentials
```

## 📋 Deployment Checklist

### For Production IDP:

1. **Build & Push Images**
   ```bash
   docker build -t registry/kubehatch-backend:v1.0 -f backend/Dockerfile.backend backend/
   docker build -t registry/kubehatch-frontend:v1.0 -f frontend/Dockerfile.frontend frontend/
   docker push registry/kubehatch-backend:v1.0
   docker push registry/kubehatch-frontend:v1.0
   ```

2. **Set Up Authentication**
   - **Option A:** OAuth2 Proxy (Recommended)
     - Deploy OAuth2 Proxy
     - Configure with your IdP (Google, Okta, etc.)
   - **Option B:** Basic Auth (Quick start)
     - Create users with htpasswd
     - Create Kubernetes secret

3. **Deploy KubeHatch**
   ```bash
   kubectl apply -f k8s/
   ```

4. **Configure Ingress**
   - Update `k8s/ingress.yaml` with your domain
   - Add TLS annotations
   - Apply ingress

5. **Set Up DNS**
   - Point domain to ingress IP
   - Wait for TLS certificate

6. **Access**
   - Users go to `https://vcluster-portal.yourdomain.com`
   - Authenticate
   - Create clusters

## 🔐 Authentication Options

### Current: Basic Auth
- ✅ Simple, quick setup
- ❌ Single password for all
- ❌ No user isolation
- ❌ Manual password management

### Recommended: OAuth/OIDC
- ✅ Each user has own credentials
- ✅ Integrates with existing IdP
- ✅ Better security
- ✅ Can add user tracking

## 👥 Multi-User Support

### Current Implementation
- All users see all clusters
- No isolation
- Shared namespace

### To Add User Isolation
1. Extract user from OAuth token
2. Store user with each cluster
3. Filter clusters by user
4. Create namespaces per user

## 🌐 Internet Exposure

### Steps:
1. Deploy Ingress Controller
2. Configure Ingress with domain
3. Set up TLS (cert-manager or manual)
4. Point DNS to ingress IP
5. Access via HTTPS

### Example:
```yaml
# Ingress exposes the portal
host: vcluster-portal.yourdomain.com
  ↓
TLS certificate (Let's Encrypt)
  ↓
OAuth authentication
  ↓
Users access portal
```

## 🔧 Kind Cluster Kubeconfig Issue

### Problem:
```
server: https://localhost:8443
Error: connection refused
```

### Solution:
Run port-forwarding in separate terminal:
```bash
vcluster connect <cluster-name> --namespace vcluster-<cluster-name>
```

Keep it running, then use kubeconfig.

### Why:
Kind clusters don't have external LoadBalancers. Port-forwarding is required for local access.

## 📖 Full Documentation

- **PRODUCTION_DEPLOYMENT.md** - Complete production guide
- **DEPLOYMENT_FLOW.md** - Step-by-step flow
- **IDP_ARCHITECTURE.md** - Architecture details
- **KUBECONFIG_KIND_FIX.md** - Kind cluster fix

## 🎯 Quick Answers

**Q: How do people access it?**
A: Deploy on Kubernetes, expose via Ingress, users access via domain URL.

**Q: How to enable login?**
A: Use OAuth2 Proxy with your IdP (Google, Okta, etc.) or Basic Auth for quick start.

**Q: Different users/teams?**
A: Current: All users share. Enhanced: Add user tracking to backend for isolation.

**Q: Internet exposure?**
A: Ingress Controller + TLS + DNS = Internet accessible.

**Q: Kind cluster kubeconfig?**
A: Run `vcluster connect` for port-forwarding.


