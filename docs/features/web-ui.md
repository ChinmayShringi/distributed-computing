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
    "can_screen_capture": true
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
