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
EdgeMesh Discovery - UDP inbound 50051 (for P2P mode)
```

### P2P Discovery (Automatic Mesh) - Enabled by Default

Automatic peer discovery via UDP broadcast on LAN. No coordinator needed - devices find each other automatically.

**Configuration via .env file:**

Both server and web load `.env` from the current directory automatically.

```bash
# .env (copy to C:\Users\chinmay\Desktop\edgecli\ on Windows)
GRPC_ADDR=:50051
WEB_ADDR=:8080
CHAT_PROVIDER=ollama
CHAT_BASE_URL=http://localhost:11434
CHAT_MODEL=llama3.2:3b
CHAT_TIMEOUT_SECONDS=120
AGENT_MAX_ITERATIONS=8
```

**Just start the servers:**
```powershell
# Windows - cd to folder with .env
cd C:\Users\chinmay\Desktop\edgecli
ollama serve              # Terminal 1
.\server-windows.exe      # Terminal 2
.\web-windows.exe         # Terminal 3
```

```bash
# Mac/Linux - from project root (uses .env automatically)
go run ./cmd/server
go run ./cmd/web
```

**To disable P2P discovery:**
```bash
P2P_DISCOVERY=false go run ./cmd/server
```

**How it works:**
1. Each device broadcasts presence on UDP port 50051 every 5 seconds
2. Other devices receive broadcasts and add to their local registry
3. Devices removed after 30s of no broadcasts (stale timeout)
4. Graceful shutdown sends LEAVE message

**Environment Variables:**
| Variable | Default | Purpose |
|----------|---------|---------|
| `P2P_DISCOVERY` | `true` | UDP broadcast discovery (set `false` to disable) |
| `DISCOVERY_PORT` | `50051` | UDP port for broadcasts |
| `SEED_PEERS` | (empty) | Comma-separated IPs for cross-subnet discovery |

**Cross-Subnet Discovery:**

UDP broadcast only works within the same subnet. For devices on different subnets, use `SEED_PEERS`:

```bash
# In .env - all team devices
SEED_PEERS=10.206.187.34,10.206.197.101,10.206.227.186,10.206.8.90,10.206.66.173,10.206.87.35,10.206.56.57
```

Each device sends announcements directly to all seed peers, enabling discovery across subnets.

**Windows Firewall (PowerShell Admin):**
```powershell
New-NetFirewallRule -DisplayName "EdgeCLI Discovery" -Direction Inbound -Protocol UDP -LocalPort 50051 -Action Allow
```

**Verify Discovery:**
```bash
# Start 2 servers on different ports
P2P_DISCOVERY=true GRPC_ADDR=:50051 go run ./cmd/server &
P2P_DISCOVERY=true GRPC_ADDR=:50052 go run ./cmd/server &

# Each should show "[INFO] discovery: found new device..."
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

The `.env` file is pre-configured for Ollama. Just run:

```powershell
cd C:\Users\chinmay\Desktop\edgecli
ollama serve              # Terminal 1
.\server-windows.exe      # Terminal 2 (loads .env automatically)
.\web-windows.exe         # Terminal 3 (loads .env automatically)
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

---

## Arduino UNO Q (Dragonwing) — EdgeMeshArduino

Added 2026-02-06 for multi-device image-generation demo.

### Board Details

| Setting | Value |
|---------|-------|
| **Board name** | EdgeMeshArduino |
| **IP address** | `10.206.56.57` |
| **WiFi password** | edgemesh |
| **Board model** | Arduino UNO Q (Qualcomm Dragonwing QRB2210) |
| **gRPC port** | 50051 (same as all mesh devices) |
| **Status** | WiFi connected, software updated |

### Finding the Arduino's IP Address

The Arduino runs Linux (Dragonwing). With shell access (`arduino@EdgeMeshArduino`), run:

```bash
ip addr
# or: hostname -I
```

Look for the `inet` address on `wlan0`.

### Deploy and Run EdgeMesh Server on Arduino

All devices on the mesh run the gRPC server on port 50051.

**Step 1: Build the Linux ARM64 binary** (on Mac):

```bash
cd /path/to/distributed-computing
GOOS=linux GOARCH=arm64 go build -o dist/edgemesh-server-linux-arm64 ./cmd/server
```

**Step 2: Copy the binary to the Arduino** (requires SSH):

```bash
scp dist/edgemesh-server-linux-arm64 arduino@10.206.56.57:/tmp/edgemesh-server
```

**Step 3: Start the server on the Arduino** (via SSH or Arduino shell):

```bash
ssh arduino@10.206.56.57
chmod +x /tmp/edgemesh-server
GRPC_ADDR=0.0.0.0:50051 /tmp/edgemesh-server
```

Or run in background:

```bash
GRPC_ADDR=0.0.0.0:50051 nohup /tmp/edgemesh-server > /tmp/edgemesh.log 2>&1 &
```

**One-liner script** (from project root):

```bash
./scripts/deploy-arduino.sh
# Uses ARDUINO_IP=10.206.56.57, ARDUINO_USER=arduino by default
# Override: ARDUINO_IP=10.206.56.57 ARDUINO_USER=arduino ./scripts/deploy-arduino.sh
```

**SSH prerequisite:** Ensure you can `ssh arduino@10.206.56.57` (password or key-based). If you use the Arduino IDE / USB shell instead, copy the binary manually (e.g. via USB storage or serial file transfer) and run the server commands in that shell.

### Register Arduino on the Mesh

From your Mac (with coordinator running and Arduino server started):

```bash
go run ./cmd/client register \
  --id "edgemesh-arduino" \
  --name "EdgeMeshArduino" \
  --self-addr "10.206.56.57:50051" \
  --platform "arduino" \
  --arch "arm64"
```

### Arduino HTTP Endpoint (TinyML — Optional)

For IMAGE_GENERATE tasks with a TinyML model, the Arduino can expose an HTTP endpoint. See [docs/IMAGE-GEN-SETUP.md](IMAGE-GEN-SETUP.md) for the sketch pattern.
