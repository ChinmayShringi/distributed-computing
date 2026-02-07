#!/bin/bash
# Deploy EdgeMesh server to Arduino UNO Q (Dragonwing)
# Prerequisites: Arduino on network, SSH access as arduino@<IP>

set -e
ARDUINO_IP="${ARDUINO_IP:-10.206.56.57}"
ARDUINO_USER="${ARDUINO_USER:-arduino}"
ARDUINO_PASS="${ARDUINO_PASS:-edgemesh}"
BINARY="edgemesh-server-linux-arm64"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"
mkdir -p dist

echo "Building Linux ARM64 server for Dragonwing..."
GOOS=linux GOARCH=arm64 go build -o "dist/$BINARY" ./cmd/server

echo "Stopping old server on Arduino (if running)..."
sshpass -p "$ARDUINO_PASS" ssh "$ARDUINO_USER@$ARDUINO_IP" "pkill -f edgemesh-server || true" || true

echo "Copying to Arduino ($ARDUINO_USER@$ARDUINO_IP)..."
sshpass -p "$ARDUINO_PASS" scp "dist/$BINARY" "$ARDUINO_USER@$ARDUINO_IP:/tmp/edgemesh-server"

echo "Copying .env configuration..."
sshpass -p "$ARDUINO_PASS" scp arduino.env "$ARDUINO_USER@$ARDUINO_IP:/tmp/.env"

echo "Starting server on Arduino (port 50051)..."
sshpass -p "$ARDUINO_PASS" ssh "$ARDUINO_USER@$ARDUINO_IP" 'chmod +x /tmp/edgemesh-server && cd /tmp && GRPC_ADDR=0.0.0.0:50051 nohup /tmp/edgemesh-server > /tmp/edgemesh.log 2>&1 &'

echo "Done. Server should be running. Check with:"
echo "  ssh $ARDUINO_USER@$ARDUINO_IP 'pgrep -a edgemesh'"
echo ""
echo "Tail logs:"
echo "  ssh $ARDUINO_USER@$ARDUINO_IP 'tail -f /tmp/edgemesh.log'"
