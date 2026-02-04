# Device Registry

The device registry manages registered devices and provides device selection algorithms for task routing.

## Overview

The registry maintains an in-memory map of devices that have registered with the orchestrator server. Each device entry includes:
- Device metadata (ID, name, platform, arch)
- Capabilities (CPU, GPU, NPU)
- gRPC address for communication
- Last-seen timestamp

## Registering Devices

### Using the CLI Client

```bash
# Register a remote device
go run ./cmd/client register \
  --name "windows-pc" \
  --self-addr "10.20.38.80:50051" \
  --platform "windows" \
  --arch "amd64"
```

### Via gRPC

```go
client.RegisterDevice(ctx, &pb.DeviceInfo{
    DeviceId:   "uuid-here",
    DeviceName: "my-device",
    Platform:   "darwin",
    Arch:       "arm64",
    HasCpu:     true,
    HasGpu:     false,
    HasNpu:     false,
    GrpcAddr:   "192.168.1.100:50051",
})
```

### Self-Registration

The server automatically registers itself on startup:
```go
func (s *OrchestratorServer) registerSelf() {
    selfInfo := s.getSelfDeviceInfo()
    s.registry.Upsert(selfInfo)
}
```

## Device Selection

### Routing Policies

| Policy | Description |
|--------|-------------|
| `BEST_AVAILABLE` | Prefers NPU > GPU > CPU |
| `REQUIRE_NPU` | Only selects NPU-capable devices, fails if none |
| `PREFER_REMOTE` | Selects non-local device if available |
| `FORCE_DEVICE_ID` | Selects specific device by ID |

### Selection Algorithm

```go
func (r *Registry) SelectDevice(policy *pb.RoutingPolicy, selfID string) SelectResult {
    switch policy.Mode {
    case pb.RoutingPolicy_BEST_AVAILABLE:
        // Try NPU first
        if device := r.findWithCapability("npu"); device != nil {
            return SelectResult{Device: device}
        }
        // Then GPU
        if device := r.findWithCapability("gpu"); device != nil {
            return SelectResult{Device: device}
        }
        // Fall back to any device
        return SelectResult{Device: r.anyDevice()}

    case pb.RoutingPolicy_PREFER_REMOTE:
        // Find non-self device
        for _, d := range r.devices {
            if d.Info.DeviceId != selfID {
                return SelectResult{Device: d.Info}
            }
        }
        // Fall back to self
        return SelectResult{Device: r.devices[selfID].Info, ExecutedLocally: true}

    case pb.RoutingPolicy_FORCE_DEVICE_ID:
        if d, ok := r.devices[policy.DeviceId]; ok {
            return SelectResult{Device: d.Info}
        }
        return SelectResult{Error: errors.New("device not found")}
    }
}
```

## Listing Devices

### Via CLI

```bash
go run ./cmd/client list
```

### Via API

```bash
curl http://localhost:8080/api/devices
```

**Response:**
```json
[
  {
    "device_id": "e452458d-a1a4-4461-996c-bad009fe33f7",
    "device_name": "macbook-pro",
    "platform": "darwin",
    "arch": "arm64",
    "capabilities": ["cpu"],
    "grpc_addr": "127.0.0.1:50051",
    "can_screen_capture": true
  },
  {
    "device_id": "f66a8dc8-2a81-4f30-a664-c0727359c7c5",
    "device_name": "windows-pc",
    "platform": "windows",
    "arch": "amd64",
    "capabilities": ["cpu"],
    "grpc_addr": "10.20.38.80:50051",
    "can_screen_capture": true
  }
]
```

## Device Info Structure

```protobuf
message DeviceInfo {
  string device_id = 1;          // Stable UUID (persisted)
  string device_name = 2;        // Human-readable name
  string platform = 3;           // "darwin", "windows", "linux"
  string arch = 4;               // "amd64", "arm64"
  bool has_cpu = 5;              // Always true
  bool has_gpu = 6;              // GPU available
  bool has_npu = 7;              // NPU available
  string grpc_addr = 8;          // Reachable address
  bool can_screen_capture = 9;   // True if device can capture screen
}
```

### Screen Capture Detection

The `can_screen_capture` flag is determined at server startup by performing a test screen capture using `kbinani/screenshot`. This tests whether the device has an active display and can capture frames. The flag is included in the device's self-registration and is used by the web UI to gate the Remote Stream feature.

## Device ID Persistence

Device IDs are persisted to survive restarts:

**Location:** `~/.edgemesh/device_id`

```go
// internal/deviceid/deviceid.go
func GetOrCreate() (string, error) {
    path := filepath.Join(homeDir, ".edgemesh", "device_id")

    // Try to read existing
    if data, err := os.ReadFile(path); err == nil {
        return string(data), nil
    }

    // Generate new
    id := uuid.New().String()
    os.WriteFile(path, []byte(id), 0644)
    return id, nil
}
```

## Internal Implementation

### Registry Structure (`internal/registry/registry.go`)

```go
type Registry struct {
    devices map[string]*DeviceEntry
    mu      sync.RWMutex
}

type DeviceEntry struct {
    Info     *pb.DeviceInfo
    LastSeen time.Time
}
```

### Key Methods

- `Upsert(info *pb.DeviceInfo)` - Add or update device
- `List() []*pb.DeviceInfo` - Get all devices
- `Get(deviceID string) *DeviceEntry` - Get specific device
- `SelectDevice(policy, selfID) SelectResult` - Select device by policy
- `SelectBestDevice() (*pb.DeviceInfo, bool)` - Select best device (NPU > GPU > CPU)
- `GetStatus(deviceID string) *pb.DeviceStatus` - Get device status

## Health Checking

Before routing commands to remote devices, the server performs a health check:

```go
healthResp, err := client.HealthCheck(ctx, &pb.Empty{})
if err != nil {
    log.Printf("[WARN] health check failed for %s", targetAddr)
}
```

Devices that fail health checks may still receive commands (best-effort), but failures are logged.
