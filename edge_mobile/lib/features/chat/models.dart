enum MessageType {
  text,
  plan,
  devices,
  result,
  approval,
  job,
  stream,
  download
}

enum MessageSender {
  user,
  assistant
}

class ChatMessage {
  final String id;
  final MessageType type;
  final MessageSender sender;
  final String? text;
  final Map<String, dynamic>? payload;
  final DateTime createdAt;

  ChatMessage({
    required this.id,
    required this.type,
    required this.sender,
    this.text,
    this.payload,
    DateTime? createdAt,
  }) : createdAt = createdAt ?? DateTime.now();
}
