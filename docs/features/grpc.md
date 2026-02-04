# gRPC Orchestrator Service

The gRPC service provides the control plane for multi-device orchestration, command execution, and distributed job management.

## Service Definition

**Package:** `edgemesh`
**File:** `proto/orchestrator.proto`
**Go Package:** `github.com/edgecli/edgecli/proto`

## RPC Methods

### Session Management

#### CreateSession
Creates an authenticated session for a client.

```protobuf
rpc CreateSession (AuthRequest) returns (SessionInfo);
```

**Request:**
```protobuf
message AuthRequest {
  string device_name = 1;    // Client device name
  string security_key = 2;   // Authentication key
}
```

**Response:**
```protobuf
message SessionInfo {
  string session_id = 1;     // UUID for session
  string host_name = 2;      // Server hostname
  int64 connected_at = 3;    // Unix timestamp
}
```

#### Heartbeat
Verifies session is still valid.

```protobuf
rpc Heartbeat (SessionInfo) returns (Empty);
```

#### ExecuteCommand
Executes an allowed command on the server.

```protobuf
rpc ExecuteCommand (CommandRequest) returns (CommandResponse);
```

**Request:**
```protobuf
message CommandRequest {
  string session_id = 1;
  string command = 2;        // Command name (e.g., "ls", "pwd")
  repeated string args = 3;  // Command arguments
}
```

**Response:**
```protobuf
message CommandResponse {
  int32 exit_code = 1;
  string stdout = 2;
  string stderr = 3;
}
```

### Device Discovery

#### RegisterDevice
Registers a device in the orchestrator's registry.

```protobuf
rpc RegisterDevice (DeviceInfo) returns (DeviceAck);
```

**Request:**
```protobuf
message DeviceInfo {
  string device_id = 1;      // Stable UUID (persisted on device)
  string device_name = 2;    // Human-readable name
  string platform = 3;       // "darwin", "windows", "linux"
  string arch = 4;           // "amd64", "arm64"
  bool has_cpu = 5;          // Always true
  bool has_gpu = 6;          // GPU available
  bool has_npu = 7;          // NPU available (Snapdragon, etc.)
  string grpc_addr = 8;      // Reachable address (e.g., "10.0.0.5:50051")
  bool can_screen_capture = 9; // True if device can capture screen (tested at startup)
}
```

#### ListDevices
Returns all registered devices.

```protobuf
rpc ListDevices (ListDevicesRequest) returns (ListDevicesResponse);
```

#### GetDeviceStatus
Returns status of a specific device.

```protobuf
rpc GetDeviceStatus (DeviceId) returns (DeviceStatus);
```

**Response:**
```protobuf
message DeviceStatus {
  string device_id = 1;
  int64 last_seen = 2;       // Unix timestamp
  double cpu_load = 3;       // 0..1 or -1 if unavailable
  uint64 mem_used_mb = 4;
  uint64 mem_total_mb = 5;
}
```

### Routed Execution

#### ExecuteRoutedCommand
Executes a command on the best available device based on routing policy.

```protobuf
rpc ExecuteRoutedCommand (RoutedCommandRequest) returns (RoutedCommandResponse);
```

**Request:**
```protobuf
message RoutedCommandRequest {
  string session_id = 1;
  RoutingPolicy policy = 2;
  string command = 3;
  repeated string args = 4;
}

message RoutingPolicy {
  enum Mode {
    BEST_AVAILABLE = 0;      // NPU > GPU > CPU
    REQUIRE_NPU = 1;         // Fail if no NPU
    PREFER_REMOTE = 2;       // Prefer non-self device
    FORCE_DEVICE_ID = 3;     // Target specific device
  }
  Mode mode = 1;
  string device_id = 2;      // Used with FORCE_DEVICE_ID
}
```

**Response:**
```protobuf
message RoutedCommandResponse {
  CommandResponse output = 1;
  string selected_device_id = 2;
  string selected_device_name = 3;
  string selected_device_addr = 4;
  double total_time_ms = 5;
  bool executed_locally = 6;
}
```

### Job Orchestration

#### SubmitJob
Submits a distributed job across multiple devices.

```protobuf
rpc SubmitJob (JobRequest) returns (JobInfo);
```

**Request:**
```protobuf
message JobRequest {
  string session_id = 1;
  string text = 2;           // Job description
  int32 max_workers = 3;     // 0 = all devices
  Plan plan = 4;             // Optional execution plan
  ReduceSpec reduce = 5;     // How to combine results
}
```

**Plan Structure:**
```protobuf
message Plan {
  repeated TaskGroup groups = 1;  // Sequential execution
}

message TaskGroup {
  int32 index = 1;               // Execution order
  repeated TaskSpec tasks = 2;   // Parallel execution within group
}

message TaskSpec {
  string task_id = 1;
  string kind = 2;               // "SYSINFO", "ECHO"
  string input = 3;
  string target_device_id = 4;   // Empty = auto-assign
}

message ReduceSpec {
  string kind = 1;               // "CONCAT"
}
```

#### GetJob
Gets the status of a job.

```protobuf
rpc GetJob (JobId) returns (JobStatus);
```

**Response:**
```protobuf
message JobStatus {
  string job_id = 1;
  string state = 2;              // QUEUED, RUNNING, DONE, FAILED
  repeated TaskStatus tasks = 3;
  string final_result = 4;
  int32 current_group = 5;
  int32 total_groups = 6;
}
```

### Plan Preview

#### PreviewPlan
Generates an execution plan without creating a job. Returns the plan, metadata about whether AI was used, and the rationale for the plan.

```protobuf
rpc PreviewPlan (PlanPreviewRequest) returns (PlanPreviewResponse);
```

**Request:**
```protobuf
message PlanPreviewRequest {
  string session_id = 1;
  string text = 2;           // Job description
  int32 max_workers = 3;     // 0 = all devices
}
```

**Response:**
```protobuf
message PlanPreviewResponse {
  bool used_ai = 1;          // Whether AI was used for plan generation
  string notes = 2;          // AI availability notes
  string rationale = 3;      // Explanation of plan generation logic
  Plan plan = 4;             // The generated execution plan
  ReduceSpec reduce = 5;     // How results would be combined
}
```

When the Windows AI Brain is available, the plan may be AI-generated. Otherwise, a deterministic fallback plan is returned (1 SYSINFO task per device).

### Worker Execution

#### RunTask
Executes a task on a worker device (called by coordinator).

```protobuf
rpc RunTask (TaskRequest) returns (TaskResult);
```

**Request:**
```protobuf
message TaskRequest {
  string task_id = 1;
  string job_id = 2;
  string kind = 3;               // "SYSINFO" or "ECHO"
  string input = 4;
}
```

**Response:**
```protobuf
message TaskResult {
  string task_id = 1;
  bool ok = 2;
  string output = 3;
  string error = 4;
  double time_ms = 5;
}
```

### WebRTC Screen Streaming

#### StartWebRTC
Creates a WebRTC peer connection and returns an offer SDP for screen streaming.

```protobuf
rpc StartWebRTC (WebRTCConfig) returns (WebRTCOffer);
```

**Request:**
```protobuf
message WebRTCConfig {
  string session_id = 1;
  int32 target_fps = 2;      // Default 8 if 0
  int32 jpeg_quality = 3;    // Default 60 if 0
  int32 monitor_index = 4;   // Default 0
}
```

**Response:**
```protobuf
message WebRTCOffer {
  string stream_id = 1;
  string sdp = 2;            // Offer SDP with ICE candidates (non-trickle)
}
```

#### CompleteWebRTC
Sets the remote description (answer SDP) to complete the WebRTC handshake.

```protobuf
rpc CompleteWebRTC (WebRTCAnswer) returns (Empty);
```

#### StopWebRTC
Closes an active stream and cleans up resources.

```protobuf
rpc StopWebRTC (WebRTCStop) returns (Empty);
```

### Health Check

#### HealthCheck
Returns server health status.

```protobuf
rpc HealthCheck (Empty) returns (HealthStatus);
```

## Regenerating Proto

```bash
make proto
# or
protoc --go_out=. --go-grpc_out=. proto/orchestrator.proto
```

## Default Port

- Server listens on `:50051` by default
- Override with `GRPC_ADDR` environment variable

## Example Usage

```go
// Connect to server
conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
client := pb.NewOrchestratorServiceClient(conn)

// Create session
session, _ := client.CreateSession(ctx, &pb.AuthRequest{
    DeviceName:  "my-client",
    SecurityKey: "dev-key",
})

// Execute routed command
resp, _ := client.ExecuteRoutedCommand(ctx, &pb.RoutedCommandRequest{
    SessionId: session.SessionId,
    Policy:    &pb.RoutingPolicy{Mode: pb.RoutingPolicy_PREFER_REMOTE},
    Command:   "pwd",
})
```
