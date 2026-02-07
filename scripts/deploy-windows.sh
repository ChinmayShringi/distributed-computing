#!/bin/bash
# Deploy EdgeMesh Server to Windows Device (QCWorkshop31)

set -e

WINDOWS_IP="${WINDOWS_IP:-192.168.1.38}"
WINDOWS_USER="${WINDOWS_USER:-rahil}"
WINDOWS_PASS="${WINDOWS_PASS:-password}"

echo "=== Deploying EdgeMesh to Windows ($WINDOWS_IP) ==="

# Build for Windows ARM64
echo "Building EdgeMesh Server for Windows ARM64..."
GOOS=windows GOARCH=arm64 go build -o edgemesh-server.exe ./cmd/server

# Copy to Windows device via ssh
echo "Copying files to Windows device..."
sshpass -p "$WINDOWS_PASS" scp edgemesh-server.exe "${WINDOWS_USER}@${WINDOWS_IP}:~/"

# Copy .env if it exists
if [ -f ".env" ]; then
    echo "Copying .env configuration..."
    sshpass -p "$WINDOWS_PASS" scp .env "${WINDOWS_USER}@${WINDOWS_IP}:~/"
fi

echo "Killing old server process..."
sshpass -p "$WINDOWS_PASS" ssh "${WINDOWS_USER}@${WINDOWS_IP}" "taskkill /F /IM edgemesh-server.exe 2>nul || echo 'No existing process'"

echo "Starting EdgeMesh server..."
sshpass -p "$WINDOWS_PASS" ssh "${WINDOWS_USER}@${WINDOWS_IP}" "cd ~ && start /B edgemesh-server.exe > edgemesh.log 2>&1"

echo ""
echo "=== Deployment Complete ==="
echo "Server deployed to: ${WINDOWS_USER}@${WINDOWS_IP}"
echo ""
echo "To test /api/assistant on Windows device, run:"
echo "  curl -X POST http://${WINDOWS_IP}:8080/api/assistant -H \"Content-Type: application/json\" -d '{\"text\": \"test\"}'"
echo ""
echo "To view logs:"
echo "  ssh ${WINDOWS_USER}@${WINDOWS_IP} 'type edgemesh.log'"
