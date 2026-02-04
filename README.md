# EdgeCLI

Edge-first CLI framework for building tools with safe mode, dangerous mode, and approval workflows for remote tool execution.

## Features

- **Tool Registry Framework** - Register and execute tools with schema validation
- **Safe/Dangerous Mode** - Two execution modes with different security controls
- **Approval Workflows** - Interactive approval for dangerous operations
- **Cross-Platform** - macOS, Linux, Windows support
- **Platform Detection** - Automatic OS, architecture, and distro detection

## Installation

### Build from Source

```bash
git clone https://github.com/edgecli/edgecli.git
cd edgecli
make build
./edgecli version
```

## Quick Start

```bash
# Show version and platform info
./edgecli version

# List registered tools
./edgecli tools

# Debug flag values
./edgecli debug flags

# Enable dangerous mode (bypasses safety controls)
./edgecli --allow-dangerous
```

## Commands

```bash
edgecli version              # Show version info
edgecli tools                # List registered tools
edgecli debug flags          # Show resolved flag values
edgecli --help               # Show help
```

## Global Flags

```bash
--verbose, -v           # Enable verbose output
--config                # Config file path (default: ~/.edgecli/config.json)
--allow-dangerous       # Enable dangerous mode
--ad                    # Alias for --allow-dangerous
--yes, -y               # Auto-confirm dangerous mode consent
```

## Tool Registration

Implement the `tools.Tool` interface to register custom tools:

```go
package main

import (
    "context"
    "encoding/json"
    "github.com/edgecli/edgecli/internal/tools"
)

type MyTool struct{}

func (t *MyTool) Name() string { return "my_tool" }
func (t *MyTool) Description() string { return "My custom tool" }
func (t *MyTool) ArgsSchema() json.RawMessage { return json.RawMessage(`{}`) }
func (t *MyTool) IsDangerous() bool { return false }
func (t *MyTool) Run(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
    return &tools.ToolResult{OK: true, Message: "Tool executed"}, nil
}

func main() {
    registry := tools.DefaultRegistry()
    registry.Register(&MyTool{})
    // Use registry.Execute() to run tools
}
```

## Dangerous Mode

The `--allow-dangerous` flag enables unrestricted command execution.

```bash
edgecli --allow-dangerous    # or --ad
edgecli --ad --yes           # Auto-confirm consent
```

**WARNING:** This mode bypasses safety controls. When invoked, you must type the exact confirmation phrase: `I UNDERSTAND AND ACCEPT THE RISK`

## Configuration

Configuration is stored in `~/.edgecli/config.json`

Default directories:
- Config: `~/.edgecli/`
- Logs: `~/.edgecli/logs/`
- Cache: `~/.edgecli/cache/`

## gRPC Control Plane

EdgeCLI includes a gRPC control plane for remote command execution.

### Prerequisites

Install protoc plugins:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Generate Proto Code

```bash
make proto
```

### Run Server

```bash
# Default port :50051
go run ./cmd/server

# Custom port
GRPC_ADDR=:9000 go run ./cmd/server
```

Server output:
```
[INFO] Orchestrator server listening on :50051
[INFO] Allowed commands: [pwd ls cat]
```

### Run Client

```bash
# Execute pwd
go run ./cmd/client --addr localhost:50051 --key dev --cmd pwd

# Execute ls
go run ./cmd/client --addr localhost:50051 --key dev --cmd ls

# Execute cat (only files under ./shared/ are allowed)
mkdir -p shared && echo "test content" > shared/test.txt
go run ./cmd/client --addr localhost:50051 --key dev --cmd cat --arg ./shared/test.txt
```

### Smoke Test

Terminal 1 (Server):
```bash
go run ./cmd/server
```

Terminal 2 (Client):
```bash
# Test pwd
go run ./cmd/client --key dev --cmd pwd
# Expected: prints current working directory

# Test ls
go run ./cmd/client --key dev --cmd ls
# Expected: directory listing with -la format

# Test cat with allowed path
mkdir -p shared && echo "hello world" > shared/test.txt
go run ./cmd/client --key dev --cmd cat --arg ./shared/test.txt
# Expected: hello world

# Test cat with disallowed path (should fail)
go run ./cmd/client --key dev --cmd cat --arg /etc/passwd
# Expected: error - absolute paths not allowed

# Test disallowed command (should fail)
go run ./cmd/client --key dev --cmd rm --arg foo
# Expected: error - command not in allowlist
```

### Allowlist

Only the following commands are permitted for remote execution:

| Command | Restrictions |
|---------|-------------|
| `pwd` | None |
| `ls` | None (runs with `-la` flag) |
| `cat` | Only files under `./shared/`, no path traversal (`..`), no absolute paths |

## Multi-Device Orchestration

EdgeCLI supports multi-device orchestration, allowing any device to act as an orchestrator.

### Device Registration

Register a device with the orchestration server:

```bash
go run ./cmd/client register --name my-laptop --self-addr 192.168.1.10:50052
```

Options:
- `--name` (required): Device name for identification
- `--self-addr` (required): This device's gRPC address
- `--http-addr`: Bulk HTTP server address for file downloads (e.g., `10.0.0.5:8081`)
- `--gpu`: Device has GPU capability
- `--npu`: Device has NPU capability

The device ID is automatically generated and persisted to `~/.edgemesh/device_id`.

### List Devices

```bash
go run ./cmd/client list-devices
```

Output:
```
DEVICE ID   NAME        PLATFORM  ARCH   CAPABILITIES  ADDRESS
---------   ----        --------  ----   ------------  -------
a1b2c3d4... my-laptop   darwin    arm64  cpu           192.168.1.10:50052
```

### Get Device Status

```bash
go run ./cmd/client status --id <device-id>
```

Output:
```
Device Status:
  Device ID: a1b2c3d4-...
  Last Seen: 2025-01-28T10:30:00Z
  CPU Load: unavailable
  Memory: 45 MB used / 128 MB total
```

### AI Task Routing (Stub)

Route an AI task to the best available device:

```bash
go run ./cmd/client --key dev route-task --task summarize --input "hello world"
```

Output:
```
Task Routing Decision:
  Selected Device ID: a1b2c3d4-...
  Selected Device Address: 192.168.1.10:50052
  Would Use NPU: false
  Result: ROUTED: summarize to my-laptop
```

Routing priority:
1. Devices with NPU (highest)
2. Devices with GPU
3. Any device with CPU (fallback)
4. Server itself (if no devices registered)

### End-to-End Demo

**Terminal 1** - Start server on laptop A:
```bash
go run ./cmd/server
# Output:
# [INFO] Orchestrator server listening on :50051
# [INFO] Server device ID: abc123...
# [INFO] Allowed commands: [pwd ls cat]
```

**Terminal 2** - Register laptop B:
```bash
# Register device
go run ./cmd/client register --name laptop-b --self-addr 192.168.1.20:50052
# Output:
# Device registered successfully!
#   Device ID: def456...
#   Name: laptop-b
#   Platform: darwin/arm64
#   Address: 192.168.1.20:50052

# List all devices
go run ./cmd/client list-devices

# Route a task
go run ./cmd/client --key dev route-task --task summarize --input "hello"
# Output:
# Task Routing Decision:
#   Selected Device ID: def456...
#   Selected Device Address: 192.168.1.20:50052
#   Would Use NPU: false
#   Result: ROUTED: summarize to laptop-b

# Legacy command execution still works
go run ./cmd/client --key dev --cmd pwd
```

## Routed Command Execution

Execute commands on the best available device with automatic routing.

### Basic Usage

```bash
# Execute command with automatic device selection
go run ./cmd/client --key dev routed-cmd --cmd ls

# Execute with arguments
go run ./cmd/client --key dev routed-cmd --cmd cat --arg ./shared/test.txt
```

### Routing Policies

```bash
# BEST_AVAILABLE (default) - prefer NPU > GPU > CPU
go run ./cmd/client --key dev routed-cmd --cmd pwd

# PREFER_REMOTE - prefer non-local device if available
go run ./cmd/client --key dev routed-cmd --cmd ls --prefer-remote

# REQUIRE_NPU - fail if no NPU device registered
go run ./cmd/client --key dev routed-cmd --cmd pwd --require-npu

# FORCE_DEVICE_ID - run on specific device
go run ./cmd/client --key dev routed-cmd --cmd ls --force-device <device-id>
```

### Output Format

```
Routed Execution:
  Selected Device: my-laptop (abc123...)
  Device Address: 127.0.0.1:50051
  Executed Locally: true
  Total Time: 12.34 ms
  Exit Code: 0
---
<command output here>
```

### Demo A: Single Machine (Local Execution)

```bash
# Terminal 1: Start server
go run ./cmd/server
# Output:
# [INFO] Self-registered as device: id=abc123... name=my-laptop addr=127.0.0.1:50051
# [INFO] Orchestrator server listening on :50051

# Terminal 2: Execute routed command
go run ./cmd/client --key dev routed-cmd --cmd pwd
# Output:
# Routed Execution:
#   Selected Device: my-laptop (abc123...)
#   Device Address: 127.0.0.1:50051
#   Executed Locally: true
#   Total Time: 5.23 ms
#   Exit Code: 0
# ---
# /path/to/working/directory
```

### Demo B: Two Machines (Remote Execution)

**Laptop A (Coordinator):**
```bash
# Start server listening on all interfaces
GRPC_ADDR=0.0.0.0:50051 go run ./cmd/server
```

**Laptop B (Worker):**
```bash
# Start server
GRPC_ADDR=0.0.0.0:50051 go run ./cmd/server

# Register with coordinator on Laptop A
go run ./cmd/client --addr 192.168.1.10:50051 register --name laptop-b --self-addr 192.168.1.20:50051
```

**Any Client:**
```bash
# List devices (shows both laptops)
go run ./cmd/client --addr 192.168.1.10:50051 list-devices

# Execute with prefer-remote (runs on Laptop B)
go run ./cmd/client --addr 192.168.1.10:50051 --key dev routed-cmd --cmd pwd --prefer-remote
# Output:
# Routed Execution:
#   Selected Device: laptop-b (def456...)
#   Device Address: 192.168.1.20:50051
#   Executed Locally: false
#   Total Time: 45.67 ms
#   Exit Code: 0
# ---
# /path/on/laptop-b
```

## Web UI Demo

EdgeCLI includes a minimal web UI for demo purposes, accessible from any browser including mobile devices.

### Prerequisites

- gRPC server running on localhost:50051

### Start Web Server

```bash
# Terminal 1: Start gRPC server
make server
# or: go run ./cmd/server

# Terminal 2: Start web server
make web
# or: go run ./cmd/web
```

### Access

Open http://localhost:8080 in your browser (or use LAN IP on phone: http://<your-ip>:8080)

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WEB_ADDR` | `:8080` | HTTP server address |
| `GRPC_ADDR` | `localhost:50051` | gRPC server to connect to |
| `DEV_KEY` | `dev` | Security key for gRPC sessions |

### REST API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Serve web UI (index.html) |
| `/api/devices` | GET | List all registered devices |
| `/api/routed-cmd` | POST | Execute command on best device |
| `/api/submit-job` | POST | Submit distributed job |
| `/api/job?id=` | GET | Get job status |
| `/api/plan` | POST | Preview execution plan without creating a job |
| `/api/request-download` | POST | Request file download ticket from a device |
| `/api/assistant` | POST | Natural language command interface |
| `/api/stream/start` | POST | Start WebRTC screen stream |
| `/api/stream/answer` | POST | Complete WebRTC handshake |
| `/api/stream/stop` | POST | Stop active stream |

#### POST /api/routed-cmd

Request:
```json
{
  "cmd": "ls",
  "args": ["-la"],
  "policy": "BEST_AVAILABLE",
  "force_device_id": ""
}
```

Response:
```json
{
  "selected_device_name": "my-laptop",
  "selected_device_id": "abc123...",
  "selected_device_addr": "127.0.0.1:50051",
  "executed_locally": true,
  "total_time_ms": 12.5,
  "exit_code": 0,
  "stdout": "...",
  "stderr": ""
}
```

#### POST /api/assistant

Request:
```json
{ "text": "list devices" }
```

Response:
```json
{
  "reply": "Found 2 devices:\n1. my-laptop (darwin/arm64) ...",
  "raw": [...]
}
```

### Multi-Device Web Demo

1. **Start coordinator on Laptop A:**
   ```bash
   GRPC_ADDR=0.0.0.0:50051 go run ./cmd/server
   ```

2. **Start worker on Laptop B:**
   ```bash
   GRPC_ADDR=0.0.0.0:50051 go run ./cmd/server
   go run ./cmd/client --addr <A-IP>:50051 register --name laptop-b --self-addr <B-IP>:50051
   ```

3. **Start web server (on any machine):**
   ```bash
   GRPC_ADDR=<A-IP>:50051 go run ./cmd/web
   ```

4. **Open on phone:**
   - Navigate to `http://<web-server-ip>:8080`
   - Click "Refresh" to see both devices
   - Run command with "Prefer Remote" policy
   - Watch it execute on Laptop B

## AI-Powered Plan Generation

EdgeCLI includes an optional AI-powered plan generation system for distributed jobs.

### How It Works

1. On Windows, a C#/.NET CLI tool (`brain/windows-ai-cli/`) wraps Windows AI APIs
2. The Go brain package (`internal/brain/`) calls the CLI via JSON stdin/stdout
3. When submitting a job, the server tries the brain first, then falls back to deterministic planning
4. Plan preview is available via `POST /api/plan` or the "Preview Plan" button in the web UI

### Configuration

```bash
# Windows PowerShell
$env:WINDOWS_AI_CLI_PATH = "C:\path\to\WindowsAiCli.exe"
$env:USE_WINDOWS_AI_PLANNER = "true"
```

### Plan Preview

Preview the execution plan before submitting a job:

```bash
curl -X POST http://localhost:8080/api/plan \
  -H "Content-Type: application/json" \
  -d '{"text":"collect status","max_workers":0}'
```

Returns the plan, whether AI was used, and the rationale for task assignments.

### Design

- **Fallback-first**: Deterministic plan generation always works, AI enhances when available
- **Platform-specific**: Uses Go build tags (`//go:build windows` / `//go:build !windows`)
- **Metadata**: Every plan includes `used_ai`, `notes`, and `rationale` for transparency

See `brain/windows-ai-cli/README.md` for the CLI tool documentation.

## Remote Streaming (v1)

Stream any device's screen to the web UI using WebRTC DataChannel (JPEG frames).

### Setup

1. **Start gRPC server on each machine:**
   ```bash
   # On machine A (coordinator)
   make server

   # On machine B (Windows/Mac/Linux)
   go run ./cmd/server
   ```

2. **Register remote device:**
   ```bash
   go run ./cmd/client register --name "windows-pc" --self-addr "192.168.1.100:50051" --platform "windows" --arch "amd64"
   ```

3. **Start web UI:**
   ```bash
   make web
   ```

4. **Open http://localhost:8080** on any device (phone, tablet, etc.)

5. **In "Remote Stream" section:**
   - Select "Prefer Remote" policy
   - Click "Start Stream"
   - Observe remote screen updating

### Stream Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| FPS | 8 | Target frames per second |
| Quality | 60 | JPEG quality (10-100) |
| Monitor | 0 | Display index for multi-monitor |

### REST API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/stream/start` | POST | Start WebRTC stream from selected device |
| `/api/stream/answer` | POST | Complete WebRTC handshake with answer SDP |
| `/api/stream/stop` | POST | Stop active stream |

#### POST /api/stream/start

Request:
```json
{
  "policy": "PREFER_REMOTE",
  "force_device_id": "",
  "fps": 8,
  "quality": 60,
  "monitor_index": 0
}
```

Response:
```json
{
  "selected_device_id": "abc123...",
  "selected_device_name": "windows-pc",
  "selected_device_addr": "192.168.1.100:50051",
  "stream_id": "def456...",
  "offer_sdp": "v=0\r\n..."
}
```

### Screen Capture Capability

Devices report `can_screen_capture` at registration time. The server tests screen capture at startup using `kbinani/screenshot`. The web UI uses this flag to:

- Show a "screen" badge on capable devices
- Disable the "Start Stream" button for devices that can't capture
- Label devices `[no screen capture]` in the stream device dropdown

### Limitations

- **LAN only** - No STUN/TURN servers configured
- **Non-trickle ICE** - May fail on complex network topologies
- **JPEG frames over DataChannel** - Not optimized for bandwidth (upgrade to video track planned)

## File Download

Download files from any registered device via the web UI. The server runs a bulk HTTP server on port `:8081` alongside the gRPC server.

### How It Works

1. The web UI sends `POST /api/request-download` with `device_id` and `path`
2. The web server calls `CreateDownloadTicket` on the target device's gRPC server
3. The device generates a one-time-use token (crypto/rand, configurable TTL)
4. The browser receives a direct download URL pointing to the device's bulk HTTP server
5. The file is served via `GET /bulk/download/<token>` on port 8081

### Usage

1. Ensure the `./shared` directory exists on the target device (or set `SHARED_DIR`)
2. In the web UI, open the "File Download" card
3. Select the target device and enter the file path
4. Click "Download"

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BULK_HTTP_ADDR` | `:8081` | Bulk HTTP server listen address |
| `BULK_TTL_SECONDS` | `60` | Download ticket expiration (seconds) |
| `SHARED_DIR` | `./shared` | Root directory for downloadable files |

### Registration

Include `--http-addr` when registering a device so the coordinator knows where to direct download requests:

```bash
go run ./cmd/client register --name "windows-pc" --self-addr "10.20.38.80:50051" --http-addr "10.20.38.80:8081"
```

## Qualcomm AI Hub CLI (optional)

[Qualcomm AI Hub](https://aihub.qualcomm.com/) CLI (`qai-hub`) lets you compile, profile, and deploy AI models targeting Qualcomm devices from any Windows x86 host. No local Qualcomm hardware required.

See [docs/qaihub.md](docs/qaihub.md) for full setup instructions.

**Quick start (Windows):**

```cmd
powershell -ExecutionPolicy Bypass -File scripts/windows/setup_qaihub.ps1
```

## Development

```bash
# Build
make build

# Run tests
make test

# Format code
make fmt

# Run linter
make lint

# Build for all platforms
make build-all
```

## Project Structure

```
edgecli/
├── brain/
│   └── windows-ai-cli/   # C#/.NET CLI for Windows AI integration
│       ├── Program.cs     # CLI entry point (capabilities, plan, summarize)
│       ├── Models.cs      # Request/response types
│       └── WindowsAiCli.csproj
├── cmd/
│   ├── edgecli/           # CLI entry point
│   │   └── commands/      # Cobra commands
│   ├── server/            # gRPC orchestrator server
│   ├── client/            # gRPC CLI client
│   └── web/               # Web UI server
│       └── index.html     # Embedded web UI
├── internal/
│   ├── allowlist/         # Command allowlist for safe execution
│   ├── approval/          # Tool approval workflows
│   ├── brain/             # Go integration with Windows AI CLI
│   │   ├── brain.go       # Types, conversion helpers, public API
│   │   ├── brain_windows.go  # Windows: shells out to CLI
│   │   └── brain_stub.go    # Non-Windows: returns error
│   ├── chat/              # Execution budget management
│   ├── config/            # Configuration management
│   ├── deviceid/          # Device ID persistence
│   ├── elevate/           # Privilege elevation
│   ├── exec/              # Command execution
│   ├── jobs/              # Job/task state machine with group execution
│   ├── mode/              # Safe/dangerous mode
│   ├── osdetect/          # Platform detection
│   ├── redact/            # Secret redaction
│   ├── registry/          # Device registry for orchestration
│   ├── sysinfo/           # System info sampling
│   ├── tools/             # Tool registry framework
│   ├── transfer/          # Download ticket manager (one-time tokens, TTL)
│   ├── ui/                # Terminal UI rendering
│   └── webrtcstream/      # WebRTC screen streaming with pion/webrtc
├── proto/                 # gRPC proto definitions
├── docs/                  # Feature documentation
├── Makefile               # Build configuration
└── go.mod                 # Go module definition
```

## Core Packages

| Package | Purpose |
|---------|---------|
| `tools/` | Tool registry and execution framework |
| `exec/` | Safe command execution with timeouts |
| `approval/` | Interactive approval for dangerous operations |
| `mode/` | Safe vs dangerous mode management |
| `osdetect/` | Platform detection (OS, arch, distro) |
| `config/` | Configuration persistence |
| `ui/` | Terminal UI components |

## License

MIT
