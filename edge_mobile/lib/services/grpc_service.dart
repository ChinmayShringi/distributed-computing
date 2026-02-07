import 'package:flutter/services.dart';
import 'package:flutter/foundation.dart';

/// Connection status for the orchestrator server
class ConnectionStatus {
  final bool connected;
  final String host;
  final int grpcPort;
  final int httpPort;
  final String deviceName;
  final int discoveredCount;

  ConnectionStatus({
    required this.connected,
    this.host = '',
    this.grpcPort = 50051,
    this.httpPort = 8080,
    this.deviceName = '',
    this.discoveredCount = 0,
  });

  factory ConnectionStatus.fromMap(Map<String, dynamic> map) {
    return ConnectionStatus(
      connected: map['connected'] as bool? ?? false,
      host: map['host'] as String? ?? '',
      grpcPort: map['grpc_port'] as int? ?? 50051,
      httpPort: map['http_port'] as int? ?? 8080,
      deviceName: map['device_name'] as String? ?? '',
      discoveredCount: map['discovered_count'] as int? ?? 0,
    );
  }
}

/// Discovered server information
class DiscoveredServer {
  final String deviceId;
  final String deviceName;
  final String grpcHost;
  final int grpcPort;
  final String httpHost;
  final int httpPort;
  final String platform;
  final bool hasLocalModel;
  final bool isActive;

  DiscoveredServer({
    required this.deviceId,
    required this.deviceName,
    required this.grpcHost,
    required this.grpcPort,
    required this.httpHost,
    required this.httpPort,
    required this.platform,
    this.hasLocalModel = false,
    this.isActive = false,
  });

  factory DiscoveredServer.fromMap(Map<String, dynamic> map) {
    return DiscoveredServer(
      deviceId: map['device_id'] as String? ?? '',
      deviceName: map['device_name'] as String? ?? '',
      grpcHost: map['grpc_host'] as String? ?? '',
      grpcPort: map['grpc_port'] as int? ?? 50051,
      httpHost: map['http_host'] as String? ?? '',
      httpPort: map['http_port'] as int? ?? 8080,
      platform: map['platform'] as String? ?? '',
      hasLocalModel: map['has_local_model'] as bool? ?? false,
      isActive: map['is_active'] as bool? ?? false,
    );
  }
}

/// GrpcService provides a Dart interface to communicate with the
/// native Kotlin gRPC client via MethodChannel.
class GrpcService {
  static const MethodChannel _channel = MethodChannel('com.example.edge_mobile/grpc');

  // Connection status notifier
  final ValueNotifier<ConnectionStatus> connectionStatus = ValueNotifier(
    ConnectionStatus(connected: false),
  );

  // Discovered servers notifier
  final ValueNotifier<List<DiscoveredServer>> discoveredServers = ValueNotifier([]);

  GrpcService() {
    // Set up method call handler for callbacks from native side
    _channel.setMethodCallHandler(_handleMethodCall);
  }

  /// Handle callbacks from native side
  Future<dynamic> _handleMethodCall(MethodCall call) async {
    switch (call.method) {
      case 'onConnectionChanged':
        final args = Map<String, dynamic>.from(call.arguments as Map);
        connectionStatus.value = ConnectionStatus.fromMap(args);
        break;

      case 'onServerDiscovered':
        final args = Map<String, dynamic>.from(call.arguments as Map);
        final server = DiscoveredServer.fromMap(args);
        final current = List<DiscoveredServer>.from(discoveredServers.value);
        // Update or add server
        final index = current.indexWhere((s) => s.deviceId == server.deviceId);
        if (index >= 0) {
          current[index] = server;
        } else {
          current.add(server);
        }
        discoveredServers.value = current;
        break;

      case 'onServerLost':
        final args = Map<String, dynamic>.from(call.arguments as Map);
        final deviceId = args['device_id'] as String?;
        if (deviceId != null) {
          final current = List<DiscoveredServer>.from(discoveredServers.value);
          current.removeWhere((s) => s.deviceId == deviceId);
          discoveredServers.value = current;
        }
        break;
    }
    return null;
  }

  /// Get current connection status
  Future<ConnectionStatus> getConnectionStatus() async {
    try {
      final result = await _channel.invokeMethod('getConnectionStatus');
      final status = ConnectionStatus.fromMap(Map<String, dynamic>.from(result));
      connectionStatus.value = status;
      return status;
    } on PlatformException catch (e) {
      throw Exception('Failed to get connection status: ${e.message}');
    }
  }

  /// Get list of discovered servers
  Future<List<DiscoveredServer>> getDiscoveredServers() async {
    try {
      final result = await _channel.invokeMethod('getDiscoveredServers');
      final servers = (result as List).map((s) {
        return DiscoveredServer.fromMap(Map<String, dynamic>.from(s));
      }).toList();
      discoveredServers.value = servers;
      return servers;
    } on PlatformException catch (e) {
      throw Exception('Failed to get discovered servers: ${e.message}');
    }
  }

  /// Set active server by device ID
  Future<bool> setActiveServer(String deviceId) async {
    try {
      final result = await _channel.invokeMethod('setActiveServer', {
        'device_id': deviceId,
      });
      return result == true;
    } on PlatformException catch (e) {
      throw Exception('Failed to set active server: ${e.message}');
    }
  }

  /// List all registered devices from the orchestrator
  Future<List<Map<String, dynamic>>> listDevices() async {
    try {
      final result = await _channel.invokeMethod('listDevices');
      return List<Map<String, dynamic>>.from(
        (result as List).map((device) => Map<String, dynamic>.from(device))
      );
    } on PlatformException catch (e) {
      throw Exception('Failed to list devices: ${e.message}');
    }
  }

  /// Create a session with the orchestrator
  Future<Map<String, dynamic>> createSession({
    String deviceName = 'android-device',
    String securityKey = 'dev',
  }) async {
    try {
      final result = await _channel.invokeMethod('createSession', {
        'device_name': deviceName,
        'security_key': securityKey,
      });
      return Map<String, dynamic>.from(result);
    } on PlatformException catch (e) {
      throw Exception('Failed to create session: ${e.message}');
    }
  }

  /// Perform a health check on the orchestrator server
  Future<Map<String, dynamic>> healthCheck() async {
    try {
      final result = await _channel.invokeMethod('healthCheck');
      return Map<String, dynamic>.from(result);
    } on PlatformException catch (e) {
      throw Exception('Failed to perform health check: ${e.message}');
    }
  }

  /// Get the status of a specific device
  Future<Map<String, dynamic>> getDeviceStatus(String deviceId) async {
    try {
      final result = await _channel.invokeMethod('getDeviceStatus', {
        'device_id': deviceId,
      });
      return Map<String, dynamic>.from(result);
    } on PlatformException catch (e) {
      throw Exception('Failed to get device status: ${e.message}');
    }
  }

  /// Execute a routed command on the best available device
  Future<Map<String, dynamic>> executeRoutedCommand({
    required String command,
    List<String> args = const [],
    String policy = 'BEST_AVAILABLE',
  }) async {
    try {
      final result = await _channel.invokeMethod('executeRoutedCommand', {
        'command': command,
        'args': args,
        'policy': policy,
      });
      return Map<String, dynamic>.from(result);
    } on PlatformException catch (e) {
      throw Exception('Failed to execute routed command: ${e.message}');
    }
  }

  /// Configure the gRPC server host and port
  /// (Note: Current implementation requires app restart to take effect)
  Future<Map<String, dynamic>> configureHost({
    required String host,
    required int port,
  }) async {
    try {
      final result = await _channel.invokeMethod('configureHost', {
        'host': host,
        'port': port,
      });
      return Map<String, dynamic>.from(result);
    } on PlatformException catch (e) {
      throw Exception('Failed to configure host: ${e.message}');
    }
  }

  /// Close the gRPC connection
  Future<Map<String, dynamic>> closeConnection() async {
    try {
      final result = await _channel.invokeMethod('closeConnection');
      return Map<String, dynamic>.from(result);
    } on PlatformException catch (e) {
      throw Exception('Failed to close connection: ${e.message}');
    }
  }

  /// Start the worker service (enables this device to execute tasks)
  Future<bool> startWorker() async {
    try {
      final result = await _channel.invokeMethod('startWorker');
      return result == true;
    } on PlatformException catch (e) {
      throw Exception('Failed to start worker: ${e.message}');
    }
  }

  /// Stop the worker service
  Future<bool> stopWorker() async {
    try {
      final result = await _channel.invokeMethod('stopWorker');
      return result == true;
    } on PlatformException catch (e) {
      throw Exception('Failed to stop worker: ${e.message}');
    }
  }

  /// Check if worker service is running
  Future<bool> isWorkerRunning() async {
    try {
      final result = await _channel.invokeMethod('isWorkerRunning');
      return result == true;
    } on PlatformException catch (e) {
      return false;
    }
  }

  /// Request screen capture permission (shows system dialog)
  Future<bool> requestScreenCapture() async {
    try {
      final result = await _channel.invokeMethod('requestScreenCapture');
      return result == true;
    } on PlatformException catch (e) {
      throw Exception('Failed to request screen capture: ${e.message}');
    }
  }

  /// Get job status by ID
  Future<Map<String, dynamic>> getJob(String jobId) async {
    try {
      final result = await _channel.invokeMethod('getJob', {
        'job_id': jobId,
      });
      return Map<String, dynamic>.from(result);
    } on PlatformException catch (e) {
      throw Exception('Failed to get job: ${e.message}');
    }
  }

  /// Send a message to the assistant (via REST /api/assistant)
  /// Returns: reply, raw (optional), mode (optional), job_id (optional), plan (optional)
  Future<Map<String, dynamic>> sendAssistantMessage(String text) async {
    try {
      final result = await _channel.invokeMethod('sendAssistantMessage', {
        'text': text,
      });
      return Map<String, dynamic>.from(result as Map);
    } on PlatformException catch (e) {
      throw Exception('Failed to send assistant message: ${e.message}');
    }
  }

  /// Get activity data (running tasks, device activities, optional metrics history)
  /// Returns: running_tasks[], device_activities[], device_metrics (if includeMetrics=true)
  Future<Map<String, dynamic>> getActivity({
    bool includeMetrics = false,
    int metricsSinceMs = 0,
  }) async {
    try {
      final result = await _channel.invokeMethod('getActivity', {
        'include_metrics': includeMetrics,
        'metrics_since_ms': metricsSinceMs,
      });
      return Map<String, dynamic>.from(result as Map);
    } on PlatformException catch (e) {
      throw Exception('Failed to get activity: ${e.message}');
    }
  }

  /// Submit a distributed job across devices
  /// Returns: job_id, created_at, summary
  Future<Map<String, dynamic>> submitJob({
    required String prompt,
    int maxWorkers = 0,
  }) async {
    try {
      final result = await _channel.invokeMethod('submitJob', {
        'prompt': prompt,
        'max_workers': maxWorkers,
      });
      return Map<String, dynamic>.from(result as Map);
    } on PlatformException catch (e) {
      throw Exception('Failed to submit job: ${e.message}');
    }
  }

  /// Get detailed job status with task timing
  /// Returns: job_id, state, tasks[] with start_at/end_at, final_result
  Future<Map<String, dynamic>> getJobDetail(String jobId) async {
    try {
      final result = await _channel.invokeMethod('getJobDetail', {
        'job_id': jobId,
      });
      return Map<String, dynamic>.from(result as Map);
    } on PlatformException catch (e) {
      throw Exception('Failed to get job detail: ${e.message}');
    }
  }

  /// Get metrics history for a specific device
  /// Returns: device_id, metrics[] with cpu_percent, memory_percent, gpu_percent, timestamp
  Future<Map<String, dynamic>> getDeviceMetrics(
    String deviceId, {
    int sinceMs = 0,
  }) async {
    try {
      final result = await _channel.invokeMethod('getDeviceMetrics', {
        'device_id': deviceId,
        'since_ms': sinceMs,
      });
      return Map<String, dynamic>.from(result as Map);
    } on PlatformException catch (e) {
      throw Exception('Failed to get device metrics: ${e.message}');
    }
  }
}
