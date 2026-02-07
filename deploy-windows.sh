#!/bin/bash
# Deploy server to Windows machine
# Usage: ./deploy-windows.sh

set -e

# Configuration
WINDOWS_HOST="10.206.87.35"
WINDOWS_USER="chinmay"
WINDOWS_PASS="root"
WINDOWS_PATH="C:/Users/chinmay/Desktop/edgecli"
GRPC_PORT="50051"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== Windows Deployment Script ===${NC}"

# Step 1: Build Windows ARM64 binary (for Snapdragon)
echo -e "\n${YELLOW}[1/5] Building Windows ARM64 binary...${NC}"
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -tags netgo -o dist/server-windows.exe ./cmd/server
echo -e "${GREEN}✓ Built dist/server-windows.exe${NC}"

# Step 2: Stop existing server on Windows
echo -e "\n${YELLOW}[2/5] Stopping existing server on Windows...${NC}"
sshpass -p "$WINDOWS_PASS" ssh -o StrictHostKeyChecking=no "$WINDOWS_USER@$WINDOWS_HOST" \
  'taskkill /F /IM server-windows.exe 2>nul || echo No server running' || true
echo -e "${GREEN}✓ Server stopped${NC}"

# Step 3: Copy binary to Windows
echo -e "\n${YELLOW}[3/5] Copying binary to Windows...${NC}"
sshpass -p "$WINDOWS_PASS" scp -o StrictHostKeyChecking=no \
  dist/server-windows.exe \
  "$WINDOWS_USER@$WINDOWS_HOST:$WINDOWS_PATH/"
echo -e "${GREEN}✓ Binary copied to $WINDOWS_PATH/${NC}"

# Step 4: Upload start script
echo -e "\n${YELLOW}[4/4] Uploading start script...${NC}"
cat > /tmp/edgecli-start-server.bat << 'BATCHEOF'
@echo off
set "GRPC_ADDR=0.0.0.0:50051"
set "WEB_ADDR=0.0.0.0:8080"
set "BULK_HTTP_ADDR=0.0.0.0:8081"
set "SHARED_DIR=C:\Users\chinmay\Desktop\edgecli\shared"
C:\Users\chinmay\Desktop\edgecli\server-windows.exe
BATCHEOF
perl -pi -e 's/\n/\r\n/' /tmp/edgecli-start-server.bat
sshpass -p "$WINDOWS_PASS" scp -o StrictHostKeyChecking=no \
  /tmp/edgecli-start-server.bat \
  "$WINDOWS_USER@$WINDOWS_HOST:$WINDOWS_PATH/start-server.bat"
echo -e "${GREEN}✓ Start script uploaded${NC}"

echo -e "\n${GREEN}=== Deployment complete! ===${NC}"
echo -e "Binary deployed to: $WINDOWS_HOST:$WINDOWS_PATH/server-windows.exe"
echo -e "\n${YELLOW}To start the server, run this on the Windows machine:${NC}"
echo -e "  cd $WINDOWS_PATH && start-server.bat"
