# KubeHatch - Internal Developer Portal for Virtual Clusters

![KubeHatch Logo](docs/images/kubehatch.jpg "KubeHatch Logo")

KubeHatch is a production-ready Internal Developer Portal (IDP) that simplifies creating and managing virtual Kubernetes clusters (vClusters) dynamically using a beautiful, user-friendly web interface. It automates deployment, management, and lifecycle operations for virtual clusters.

## Overview

KubeHatch enables teams to easily provision isolated ephemeral Kubernetes clusters (vClusters) for development, testing, validation, and automation scenarios. With its intuitive web UI, developers can create, manage, and delete virtual clusters without needing deep Kubernetes expertise.

## Architecture

![Architecture Diagram](docs/images/architecture.png "KubeHatch Architecture")

## Features

### Core Functionality
- ✅ **Create Virtual Clusters** - Provision isolated Kubernetes clusters on-demand
- ✅ **Dashboard View** - Beautiful dashboard showing all your virtual clusters with real-time status
- ✅ **Cluster Management** - View, download kubeconfig, and delete clusters with ease
- ✅ **High Availability** - Optional HA setup with 3 replicas for production workloads
- ✅ **LoadBalancer Support** - Automated exposure of vClusters via LoadBalancer services
- ✅ **Kubeconfig Management** - Easy download and viewing of cluster configurations

### User Experience
- 🎨 **Modern UI** - Beautiful, responsive design with dark theme
- 📊 **Statistics Dashboard** - Overview cards showing cluster counts, status, and HA metrics
- 🔄 **Real-time Updates** - Auto-refreshing dashboard with cluster status
- ⚡ **Fast Operations** - Streamlined workflows for common tasks
- 📱 **Mobile Friendly** - Responsive design that works on all devices

### Developer Features
- 🔐 **Flexible Authentication** - Support for custom kubeconfig or default cluster config
- 🚀 **Quick Provisioning** - Create clusters in minutes
- 🗑️ **Easy Cleanup** - One-click cluster deletion with namespace cleanup
- 📥 **Kubeconfig Export** - Download or copy kubeconfig files directly from the UI

## Local Development & Testing

Want to test locally before deploying? Check out our [Local Development Guide](LOCAL_DEVELOPMENT.md) for step-by-step instructions.

**Quick local start:**
```bash
# Terminal 1: Start backend
./scripts/dev-backend.sh

# Terminal 2: Start frontend (or use proxy)
./scripts/dev-frontend.sh
# OR use proxy (recommended):
./scripts/dev-proxy.sh
```

## Try It Out Now

Go to the [Quickstart Guide](https://loftlabs-experiments.github.io/kubehatch/QUICKSTART/) or follow the steps below to get started quickly.

### Quick Start

1. Clone the repo:
   ```bash
   git clone https://github.com/LoftLabs-Experiments/kubehatch.git
   cd kubehatch
   ```

2. Deploy with Kubernetes:
   ```bash
   kubectl apply -f k8s/
   ```

3. Visit the UI at your ingress URL (see Quickstart for details).

### Using the Portal

#### Dashboard
- View all your virtual clusters in a beautiful grid layout
- See real-time status (Running, Pending, Error)
- Monitor cluster statistics at a glance
- Access quick actions for each cluster

#### Creating a Cluster
1. Navigate to the "Create Cluster" tab
2. Enter a cluster name (lowercase alphanumeric with hyphens)
3. Optionally upload a host kubeconfig file
4. Choose options:
   - Enable High Availability (3 replicas)
   - Expose via LoadBalancer (for external access)
5. Click "Create Cluster" and wait for provisioning
6. Download or copy your kubeconfig when ready

#### Managing Clusters
- **View Details**: See cluster namespace, HA status, LoadBalancer endpoint, and creation time
- **Download Kubeconfig**: Get the kubeconfig file for kubectl access
- **View Kubeconfig**: Preview the configuration in the UI
- **Delete Cluster**: Remove clusters and their namespaces with one click

## API Endpoints

The backend provides a RESTful API:

- `POST /api/vcluster` - Create a new virtual cluster
- `GET /api/vclusters` - List all virtual clusters
- `GET /api/vcluster/{name}/kubeconfig` - Get kubeconfig for a cluster
- `DELETE /api/vcluster/{name}` - Delete a virtual cluster

## Documentation

Full documentation is available at [KubeHatch Docs](https://loftlabs-experiments.github.io/kubehatch/).

## Contributing

Contributions welcome! See [Build from Source](https://loftlabs-experiments.github.io/kubehatch/BUILD/) to get started.

## License

See [LICENSE](LICENSE) file for details.
