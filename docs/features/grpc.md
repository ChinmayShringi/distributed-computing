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
  string http_addr = 10;     // Bulk HTTP server address (e.g., "10.0.0.5:8081")
  double llm_prefill_toks_per_s = 11; // LLM prefill throughput (0 = use default)
  double llm_decode_toks_per_s = 12;  // LLM decode throughput (0 = use default)
  uint64 ram_free_mb = 13;   // Free RAM in MB (0 = unknown)
  bool has_local_model = 14;        // True if Ollama/local LLM is running
  string local_model_name = 15;     // Loaded model name (e.g., "llama3.2:3b")
  string local_chat_endpoint = 16;  // Chat endpoint URL (e.g., "http://localhost:11434")
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
    PREFER_LOCAL_MODEL = 4;  // Prefer device with local LLM model
    REQUIRE_LOCAL_MODEL = 5; // Fail if no local LLM model available
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
  string kind = 2;               // "SYSINFO", "ECHO", "LLM_GENERATE"
  string input = 3;
  string target_device_id = 4;   // Empty = auto-assign
  int32 prompt_tokens = 5;       // For LLM_GENERATE: estimated prompt tokens
  int32 max_output_tokens = 6;   // For LLM_GENERATE: max output tokens
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

#### PreviewPlanCost
Estimates execution cost (latency, memory) for a plan without running it. Returns per-device cost breakdowns and recommends the best device.

```protobuf
rpc PreviewPlanCost (PlanCostRequest) returns (PlanCostResponse);
```

**Request:**
```protobuf
message PlanCostRequest {
  string session_id = 1;
  Plan plan = 2;                      // Plan to estimate
  repeated string device_ids = 3;     // Optional: limit to these devices
}
```

**Response:**
```protobuf
message PlanCostResponse {
  double total_predicted_ms = 1;               // Best device's total latency
  repeated DeviceCostEstimate device_costs = 2;
  string recommended_device_id = 3;
  string recommended_device_name = 4;
  bool has_unknown_costs = 5;                  // True if any step had unknown cost
  string warning = 6;                          // Warning message if applicable
}

message DeviceCostEstimate {
  string device_id = 1;
  string device_name = 2;
  double total_ms = 3;
  repeated StepCostEstimate step_costs = 4;
  uint64 estimated_peak_ram_mb = 5;
  bool ram_sufficient = 6;                     // False if estimated RAM > device free RAM
}

message StepCostEstimate {
  string task_id = 1;
  string kind = 2;
  double predicted_ms = 3;
  double predicted_memory_mb = 4;
  bool unknown_cost = 5;                       // True if step type not recognized
  string notes = 6;                            // e.g., "using default prefill TPS"
}
```

**Cost Estimation Formula:**
- For `LLM_GENERATE` steps: `predicted_ms = (prompt_tokens / prefill_tps + max_output_tokens / decode_tps) * 1000`
- For `SYSINFO`, `ECHO`: ~10ms (local operations)
- For unknown step types: 250ms penalty

**Device Throughput Defaults:**
- Laptop (macos/windows/linux): prefill=300 tps, decode=30 tps
- Phone (android/ios): prefill=120 tps, decode=12 tps

**Execution Model:**
- Tasks within a group execute in parallel: `group_cost = max(task_costs)`
- Groups execute sequentially: `total_cost = sum(group_costs)`

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

#### RunLLMTask
Executes an LLM inference task on the device's local model (Ollama).

```protobuf
rpc RunLLMTask (LLMTaskRequest) returns (LLMTaskResponse);
```

**Request:**
```protobuf
message LLMTaskRequest {
  string prompt = 1;         // User prompt
  string model = 2;          // Optional: specific model name
  int32 max_tokens = 3;      // Optional: max tokens to generate
}
```

**Response:**
```protobuf
message LLMTaskResponse {
  string output = 1;         // Generated text
  string model_used = 2;     // Model that was used
  int64 tokens_generated = 3; // Approximate tokens generated
  string error = 4;          // Error message if failed
}
```

The device must have `has_local_model = true` for this RPC to succeed. Use `PREFER_LOCAL_MODEL` or `REQUIRE_LOCAL_MODEL` routing policies to target devices with local models.

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

### Bulk File Transfer

#### CreateDownloadTicket
Creates a one-time-use download ticket for a file on the device. The ticket can be redeemed via the bulk HTTP server.

```protobuf
rpc CreateDownloadTicket (DownloadTicketRequest) returns (DownloadTicket);
```

**Request:**
```protobuf
message DownloadTicketRequest {
  string path = 1;             // File path on the device
}
```

**Response:**
```protobuf
message DownloadTicket {
  string token = 1;            // One-time-use download token
  string filename = 2;         // Base filename
  int64 size_bytes = 3;        // File size
  int64 expires_unix_ms = 4;   // Token expiration timestamp
}
```

The token is redeemed via HTTP GET on the device's bulk HTTP server: `http://<http_addr>/bulk/download/<token>`

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
