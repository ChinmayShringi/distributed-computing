import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:uuid/uuid.dart';
import 'models.dart';
import '../../services/grpc_service.dart';

// Using basic Provider to hold ValueNotifier/ChangeNotifier
// which is the most compatible way across all Riverpod versions.

final chatMessagesProvider = Provider((ref) => ValueNotifier<List<ChatMessage>>([]));

final chatThinkingProvider = Provider((ref) => ValueNotifier<bool>(false));

class ChatController {
  final WidgetRef ref;
  final _uuid = const Uuid();
  final _grpcService = GrpcService();

  ChatController(this.ref);

  String _deviceType(String? platform) {
    if (platform == null) return 'desktop';
    final p = platform.toLowerCase();
    if (p.contains('android') || p.contains('ios')) return 'mobile';
    if (p.contains('linux')) return 'server';
    return 'desktop';
  }

  Future<void> handleUserMessage(String text) async {
    final userMsg = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.text,
      sender: MessageSender.user,
      text: text,
    );

    final messagesNotifier = ref.read(chatMessagesProvider);
    messagesNotifier.value = [...messagesNotifier.value, userMsg];

    final thinkingNotifier = ref.read(chatThinkingProvider);
    thinkingNotifier.value = true;

    try {
      // Send message to the real assistant API
      final response = await _grpcService.sendAssistantMessage(text);

      // Parse response and determine message type
      final reply = response['reply'] as String? ?? 'No response from assistant';
      final raw = response['raw'];
      final jobId = response['job_id'] as String?;
      final mode = response['mode'] as String?;

      // Determine how to display the response
      if (raw is List && raw.isNotEmpty && mode == 'command' && _isDeviceList(raw)) {
        // Device list response
        _showDeviceListResponse(reply, raw);
      } else if (jobId != null && jobId.isNotEmpty) {
        // Job was submitted - show plan card
        _showJobSubmittedResponse(reply, response);
      } else {
        // Regular text response
        _showTextResponse(reply);
      }
    } catch (e) {
      // Show error message
      _showTextResponse('Sorry, I encountered an error: $e');
    } finally {
      thinkingNotifier.value = false;
    }
  }

  /// Check if raw data looks like a device list
  bool _isDeviceList(List raw) {
    if (raw.isEmpty) return false;
    final first = raw[0];
    if (first is Map) {
      return first.containsKey('device_id') || first.containsKey('device_name');
    }
    return false;
  }

  /// Show a text-only response
  void _showTextResponse(String text) {
    final assistantMsg = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.text,
      sender: MessageSender.assistant,
      text: text,
    );

    final notifier = ref.read(chatMessagesProvider);
    notifier.value = [...notifier.value, assistantMsg];
  }

  /// Show a device list response
  void _showDeviceListResponse(String introText, List devices) {
    final assistantMsg = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.text,
      sender: MessageSender.assistant,
      text: introText.isNotEmpty ? introText : 'Here are the devices connected to your mesh:',
    );

    final transformedDevices = devices.map((d) {
      final device = d as Map;
      return {
        'name': device['device_name'] ?? 'Unknown',
        'type': _deviceType(device['platform']?.toString()),
        'status': 'online',
        'os': '${device['platform'] ?? ''} ${device['arch'] ?? ''}'.trim(),
        'device_id': device['device_id'],
      };
    }).toList();

    final deviceCard = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.devices,
      sender: MessageSender.assistant,
      payload: {'devices': transformedDevices},
    );

    final notifier = ref.read(chatMessagesProvider);
    notifier.value = [...notifier.value, assistantMsg, deviceCard];
  }

  /// Show a job submitted response with plan card
  void _showJobSubmittedResponse(String introText, Map<String, dynamic> response) {
    final jobId = response['job_id'] as String? ?? '';
    final plan = response['plan'];

    final assistantMsg = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.text,
      sender: MessageSender.assistant,
      text: introText.isNotEmpty ? introText : 'I\'ve created a job to handle your request.',
    );

    // Extract plan details if available
    List<String> steps = ['Analyzing request', 'Selecting best device', 'Executing task'];
    String device = 'Auto-selected';
    String policy = 'BEST_AVAILABLE';

    if (plan is Map) {
      if (plan['groups'] is List) {
        steps = (plan['groups'] as List).map((g) {
          if (g is Map && g['tasks'] is List) {
            final tasks = g['tasks'] as List;
            return tasks.map((t) => t['kind']?.toString() ?? 'Task').join(', ');
          }
          return 'Task group';
        }).toList();
      }
    }

    final planCard = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.plan,
      sender: MessageSender.assistant,
      payload: {
        'steps': steps,
        'job_id': jobId,
        'device': device,
        'policy': policy,
        'risk': 'Low',
      },
    );

    final notifier = ref.read(chatMessagesProvider);
    notifier.value = [...notifier.value, assistantMsg, planCard];
  }

  /// Execute a plan (called when user taps "Execute" on a plan card)
  Future<void> executePlan(Map<String, dynamic> payload) async {
    final thinkingNotifier = ref.read(chatThinkingProvider);
    thinkingNotifier.value = true;

    final jobId = payload['job_id'] as String?;

    try {
      if (jobId != null && jobId.isNotEmpty) {
        // Poll for job completion
        Map<String, dynamic>? jobResult;
        for (int i = 0; i < 30; i++) { // Max 30 seconds
          await Future.delayed(const Duration(seconds: 1));
          jobResult = await _grpcService.getJob(jobId);
          final state = jobResult['state'] as String? ?? '';
          if (state == 'DONE' || state == 'FAILED') {
            break;
          }
        }

        if (jobResult != null) {
          final state = jobResult['state'] as String? ?? 'UNKNOWN';
          final result = jobResult['final_result'] as String? ?? '';

          final resultCard = ChatMessage(
            id: _uuid.v4(),
            type: MessageType.result,
            sender: MessageSender.assistant,
            payload: {
              'cmd': payload['cmd'] ?? 'job',
              'device': payload['device'] ?? 'Distributed',
              'host_compute': 'CPU',
              'time': 'completed',
              'exit_code': state == 'DONE' ? 0 : 1,
              'output': result,
            },
          );

          final messagesNotifier = ref.read(chatMessagesProvider);
          messagesNotifier.value = [...messagesNotifier.value, resultCard];
        }
      } else {
        // No job ID, just show mock result
        _showMockExecutionResult(payload);
      }
    } catch (e) {
      _showTextResponse('Error executing plan: $e');
    } finally {
      thinkingNotifier.value = false;
    }
  }

  /// Show mock execution result (fallback when no job ID)
  void _showMockExecutionResult(Map<String, dynamic> payload) {
    final resultCard = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.result,
      sender: MessageSender.assistant,
      payload: {
        'cmd': payload['cmd'] ?? 'command',
        'device': payload['device'] ?? 'Selected Device',
        'host_compute': 'CPU',
        'time': '124ms',
        'exit_code': 0,
        'output': 'Command executed successfully.',
      },
    );

    final messagesNotifier = ref.read(chatMessagesProvider);
    messagesNotifier.value = [...messagesNotifier.value, resultCard];
  }

  /// Legacy method for stream queries (still uses device list from gRPC)
  Future<void> handleStreamQuery() async {
    final thinkingNotifier = ref.read(chatThinkingProvider);
    thinkingNotifier.value = true;

    try {
      final devices = await _grpcService.listDevices();

      final assistantMsg = ChatMessage(
        id: _uuid.v4(),
        type: MessageType.text,
        sender: MessageSender.assistant,
        text: 'Select a device to start streaming:',
      );

      final transformedDevices = devices.map((d) => {
        'name': d['device_name'] ?? 'Unknown',
        'type': _deviceType(d['platform']),
        'status': 'online',
        'os': '${d['platform'] ?? ''} ${d['arch'] ?? ''}',
        'device_id': d['device_id'],
      }).toList();

      final streamCard = ChatMessage(
        id: _uuid.v4(),
        type: MessageType.stream,
        sender: MessageSender.assistant,
        payload: {'devices': transformedDevices},
      );

      final notifier = ref.read(chatMessagesProvider);
      notifier.value = [...notifier.value, assistantMsg, streamCard];
    } catch (e) {
      _showTextResponse('Error fetching devices for streaming: $e');
    } finally {
      thinkingNotifier.value = false;
    }
  }
}
