# Deployment

Guide for deploying EdgeCLI across multiple machines.

## Local Development

```bash
# Start gRPC server
make server

# Start web UI (separate terminal)
make web

# Open browser
open http://localhost:8080
```

## Multi-Machine Setup

### Architecture

```
┌─────────────────┐         ┌─────────────────┐
│   Mac (Coord)   │◀───────▶│    Windows      │
│   Server:50051  │  gRPC   │   Server:50051  │
│   Web:8080      │         │                 │
└─────────────────┘         └─────────────────┘
        ▲
        │ HTTP
        │
   ┌────┴────┐
   │ Browser │
   └─────────┘
```

### Step 1: Start Coordinator (Mac)

```bash
cd /path/to/project

# Start gRPC server
go run ./cmd/server

# In another terminal, start web UI
go run ./cmd/web
```

### Step 2: Deploy to Windows

Use the deployment script:

```bash
./deploy-windows.sh
```

Or manually:

```bash
# Build Windows binary
GOOS=windows GOARCH=amd64 go build -o dist/server-windows.exe ./cmd/server

# Copy to Windows
sshpass -p 'password' scp dist/server-windows.exe user@windows-ip:C:/Users/user/

# Start on Windows
sshpass -p 'password' ssh user@windows-ip \
  'cmd /c "set GRPC_ADDR=0.0.0.0:50051&& start /B C:\Users\user\server-windows.exe"'
```

### Step 3: Register Windows Device

```bash
go run ./cmd/client register \
  --name "windows-pc" \
  --self-addr "WINDOWS_IP:50051" \
  --platform "windows" \
  --arch "amd64"
```

### Step 4: Verify

```bash
# List devices
curl http://localhost:8080/api/devices

# Test distributed job
curl -X POST http://localhost:8080/api/submit-job \
  -H "Content-Type: application/json" \
  -d '{"text":"collect status","max_workers":0}'
```

## Windows Deployment Script

**File:** `deploy-windows.sh`

```bash
#!/bin/bash
# Configuration
WINDOWS_HOST="10.20.38.80"
WINDOWS_USER="sshuser"
WINDOWS_PASS="root"
WINDOWS_PATH="C:/Users/sshuser.Batman"
GRPC_PORT="50051"

# 1. Build Windows binary
GOOS=windows GOARCH=amd64 go build -o dist/server-windows.exe ./cmd/server

# 2. Stop existing server
sshpass -p "$WINDOWS_PASS" ssh "$WINDOWS_USER@$WINDOWS_HOST" \
  'taskkill /F /IM server-windows.exe 2>nul'

# 3. Copy binary
sshpass -p "$WINDOWS_PASS" scp dist/server-windows.exe \
  "$WINDOWS_USER@$WINDOWS_HOST:$WINDOWS_PATH/"

# 4. Start server
sshpass -p "$WINDOWS_PASS" ssh "$WINDOWS_USER@$WINDOWS_HOST" \
  "cmd /c \"set GRPC_ADDR=0.0.0.0:$GRPC_PORT&& start /B $WINDOWS_PATH/server-windows.exe\""

# 5. Verify
sshpass -p "$WINDOWS_PASS" ssh "$WINDOWS_USER@$WINDOWS_HOST" \
  "netstat -an | findstr \"$GRPC_PORT.*LISTEN\""
```

**Prerequisites:**
- `sshpass` installed (`brew install sshpass`)
- SSH access to Windows machine
- Windows OpenSSH server enabled

## Cross-Platform Builds

```bash
# All platforms
make build-all

# Individual platforms
GOOS=darwin GOARCH=arm64 go build -o dist/server-darwin-arm64 ./cmd/server
GOOS=darwin GOARCH=amd64 go build -o dist/server-darwin-amd64 ./cmd/server
GOOS=linux GOARCH=arm64 go build -o dist/server-linux-arm64 ./cmd/server
GOOS=linux GOARCH=amd64 go build -o dist/server-linux-amd64 ./cmd/server
GOOS=windows GOARCH=amd64 go build -o dist/server-windows.exe ./cmd/server
```

## Environment Variables

### Server

| Variable | Default | Description |
|----------|---------|-------------|
| `GRPC_ADDR` | `:50051` | Listen address |
| `DEVICE_ID` | (auto) | Override device ID |

### Web Server

| Variable | Default | Description |
|----------|---------|-------------|
| `WEB_ADDR` | `:8080` | HTTP listen address |
| `GRPC_ADDR` | `localhost:50051` | gRPC server to connect to |

## Firewall Configuration

### Mac

```bash
# Allow incoming on port 50051
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --add /path/to/server
```

### Windows

```powershell
# Allow incoming on port 50051
netsh advfirewall firewall add rule name="EdgeCLI gRPC" dir=in action=allow protocol=TCP localport=50051
```

### Linux

```bash
# Allow incoming on port 50051
sudo ufw allow 50051/tcp
```

## Troubleshooting

### Connection Refused

1. Check server is running: `lsof -i :50051`
2. Check firewall allows the port
3. Verify IP address is correct

### Device Not Registering

1. Ensure server is reachable from coordinator
2. Check network connectivity: `ping DEVICE_IP`
3. Test gRPC connection: `grpcurl -plaintext DEVICE_IP:50051 list`

### Windows SSH Issues

1. Ensure OpenSSH Server is installed
2. Check SSH service is running: `Get-Service sshd`
3. Verify credentials work: `ssh user@windows-ip`

### Task Execution Fails

1. Check device has `RunTask` RPC (needs updated binary)
2. Verify device is registered: `curl localhost:8080/api/devices`
3. Check server logs for errors
