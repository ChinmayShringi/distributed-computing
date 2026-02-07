# Feature Documentation

Documentation for EdgeCLI features and components.

## Core Features

| Document | Description |
|----------|-------------|
| [gRPC Service](grpc.md) | gRPC orchestrator service, RPC methods, proto definitions |
| [Web UI](web-ui.md) | Browser-based interface, REST API endpoints |
| [CLI Tools](cli.md) | Command-line tools (server, client, web, edgecli) |

## AI & Planning

| Document | Description |
|----------|-------------|
| [Windows AI CLI](../../brain/windows-ai-cli/README.md) | C#/.NET CLI tool for AI-powered plan generation (Windows only) |
| [QAI Hub Integration](../qaihub.md) | Qualcomm AI Hub CLI for model compilation (cloud-based) |
| [Local Chat Runtime](../chat.md) | Ollama/LM Studio integration for local chat on Mac |

## Setup & Hackathon

| Document | Description |
|----------|-------------|
| [QAI Setup Guide](../setup-guide-qai.md) | Full setup: Mac coordinator + Windows Snapdragon worker + QAI Hub |
| [QAI Roadmap](../plans/qai-roadmap.md) | Deliverables, demo scenarios, architecture for QAI workstream |
| [Connection Details](../connection.md) | SSH credentials, Windows machine specs, firewall config |

The brain package (`internal/brain`) integrates with the Windows AI CLI to generate execution plans. On non-Windows platforms, it falls back to deterministic plan generation. Plans can be previewed before submission via the `PreviewPlan` gRPC RPC or `POST /api/plan` web endpoint.

The chat feature (`internal/llm`) provides a local LLM runtime via Ollama or OpenAI-compatible APIs (LM Studio). This is separate from the Qualcomm pipeline, which compiles models for Snapdragon devices but cannot run them on Mac.

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

## Mobile App

| Document | Description |
|----------|-------------|
| [Mobile App](mobile.md) | Flutter cross-platform mobile UI, Android gRPC integration |

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
┌─────────────┐     gRPC      ┌──────┴──────┐     gRPC      ┌─────────────┐
│   Client    │──────────────▶│   Server    │◀─────────────▶│   Worker    │
└─────────────┘               │(coordinator)│               │  (remote)   │
                              └──────┬──────┘               └─────────────┘
                                     │ gRPC
                              ┌──────┴──────┐
                              │ Mobile App  │
                              │ (Flutter)   │
                              └─────────────┘
```

## Key Concepts

- **Device**: A machine running the EdgeCLI server
- **Coordinator**: The main server that manages device registry and routes commands
- **Worker**: A remote device that executes tasks
- **Job**: A collection of tasks distributed across devices
- **Task Group**: Tasks that execute in parallel; groups execute sequentially
- **Routing Policy**: Rules for selecting which device executes a command
