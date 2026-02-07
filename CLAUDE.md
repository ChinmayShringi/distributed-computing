# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> **IMPORTANT:** Read `prework.md` BEFORE starting any task. Read `postwork.md` AFTER completing any task.

## Build Commands

```bash
make build          # Build CLI binary
make server         # Run gRPC server (go run ./cmd/server)
make web            # Run web UI server (go run ./cmd/web)
make proto          # Generate gRPC code from proto/orchestrator.proto
make test           # Run tests (go test -v ./...)
make lint           # Run golangci-lint
make fmt            # Format code (go fmt + goimports)
make build-all      # Cross-platform builds (darwin/linux/windows)
```

**Deploy to Windows:**
```bash
./deploy-windows.sh  # Build, copy, and start server on Windows machine (10.20.38.80)
```

This script:
1. Builds `dist/server-windows.exe`
2. Stops existing server on Windows
3. Copies binary via SCP
4. Starts server with `GRPC_ADDR=0.0.0.0:50051`
5. Verifies server is listening

See `docs/connection.md` for detailed Windows machine setup (SSH credentials, manual deployment, multi-device demo).

## Documentation

Feature documentation should be placed under `docs/` directory:
- `docs/plans/` - Implementation plans
- `docs/features/` - Feature documentation and guides
- `docs/features/README.md` - Feature index (update when adding new feature areas)

## Architecture

EdgeCLI is a distributed orchestration system with a gRPC control plane for multi-device command execution, AI task routing, and screen streaming.

### Components

| Binary | Purpose | Default Port |
|--------|---------|--------------|
| `cmd/server` | gRPC orchestrator (device registry, job management, task routing, WebRTC streaming, bulk HTTP file server) | `:50051` (gRPC), `:8081` (HTTP bulk) |
| `cmd/web` | HTTP server with embedded web UI (REST-to-gRPC bridge) | `:8080` |
| `cmd/client` | CLI client for device registration and commands | - |
| `cmd/edgecli` | Main CLI with safe/dangerous mode execution | - |
| `brain/windows-ai-cli` | C#/.NET CLI for AI-powered plan generation (Windows only, separate `dotnet build`) | - |

### Control Flow

```
┌─────────────┐     HTTP      ┌─────────────┐     gRPC      ┌─────────────┐
│   Browser   │──────────────▶│  Web Server  │──────────────▶│   Server    │
│             │               │  (cmd/web)   │               │(coordinator)│
└─────────────┘               └──────────────┘               └──────┬──────┘
                                                                    │
                                                     ┌──────────────┼──────────────┐
                                                     │              │              │
                                                     ▼              ▼              ▼
                                               ┌──────────┐  ┌──────────┐  ┌────────────┐
                                               │ Worker A  │  │ Worker B │  │  Brain CLI │
                                               │ (remote)  │  │ (remote) │  │ (Windows)  │
                                               └──────────┘  └──────────┘  └────────────┘
```

### Key Flows

| Flow | Path |
|------|------|
| Device registration | `client` -> gRPC `RegisterDevice` -> server registry |
| Routed command | Web UI -> `POST /api/routed-cmd` -> gRPC `ExecuteRoutedCommand` -> selected device |
| Job submission | Web UI -> `POST /api/submit-job` -> gRPC `SubmitJob` -> brain (optional) -> `CreateJob` -> parallel task execution |
| Plan preview | Web UI -> `POST /api/plan` -> gRPC `PreviewPlan` -> brain (optional) -> return plan without execution |
| Screen streaming | Web UI -> `POST /api/stream/start` -> gRPC `StartWebRTC` on target device -> WebRTC DataChannel -> browser |

### Routing Policies

Commands can be routed to devices using policies:
- `BEST_AVAILABLE` - NPU > GPU > CPU preference
- `PREFER_REMOTE` - Prefer non-local devices
- `REQUIRE_NPU` - Fail if no NPU device
- `FORCE_DEVICE_ID` - Target specific device

### Job Execution Model

Jobs contain **task groups** that execute sequentially. Tasks within a group execute in parallel across devices. Results are combined using a reduce operation (currently CONCAT).

```
Job -> [Group 0] -> [Group 1] -> ... -> Reduce -> Final Result
          |            |
     [Task A, B]  [Task C, D]  (parallel within group)
```

Plan generation priority:
1. Explicit plan in `JobRequest` (if provided)
2. Windows AI Brain (if enabled and available)
3. Default deterministic plan (1 SYSINFO per device)

## Key Packages

| Package | Purpose |
|---------|---------|
| `internal/registry` | In-memory device registry with selection algorithms |
| `internal/jobs` | Job/task state machine with group execution |
| `internal/brain` | Windows AI CLI integration for plan generation (platform-specific via build tags) |
| `internal/cost` | Plan cost estimation (latency, memory) with device recommendations |
| `internal/llm` | LLM provider interfaces (planning + chat), Ollama/OpenAI-compatible clients |
| `internal/qaihub` | Qualcomm AI Hub CLI wrapper (model compilation, doctor checks) |
| `internal/exec` | Command execution with timeouts |
| `internal/allowlist` | Command whitelist for safe mode |
| `internal/mode` | Safe vs dangerous mode management |
| `internal/approval` | Interactive approval workflows |
| `internal/tools` | Tool registry interface |
| `internal/webrtcstream` | WebRTC screen streaming with pion/webrtc |
| `internal/transfer` | Download ticket manager (one-time-use tokens, TTL, crypto/rand) |
| `internal/sysinfo` | Host metrics sampling (CPU, memory) |
| `internal/deviceid` | Device ID persistence (`~/.edgemesh/device_id`) |
| `brain/windows-ai-cli` | C#/.NET CLI tool wrapping Windows AI APIs (Windows only, separate build) |

### Platform-Specific Code

The brain package uses Go build tags for platform-specific behavior:
- `internal/brain/brain.go` - Shared types, public API, conversion helpers
- `internal/brain/brain_windows.go` - `//go:build windows` - Calls `WindowsAiCli.exe` via `exec.Command`
- `internal/brain/brain_stub.go` - `//go:build !windows` - Returns error (brain unavailable)

When changing the brain API (types, signatures), all three files must be updated.

## Proto Changes

When modifying `proto/orchestrator.proto`:
1. Run `make proto` to regenerate
2. Generated files go to `proto/*.pb.go`
3. Update corresponding handlers in `cmd/server/main.go`
4. Update REST bridge in `cmd/web/main.go` if new RPC is web-exposed
5. Update `cmd/web/index.html` if new UI needed
6. Update `docs/features/grpc.md`

### Current gRPC RPCs (16 total)

| RPC | Purpose |
|-----|---------|
| `CreateSession` | Authenticate client, get session ID |
| `Heartbeat` | Verify session is alive |
| `ExecuteCommand` | Run allowed command on server |
| `RegisterDevice` | Register device in registry |
| `ListDevices` | List all registered devices |
| `GetDeviceStatus` | Get device metrics |
| `RunAITask` | Route AI task to best device (stub) |
| `ExecuteRoutedCommand` | Run command on best available device via policy |
| `SubmitJob` | Submit distributed job across devices |
| `GetJob` | Get job status and results |
| `PreviewPlan` | Preview execution plan without creating a job |
| `PreviewPlanCost` | Estimate execution cost for a plan before running |
| `RunTask` | Execute a task locally (worker RPC) |
| `CreateDownloadTicket` | Create one-time-use file download token |
| `ReadFile` | Read file from device (for LLM tool calling) |
| `StartWebRTC` / `CompleteWebRTC` / `StopWebRTC` | Screen streaming via WebRTC |
| `HealthCheck` | Server health status |

### Current REST Endpoints (18 total)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/` | GET | Serve embedded web UI |
| `/api/devices` | GET | List devices (includes `can_screen_capture`) |
| `/api/routed-cmd` | POST | Execute routed command |
| `/api/submit-job` | POST | Submit distributed job |
| `/api/job?id=` | GET | Get job status |
| `/api/plan` | POST | Preview execution plan |
| `/api/plan-cost` | POST | Estimate execution cost for a plan |
| `/api/assistant` | POST | Natural language interface |
| `/api/request-download` | POST | Request file download ticket from device |
| `/api/stream/start` | POST | Start WebRTC stream |
| `/api/stream/answer` | POST | Complete WebRTC handshake |
| `/api/stream/stop` | POST | Stop stream |
| `/api/chat/health` | GET | Chat runtime health check |
| `/api/chat` | POST | Send chat message (local LLM) |
| `/api/agent` | POST | LLM agent with tool calling |
| `/api/agent/health` | GET | Agent health check |
| `/api/qaihub/doctor` | GET | QAI Hub CLI health check |
| `/api/qaihub/compile` | POST | Compile model with Qualcomm AI Hub (via CLI) |
| `/api/qaihub/devices` | GET | List QAI Hub target devices (filter: name, chipset, vendor) |
| `/api/qaihub/job-status` | GET/POST | Check QAI Hub compile job status (via Python SDK) |
| `/api/qaihub/submit-compile` | POST | Submit compile job via Python SDK |

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `GRPC_ADDR` | `:50051` | Server listen address |
| `WEB_ADDR` | `:8080` | Web server address |
| `DEVICE_ID` | auto-generated | Override device ID |
| `DEV_KEY` | `dev` | Security key for gRPC auth |
| `WINDOWS_AI_CLI_PATH` | (empty) | Path to `WindowsAiCli.exe` (Windows only) |
| `USE_WINDOWS_AI_PLANNER` | `false` | Set to `"true"` to enable Windows AI brain |
| `BULK_HTTP_ADDR` | `:8081` | Bulk file download HTTP server address |
| `BULK_TTL_SECONDS` | `60` | Download ticket time-to-live in seconds |
| `SHARED_DIR` | `./shared` | Directory served for bulk file downloads |
| `CHAT_PROVIDER` | `ollama` | Chat provider: `ollama`, `openai`, or `echo` |
| `CHAT_BASE_URL` | `http://localhost:11434` | Chat API base URL (Ollama default) |
| `CHAT_MODEL` | `llama2` | Model name for chat |
| `CHAT_API_KEY` | (empty) | API key (optional, for OpenAI-compatible) |
| `CHAT_TIMEOUT_SECONDS` | `60` | Chat request timeout |
| `AGENT_MAX_ITERATIONS` | `8` | Max tool calling iterations for agent |
| `P2P_DISCOVERY` | `true` | UDP broadcast peer discovery (set `false` to disable) |
| `DISCOVERY_PORT` | `50050` | UDP port for P2P discovery broadcasts |
| `SEED_PEERS` | (empty) | Comma-separated IPs for cross-subnet discovery |

## Multi-Device Setup

1. Start coordinator: `make server`
2. Start web UI: `make web`
3. Register remote device: `go run ./cmd/client register --name "device-name" --self-addr "IP:50051" --platform "windows" --arch "amd64"`
4. Open http://localhost:8080

## Windows Machine

```
Host: 10.20.38.80
User: sshuser
Pass: root
```

See `docs/connection.md` for full details.

## Execution Modes

- **Safe Mode** (default): Command allowlist enforced, approval workflows active
- **Dangerous Mode** (`--allow-dangerous`): No restrictions, requires explicit confirmation phrase
