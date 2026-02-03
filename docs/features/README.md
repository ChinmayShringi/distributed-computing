# Feature Documentation

Documentation for EdgeCLI features and components.

## Core Features

| Document | Description |
|----------|-------------|
| [gRPC Service](grpc.md) | gRPC orchestrator service, RPC methods, proto definitions |
| [Web UI](web-ui.md) | Browser-based interface, REST API endpoints |
| [CLI Tools](cli.md) | Command-line tools (server, client, web, edgecli) |

## Orchestration

| Document | Description |
|----------|-------------|
| [Device Registry](device-registry.md) | Device registration, discovery, selection algorithms |
| [Command Routing](routing.md) | Routing policies, remote execution, forwarding |
| [Distributed Jobs](jobs.md) | Job/task system, groups, parallel execution, reduce |

## Operations

| Document | Description |
|----------|-------------|
| [Deployment](deployment.md) | Multi-machine setup, Windows deployment, cross-platform builds |
| [Execution Modes](modes.md) | Safe mode, dangerous mode, allowlists, approvals |

## Quick Links

### Starting the System

```bash
# Start coordinator
make server

# Start web UI
make web

# Open browser
open http://localhost:8080
```

### Adding a Remote Device

```bash
# Deploy to Windows
./deploy-windows.sh

# Register with coordinator
go run ./cmd/client register \
  --name "device-name" \
  --self-addr "IP:50051" \
  --platform "windows" \
  --arch "amd64"
```

### Running a Distributed Job

```bash
curl -X POST http://localhost:8080/api/submit-job \
  -H "Content-Type: application/json" \
  -d '{"text":"collect status","max_workers":0}'
```

## Architecture Overview

```
┌─────────────┐     HTTP      ┌─────────────┐
│   Browser   │──────────────▶│  Web Server │
└─────────────┘               └──────┬──────┘
                                     │ gRPC
                                     ▼
┌─────────────┐     gRPC      ┌─────────────┐     gRPC      ┌─────────────┐
│   Client    │──────────────▶│   Server    │◀─────────────▶│   Worker    │
└─────────────┘               │(coordinator)│               │  (remote)   │
                              └─────────────┘               └─────────────┘
```

## Key Concepts

- **Device**: A machine running the EdgeCLI server
- **Coordinator**: The main server that manages device registry and routes commands
- **Worker**: A remote device that executes tasks
- **Job**: A collection of tasks distributed across devices
- **Task Group**: Tasks that execute in parallel; groups execute sequentially
- **Routing Policy**: Rules for selecting which device executes a command
