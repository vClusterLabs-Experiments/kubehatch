# Fixing Kubeconfig for Kind Clusters

## The Problem

When using KubeHatch with a **local kind cluster**, the generated kubeconfig has:
```
server: https://localhost:8443
```

This endpoint requires **port-forwarding** to be active. Without it, you'll get:
```
connection refused - did you specify the right host or port?
```

## Why This Happens

Kind clusters don't have external LoadBalancers. The vcluster service is only accessible:
1. From within the cluster (using service FQDN)
2. Via port-forwarding (localhost:8443)

## Solution: Run Port-Forwarding

### Quick Fix

**In a separate terminal, run:**

```bash
vcluster connect <cluster-name> --namespace vcluster-<cluster-name>
```

**Example:**
```bash
vcluster connect de --namespace vcluster-de
```

**Keep this terminal running**, then use your kubeconfig:
```bash
kubectl --kubeconfig kubeconfig-de.yaml get nodes
```

### Alternative: Manual Port-Forward

If `vcluster connect` doesn't work:

```bash
kubectl port-forward -n vcluster-<cluster-name> svc/<cluster-name> 8443:443
```

## For Production Clusters

In production (with LoadBalancer or external access):
- The kubeconfig will automatically use the external endpoint
- No port-forwarding needed
- Works from anywhere

## Making It Automatic (Future Enhancement)

We could enhance KubeHatch to:
1. Detect if it's a kind cluster
2. Automatically set up port-forwarding
3. Or provide a helper script that does both

For now, users need to run `vcluster connect` manually.


