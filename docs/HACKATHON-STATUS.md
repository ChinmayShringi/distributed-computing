# EdgeMesh Hackathon Status — Team EdgeMesh (QAI Workstream)

**Last updated**: 2026-02-06 15:10 EST  
**Branch**: `rahil-qai`  
**Owner**: Rahil Singhi  
**Coordinator**: Rahil's Mac at `10.206.187.34:50051`

---

## What We Built (Working Right Now)

### 1. Distributed Orchestrator System ✅

**The core**: A distributed task orchestrator that routes work across multiple devices intelligently.

| Component | Status | Details |
|---|---|---|
| **Device Registry** | ✅ LIVE | Tracks 6 devices with capabilities (CPU/GPU/NPU) |
| **Auto-Registration** | ✅ LIVE | Devices join mesh automatically with `COORDINATOR_ADDR` env var |
| **Cross-Device Routing** | ✅ LIVE | Mac coordinator routes tasks to Windows Snapdragon via SSH tunnel |
| **Distributed Jobs** | ✅ LIVE | Fan-out tasks to all devices in parallel, collect + reduce results |
| **Cost Estimation** | ✅ LIVE | Predict latency and RAM usage per device, recommend best device |
| **LLM_GENERATE Handler** | ✅ BUILT | Devices can execute LLM inference tasks (connects to Ollama/LM Studio) |

### 2. Qualcomm AI Hub (QAI) Integration ✅

**The QAI piece**: Compile AI models for Qualcomm Snapdragon NPU using QAI Hub cloud API.

| Feature | Endpoint | Status |
|---|---|---|
| **CLI Health Check** | `GET /api/qaihub/doctor` | ✅ LIVE |
| **Device Catalog** | `GET /api/qaihub/devices?chipset=...` | ✅ LIVE (100+ Qualcomm targets) |
| **Compile Submission** | `POST /api/qaihub/submit-compile` | ✅ LIVE |
| **Job Status Poll** | `GET /api/qaihub/job-status?job_id=...` | ✅ LIVE |
| **Artifact Download** | `scripts/qaihub_download_job.py` | ✅ LIVE |

**Verified**: Account has 2 successful MobileNetV2 compile jobs (`jpyd4k40p`, `jp2m7q765`).

### 3. Multi-Device Mesh (Team)

| Device | Owner | Capabilities | Address | Status |
|---|---|---|---|---|
| **Rahils-MacBook-Pro** | Rahil | CPU | `10.206.187.34:50051` | ✅ LIVE (coordinator) |
| **QCWorkshop31-Snapdragon-NPU** | Shared | CPU, GPU, NPU | `10.206.87.35:50051` | ✅ LIVE (via SSH tunnel) |
| Bharath-MacBook | Bharath | CPU | `10.206.197.101:50051` | 📝 Registered (server not running) |
| Manav-MacBook | Manav | CPU | `10.206.227.186:50051` | 📝 Registered (server not running) |
| Sariya-MacBook | Sariya | CPU | `10.206.8.90:50051` | 📝 Registered (server not running) |
| Chinmay-MacBook | Chinmay | CPU | `10.206.66.173:50051` | 📝 Registered (server not running) |

---

## Demo-Ready Features

### Demo 1: Multi-Device Distributed Jobs ✅

**What it shows**: Task orchestration across heterogeneous devices.

1. Open Web UI: http://10.206.187.34:8080 (accessible to anyone on network)
2. Navigate to "Distributed Jobs" card
3. Click "Submit Job" → auto-distributes SYSINFO tasks to all running devices
4. Results appear showing stats from Mac + Windows Snapdragon in parallel

**What's impressive**: 
- NPU-capable device (Windows Snapdragon) gets purple NPU badge
- Device selection UI shows CPU/GPU/NPU capabilities
- Jobs fan out to all devices simultaneously, results concatenate

### Demo 2: Intelligent Device Routing ✅

**What it shows**: The orchestrator picks the best device for the task.

1. "Routed Command" card in Web UI
2. Command: `pwd`
3. Policy: **REQUIRE_NPU** → routes to Windows Snapdragon (only NPU device)
4. Policy: **PREFER_REMOTE** → routes away from the Mac
5. Policy: **BEST_AVAILABLE** → chooses NPU > GPU > CPU

### Demo 3: Qualcomm AI Hub Pipeline ✅

**What it shows**: We can compile models for Snapdragon chips via Qualcomm cloud.

**Via Web UI** (if we add a UI card):
1. Click "Compile Model"
2. Enter model path (ONNX file or QAI Hub model ID)
3. Select target: "Samsung Galaxy S24 (Family)"
4. Job submits to Qualcomm cloud
5. Poll status, download compiled artifacts

**Via API** (works now):
```bash
# List target devices
curl "http://localhost:8080/api/qaihub/devices?vendor=samsung&chipset=8gen3"

# Submit compile (need ONNX file)
curl -X POST http://localhost:8080/api/qaihub/submit-compile \
  -H "Content-Type: application/json" \
  -d '{"model":"model.onnx","device_name":"Samsung Galaxy S24 (Family)"}'

# Check status
curl "http://localhost:8080/api/qaihub/job-status?job_id=jpyd4k40p"
```

---

## How It All Fits Together

```
┌─────────────────────────────────────────────────────────────────┐
│                    USER REQUEST                                 │
│  Web UI or CLI: "Summarize this text with Qwen3-8B"             │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│              ORCHESTRATOR (Rahil's Mac)                         │
│                                                                 │
│  1. Parse request → task kind: LLM_GENERATE, model: Qwen3-8B    │
│  2. Query device registry → who has NPU + enough RAM?           │
│  3. Cost estimation → Windows Snapdragon is fastest (NPU)       │
│  4. Route task to Windows Snapdragon via gRPC                   │
└────────────────────────┬────────────────────────────────────────┘
                         │ gRPC: RunTask(kind="LLM_GENERATE")
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│         WORKER (Windows Snapdragon — QCWorkshop31)              │
│                                                                 │
│  1. Receive LLM_GENERATE task                                   │
│  2. Load compiled Qwen3-8B model from local storage             │
│     (either via Ollama OR QNN runtime for QAI Hub compiled)     │
│  3. Run inference on Qualcomm Hexagon NPU                       │
│  4. Return result                                               │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│              RESULT BACK TO USER                                │
└─────────────────────────────────────────────────────────────────┘
```

---

## What's Left for Hackathon

### High Priority (Next 12 Hours)

| Task | Why | Estimate |
|---|---|---|
| **Get ONNX models** | Need Qwen3-8B.onnx, Gemma-4B.onnx, QwenCode-14B.onnx | Find/export |
| **Submit 3 compile jobs** | Show QAI Hub compiling for Snapdragon | 30 min |
| **Test LLM_GENERATE end-to-end** | Install Ollama on Windows, test inference | 1 hour |
| **Model routing logic** | Map request → model (summarize→Qwen3, code→QwenCode, img→Gemma) | 2 hours |
| **Deploy compiled models** | Push .bin to Windows shared/ dir | 30 min |
| **Update Web UI** | QAI Hub card with device catalog, job submission form | 2 hours |

### Medium Priority (If Time Permits)

| Task | Why |
|---|---|
| Add phone/Arduino to mesh | Show 4+ device orchestration |
| Windows AI CLI (.NET) | Enable AI-powered plan generation on Snapdragon |
| Benchmark NPU vs CPU | Show speedup on Snapdragon NPU vs Mac CPU |
| Model artifact viewer | Show compiled .bin files in Web UI |

### Low Priority (Future)

| Task | Why |
|---|---|
| Persistent registry (database) | Survives coordinator restarts |
| Heartbeat/health monitoring | Auto-remove dead devices |
| P2P mesh (no fixed coordinator) | Any device can be coordinator |

---

## Technical Constraints (Good to Know)

| Constraint | Impact | Workaround |
|---|---|---|
| **Network blocks port 50051 directly** | Mac → Windows requires SSH tunnel | `ssh -f -N -L 50052:127.0.0.1:50051` |
| **Windows Defender firewall** | Blocks gRPC when enabled | Turned off for hackathon |
| **SSH tunnel slow (30s)** | gRPC dial needs 15s timeout | Changed from 2s to 15s |
| **QAI Hub compiles in cloud** | Can't run compiled models on Mac | Models target Snapdragon NPU only |
| **Registry is in-memory** | Coordinator restart = device list reset | Re-register on startup |

---

## Commands to Keep Services Running

**On Mac (Coordinator)**:
```bash
# Terminal 1: gRPC + Coordinator
GRPC_ADDR=0.0.0.0:50051 go run ./cmd/server

# Terminal 2: Web UI
go run ./cmd/web

# Terminal 3: SSH tunnel to Windows
ssh -f -N -L 50052:127.0.0.1:50051 chinmay@10.206.87.35
```

**On Windows (QCWorkshop31)**:
```cmd
C:\Users\chinmay\start-server.bat
```

**On Teammates' Macs** (when they join):
```bash
git pull origin rahil-qai
COORDINATOR_ADDR=10.206.187.34:50051 GRPC_ADDR=0.0.0.0:50051 go run ./cmd/server
```

---

## Key URLs

| Service | URL | Access |
|---|---|---|
| **Web UI** | http://localhost:8080 | Local only |
| **Web UI (network)** | http://10.206.187.34:8080 | Anyone on venue WiFi |
| **gRPC Coordinator** | `10.206.187.34:50051` | Teammates register here |
| **QAI Hub Account** | https://aihub.qualcomm.com/ | Token: `b3yhelucambdm13uz9usknrhu98l6pln1dzboooy` |

---

## Next Steps (Immediate)

1. **Get the 3 ONNX models** (Qwen3-8B, Gemma-4B, QwenCode-14B) — do you have these or need to export them?
2. **Install Ollama on Windows** — so `LLM_GENERATE` tasks can actually run inference
3. **Submit compile jobs** — compile all 3 models for Samsung Galaxy S24 via QAI Hub
4. **Test end-to-end inference** — send text prompt → Windows Snapdragon runs model → returns result

Which do you want to tackle first?
