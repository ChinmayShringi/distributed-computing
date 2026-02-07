import 'package:flutter/services.dart';

/// GrpcService provides a Dart interface to communicate with the
/// native Kotlin gRPC client via MethodChannel.
class GrpcService {
  static const MethodChannel _channel = MethodChannel('com.example.edge_mobile/grpc');

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
}
