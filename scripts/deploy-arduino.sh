#!/bin/bash
# Deploy EdgeMesh server to Arduino UNO Q (Dragonwing)
# Prerequisites: Arduino on network, SSH access as arduino@<IP>

set -e
ARDUINO_IP="${ARDUINO_IP:-10.206.56.57}"
ARDUINO_USER="${ARDUINO_USER:-arduino}"
BINARY="edgemesh-server-linux-arm64"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"
mkdir -p dist

echo "Building Linux ARM64 server for Dragonwing..."
GOOS=linux GOARCH=arm64 go build -o "dist/$BINARY" ./cmd/server

echo "Copying to Arduino ($ARDUINO_USER@$ARDUINO_IP)..."
scp "dist/$BINARY" "$ARDUINO_USER@$ARDUINO_IP:/tmp/edgemesh-server"

echo "Starting server on Arduino (port 50051)..."
ssh "$ARDUINO_USER@$ARDUINO_IP" 'chmod +x /tmp/edgemesh-server && GRPC_ADDR=0.0.0.0:50051 nohup /tmp/edgemesh-server > /tmp/edgemesh.log 2>&1 &'

echo "Done. Server should be running. Check with:"
echo "  ssh $ARDUINO_USER@$ARDUINO_IP 'pgrep -a edgemesh'"
echo ""
echo "Register from Mac:"
echo "  go run ./cmd/client register --id edgemesh-arduino --name EdgeMeshArduino --self-addr $ARDUINO_IP:50051 --platform arduino --arch arm64"
