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
├── cmd/edgecli/           # CLI entry point
│   └── commands/          # Cobra commands
├── internal/
│   ├── approval/          # Tool approval workflows
│   ├── chat/              # Execution budget management
│   ├── config/            # Configuration management
│   ├── elevate/           # Privilege elevation
│   ├── exec/              # Command execution
│   ├── mode/              # Safe/dangerous mode
│   ├── osdetect/          # Platform detection
│   ├── redact/            # Secret redaction
│   ├── tools/             # Tool registry framework
│   └── ui/                # Terminal UI rendering
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
