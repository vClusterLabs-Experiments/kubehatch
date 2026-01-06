#!/bin/bash

# Local Frontend Development Script
# This script helps you run the KubeHatch frontend locally

set -e

echo "🎨 Starting KubeHatch Frontend for Local Development"
echo "==================================================="

# Get the script directory and project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
FRONTEND_DIR="$PROJECT_ROOT/frontend"

cd "$FRONTEND_DIR"

PORT=${PORT:-8080}

# Check for Python 3
if command -v python3 &> /dev/null; then
    echo "✅ Using Python 3 HTTP server"
    echo ""
    echo "🌐 Frontend will be available at: http://localhost:$PORT"
    echo "   Press Ctrl+C to stop"
    echo ""
    python3 -m http.server "$PORT"
# Check for Node.js http-server
elif command -v http-server &> /dev/null; then
    echo "✅ Using http-server (Node.js)"
    echo ""
    echo "🌐 Frontend will be available at: http://localhost:$PORT"
    echo "   Press Ctrl+C to stop"
    echo ""
    http-server -p "$PORT" --cors
# Check for npx (Node.js)
elif command -v npx &> /dev/null; then
    echo "✅ Using npx http-server"
    echo ""
    echo "🌐 Frontend will be available at: http://localhost:$PORT"
    echo "   Press Ctrl+C to stop"
    echo ""
    npx http-server -p "$PORT" --cors
# Check for PHP
elif command -v php &> /dev/null; then
    echo "✅ Using PHP built-in server"
    echo ""
    echo "🌐 Frontend will be available at: http://localhost:$PORT"
    echo "   Press Ctrl+C to stop"
    echo ""
    php -S "localhost:$PORT"
else
    echo "❌ Error: No suitable HTTP server found."
    echo ""
    echo "Please install one of the following:"
    echo "  - Python 3 (usually pre-installed): python3 -m http.server"
    echo "  - Node.js http-server: npm install -g http-server"
    echo "  - PHP: php -S localhost:$PORT"
    echo ""
    echo "Or manually serve the frontend directory on port $PORT"
    exit 1
fi



