# Mobile App (edge_mobile)

Flutter cross-platform mobile application for the EdgeCLI orchestration system. Provides a rich native UI for device management, job monitoring, command execution, and screen streaming.

## Features

| Feature | Description |
|---------|-------------|
| Dashboard | Real-time device and job statistics with charts |
| Device Registry | View registered devices, capabilities, and status |
| Command Execution | Execute routed commands with policy selection |
| Job Monitoring | Track distributed job progress and results |
| Chat | LLM chat interface (via orchestrator) |
| Screen Streaming | WebRTC-based remote screen viewing |
| File Download | Download files from remote devices |
| Worker Mode | Run as a worker node in the mesh |
| P2P Discovery | UDP broadcast for automatic device discovery |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Flutter (Dart)                          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  lib/                                                │   │
│  │  ├── features/    (UI screens)                       │   │
│  │  ├── services/    (GrpcService)                      │   │
│  │  └── router/      (GoRouter navigation)              │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                 │
│               MethodChannel (Platform Channel)              │
│             'com.example.edge_mobile/grpc'                  │
│                           │                                 │
└───────────────────────────┼─────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   Android (Kotlin)                          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  GrpcMethodChannelHandler                            │   │
│  │    ↓ routes calls                                    │   │
│  │  OrchestratorGrpcClient  ─────→ EdgeCLI Server       │   │
│  │    ↓ (gRPC stub)                (Mac/Windows)        │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  WorkerService (Foreground Service)                  │   │
│  │    ├── OrchestratorServerService (local gRPC server) │   │
│  │    ├── DiscoveryService (UDP broadcast)              │   │
│  │    └── TaskExecutor (command execution)              │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Screens

| Screen | Path | Description |
|--------|------|-------------|
| Connect | `/` | Device discovery, server connection setup |
| Dashboard | `/dashboard` | Stats overview, quick actions |
| Devices | `/devices` | Device list with capabilities |
| Device Detail | `/devices/:id` | Device metrics, status, actions |
| Run | `/run` | Command execution with routing policy |
| Jobs | `/jobs` | Job list and status tracking |
| Chat | `/chat` | LLM conversation interface |
| Settings | `/settings` | App configuration |
| Approvals | `/approvals` | Pending task approvals |
| Download | `/download` | File download management |

## Kotlin Components

### GrpcMethodChannelHandler

Platform channel bridge between Flutter and native gRPC. Routes method calls:

| Method | Purpose |
|--------|---------|
| `listDevices` | List all registered devices |
| `createSession` | Create authenticated session |
| `healthCheck` | Verify server connectivity |
| `getDeviceStatus` | Get device metrics |
| `executeRoutedCommand` | Execute command with routing policy |
| `configureHost` | Update server host/port |
| `closeConnection` | Close gRPC connection |
| `startWorker` | Start worker service |
| `stopWorker` | Stop worker service |
| `isWorkerRunning` | Check worker status |
| `getJob` | Get job status by ID |
| `requestScreenCapture` | Request screen capture permission |

### OrchestratorGrpcClient

gRPC client wrapper that communicates with the EdgeCLI server:

```kotlin
class OrchestratorGrpcClient(
    host: String = "10.0.2.2",  // Android emulator's host
    port: Int = 50051
)
```

Uses generated protobuf stubs from `proto/orchestrator.proto`.

### WorkerService

Android foreground service that enables the device to act as a worker node:

- Runs a local gRPC server (port 50051)
- Broadcasts presence via UDP discovery
- Executes tasks received from the orchestrator
- Shows persistent notification when active

### DiscoveryService

UDP broadcast service for P2P device discovery:

- Broadcasts `ANNOUNCE` messages every 5 seconds
- Listens for peer announcements on UDP port 50052
- Tracks device last-seen times for stale detection
- Sends `LEAVE` message on shutdown

Message format (JSON):
```json
{
  "type": "ANNOUNCE",
  "version": 1,
  "ts": 1707340800000,
  "device": {
    "device_id": "abc123...",
    "device_name": "pixel-7",
    "grpc_addr": "192.168.1.50:50051",
    "platform": "android",
    "arch": "arm64",
    "has_npu": true
  }
}
```

### DeviceInfoCollector

Collects device information including:
- Device ID (persisted)
- Device name (model name)
- Platform/architecture
- Capabilities (CPU, GPU, NPU)
- Network addresses

## Project Structure

```
edge_mobile/
├── lib/
│   ├── main.dart              # App entry point
│   ├── app.dart               # Root widget (EdgeMeshApp)
│   ├── theme/
│   │   ├── app_colors.dart    # Color palette
│   │   └── app_theme.dart     # Theme configuration
│   ├── features/
│   │   ├── auth/              # Connect screen
│   │   ├── dashboard/         # Main dashboard
│   │   ├── devices/           # Device list & details
│   │   ├── execution/         # Command execution (run_screen)
│   │   ├── jobs/              # Job status display
│   │   ├── chat/              # LLM chat interface
│   │   ├── download/          # File downloads
│   │   ├── approvals/         # Task approvals
│   │   └── settings/          # App settings
│   ├── services/
│   │   └── grpc_service.dart  # Platform channel client
│   ├── router/
│   │   └── app_router.dart    # GoRouter navigation
│   └── shared/
│       └── widgets/           # Reusable UI components
│
├── android/app/src/main/kotlin/com/example/edge_mobile/
│   ├── MainActivity.kt              # Flutter activity
│   ├── GrpcMethodChannelHandler.kt  # Platform channel handler
│   ├── OrchestratorGrpcClient.kt    # gRPC client wrapper
│   ├── OrchestratorServerService.kt # Local gRPC server
│   ├── WorkerService.kt             # Background worker service
│   ├── DiscoveryService.kt          # UDP P2P discovery
│   ├── TaskExecutor.kt              # Command execution
│   └── DeviceInfoCollector.kt       # Device info gathering
│
├── ios/                       # iOS implementation (Swift)
├── macos/                     # macOS implementation
├── linux/                     # Linux implementation
├── windows/                   # Windows implementation
└── pubspec.yaml               # Flutter dependencies
```

## Dependencies

| Package | Purpose |
|---------|---------|
| flutter_riverpod | State management |
| go_router | Navigation |
| google_fonts | Typography |
| fl_chart | Charts and graphs |
| flutter_animate | Animations |
| lucide_icons | Icon set |
| uuid | UUID generation |
| flutter_webrtc | WebRTC streaming |

## Device Setup

### Prerequisites

- Android device with USB debugging enabled
- Flutter SDK 3.10+
- Android Studio (for Android builds)
- Xcode (for iOS builds, macOS only)

### Enable Developer Options (Android)

1. Go to **Settings > About Phone > Software Information**
2. Tap **Build Number** 7 times
3. Go to **Settings > Developer Options**
4. Enable **USB Debugging**

### Verify Connection

```bash
# Add Android SDK to PATH
export PATH="$PATH:$HOME/Library/Android/sdk/platform-tools"

# Verify device is connected
adb devices
# Should show: <device_id>    device
```

## Build & Run

```bash
cd edge_mobile

# Install dependencies
flutter pub get

# Run on connected device
flutter run

# Build release APK
flutter build apk --release

# Build iOS (macOS only)
flutter build ios --release
```

### Hot Reload

Press `r` in the terminal during `flutter run` to hot reload changes.

### Debugging

```bash
# Open DevTools
flutter pub global run devtools

# Run with verbose logging
flutter run -v
```

## Connecting to Server

The mobile app connects to the EdgeCLI gRPC server. Configure the server address:

1. **Emulator**: Default `10.0.2.2:50051` (Android emulator's host alias)
2. **Physical device**: Use the Mac/Windows server's LAN IP (e.g., `192.168.1.195:50051`)

The Connect screen allows entering the server address before connecting.

## Worker Mode

Enable worker mode to participate in the distributed mesh:

1. Open the app
2. Connect to an orchestrator
3. Go to Settings
4. Enable "Worker Mode"

The device will:
- Start a local gRPC server
- Broadcast its presence via UDP
- Accept and execute tasks from the orchestrator
- Show a persistent notification while active

## Design

- **Theme**: Premium dark mode with Indigo accents
- **Typography**: Google Fonts
- **Icons**: Lucide icon set
- **Charts**: fl_chart for activity visualization
- **Animations**: flutter_animate for smooth transitions

## Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| Android | Full | Native gRPC via Kotlin |
| iOS | Partial | Swift implementation needed |
| macOS | Stub | Desktop layout planned |
| Linux | Stub | Desktop layout planned |
| Windows | Stub | Desktop layout planned |
| Web | Stub | WebSocket fallback needed |
