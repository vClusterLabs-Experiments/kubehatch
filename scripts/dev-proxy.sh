#!/bin/bash

# Simple Reverse Proxy for Local Development
# This script creates a simple proxy to avoid CORS issues
# It serves the frontend and proxies /api requests to the backend

set -e

echo "🔀 Starting KubeHatch Development Proxy"
echo "======================================="

# Get the script directory and project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
FRONTEND_DIR="$PROJECT_ROOT/frontend"

PORT=${PORT:-8080}
BACKEND_PORT=${BACKEND_PORT:-8081}

# Check if Go is installed (needed for the proxy)
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is required for the proxy script."
    echo "   Please install Go or use the frontend/backend scripts separately."
    exit 1
fi

# Create a temporary Go proxy server
cat > /tmp/kubehatch-proxy.go << 'EOF'
package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	frontendDir = flag.String("frontend", ".", "Frontend directory")
	port        = flag.String("port", "8080", "Port to listen on")
	backendURL  = flag.String("backend", "http://localhost:8081", "Backend URL")
)

func main() {
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") {
			// Proxy API requests to backend
			proxyURL := *backendURL + r.URL.Path
			log.Printf("Proxying %s -> %s", r.URL.Path, proxyURL)
			
			req, err := http.NewRequest(r.Method, proxyURL, r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			
			// Copy headers
			for key, values := range r.Header {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}
			
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			defer resp.Body.Close()
			
			// Copy response headers
			for key, values := range resp.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
			return
		}
		
		// Serve static files
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		
		filePath := filepath.Join(*frontendDir, path)
		http.ServeFile(w, r, filePath)
	})
	
	log.Printf("🚀 Proxy server starting on http://localhost:%s", *port)
	log.Printf("   Frontend: %s", *frontendDir)
	log.Printf("   Backend:  %s", *backendURL)
	log.Printf("   Press Ctrl+C to stop")
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
EOF

cd "$FRONTEND_DIR"

# Run the proxy
go run /tmp/kubehatch-proxy.go \
	-frontend="$FRONTEND_DIR" \
	-port="$PORT" \
	-backend="http://localhost:$BACKEND_PORT"

# Cleanup
rm -f /tmp/kubehatch-proxy.go



