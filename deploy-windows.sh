#!/bin/bash
# Deploy server to Windows machine
# Usage: ./deploy-windows.sh

set -e

# Configuration
WINDOWS_HOST="10.20.38.80"
WINDOWS_USER="sshuser"
WINDOWS_PASS="root"
WINDOWS_PATH="C:/Users/sshuser.Batman"
GRPC_PORT="50051"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== Windows Deployment Script ===${NC}"

# Step 1: Build Windows binary
echo -e "\n${YELLOW}[1/5] Building Windows binary...${NC}"
GOOS=windows GOARCH=amd64 go build -o dist/server-windows.exe ./cmd/server
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

# Step 4: Start server on Windows
echo -e "\n${YELLOW}[4/5] Starting server on Windows...${NC}"
# Create a wrapper script (avoiding trailing spaces with Windows echo)
sshpass -p "$WINDOWS_PASS" ssh -o StrictHostKeyChecking=no "$WINDOWS_USER@$WINDOWS_HOST" \
  "echo @echo off> $WINDOWS_PATH/start-server.bat"
sshpass -p "$WINDOWS_PASS" ssh -o StrictHostKeyChecking=no "$WINDOWS_USER@$WINDOWS_HOST" \
  "echo set GRPC_ADDR=0.0.0.0:$GRPC_PORT>> $WINDOWS_PATH/start-server.bat"
sshpass -p "$WINDOWS_PASS" ssh -o StrictHostKeyChecking=no "$WINDOWS_USER@$WINDOWS_HOST" \
  "echo $WINDOWS_PATH/server-windows.exe>> $WINDOWS_PATH/start-server.bat"
# Run via task scheduler for proper detachment
sshpass -p "$WINDOWS_PASS" ssh -o StrictHostKeyChecking=no "$WINDOWS_USER@$WINDOWS_HOST" \
  'schtasks /delete /tn "EdgeCLI-Server" /f 2>nul & schtasks /create /tn "EdgeCLI-Server" /tr "C:\Users\sshuser.Batman\start-server.bat" /sc once /st 00:00 /f >nul && schtasks /run /tn "EdgeCLI-Server" >nul'
sleep 3
echo -e "${GREEN}✓ Server started${NC}"

# Step 5: Verify server is running
echo -e "\n${YELLOW}[5/5] Verifying server is listening...${NC}"
LISTENING=$(sshpass -p "$WINDOWS_PASS" ssh -o StrictHostKeyChecking=no "$WINDOWS_USER@$WINDOWS_HOST" \
  "netstat -an | findstr \"$GRPC_PORT.*LISTEN\"" 2>/dev/null || true)

if [[ "$LISTENING" == *"LISTENING"* ]]; then
  echo -e "${GREEN}✓ Server is listening on port $GRPC_PORT${NC}"
  echo -e "\n${GREEN}=== Deployment successful! ===${NC}"
  echo -e "Windows server: $WINDOWS_HOST:$GRPC_PORT"
else
  echo -e "${RED}✗ Server may not be running. Check manually.${NC}"
  exit 1
fi
