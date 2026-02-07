# CLI Tools

EdgeCLI provides several command-line tools for orchestration, device management, and command execution.

## Binaries

| Binary | Purpose | Command |
|--------|---------|---------|
| `edgecli` | Main CLI with safe/dangerous modes | `go run ./cmd/edgecli` |
| `server` | gRPC orchestrator server | `go run ./cmd/server` |
| `client` | gRPC client for device ops | `go run ./cmd/client` |
| `web` | HTTP server with web UI | `go run ./cmd/web` |

## Server (`cmd/server`)

Runs the gRPC orchestrator server.

```bash
# Start with defaults (port 50051)
go run ./cmd/server

# Custom port
GRPC_ADDR=:9000 go run ./cmd/server

# With custom device ID
DEVICE_ID=my-custom-id go run ./cmd/server
```

**Output:**
```
[INFO] Self-registered as device: id=e452458d-... name=hostname addr=127.0.0.1:50051
[INFO] Orchestrator server listening on :50051
[INFO] Server device ID: e452458d-...
[INFO] Server gRPC address: 127.0.0.1:50051
[INFO] Allowed commands: [ls cat pwd]
```

## Client (`cmd/client`)

CLI client for device registration and commands.

### Register Device

```bash
# Register a remote device with the coordinator
go run ./cmd/client register \
  --name "windows-pc" \
  --self-addr "10.20.38.80:50051" \
  --platform "windows" \
  --arch "amd64"
```

**Flags:**
- `--name` - Human-readable device name
- `--self-addr` - Device's gRPC address (reachable from coordinator)
- `--platform` - OS platform (darwin, windows, linux)
- `--arch` - Architecture (amd64, arm64)
- `--server` - Coordinator address (default: localhost:50051)

### List Devices

```bash
go run ./cmd/client list
```

### Execute Command

```bash
go run ./cmd/client exec pwd
go run ./cmd/client exec ls -la
```

## Web Server (`cmd/web`)

Runs HTTP server with embedded web UI.

```bash
# Start with defaults (port 8080, connects to localhost:50051)
go run ./cmd/web

# Custom ports
WEB_ADDR=:3000 GRPC_ADDR=server:50051 go run ./cmd/web
```

**Output:**
```
[INFO] Connecting to gRPC server at localhost:50051
[INFO] Web server listening on :8080
[INFO] Connected to gRPC server at localhost:50051
[INFO] Open http://localhost:8080 in your browser
```

## EdgeCLI (`cmd/edgecli`)

Main CLI with safe/dangerous mode execution.

### Modes

**Safe Mode (default):**
- Command allowlist enforced
- Approval workflows for dangerous operations
- Schema validation

**Dangerous Mode:**
- No restrictions
- Requires explicit confirmation

```bash
# Safe mode
go run ./cmd/edgecli

# Dangerous mode
go run ./cmd/edgecli --allow-dangerous
```

### Debug Commands

```bash
go run ./cmd/edgecli debug
```

## Building Binaries

```bash
# Build all
make build-all

# Individual builds
go build -o edgecli ./cmd/edgecli
go build -o server ./cmd/server
go build -o client ./cmd/client
go build -o web ./cmd/web

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build -o dist/server-windows.exe ./cmd/server
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GRPC_ADDR` | `:50051` | gRPC server listen address |
| `WEB_ADDR` | `:8080` | Web server listen address |
| `DEVICE_ID` | (auto) | Override device ID |
| `DEV_KEY` | - | Security key for auth |

## Makefile Targets

```bash
make server     # Run gRPC server
make web        # Run web server
make build      # Build edgecli binary
make build-all  # Cross-platform builds
make test       # Run tests
make lint       # Run linter
make proto      # Regenerate proto files
```

## Multi-Device Workflow

```bash
# Terminal 1: Start coordinator (Mac)
make server

# Terminal 2: Start web UI
make web

# Terminal 3: Start worker (Windows via SSH)
./deploy-windows.sh

# Terminal 4: Register Windows with Mac
go run ./cmd/client register \
  --name "windows-pc" \
  --self-addr "10.20.38.80:50051" \
  --platform "windows" \
  --arch "amd64"

# Open browser
open http://localhost:8080
```

## Deployment Script

Deploy server to Windows machine:

```bash
./deploy-windows.sh
```

This script:
1. Builds Windows binary
2. Stops existing server on Windows
3. Copies binary via SCP
4. Starts server
5. Verifies it's listening
