# Web UI

The web UI provides a browser-based interface for device management, command execution, and job orchestration.

## Running the Web Server

```bash
make web
# or
go run ./cmd/web
```

**Default Address:** `http://localhost:8080`
**Environment Variable:** `WEB_ADDR` (e.g., `WEB_ADDR=:3000`)

## Features

### 1. Device List
- Displays all registered devices
- Shows device name, platform/arch, gRPC address
- Displays capabilities (CPU, GPU, NPU) as badges
- Shows "screen" badge for devices that support screen capture
- Click "Refresh" to reload device list

### 2. Routed Command Execution
- Enter command (e.g., `pwd`, `ls`, `cat`)
- Enter arguments (comma-separated)
- Select routing policy:
  - **Best Available** - NPU > GPU > CPU preference
  - **Prefer Remote** - Route to non-local device
  - **Require NPU** - Only NPU-capable devices
  - **Prefer Local Model** - Prefer device with Ollama/local LLM
  - **Require Local Model** - Only devices with local LLM
  - **Force Device ID** - Select specific device from dropdown
- View output with execution metadata (device, time, exit code)

### 3. Distributed Jobs
- Enter job description
- Set max workers (0 = all devices)
- **Preview Plan** - See the execution plan before submitting (shows rationale, AI usage, and task assignments)
- Submit job and monitor status
- View task progress per device
- See final concatenated result

### 4. File Download
- Select target device from dropdown
- Enter file path on the remote device
- Click "Download" to request a one-time download ticket
- Browser initiates download via the device's bulk HTTP server

### 5. Assistant Chat
- Natural language interface
- Commands: "list devices", "pwd", "ls"
- Interactive command execution

### 6. Qualcomm Model Pipeline (qai-hub)
- Run qai-hub doctor to check CLI installation
- Compile ONNX models for Qualcomm Snapdragon devices
- Note: Compiled models target Snapdragon and cannot run locally on Mac

### 7. Local Chat Runtime
- Chat powered by local LLM (Ollama or LM Studio)
- Independent of Qualcomm pipeline
- Health status indicator
- Multi-turn conversation support

## REST API Endpoints

### GET /api/devices
Returns list of registered devices.

**Response:**
```json
[
  {
    "device_id": "e452458d-...",
    "device_name": "macbook-pro",
    "platform": "darwin",
    "arch": "arm64",
    "capabilities": ["cpu"],
    "grpc_addr": "127.0.0.1:50051",
    "can_screen_capture": true,
    "has_local_model": true,
    "local_model_name": "llama3.2:3b",
    "local_chat_endpoint": "http://localhost:11434"
  }
]
```

### POST /api/routed-cmd
Execute a routed command.

**Request:**
```json
{
  "cmd": "pwd",
  "args": [],
  "policy": "PREFER_REMOTE",
  "force_device_id": ""
}
```

**Response:**
```json
{
  "stdout": "/Users/user\n",
  "stderr": "",
  "exit_code": 0,
  "selected_device_id": "e452458d-...",
  "selected_device_name": "macbook-pro",
  "total_time_ms": 12.5,
  "executed_locally": true
}
```

### POST /api/submit-job
Submit a distributed job.

**Request:**
```json
{
  "text": "collect status",
  "max_workers": 0
}
```

**Response:**
```json
{
  "job_id": "8155c378-...",
  "created_at": 1770149334,
  "summary": "distributed to 2 devices in 1 group(s)"
}
```

### POST /api/plan
Preview an execution plan without creating a job. Shows which devices would receive tasks, whether AI was used for plan generation, and the rationale.

**Request:**
```json
{
  "text": "collect status",
  "max_workers": 0
}
```

**Response:**
```json
{
  "used_ai": false,
  "notes": "Brain not available (non-Windows or disabled)",
  "rationale": "Default: 1 SYSINFO per device, 2 of 2 devices selected",
  "plan": {
    "groups": [
      {
        "index": 0,
        "tasks": [
          {"task_id": "uuid-1", "kind": "SYSINFO", "input": "collect_status", "target_device_id": "e452458d-..."},
          {"task_id": "uuid-2", "kind": "SYSINFO", "input": "collect_status", "target_device_id": "f66a8dc8-..."}
        ]
      }
    ]
  },
  "reduce": {"kind": "CONCAT"}
}
```

**Response fields:**
- `used_ai` - Whether the Windows AI Brain was used for plan generation
- `notes` - AI availability information
- `rationale` - Human-readable explanation of plan generation logic
- `plan` - The execution plan (groups of parallel tasks)
- `reduce` - How results would be combined

### POST /api/plan-cost
Estimate execution cost (latency, memory) for a plan. Returns per-device cost breakdowns and recommends the best device for execution.

**Request:**
```json
{
  "plan": {
    "groups": [
      {
        "index": 0,
        "tasks": [
          {"task_id": "step-1", "kind": "LLM_GENERATE", "prompt_tokens": 500, "max_output_tokens": 200},
          {"task_id": "step-2", "kind": "LLM_GENERATE", "prompt_tokens": 100, "max_output_tokens": 50}
        ]
      }
    ]
  },
  "device_ids": []
}
```

**Response:**
```json
{
  "total_predicted_ms": 4166.67,
  "device_costs": [
    {
      "device_id": "e452458d-...",
      "device_name": "macbook-pro",
      "total_ms": 8333.33,
      "step_costs": [
        {"task_id": "step-1", "kind": "LLM_GENERATE", "predicted_ms": 8333.33, "predicted_memory_mb": 2048, "unknown_cost": false, "notes": "using default throughput for platform"},
        {"task_id": "step-2", "kind": "LLM_GENERATE", "predicted_ms": 2000.00, "predicted_memory_mb": 2048, "unknown_cost": false, "notes": "using default throughput for platform"}
      ],
      "estimated_peak_ram_mb": 2048,
      "ram_sufficient": true
    },
    {
      "device_id": "f66a8dc8-...",
      "device_name": "fast-device",
      "total_ms": 4166.67,
      "step_costs": [...],
      "estimated_peak_ram_mb": 2048,
      "ram_sufficient": true
    }
  ],
  "recommended_device_id": "f66a8dc8-...",
  "recommended_device_name": "fast-device",
  "has_unknown_costs": false,
  "warning": ""
}
```

**Request fields:**
- `plan` - The execution plan (same structure as job submission)
- `device_ids` - Optional array of device IDs to limit estimation to (empty = all devices)

**Response fields:**
- `total_predicted_ms` - Best device's total predicted latency
- `device_costs` - Per-device breakdown with step costs
- `recommended_device_id/name` - Device with lowest total cost
- `has_unknown_costs` - True if any step type was unrecognized
- `warning` - Warning message if applicable

### GET /api/job?id={job_id}
Get job status.

**Response:**
```json
{
  "job_id": "8155c378-...",
  "state": "DONE",
  "tasks": [
    {
      "task_id": "3f0b6a31-...",
      "assigned_device_id": "e452458d-...",
      "assigned_device_name": "macbook-pro",
      "state": "DONE",
      "result": "Device: macbook-pro\n...",
      "error": ""
    }
  ],
  "final_result": "=== macbook-pro (e452458d) ===\n...",
  "current_group": 0,
  "total_groups": 1
}
```

### POST /api/assistant
Send message to assistant.

**Request:**
```json
{
  "text": "list devices"
}
```

**Response:**
```json
{
  "reply": "Found 2 devices:\n..."
}
```

### POST /api/request-download
Request a file download from a specific device. The web server calls `CreateDownloadTicket` on the target device's gRPC server and returns a direct download URL.

**Request:**
```json
{
  "device_id": "f66a8dc8-...",
  "path": "./shared/report.csv"
}
```

**Response:**
```json
{
  "download_url": "http://10.20.38.80:8081/bulk/download/abc123token",
  "filename": "report.csv",
  "size_bytes": 204800,
  "expires_unix_ms": 1770149394000
}
```

### GET /api/chat/health
Check the local chat runtime health.

**Response:**
```json
{
  "ok": true,
  "provider": "ollama",
  "base_url": "http://localhost:11434",
  "model": "llama2"
}
```

### POST /api/chat
Send a chat message to the local LLM runtime.

**Request:**
```json
{
  "messages": [
    {"role": "user", "content": "Hello!"}
  ]
}
```

**Response:**
```json
{
  "reply": "Hello! How can I help you today?"
}
```

Supports multi-turn conversations by including the full message history.

### POST /api/llm-task
Route an LLM inference task to a device with a local model (Ollama). Uses `PREFER_LOCAL_MODEL` routing by default.

**Request:**
```json
{
  "prompt": "What is the capital of France?",
  "model": "",
  "device_id": ""
}
```

**Response:**
```json
{
  "output": "The capital of France is Paris.",
  "model_used": "llama3.2:3b",
  "tokens_generated": 8,
  "device_id": "e452458d-...",
  "device_name": "macbook-pro"
}
```

**Request fields:**
- `prompt` - The user prompt to send to the LLM
- `model` - Optional: specific model name (uses device's default if empty)
- `device_id` - Optional: target specific device (uses `PREFER_LOCAL_MODEL` if empty)

**Errors:**
- 404: "No device with local model available"
- 502: Connection or RPC failure

### GET /api/qaihub/doctor
Check the qai-hub CLI installation and configuration.

**Response:**
```json
{
  "qai_hub_found": true,
  "qai_hub_version": "0.18.0",
  "token_env_present": true,
  "notes": ["qai-hub binary found at: /usr/local/bin/qai-hub", "CLI commands appear functional"]
}
```

### POST /api/qaihub/compile
Compile an ONNX model using Qualcomm AI Hub.

**Request:**
```json
{
  "onnx_path": "/path/to/model.onnx",
  "target": "Samsung Galaxy S24",
  "runtime": "precompiled_qnn_onnx"
}
```

**Response:**
```json
{
  "submitted": true,
  "job_id": "abc123",
  "out_dir": "/path/to/artifacts/qaihub/20250205-120000",
  "raw_output_path": "/path/to/artifacts/qaihub/20250205-120000/compile.log",
  "notes": ["Compile command completed"]
}
```

Note: Compiled models target Qualcomm Snapdragon devices and cannot run on Mac.

## UI Styling

- White background with black text
- Responsive design (max-width: 800px)
- Card-based layout
- Color-coded capability badges:
  - CPU: Gray
  - GPU: Green
  - NPU: Purple
  - Screen: Blue (devices that support screen capture)

## Stream Capability Gating

The web UI checks the `can_screen_capture` field on each device before allowing screen streaming:

- Devices without screen capture support are labeled `[no screen capture]` in the stream device dropdown
- If a user selects a device with `can_screen_capture: false` and FORCE_DEVICE_ID policy, the UI shows an error and prevents the stream from starting
- The `can_screen_capture` flag is set at server startup by performing a test screen capture using `kbinani/screenshot`

## Architecture

The web server:
1. Serves embedded HTML/CSS/JS from `cmd/web/index.html`
2. Connects to gRPC server on startup
3. Translates REST requests to gRPC calls
4. Returns JSON responses

```
Browser → HTTP (REST) → Web Server → gRPC → Orchestrator Server
```

## Embedded HTML

The HTML is embedded using Go's `embed` package:

```go
//go:embed index.html
var indexHTML string
```

Changes to `index.html` require rebuilding the web server.
