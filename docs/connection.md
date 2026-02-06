# Windows Snapdragon Device Setup

## Connection Details

```
Host: 10.206.87.35
User: chinmay
Pass: root
Hardware: Snapdragon (ARM64) with NPU + GPU
```

## Using sshpass (recommended)

```bash
# Install sshpass on Mac
brew install sshpass

# Connect
sshpass -p 'root' ssh -o StrictHostKeyChecking=no chinmay@10.206.87.35
```

## Deploying EdgeCLI Server to Windows

**Important:** Use native Windows binaries, not WSL. WSL2 uses NAT networking which makes external access unreliable.

### 1. Build Windows ARM64 binaries on Mac

```bash
cd /path/to/project
GOOS=windows GOARCH=arm64 go build -o dist/server-windows.exe ./cmd/server
GOOS=windows GOARCH=arm64 go build -o dist/client-windows.exe ./cmd/client
```

### 2. Copy to Windows

```bash
sshpass -p 'root' scp -o StrictHostKeyChecking=no \
  dist/server-windows.exe dist/client-windows.exe \
  "chinmay@10.206.87.35:C:/Users/chinmay/Desktop/edgecli/"
```

### 3. Start Windows gRPC Server

```bash
# Via SSH - note the special cmd syntax for environment variables
sshpass -p 'root' ssh chinmay@10.206.87.35 \
  'cmd /c "set GRPC_ADDR=0.0.0.0:50051&& C:\Users\chinmay\Desktop\edgecli\server-windows.exe"'
```

**Note:** The `&&` must immediately follow the env value (no space) or cmd eats the colon.

### 4. Verify Server is Listening

```bash
sshpass -p 'root' ssh chinmay@10.206.87.35 'netstat -an | findstr 50051'
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
# In another terminal - use the batch script
sshpass -p 'root' ssh chinmay@10.206.87.35 \
  'C:\Users\chinmay\Desktop\edgecli\start-server.bat'
```

### Step 3: Register Windows Snapdragon with Mac

```bash
# The client auto-generates a unique device ID for remote addresses
go run ./cmd/client register \
  --name "windows-snapdragon" \
  --self-addr "10.206.87.35:50051" \
  --platform "windows" \
  --arch "arm64" \
  --has-npu \
  --has-gpu
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

## Verified Configuration (2026-02-06)

- SSH connection: Working
- Native Windows ARM64 binary: server-windows.exe
- gRPC server: Listening on 0.0.0.0:50051
- Device name: windows-snapdragon
- Hardware: Snapdragon NPU + GPU
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
| **EdgeCLI dir** | `C:\Users\chinmay\Desktop\edgecli\` |
| gRPC Server | `C:\Users\chinmay\Desktop\edgecli\server-windows-arm64.exe` |
| gRPC Client | `C:\Users\chinmay\Desktop\edgecli\client-windows-arm64.exe` |
| Web Server | `C:\Users\chinmay\Desktop\edgecli\web-windows.exe` |
| Shared dir | `C:\Users\chinmay\Desktop\edgecli\shared\` |
| Device ID | `C:\Users\chinmay\Desktop\edgecli\.edgemesh\` |
| Server batch | `C:\Users\chinmay\Desktop\edgecli\start-server.bat` |
| Web batch | `C:\Users\chinmay\Desktop\edgecli\start-web.bat` |

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
```

### Ollama Setup (Local LLM)

Ollama is installed on this machine for local chat and agent functionality.

**Installation:**
```powershell
winget install Ollama.Ollama
```

**Available Models:**
| Model | Size | Use Case |
|-------|------|----------|
| `llama3.2:3b` | 2.0GB | Chat + tool calling (agent) |
| `phi3:mini` | 2.2GB | Fast chat only (no tool support) |
| `mistral:7b` | 4.1GB | High quality chat + tools |

**Starting Ollama:**
```powershell
# Start Ollama service (runs in background)
ollama serve

# Verify it's running
curl http://localhost:11434
# Returns: "Ollama is running"
```

**Starting EdgeCLI with Ollama:**
```powershell
# Terminal 1 - gRPC Server
cd C:\Users\chinmay\Desktop\edgecli
set GRPC_ADDR=:50051
set CHAT_PROVIDER=ollama
set CHAT_MODEL=llama3.2:3b
server-windows.exe

# Terminal 2 - Web UI
cd C:\Users\chinmay\Desktop\edgecli
set WEB_ADDR=0.0.0.0:8080
set GRPC_ADDR=localhost:50051
set CHAT_PROVIDER=ollama
set CHAT_MODEL=llama3.2:3b
web-windows.exe
```

**Verify Chat & Agent:**
```bash
# From Mac
curl http://10.206.87.35:8080/api/chat/health
# {"ok":true,"provider":"ollama","model":"llama3.2:3b"}

curl http://10.206.87.35:8080/api/agent/health
# {"ok":true,"provider":"openai","model":"llama3.2:3b"}

curl -X POST http://10.206.87.35:8080/api/agent \
  -H "Content-Type: application/json" \
  -d '{"message": "What devices are in the mesh?"}'
```

### Full Setup Guide

See [docs/setup-guide-qai.md](setup-guide-qai.md) for step-by-step instructions.
