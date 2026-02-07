# Windows Machine Setup

## Connection Details

```
Host: 10.206.87.35
User: sshuser
Pass: root
```

## Using sshpass (recommended)

```bash
# Install sshpass on Mac
brew install sshpass

# Connect
sshpass -p 'root' ssh -o StrictHostKeyChecking=no sshuser@10.206.87.35
```

## Deploying EdgeCLI Server to Windows

**Important:** Use native Windows binaries, not WSL. WSL2 uses NAT networking which makes external access unreliable.

### 1. Build Windows binaries on Mac

```bash
cd /path/to/project
GOOS=windows GOARCH=amd64 go build -o dist/server-windows.exe ./cmd/server
GOOS=windows GOARCH=amd64 go build -o dist/client-windows.exe ./cmd/client
```

### 2. Copy to Windows

```bash
sshpass -p 'root' scp -o StrictHostKeyChecking=no \
  dist/server-windows.exe dist/client-windows.exe \
  "sshuser@10.206.87.35:C:/Users/sshuser.Batman/"
```

### 3. Start Windows gRPC Server

```bash
# Via SSH - note the special cmd syntax for environment variables
sshpass -p 'root' ssh sshuser@10.206.87.35 \
  'cmd /c "set GRPC_ADDR=0.0.0.0:50051&& C:\Users\sshuser.Batman\server-windows.exe"'
```

**Note:** The `&&` must immediately follow the env value (no space) or cmd eats the colon.

### 4. Verify Server is Listening

```bash
sshpass -p 'root' ssh sshuser@10.206.87.35 'netstat -an | findstr 50051'
# Expected: TCP 0.0.0.0:50051 LISTENING
```

## Multi-Device Demo (Verified Working)

### Step 1: Start Mac gRPC Server (Coordinator)

```bash
cd /path/to/project
go run ./cmd/server
# Output: Server device ID: e452458d-...
```

### Step 2: Start Windows gRPC Server

```bash
# In another terminal
sshpass -p 'root' ssh sshuser@10.206.87.35 \
  'cmd /c "set GRPC_ADDR=0.0.0.0:50051&& C:\Users\sshuser.Batman\server-windows.exe"'
```

### Step 3: Register Windows with Mac

```bash
# The client auto-generates a unique device ID for remote addresses
go run ./cmd/client register \
  --name "windows-batman" \
  --self-addr "10.206.87.35:50051" \
  --platform "windows" \
  --arch "amd64"
# Output: Generated new device ID for remote device: 23d1b497-...
```

### Step 4: Start Web UI

```bash
go run ./cmd/web
# Open http://localhost:8080
```

### Step 5: Test via Web UI

1. Click "Refresh Devices" - should show both Mac and Windows
2. Enter command: `pwd`
3. Select policy: "PREFER_REMOTE"
4. Click "Run" - should execute on Windows (output: `C:\Users\sshuser.Batman`)

### Alternative: Test via curl

```bash
# List devices
curl http://localhost:8080/api/devices

# Run command on remote device
curl -X POST http://localhost:8080/api/routed-cmd \
  -H "Content-Type: application/json" \
  -d '{"cmd":"pwd","args":[],"policy":"PREFER_REMOTE"}'
# Response includes: "stdout":"C:\\Users\\sshuser.Batman\r\n"

# Use assistant
curl -X POST http://localhost:8080/api/assistant \
  -H "Content-Type: application/json" \
  -d '{"text":"list devices"}'
```

## Verified Configuration (2026-02-03)

- SSH connection: Working
- Native Windows binary: server-windows.exe
- gRPC server: Listening on 0.0.0.0:50051
- Device name: Batman (windows-batman when registered)
- Multi-device routing: PREFER_REMOTE correctly routes to Windows
- Web UI: Accessible at http://localhost:8080

---

## Snapdragon ARM64 Machine (QAI Workstream)

Added 2026-02-06 by Rahil for the QAI hackathon workstream.

### Connection Details

```
Host: 10.206.87.35
User: chinmay
Pass: root
```

### Machine Specs

- **Hostname**: QCWorkshop31
- **OS**: Windows 11 Pro, Build 26100 (24H2)
- **Architecture**: ARM64 (Snapdragon X Elite)
- **NPU**: Yes (Qualcomm Hexagon)

### What's Installed

| Component | Path |
|-----------|------|
| Python 3.12 | `C:\Users\chinmay\Python312\` |
| QAI Hub venv | `C:\Users\chinmay\venv-qaihub\` |
| gRPC Server | `C:\Users\chinmay\server-windows-arm64.exe` |
| gRPC Client | `C:\Users\chinmay\client-windows-arm64.exe` |
| Shared dir | `C:\Users\chinmay\shared\` |
| Server batch | `C:\Users\chinmay\start-server.bat` |

### Building for ARM64

**Important**: This machine is ARM64, not AMD64. Build with:

```bash
GOOS=windows GOARCH=arm64 go build -o dist/server-windows-arm64.exe ./cmd/server
GOOS=windows GOARCH=arm64 go build -o dist/client-windows-arm64.exe ./cmd/client
```

### Firewall Rules Added

```
EdgeMesh gRPC  - TCP inbound 50051
EdgeMesh HTTP  - TCP inbound 8081
EdgeMesh Discovery - UDP inbound 50050 (for P2P mode)
```

### P2P Discovery (Automatic Mesh) - Enabled by Default

Automatic peer discovery via UDP broadcast on LAN. No coordinator needed - devices find each other automatically.

**Just start the server:**
```powershell
# Windows
server-windows.exe
```

```bash
# Mac/Linux
go run ./cmd/server
```

**To disable P2P discovery:**
```bash
P2P_DISCOVERY=false go run ./cmd/server
```

**How it works:**
1. Each device broadcasts presence on UDP port 50050 every 5 seconds
2. Other devices receive broadcasts and add to their local registry
3. Devices removed after 30s of no broadcasts (stale timeout)
4. Graceful shutdown sends LEAVE message

**Environment Variables:**
| Variable | Default | Purpose |
|----------|---------|---------|
| `P2P_DISCOVERY` | `true` | UDP broadcast discovery (set `false` to disable) |
| `DISCOVERY_PORT` | `50050` | UDP port for broadcasts |

**Windows Firewall (PowerShell Admin):**
```powershell
New-NetFirewallRule -DisplayName "EdgeCLI Discovery" -Direction Inbound -Protocol UDP -LocalPort 50050 -Action Allow
```

**Verify Discovery:**
```bash
# Start 2 servers on different ports
P2P_DISCOVERY=true GRPC_ADDR=:50051 go run ./cmd/server &
P2P_DISCOVERY=true GRPC_ADDR=:50052 go run ./cmd/server &

# Each should show "[INFO] discovery: found new device..."
```

### Full Setup Guide

See [docs/setup-guide-qai.md](setup-guide-qai.md) for step-by-step instructions.
