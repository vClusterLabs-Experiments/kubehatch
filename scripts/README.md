# Development Scripts

Helper scripts for local development and testing.

## Available Scripts

### `dev-backend.sh`
Starts the backend server locally on port 8081.

**Usage:**
```bash
./scripts/dev-backend.sh
```

**What it does:**
- Checks for required tools (Go, kubectl, vcluster CLI)
- Sets up environment variables
- Installs Go dependencies
- Runs the backend server

**Requirements:**
- Go 1.22+
- kubectl (optional, but recommended)
- vcluster CLI (optional, but required for cluster creation)

### `dev-frontend.sh`
Starts a local HTTP server for the frontend on port 8080.

**Usage:**
```bash
./scripts/dev-frontend.sh
```

**What it does:**
- Automatically detects and uses available HTTP server (Python, Node.js, PHP)
- Serves the frontend directory
- Enables CORS if using http-server

**Requirements:**
- Python 3, Node.js (http-server), or PHP (any one of these)

### `dev-proxy.sh`
Starts a reverse proxy that serves the frontend and proxies API requests to the backend.

**Usage:**
```bash
# Terminal 1: Start backend
./scripts/dev-backend.sh

# Terminal 2: Start proxy (which serves frontend)
./scripts/dev-proxy.sh
```

**What it does:**
- Creates a single entry point on port 8080
- Serves frontend files
- Proxies `/api/*` requests to backend on port 8081
- Avoids CORS issues

**Requirements:**
- Go (for the proxy server)
- Backend must be running on port 8081

## Quick Start

**Option 1: Separate Frontend and Backend (Simple)**
```bash
# Terminal 1
./scripts/dev-backend.sh

# Terminal 2
./scripts/dev-frontend.sh
# Then open http://localhost:8080
# Note: You may need to configure the frontend to use http://localhost:8081/api
```

**Option 2: Using Proxy (Recommended)**
```bash
# Terminal 1
./scripts/dev-backend.sh

# Terminal 2
./scripts/dev-proxy.sh
# Then open http://localhost:8080
# Everything works from one port!
```

## Environment Variables

- `KUBECONFIG` - Path to your kubeconfig file (default: `~/.kube/config`)
- `PORT` - Port for frontend/proxy (default: 8080)
- `BACKEND_PORT` - Port for backend (default: 8081, set in backend code)

## Troubleshooting

See [LOCAL_DEVELOPMENT.md](../LOCAL_DEVELOPMENT.md) for detailed troubleshooting guide.



