# 🧪 Local Development & Testing Guide

This guide will help you run and test KubeHatch locally on your machine.

## 📌 Prerequisites

Before you start, ensure you have the following installed:

- **Go** (1.22 or later) - [Download Go](https://go.dev/dl/)
- **kubectl** - [Install kubectl](https://kubernetes.io/docs/tasks/tools/)
- **vcluster CLI** - [Install vcluster](https://www.vcluster.com/docs/getting-started/setup)
- **A Kubernetes cluster** (local or remote) with kubectl configured
- **Node.js** (optional, for serving frontend) or any static file server

## 🚀 Quick Start

### Option 1: Using the Helper Scripts (Recommended)

We've provided helper scripts to make local development easier:

```bash
# Make scripts executable
chmod +x scripts/dev-backend.sh scripts/dev-frontend.sh

# Terminal 1: Start the backend
./scripts/dev-backend.sh

# Terminal 2: Start the frontend
./scripts/dev-frontend.sh
```

### Option 2: Manual Setup

#### Step 1: Set Up Backend

1. **Navigate to backend directory:**
   ```bash
   cd backend
   ```

2. **Install Go dependencies:**
   ```bash
   go mod download
   ```

3. **Set your kubeconfig path (optional):**
   ```bash
   # If you want to use a specific kubeconfig file
   export KUBECONFIG=/path/to/your/kubeconfig
   
   # Or use the default (~/.kube/config)
   # The backend will use in-cluster config if KUBECONFIG is not set
   ```

4. **Create the requests directory:**
   ```bash
   mkdir -p requests
   ```

5. **Run the backend:**
   ```bash
   go run main.go
   ```

   The backend will start on `http://localhost:8081`

#### Step 2: Set Up Frontend

**Option A: Using Python's built-in server (simplest):**
```bash
cd frontend
python3 -m http.server 8080
```

**Option B: Using Node.js http-server:**
```bash
# Install http-server globally (if not already installed)
npm install -g http-server

# Run the server
cd frontend
http-server -p 8080 --cors
```

**Option C: Using npx (no installation needed):**
```bash
cd frontend
npx http-server -p 8080 --cors
```

**Option D: Using any static file server:**
- Serve the `frontend/` directory on port 8080
- Make sure CORS is enabled if serving from a different origin

The frontend will be available at `http://localhost:8080`

#### Step 3: Configure Frontend API Endpoint

The frontend is configured to use `/api` as the base path by default. 

**For local development with backend on port 8081:**

**Option A: Use URL parameter (easiest for testing)**
- Open the frontend with: `http://localhost:8080?api_base=http://localhost:8081/api`
- This tells the frontend to use the full backend URL

**Option B: Use a reverse proxy (recommended)**
- See Option 4 below for proxy setup
- This avoids CORS issues and provides a single entry point

**Option C: Modify frontend code**
- Edit `frontend/index.html` and change the `API_BASE` constant

### Option 3: Using Docker Compose (Coming Soon)

For a more production-like setup, you can use Docker Compose to run both services.

### Option 4: Using a Reverse Proxy (Recommended for Local Dev)

To avoid CORS issues and have a single entry point, you can use a simple reverse proxy:

**Using nginx (if installed):**
```nginx
# Create nginx.conf
server {
    listen 8080;
    
    location /api {
        proxy_pass http://localhost:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    
    location / {
        root /path/to/kubehatch/frontend;
        try_files $uri $uri/ /index.html;
    }
}
```

**Using a simple Go proxy (included in scripts):**
```bash
./scripts/dev-proxy.sh
```

## 🧪 Testing

### 1. Test Backend API Directly

```bash
# List all vclusters
curl http://localhost:8081/api/vclusters

# Create a vcluster
curl -X POST http://localhost:8081/api/vcluster \
  -F "clusterName=test-cluster" \
  -F "ha=on" \
  -F "loadbalancer=on"

# Get kubeconfig for a cluster
curl http://localhost:8081/api/vcluster/test-cluster/kubeconfig

# Delete a cluster
curl -X DELETE http://localhost:8081/api/vcluster/test-cluster
```

### 2. Test Frontend

1. Open `http://localhost:8080` in your browser
2. Navigate to the Dashboard tab - you should see your existing clusters
3. Navigate to Create Cluster tab and create a test cluster
4. Wait for the cluster to be created (may take 1-2 minutes)
5. Download or view the kubeconfig
6. Test deleting a cluster from the dashboard

### 3. Verify Cluster Creation

After creating a cluster, verify it was created:

```bash
# List vcluster namespaces
kubectl get namespaces | grep vcluster

# Check the StatefulSet
kubectl get statefulset -n vcluster-<cluster-name>

# Check the service
kubectl get svc -n vcluster-<cluster-name>
```

## 🔧 Troubleshooting

### Backend Issues

**Issue: "kubectl: command not found"**
- Solution: Make sure kubectl is installed and in your PATH
- Verify: `which kubectl`

**Issue: "vcluster: command not found"**
- Solution: Install vcluster CLI
- Install: `curl -L -o vcluster https://github.com/loft-sh/vcluster/releases/latest/download/vcluster-linux-amd64 && chmod +x vcluster && sudo mv vcluster /usr/local/bin/`

**Issue: "Permission denied" or RBAC errors**
- Solution: Make sure your kubeconfig has proper permissions
- Test: `kubectl get namespaces` should work

**Issue: Backend can't find kubeconfig**
- Solution: Set KUBECONFIG environment variable or ensure default kubeconfig exists at `~/.kube/config`
- Or: Upload a kubeconfig file through the UI

### Frontend Issues

**Issue: CORS errors**
- Solution: Make sure CORS is enabled in the backend (it should be by default)
- Or: Use a reverse proxy to serve both from the same origin

**Issue: API calls failing**
- Solution: Check browser console for errors
- Verify: Backend is running on port 8081
- Check: Network tab in browser dev tools

**Issue: Frontend not loading**
- Solution: Make sure you're serving the frontend directory, not just opening the HTML file
- Use: `python3 -m http.server` or similar

### Cluster Creation Issues

**Issue: Cluster creation times out**
- Solution: Check backend logs for errors
- Verify: vcluster CLI is working: `vcluster --version`
- Check: Kubernetes cluster is accessible: `kubectl cluster-info`

**Issue: Kubeconfig not generated**
- Solution: Wait longer (cluster creation can take 1-2 minutes)
- Check: Backend logs for secret retrieval errors
- Verify: Secret exists: `kubectl get secret vc-<cluster-name> -n vcluster-<cluster-name>`

## 📝 Development Tips

1. **Backend Logging**: The backend logs all operations. Watch the terminal for detailed logs.

2. **Hot Reload**: For backend development, use tools like `air` or `reflex`:
   ```bash
   # Install air
   go install github.com/cosmtrek/air@latest
   
   # Run with hot reload
   cd backend
   air
   ```

3. **Frontend Development**: Edit `frontend/index.html` directly. Refresh the browser to see changes.

4. **API Testing**: Use tools like Postman or `curl` to test API endpoints directly.

5. **Debugging**: Check browser console (F12) for frontend errors and backend terminal for server errors.

## 🎯 Next Steps

Once you have everything running locally:

1. Test creating clusters with different configurations
2. Test the dashboard features (list, view, delete)
3. Test error scenarios (invalid names, missing kubeconfig, etc.)
4. Verify LoadBalancer endpoints work (if your cluster supports it)
5. Test HA mode with 3 replicas

## 🚀 Production Deployment

When you're ready to deploy to production, see the [Quickstart Guide](docs/QUICKSTART.md) for Kubernetes deployment instructions.

