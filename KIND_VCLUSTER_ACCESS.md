# Accessing Virtual Clusters from Kind

## The Challenge

When using **kind** (Kubernetes in Docker) as your host cluster, vclusters don't have external IPs. You need **port-forwarding** to access them from your local machine.

## Quick Start

### Step 1: Download Kubeconfig from KubeHatch UI

1. Open http://localhost:8080
2. Go to your cluster in the dashboard
3. Click "Download Kubeconfig"
4. Save the file (e.g., `kubeconfig-de.yaml`)

### Step 2: Set Up Port-Forwarding

**Option A: Using Helper Script (Easiest)**

```bash
./scripts/connect-vcluster.sh <cluster-name>
```

**Example:**
```bash
./scripts/connect-vcluster.sh de
```

This will:
- Set up port-forwarding automatically
- Switch your kubectl context to the vcluster
- Keep running until you press Ctrl+C

**Option B: Using vcluster CLI Directly**

```bash
vcluster connect <cluster-name> --namespace vcluster-<cluster-name>
```

**Example:**
```bash
vcluster connect de --namespace vcluster-de
```

**Option C: Manual Port-Forward**

If the above don't work:

```bash
kubectl port-forward -n vcluster-<cluster-name> svc/<cluster-name> 8443:443
```

**Example:**
```bash
kubectl port-forward -n vcluster-de svc/de 8443:443
```

### Step 3: Use the Kubeconfig

**Keep the port-forwarding terminal running**, then in another terminal:

```bash
# Using the downloaded kubeconfig
kubectl --kubeconfig kubeconfig-de.yaml get nodes

# Or set it as your current context
export KUBECONFIG=kubeconfig-de.yaml
kubectl get nodes
```

## Complete Example

```bash
# Terminal 1: Set up port-forwarding
./scripts/connect-vcluster.sh de

# Terminal 2: Use the vcluster
kubectl --kubeconfig ~/Downloads/kubeconfig-de.yaml get nodes
kubectl --kubeconfig ~/Downloads/kubeconfig-de.yaml get pods -A
```

## Troubleshooting

### Error: "connection refused"

**Problem:** Port-forwarding is not active.

**Solution:**
1. Make sure `vcluster connect` is running in a separate terminal
2. Check if the port is already in use: `lsof -i :8443`
3. Try a different port: `vcluster connect de --namespace vcluster-de --server-port 8444`

### Error: "couldn't get current server API group list"

**Problem:** The vcluster might not be fully ready.

**Solution:**
1. Check vcluster status: `kubectl get pods -n vcluster-<cluster-name>`
2. Wait for pods to be Running: `kubectl wait --for=condition=ready pod -n vcluster-<cluster-name> --all --timeout=5m`
3. Try connecting again

### Error: "context not found"

**Problem:** The kubeconfig might be pointing to the wrong endpoint.

**Solution:**
1. Check the kubeconfig server endpoint: `grep server kubeconfig-de.yaml`
2. If it shows `localhost:8443`, make sure port-forwarding is active
3. If it shows a service FQDN, you might need to run `vcluster connect` to get a proper kubeconfig

### Multiple Clusters

If you need to access multiple vclusters simultaneously:

```bash
# Terminal 1: Connect to cluster 1
vcluster connect de --namespace vcluster-de --server-port 8443

# Terminal 2: Connect to cluster 2  
vcluster connect test-cluster --namespace vcluster-test-cluster --server-port 8444

# Terminal 3: Use kubeconfigs with specific ports
kubectl --kubeconfig kubeconfig-de.yaml get nodes
kubectl --kubeconfig kubeconfig-test-cluster.yaml get nodes
```

## Understanding the Kubeconfig

When you download a kubeconfig from KubeHatch for a kind cluster, it will have:

```yaml
server: https://localhost:8443
```

or

```yaml
server: https://127.0.0.1:<random-port>
```

Both require port-forwarding to be active. The `vcluster connect` command:
1. Sets up the port-forward automatically
2. Updates your kubeconfig with the correct port
3. Switches your kubectl context

## Production vs Development

- **Kind (Development):** Requires port-forwarding
- **Production (with LoadBalancer):** No port-forwarding needed, works from anywhere
- **Production (with Ingress):** Uses Ingress endpoint, no port-forwarding needed

## Pro Tips

1. **Use tmux/screen** to keep port-forwarding sessions running:
   ```bash
   tmux new -s vcluster-de
   ./scripts/connect-vcluster.sh de
   # Detach: Ctrl+B, then D
   # Reattach: tmux attach -t vcluster-de
   ```

2. **Check if port-forwarding is active:**
   ```bash
   lsof -i :8443
   ```

3. **List all your vclusters:**
   ```bash
   vcluster list
   ```

4. **Disconnect from a vcluster:**
   ```bash
   vcluster disconnect
   ```

