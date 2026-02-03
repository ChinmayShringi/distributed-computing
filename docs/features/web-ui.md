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
- Submit job and monitor status
- View task progress per device
- See final concatenated result

### 4. Assistant Chat
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
    "grpc_addr": "127.0.0.1:50051"
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

## UI Styling

- White background with black text
- Responsive design (max-width: 800px)
- Card-based layout
- Color-coded capability badges:
  - CPU: Gray
  - GPU: Green
  - NPU: Purple

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
