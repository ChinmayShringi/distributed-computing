# EdgeMesh QAI Setup Guide

Step-by-step guide to reproduce the full EdgeMesh + Qualcomm AI Hub setup from scratch.

## Prerequisites

- macOS with Homebrew
- SSH access to a Windows ARM64 machine (Snapdragon X Elite)
- Qualcomm AI Hub account (https://aihub.qualcomm.com/)

## Architecture

```
┌──────────────────────┐                    ┌──────────────────────────────┐
│   Mac (Coordinator)  │                    │  Windows Snapdragon (Worker) │
│                      │     gRPC :50051    │                              │
│  gRPC Server ────────┼───────────────────▶│  gRPC Server                 │
│  Web UI :8080        │                    │  QAI Hub CLI                 │
│  QAI Hub CLI         │                    │  Python 3.12 + venv          │
│  Python 3.13 + venv  │                    │  NPU capability              │
│                      │                    │                              │
└──────────────────────┘                    └──────────────────────────────┘
        ▲                                            ▲
        │  HTTP :8080                                │
        │                                            │
   ┌─────────┐                              ┌─────────────────┐
   │ Browser │                              │ QAI Hub Cloud   │
   └─────────┘                              │ (Qualcomm)      │
                                            └─────────────────┘
```

## Step 1: Install Go (Mac)

```bash
brew install go
go version  # should show go1.25+
```

## Step 2: Build the Project

```bash
cd /path/to/distributed-computing
go build ./...   # downloads deps and compiles everything
```

## Step 3: Set Up QAI Hub on Mac

```bash
# Create Python venv and install qai-hub
python3 -m venv .venv-qaihub
source .venv-qaihub/bin/activate
pip install --upgrade pip
pip install qai-hub

# Configure API token (default hackathon token)
qai-hub configure --api_token b3yhelucambdm13uz9usknrhu98l6pln1dzboooy

# Verify - should list 100+ Qualcomm target devices
qai-hub list-devices
```

## Step 4: Start Backend Services on Mac

**Terminal 1** - gRPC Orchestrator Server:
```bash
mkdir -p shared
go run ./cmd/server
# Output: Orchestrator server listening on :50051
# Output: Self-registered as device: name=<hostname> grpc=127.0.0.1:50051
```

**Terminal 2** - Web UI Server:
```bash
go run ./cmd/web
# Output: Web server listening on :8080
# Output: QAI Hub CLI: available at .venv-qaihub/bin/qai-hub
```

Open http://localhost:8080 in your browser.

## Step 5: Verify Mac Setup

```bash
# List devices (should show your Mac)
curl -s http://localhost:8080/api/devices | python3 -m json.tool

# Run QAI Hub doctor
curl -s http://localhost:8080/api/qaihub/doctor | python3 -m json.tool

# Or via CLI
go run ./cmd/edgecli qaihub doctor

# Submit a test job
curl -s -X POST http://localhost:8080/api/submit-job \
  -H "Content-Type: application/json" \
  -d '{"text":"collect status","max_workers":0}' | python3 -m json.tool
```

## Step 6: Set Up Windows Snapdragon Device

### Prerequisites
- `sshpass` on Mac: `brew install hudochenkov/sshpass/sshpass`
- SSH access to Windows ARM64 machine

### 6a. Build Windows ARM64 Binaries (on Mac)

```bash
GOOS=windows GOARCH=arm64 go build -o dist/server-windows-arm64.exe ./cmd/server
GOOS=windows GOARCH=arm64 go build -o dist/client-windows-arm64.exe ./cmd/client
```

### 6b. Deploy to Windows

```bash
# Replace with your Windows machine credentials
WINDOWS_HOST="10.206.87.35"
WINDOWS_USER="chinmay"
WINDOWS_PASS="root"

# Copy binaries
sshpass -p "$WINDOWS_PASS" scp -o StrictHostKeyChecking=no \
  dist/server-windows-arm64.exe dist/client-windows-arm64.exe \
  "$WINDOWS_USER@$WINDOWS_HOST:C:/Users/$WINDOWS_USER/"
```

### 6c. Install Python on Windows

```bash
# Download ARM64 Python installer
sshpass -p "$WINDOWS_PASS" ssh -o StrictHostKeyChecking=no \
  "$WINDOWS_USER@$WINDOWS_HOST" \
  "powershell -Command \"Invoke-WebRequest -Uri 'https://www.python.org/ftp/python/3.12.9/python-3.12.9-arm64.exe' -OutFile 'C:\Users\\$WINDOWS_USER\python-installer.exe'\""

# Install silently
sshpass -p "$WINDOWS_PASS" ssh -o StrictHostKeyChecking=no \
  "$WINDOWS_USER@$WINDOWS_HOST" \
  "C:\Users\\$WINDOWS_USER\python-installer.exe /quiet InstallAllUsers=0 PrependPath=1 Include_pip=1 TargetDir=C:\Users\\$WINDOWS_USER\Python312"
```

### 6d. Install QAI Hub on Windows

```bash
sshpass -p "$WINDOWS_PASS" ssh -o StrictHostKeyChecking=no \
  "$WINDOWS_USER@$WINDOWS_HOST" \
  "C:\Users\\$WINDOWS_USER\Python312\python.exe -m venv C:\Users\\$WINDOWS_USER\venv-qaihub && C:\Users\\$WINDOWS_USER\venv-qaihub\Scripts\pip.exe install qai-hub"

# Configure API token
sshpass -p "$WINDOWS_PASS" ssh -o StrictHostKeyChecking=no \
  "$WINDOWS_USER@$WINDOWS_HOST" \
  "C:\Users\\$WINDOWS_USER\venv-qaihub\Scripts\qai-hub.exe configure --api_token b3yhelucambdm13uz9usknrhu98l6pln1dzboooy"
```

### 6e. Start Server on Windows

```bash
# Create shared directory
sshpass -p "$WINDOWS_PASS" ssh -o StrictHostKeyChecking=no \
  "$WINDOWS_USER@$WINDOWS_HOST" "mkdir C:\Users\\$WINDOWS_USER\shared 2>nul"

# Create startup batch file
sshpass -p "$WINDOWS_PASS" ssh -o StrictHostKeyChecking=no \
  "$WINDOWS_USER@$WINDOWS_HOST" \
  "powershell -Command \"Set-Content -Path 'C:\Users\\$WINDOWS_USER\start-server.bat' -Value @('set GRPC_ADDR=0.0.0.0:50051','set SHARED_DIR=C:\Users\\$WINDOWS_USER\shared','C:\Users\\$WINDOWS_USER\server-windows-arm64.exe') -Encoding ASCII\""

# Open firewall ports
sshpass -p "$WINDOWS_PASS" ssh -o StrictHostKeyChecking=no \
  "$WINDOWS_USER@$WINDOWS_HOST" \
  "netsh advfirewall firewall add rule name=\"EdgeMesh gRPC\" dir=in action=allow protocol=tcp localport=50051"
sshpass -p "$WINDOWS_PASS" ssh -o StrictHostKeyChecking=no \
  "$WINDOWS_USER@$WINDOWS_HOST" \
  "netsh advfirewall firewall add rule name=\"EdgeMesh HTTP\" dir=in action=allow protocol=tcp localport=8081"

# Start server (runs detached)
sshpass -p "$WINDOWS_PASS" ssh -o StrictHostKeyChecking=no \
  "$WINDOWS_USER@$WINDOWS_HOST" \
  "powershell -Command \"Start-Process -FilePath 'C:\Users\\$WINDOWS_USER\start-server.bat' -WorkingDirectory 'C:\Users\\$WINDOWS_USER' -WindowStyle Hidden\""
```

### 6f. Register Windows Device with Mac Coordinator

```bash
go run ./cmd/client register \
  --name "QCWorkshop31-snapdragon" \
  --self-addr "10.206.87.35:50051" \
  --platform "windows" \
  --arch "arm64" \
  --npu
```

## Step 7: Verify Multi-Device Mesh

```bash
# List all devices (should show Mac + Windows)
curl -s http://localhost:8080/api/devices | python3 -m json.tool

# Execute command on Windows (remote execution)
curl -s -X POST http://localhost:8080/api/routed-cmd \
  -H "Content-Type: application/json" \
  -d '{"cmd":"pwd","args":[],"policy":"PREFER_REMOTE"}' | python3 -m json.tool
# Should return: "stdout": "C:\\Users\\chinmay\r\n"

# Distributed job across both devices
curl -s -X POST http://localhost:8080/api/submit-job \
  -H "Content-Type: application/json" \
  -d '{"text":"collect status","max_workers":0}' | python3 -m json.tool
```

## Current Device Inventory

| Device | Hostname | Platform | Capabilities | Address | Notes |
|--------|----------|----------|-------------|---------|-------|
| Mac (Coordinator) | Rahils-MacBook-Pro.local | darwin/arm64 | cpu | 127.0.0.1:50051 | Runs coordinator + web UI |
| Windows Snapdragon | QCWorkshop31 | windows/arm64 | cpu, **npu** | 10.206.87.35:50051 | Qualcomm Snapdragon X Elite, Win 11 24H2 (Build 26100) |

## Connecting Additional Devices

Any teammate can add their device to the mesh:

1. Build the server binary for their platform
2. Start the server on their machine with `GRPC_ADDR=0.0.0.0:50051`
3. Open firewall port 50051
4. Register with the coordinator:
   ```bash
   go run ./cmd/client --addr <coordinator-ip>:50051 register \
     --name "my-device" \
     --self-addr "<my-ip>:50051" \
     --npu  # if device has NPU
   ```

All registered devices will appear in `GET /api/devices` and participate in distributed jobs.

## Troubleshooting

| Issue | Fix |
|-------|-----|
| `qai-hub: command not found` | Activate venv: `source .venv-qaihub/bin/activate` |
| Windows firewall blocking | Run `netsh advfirewall firewall add rule name="EdgeMesh" dir=in action=allow protocol=tcp localport=50051` |
| `context deadline exceeded` on remote commands | Check firewall, verify server is running: `netstat -an \| findstr 50051` |
| Python "Access is denied" on Windows | The Microsoft Store stub — install real Python via the ARM64 installer |
| Server exits when SSH disconnects | Use `Start-Process` with `-WindowStyle Hidden` or a batch file |
