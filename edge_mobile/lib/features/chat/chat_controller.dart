import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:uuid/uuid.dart';
import 'models.dart';
import '../../data/mock_data.dart';

// Using basic Provider to hold ValueNotifier/ChangeNotifier
// which is the most compatible way across all Riverpod versions.

final chatMessagesProvider = Provider((ref) => ValueNotifier<List<ChatMessage>>([]));

final chatThinkingProvider = Provider((ref) => ValueNotifier<bool>(false));

class ChatController {
  final WidgetRef ref;
  final _uuid = const Uuid();

  ChatController(this.ref);

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

    await Future.delayed(const Duration(milliseconds: 1500));

    final query = text.toLowerCase();
    
    if (query.contains('device')) {
      _handleDeviceQuery();
    } else if (query.contains('run') || query.contains('ls') || query.contains('python')) {
      _handleRunQuery(text);
    } else if (query.contains('stream')) {
      _handleStreamQuery();
    } else if (query.contains('download')) {
      _handleDownloadQuery();
    } else {
      _handleGenericQuery();
    }

    thinkingNotifier.value = false;
  }

  void _handleDeviceQuery() {
    final assistantMsg = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.text,
      sender: MessageSender.assistant,
      text: 'Right away. Here are the devices currently connected to your mesh:',
    );

    final deviceCard = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.devices,
      sender: MessageSender.assistant,
      payload: {'devices': MockData.devices},
    );

    final notifier = ref.read(chatMessagesProvider);
    notifier.value = [...notifier.value, assistantMsg, deviceCard];
  }

  void _handleRunQuery(String text) {
    final assistantMsg = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.text,
      sender: MessageSender.assistant,
      text: 'I understand. I\'ll prepare a plan to execute that command.',
    );

    final planCard = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.plan,
      sender: MessageSender.assistant,
      payload: {
        'steps': [
          'Identify best available device (Samsung Galaxy S24)',
          'Check execution policy (SAFE)',
          'Initialize remote shell',
          'Execute command and stream output'
        ],
        'cmd': text,
        'device': 'Samsung Galaxy S24',
        'policy': 'SAFE',
        'risk': 'Low',
      },
    );

    final notifier = ref.read(chatMessagesProvider);
    notifier.value = [...notifier.value, assistantMsg, planCard];
  }

  void _handleStreamQuery() {
    final assistantMsg = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.text,
      sender: MessageSender.assistant,
      text: 'Sure. Select a device to start streaming its desktop or terminal:',
    );

    final streamCard = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.stream,
      sender: MessageSender.assistant,
      payload: {'devices': MockData.devices},
    );

    final notifier = ref.read(chatMessagesProvider);
    notifier.value = [...notifier.value, assistantMsg, streamCard];
  }

  void _handleDownloadQuery() {
    final assistantMsg = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.text,
      sender: MessageSender.assistant,
      text: 'Accessing shared file system...',
    );

    final downloadCard = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.download,
      sender: MessageSender.assistant,
      payload: {
        'file': 'shared/test.txt',
        'device': 'MacBook Pro M3',
        'size': '2.4 MB',
      },
    );

    final notifier = ref.read(chatMessagesProvider);
    notifier.value = [...notifier.value, assistantMsg, downloadCard];
  }

  void _handleGenericQuery() {
    final assistantMsg = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.text,
      sender: MessageSender.assistant,
      text: 'I\'m not quite sure how to help with that yet. Try asking me to "show devices" or "run a command".',
    );

    final notifier = ref.read(chatMessagesProvider);
    notifier.value = [...notifier.value, assistantMsg];
  }

  void executePlan(Map<String, dynamic> payload) async {
    final thinkingNotifier = ref.read(chatThinkingProvider);
    thinkingNotifier.value = true;
    
    await Future.delayed(const Duration(milliseconds: 2000));
    
    final resultCard = ChatMessage(
      id: _uuid.v4(),
      type: MessageType.result,
      sender: MessageSender.assistant,
      payload: {
        'cmd': payload['cmd'],
        'device': payload['device'],
        'host_compute': 'CPU',
        'time': '124ms',
        'exit_code': 0,
        'output': 'Active Internet connections (w/o servers)\nProto Recv-Q Send-Q Local Address           Foreign Address         State\ntcp        0      0 192.168.1.10:50051      192.168.1.12:44342      ESTABLISHED',
      },
    );

    final messagesNotifier = ref.read(chatMessagesProvider);
    messagesNotifier.value = [...messagesNotifier.value, resultCard];
    thinkingNotifier.value = false;
  }
}
