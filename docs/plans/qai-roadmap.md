# QAI Workstream Roadmap — Team EdgeMesh

**Owner**: Rahil Singhi
**Branch**: `rahil-qai`
**Last updated**: 2026-02-06

---

## Mission

Enable Qualcomm AI Hub (QAI) as a first-class citizen in the EdgeMesh distributed computing framework — so that AI models can be compiled, profiled, and eventually deployed to Snapdragon devices through the mesh.

---

## What We Are Delivering

### 1. Infrastructure (DONE)

| Deliverable | Status | Details |
|---|---|---|
| Go toolchain + project build | Done | Go 1.25, all packages compile |
| QAI Hub CLI on Mac | Done | Python venv, qai-hub 0.44.0, API token configured |
| QAI Hub CLI on Windows Snapdragon | Done | Python 3.12, ARM64, venv, token configured |
| gRPC Server on Mac (coordinator) | Done | Port 50051, self-registered |
| gRPC Server on Windows (worker) | Done | Port 50051, firewall opened, ARM64 binary |
| Web UI | Done | Port 8080, all endpoints active |
| Multi-device mesh | Done | Mac + Windows Snapdragon, cross-device jobs working |
| Device registration with NPU flag | Done | Windows registered with `npu` capability |

### 2. QAI Hub Integration (IN PROGRESS)

| Deliverable | Status | Details |
|---|---|---|
| `qai-hub list-devices` from both machines | Done | Returns 100+ Qualcomm target devices |
| `/api/qaihub/doctor` endpoint | Done | Web API health check |
| `edgecli qaihub doctor` CLI | Done | CLI health check |
| `/api/qaihub/compile` endpoint | Available | Needs ONNX model to demo |
| `edgecli qaihub compile` CLI | Available | Compile ONNX for Snapdragon |
| QAI Hub compile demo (MobileNetV2) | TODO | Submit compile job via API |
| QAI Hub profiling integration | TODO | Profile model on target device |

### 3. Python Backend (TODO)

| Deliverable | Status | Details |
|---|---|---|
| Python orchestrator service | TODO | FastAPI/Flask backend for QAI-specific workflows |
| Model registry API | TODO | Track compiled models, artifacts, target devices |
| QAI Hub job status polling | TODO | Watch compile jobs, report progress |
| Artifact management | TODO | Download/store compiled model artifacts |

---

## Questions We Are Answering

1. **Can we compile AI models for Snapdragon from a Mac?**
   - Yes. QAI Hub is a cloud API. We submit from Mac, compilation runs on Qualcomm cloud infra.

2. **Can we distribute model compilation across the mesh?**
   - The compile job itself runs in Qualcomm's cloud. But we can orchestrate submissions from any device in the mesh, and route compiled artifacts to specific devices.

3. **Can a Snapdragon device with NPU be targeted specifically?**
   - Yes. The device registry tracks NPU capability. Routing policies (`REQUIRE_NPU`, `BEST_AVAILABLE`) prefer NPU-equipped devices. QAI Hub lets you compile for specific Snapdragon chipsets.

4. **How does QAI Hub fit into the EdgeMesh architecture?**
   - See architecture diagram below.

5. **Can multiple teammates' devices participate?**
   - Yes. Any device that registers with the coordinator appears in the mesh. All registered devices participate in distributed jobs and can be targeted for model deployment.

---

## Architecture: QAI Hub in EdgeMesh

```
                              ┌─────────────────────────┐
                              │   Qualcomm AI Hub Cloud  │
                              │  (compile, profile, run) │
                              └────────────┬────────────┘
                                           │ HTTPS API
                                           │
┌────────────────────────────────────────────────────────────────┐
│                        EdgeMesh Network                        │
│                                                                │
│  ┌───────────────────┐          ┌───────────────────────────┐  │
│  │  Mac (Coordinator) │  gRPC   │  Snapdragon Windows (NPU) │  │
│  │                    │◀───────▶│                            │  │
│  │  - Web UI :8080    │         │  - gRPC Server :50051      │  │
│  │  - gRPC :50051     │         │  - QAI Hub CLI             │  │
│  │  - QAI Hub CLI     │         │  - NPU acceleration        │  │
│  │  - Python backend  │         │  - Windows AI APIs         │  │
│  └───────────────────┘          └───────────────────────────┘  │
│           ▲                              ▲                      │
│           │                              │                      │
│  ┌────────┴─────────┐          ┌────────┴──────────┐          │
│  │  Teammate Device  │          │  Teammate Device   │          │
│  │  (registered)     │          │  (registered)      │          │
│  └──────────────────┘          └───────────────────┘          │
└────────────────────────────────────────────────────────────────┘
```

---

## Interaction with Other Workstreams

### Web UI
- **Current**: Device list shows NPU badge, QAI Hub doctor card, compile form
- **Enhancement**: Show QAI Hub compile job status, model artifact downloads, per-device model deployment status
- **Multi-device view**: When multiple devices are registered, the UI shows all of them with capabilities. Jobs fan out to all devices.

### Windows AI CLI (C#)
- The Windows machine runs Build 26100 (24H2) — the exact version needed for Windows AI APIs
- The C# CLI (`brain/windows-ai-cli/`) can be built on Windows with .NET SDK to enable AI-powered plan generation
- Currently .NET is not installed; adding it would unlock `used_ai: true` plans

### LLM Agent
- The agent (`/api/agent`) can call tools like `get_capabilities` and `execute_shell_cmd` across the mesh
- QAI Hub operations could be exposed as agent tools (e.g., "compile MobileNetV2 for Galaxy S24")

### Job System
- QAI Hub compile jobs could be submitted as EdgeMesh tasks of kind `QAI_COMPILE`
- The orchestrator would route compile tasks to devices with QAI Hub configured
- Results (compiled artifacts) would be stored in the device's `shared/` directory for download

---

## Demo Scenarios for Hackathon

### Demo 1: Multi-Device Mesh (READY)
1. Open Web UI at http://localhost:8080
2. Show 2+ devices registered (Mac + Windows Snapdragon)
3. Run distributed SYSINFO job — shows results from both devices
4. Route command to Windows with PREFER_REMOTE policy

### Demo 2: QAI Hub Model Pipeline (IN PROGRESS)
1. Show `qai-hub list-devices` — catalog of Qualcomm target devices
2. Submit compile job for MobileNetV2 targeting Samsung Galaxy S24
3. Show job progress and artifact download
4. Demonstrate the cloud-to-edge pipeline

### Demo 3: Intelligent Task Routing (READY)
1. Show device with NPU capability in registry
2. Submit job with REQUIRE_NPU policy — routes to Snapdragon device
3. Show BEST_AVAILABLE routing prefers NPU > GPU > CPU

---

## Technical Notes

### QAI Hub API Token
- Default hackathon token: `b3yhelucambdm13uz9usknrhu98l6pln1dzboooy`
- Configured in `~/.qai_hub/client.ini` on each machine
- The Go client auto-discovers the venv at `.venv-qaihub/`

### Windows Machine Details
- **Host**: 10.206.87.35 (SSH: `chinmay` / `root`)
- **OS**: Windows 11 Pro, Build 26100 (24H2)
- **Arch**: ARM64 (Snapdragon X Elite)
- **Python**: 3.12.9 at `C:\Users\chinmay\Python312\`
- **QAI Hub venv**: `C:\Users\chinmay\venv-qaihub\`
- **Server binary**: `C:\Users\chinmay\server-windows-arm64.exe`

### Port Map
| Port | Service | Machine |
|------|---------|---------|
| 50051 | gRPC orchestrator | Mac + Windows |
| 8080 | Web UI (HTTP) | Mac |
| 8081 | Bulk file download | Mac + Windows |
