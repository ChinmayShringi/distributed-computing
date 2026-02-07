# EdgeMobile

Flutter mobile app for the EdgeCLI distributed orchestration system. Provides a native mobile interface for device management, job monitoring, and command execution.

## Features

- **Dashboard** - Real-time device and job statistics
- **Device Registry** - View devices, capabilities, and status
- **Command Execution** - Run commands with routing policies
- **Job Monitoring** - Track distributed job progress
- **Chat** - LLM conversation interface
- **Screen Streaming** - WebRTC remote screen viewing
- **Worker Mode** - Participate as a mesh worker node

## Prerequisites

- Flutter SDK 3.10+
- Android Studio (for Android)
- Xcode (for iOS, macOS only)
- ADB for physical device debugging

## Quick Start

```bash
# Install dependencies
flutter pub get

# Run on connected device
flutter run

# Build release APK
flutter build apk --release
```

## Device Setup (Android)

1. Enable **Developer Options** (tap Build Number 7 times)
2. Enable **USB Debugging** in Developer Options
3. Connect device and authorize debugging
4. Verify: `adb devices`

## Connecting to Server

The app connects to an EdgeCLI gRPC server:

- **Emulator**: `10.0.2.2:50051` (default)
- **Physical device**: Use server's LAN IP (e.g., `192.168.1.195:50051`)

Enter the address on the Connect screen.

## Project Structure

```
lib/
├── features/           # UI screens
│   ├── dashboard/      # Main dashboard
│   ├── devices/        # Device list
│   ├── chat/           # LLM chat
│   └── ...
├── services/           # Business logic
│   └── grpc_service.dart
└── router/             # Navigation

android/.../kotlin/     # Native gRPC implementation
├── GrpcMethodChannelHandler.kt
├── OrchestratorGrpcClient.kt
├── WorkerService.kt
└── DiscoveryService.kt
```

## Documentation

Full documentation: [docs/features/mobile.md](../docs/features/mobile.md)

## Platform Support

| Platform | Status |
|----------|--------|
| Android | Full support (native gRPC) |
| iOS | Partial |
| Desktop | Stub |
