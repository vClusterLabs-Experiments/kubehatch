# Using Kubeconfig with Kind Clusters

## Important Note for Local Kind Clusters

When using KubeHatch with a local kind cluster, the generated kubeconfig uses `localhost:8443` as the server endpoint. This requires **port-forwarding** to be active.

## How to Use the Kubeconfig

### Option 1: Use vcluster connect (Recommended)

Run this command in a separate terminal to set up port-forwarding:

```bash
vcluster connect <cluster-name> --namespace vcluster-<cluster-name>
```

**Example:**
```bash
vcluster connect de --namespace vcluster-de
```

Keep this terminal running. Then use the kubeconfig:

```bash
kubectl --kubeconfig kubeconfig-de.yaml get nodes
```

### Option 2: Manual Port-Forwarding

If you prefer manual port-forwarding:

```bash
kubectl port-forward -n vcluster-<cluster-name> svc/<cluster-name> 8443:443
```

Then use the kubeconfig (it already points to `localhost:8443`).

### Option 3: Use Service FQDN (From Within Cluster)

If you're accessing from within a pod in the cluster, the kubeconfig will be updated to use the service FQDN:
```
https://<cluster-name>.vcluster-<cluster-name>.svc.cluster.local:443
```

## For Production Clusters

If you're using a LoadBalancer:
- The kubeconfig will automatically use the LoadBalancer endpoint
- No port-forwarding needed
- Works from anywhere

## Troubleshooting

**Error: "connection refused"**
- Make sure `vcluster connect` is running in the background
- Or set up manual port-forwarding

**Error: "couldn't get current server API group list"**
- Verify the vcluster is running: `kubectl get pods -n vcluster-<cluster-name>`
- Check port-forwarding is active
- Try restarting `vcluster connect`



