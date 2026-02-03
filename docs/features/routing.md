# Command Routing

Command routing enables executing commands on the best available device based on capabilities and policies.

## Routing Policies

### BEST_AVAILABLE (Default)
Selects the device with the best compute capabilities.

**Priority:** NPU > GPU > CPU

```bash
curl -X POST http://localhost:8080/api/routed-cmd \
  -H "Content-Type: application/json" \
  -d '{"cmd":"pwd","args":[],"policy":"BEST_AVAILABLE"}'
```

### PREFER_REMOTE
Prefers non-local devices. Useful when you want to offload work to other machines.

```bash
curl -X POST http://localhost:8080/api/routed-cmd \
  -H "Content-Type: application/json" \
  -d '{"cmd":"pwd","args":[],"policy":"PREFER_REMOTE"}'
```

If only the local device is available, falls back to local execution.

### REQUIRE_NPU
Only executes on NPU-capable devices. Fails if no NPU device is registered.

```bash
curl -X POST http://localhost:8080/api/routed-cmd \
  -H "Content-Type: application/json" \
  -d '{"cmd":"pwd","args":[],"policy":"REQUIRE_NPU"}'
```

**Error if no NPU:**
```json
{"error": "no device with NPU capability found"}
```

### FORCE_DEVICE_ID
Targets a specific device by ID.

```bash
curl -X POST http://localhost:8080/api/routed-cmd \
  -H "Content-Type: application/json" \
  -d '{
    "cmd":"pwd",
    "args":[],
    "policy":"FORCE_DEVICE_ID",
    "force_device_id":"f66a8dc8-2a81-4f30-a664-c0727359c7c5"
  }'
```

In the Web UI, select "Force Device ID" from the dropdown and choose the target device.

## Execution Flow

```
1. Client sends RoutedCommandRequest
       ↓
2. Server validates session
       ↓
3. Server selects device based on policy
       ↓
4. If local → Execute directly
   If remote → Forward via gRPC
       ↓
5. Return result with routing metadata
```

## Response Structure

```json
{
  "stdout": "/Users/user\n",
  "stderr": "",
  "exit_code": 0,
  "selected_device_id": "e452458d-...",
  "selected_device_name": "macbook-pro",
  "selected_device_addr": "127.0.0.1:50051",
  "total_time_ms": 12.5,
  "executed_locally": true
}
```

**Fields:**
- `stdout/stderr` - Command output
- `exit_code` - Process exit code
- `selected_device_*` - Which device executed the command
- `total_time_ms` - Total time including network overhead
- `executed_locally` - Whether command ran on the coordinator

## Remote Forwarding

When a command is routed to a remote device:

1. Server dials remote device's gRPC address
2. Creates a session on the remote device
3. Executes command via `ExecuteCommand` RPC
4. Returns result to original caller

```go
func (s *OrchestratorServer) forwardCommand(ctx context.Context, targetAddr string, req *pb.RoutedCommandRequest) (*pb.CommandResponse, error) {
    // Dial remote
    conn, _ := grpc.DialContext(ctx, targetAddr, grpc.WithInsecure())
    client := pb.NewOrchestratorServiceClient(conn)

    // Health check
    client.HealthCheck(ctx, &pb.Empty{})

    // Create session
    session, _ := client.CreateSession(ctx, &pb.AuthRequest{
        DeviceName:  "coordinator-forward",
        SecurityKey: "internal-routing",
    })

    // Execute
    return client.ExecuteCommand(ctx, &pb.CommandRequest{
        SessionId: session.SessionId,
        Command:   req.Command,
        Args:      req.Args,
    })
}
```

## Allowed Commands

Commands are validated against an allowlist before execution:

**Default allowed:** `ls`, `cat`, `pwd`

```go
// internal/allowlist/allowlist.go
var defaultAllowed = []string{"ls", "cat", "pwd"}
```

Attempting to run a disallowed command returns an error:
```json
{"error": "command not allowed: rm"}
```

## Web UI Integration

The Web UI provides a dropdown to select routing policy:

1. **Best Available** - `BEST_AVAILABLE`
2. **Prefer Remote** - `PREFER_REMOTE`
3. **Require NPU** - `REQUIRE_NPU`
4. **Force Device ID** - `FORCE_DEVICE_ID` (shows device dropdown)

When "Force Device ID" is selected, a second dropdown appears populated with registered devices.

## CLI Usage

```bash
# List available devices
go run ./cmd/client list

# Execute with routing policy
go run ./cmd/client exec --policy PREFER_REMOTE pwd
```

## Debugging

Check server logs for routing decisions:

```
[INFO] ExecuteRoutedCommand: selected device=f66a8dc8-... name=windows-pc addr=10.20.38.80:50051 local=false
[DEBUG] forwardCommand: health check ok for 10.20.38.80:50051
[INFO] forwardCommand: command completed on 10.20.38.80:50051 exit_code=0
```
