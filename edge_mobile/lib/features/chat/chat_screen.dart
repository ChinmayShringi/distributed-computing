import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:lucide_icons/lucide_icons.dart';
import 'package:flutter_animate/flutter_animate.dart';
import '../../theme/app_colors.dart';
import '../../shared/widgets/edge_mesh_logo.dart';
import 'chat_controller.dart';
import 'models.dart';
import 'widgets/chat_cards.dart';

class ChatScreen extends ConsumerStatefulWidget {
  const ChatScreen({super.key});

  @override
  ConsumerState<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends ConsumerState<ChatScreen> {
  final TextEditingController _messageController = TextEditingController();
  final ScrollController _scrollController = ScrollController();

  void _sendMessage() {
    final text = _messageController.text.trim();
    if (text.isEmpty) return;

    ChatController(ref).handleUserMessage(text);
    _messageController.clear();
    
    // Auto scroll to bottom
    Future.delayed(const Duration(milliseconds: 100), () {
      if (_scrollController.hasClients) {
        _scrollController.animateTo(
          _scrollController.position.maxScrollExtent,
          duration: const Duration(milliseconds: 300),
          curve: Curves.easeOut,
        );
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final messagesNotifier = ref.watch(chatMessagesProvider);
    final thinkingNotifier = ref.watch(chatThinkingProvider);

    return Column(
      children: [
        Expanded(
          child:             ValueListenableBuilder<List<ChatMessage>>(
            valueListenable: messagesNotifier,
            builder: (context, messages, child) {
              if (messages.isEmpty) {
                return _ChatBody(ref: ref);
              }
              return ValueListenableBuilder<bool>(
                valueListenable: thinkingNotifier,
                builder: (context, isThinking, child) {
                  return ListView.builder(
                    controller: _scrollController,
                    padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
                    itemCount: messages.length + (isThinking ? 1 : 0),
                    itemBuilder: (context, index) {
                      if (index == messages.length) {
                        return _ThinkingIndicator().animate().fadeIn();
                      }
                      return _MessageItem(message: messages[index]);
                    },
                  );
                },
              );
            },
          ),
        ),
        _ComposerBar(
          controller: _messageController,
          onSend: _sendMessage,
          onOpenActions: () {
            showModalBottomSheet(
              context: context,
              backgroundColor: const Color(0xFF0F1623),
              shape: const RoundedRectangleBorder(
                borderRadius: BorderRadius.vertical(top: Radius.circular(22)),
              ),
              builder: (ctx) => Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    _ActionSheetItem(
                      icon: LucideIcons.server,
                      label: 'List Devices',
                      onTap: () {
                        Navigator.pop(ctx);
                        ChatController(ref).handleUserMessage('Show my devices');
                      },
                    ),
                    _ActionSheetItem(
                      icon: LucideIcons.terminal,
                      label: 'Run Command',
                      onTap: () {
                        Navigator.pop(ctx);
                        ChatController(ref).handleUserMessage('Run ls -la');
                      },
                    ),
                    _ActionSheetItem(
                      icon: LucideIcons.playCircle,
                      label: 'Start Stream',
                      onTap: () {
                        Navigator.pop(ctx);
                        ChatController(ref).handleUserMessage('Start stream');
                      },
                    ),
                    _ActionSheetItem(
                      icon: LucideIcons.download,
                      label: 'Download File',
                      onTap: () {
                        Navigator.pop(ctx);
                        ChatController(ref).handleUserMessage('Download shared file');
                      },
                    ),
                  ],
                ),
              ),
            );
          },
          ref: ref,
        ),
      ],
    );
  }
}

class _ChatBody extends StatelessWidget {
  final WidgetRef ref;
  const _ChatBody({required this.ref});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 28),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const EdgeMeshLogo(size: 68),
            const SizedBox(height: 24),
            Text(
              'Hello! I am EDGE MESH.',
              textAlign: TextAlign.center,
              style: GoogleFonts.inter(
                color: const Color(0xFFE6EDF6),
                fontSize: 22,
                fontWeight: FontWeight.w700,
                letterSpacing: -0.5,
              ),
            ),
            const SizedBox(height: 10),
            Text(
              'Ask me to run commands, check devices, or manage your network.',
              textAlign: TextAlign.center,
              style: GoogleFonts.inter(
                color: const Color(0xFFA7B1C2),
                fontSize: 15,
                height: 1.4,
                fontWeight: FontWeight.w500,
              ),
            ),
            const SizedBox(height: 32),
            Wrap(
              spacing: 12,
              runSpacing: 12,
              alignment: WrapAlignment.center,
              children: [
                _SuggestionChip('Show my devices', onTap: () => ChatController(ref).handleUserMessage('Show my devices')),
                _SuggestionChip('Run status check', onTap: () => ChatController(ref).handleUserMessage('Run status check')),
                _SuggestionChip('Start stream on laptop', onTap: () => ChatController(ref).handleUserMessage('Start stream')),
                _SuggestionChip('Download shared/report.txt', onTap: () => ChatController(ref).handleUserMessage('Download shared/report.txt')),
              ],
            ),
          ],
        ),
      ),
    ).animate().fadeIn(duration: 600.ms).scale(begin: const Offset(0.98, 0.98), curve: Curves.easeOut);
  }
}

class _SuggestionChip extends StatelessWidget {
  final String text;
  final VoidCallback onTap;
  const _SuggestionChip(this.text, {required this.onTap});

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(999),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: const Color(0xFF121B2B).withOpacity(0.45),
          borderRadius: BorderRadius.circular(999),
          border: Border.all(color: const Color(0xFF233043).withOpacity(0.9)),
        ),
        child: Text(text, style: const TextStyle(color: Color(0xFFE6EDF6), fontSize: 13)),
      ),
    );
  }
}

class _ComposerBar extends StatelessWidget {
  final TextEditingController controller;
  final VoidCallback onSend;
  final VoidCallback onOpenActions;
  final WidgetRef ref;

  const _ComposerBar({
    required this.controller,
    required this.onSend,
    required this.onOpenActions,
    required this.ref,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 10, 12, 14),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.end,
        children: [
          _CircleIconBtn(
            icon: Icons.add_rounded,
            onTap: onOpenActions,
          ),
          const SizedBox(width: 10),

          Expanded(
            child: Container(
              constraints: const BoxConstraints(minHeight: 46),
              padding: const EdgeInsets.symmetric(horizontal: 14),
              decoration: BoxDecoration(
                color: const Color(0xFF121B2B).withOpacity(0.55),
                borderRadius: BorderRadius.circular(24),
                border: Border.all(color: const Color(0xFF233043).withOpacity(0.9)),
              ),
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: controller,
                      maxLines: 5,
                      minLines: 1,
                      style: const TextStyle(color: Color(0xFFE6EDF6), fontSize: 14),
                      decoration: const InputDecoration(
                        hintText: 'Message EDGE MESH…',
                        hintStyle: TextStyle(color: Color(0xFF7C879A), fontSize: 14),
                        border: InputBorder.none,
                        contentPadding: EdgeInsets.symmetric(vertical: 12),
                      ),
                    ),
                  ),
                  Icon(Icons.mic_none_rounded, color: const Color(0xFFA7B1C2), size: 20),
                ],
              ),
            ),
          ),

          const SizedBox(width: 10),

          // Send button like screenshot
          Container(
            width: 46,
            height: 46,
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(999),
              color: const Color(0xFFE6EDF6),
            ),
            child: IconButton(
              icon: const Icon(Icons.arrow_upward_rounded, color: Color(0xFF0A0D12), size: 22),
              onPressed: onSend,
            ),
          ),
        ],
      ),
    );
  }
}

class _CircleIconBtn extends StatelessWidget {
  final IconData icon;
  final VoidCallback onTap;
  const _CircleIconBtn({required this.icon, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return InkWell(
      borderRadius: BorderRadius.circular(999),
      onTap: onTap,
      child: Container(
        width: 46,
        height: 46,
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          color: const Color(0xFF121B2B).withOpacity(0.55),
          border: Border.all(color: const Color(0xFF233043).withOpacity(0.9)),
        ),
        child: Icon(icon, color: const Color(0xFFE6EDF6), size: 22),
      ),
    );
  }
}

class _ActionSheetItem extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  const _ActionSheetItem({required this.icon, required this.label, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: Icon(icon, color: const Color(0xFFE6EDF6), size: 20),
      title: Text(label, style: const TextStyle(color: Color(0xFFE6EDF6), fontSize: 16)),
      onTap: onTap,
    );
  }
}

class _MessageItem extends StatelessWidget {
  final ChatMessage message;
  const _MessageItem({required this.message});

  @override
  Widget build(BuildContext context) {
    final isAssistant = message.sender == MessageSender.assistant;

    return Padding(
      padding: const EdgeInsets.only(bottom: 24),
      child: Column(
        crossAxisAlignment: isAssistant ? CrossAxisAlignment.start : CrossAxisAlignment.end,
        children: [
          if (message.type == MessageType.text)
            _TextBubble(
              text: message.text ?? '',
              isAssistant: isAssistant,
            )
          else
            _CardWrapper(
              child: _buildCard(context),
            ),
        ],
      ),
    );
  }

  Widget _buildCard(BuildContext context) {
    switch (message.type) {
      case MessageType.plan:
        return Consumer(builder: (context, ref, child) {
          return PlanCard(
            payload: message.payload!,
            onRun: () => ChatController(ref).executePlan(message.payload!),
            onEdit: () {},
          );
        });
      case MessageType.devices:
        return DeviceListCard(payload: message.payload!);
      case MessageType.result:
        return ResultCard(payload: message.payload!);
      default:
        return const SizedBox();
    }
  }
}

class _TextBubble extends StatelessWidget {
  final String text;
  final bool isAssistant;

  const _TextBubble({required this.text, required this.isAssistant});

  @override
  Widget build(BuildContext context) {
    return Container(
      constraints: BoxConstraints(maxWidth: MediaQuery.of(context).size.width * 0.8),
      padding: isAssistant ? EdgeInsets.zero : const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: isAssistant ? null : BoxDecoration(
        color: AppColors.surface2,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Text(
        text,
        style: GoogleFonts.inter(
          fontSize: 15,
          color: AppColors.textPrimary,
          height: 1.5,
          fontWeight: isAssistant ? FontWeight.w500 : FontWeight.normal,
        ),
      ),
    ).animate().fadeIn(duration: 400.ms).slideX(begin: isAssistant ? -0.05 : 0.05);
  }
}

class _CardWrapper extends StatelessWidget {
  final Widget child;
  const _CardWrapper({required this.child});

  @override
  Widget build(BuildContext context) {
    return Container(
      constraints: BoxConstraints(maxWidth: MediaQuery.of(context).size.width * 0.9),
      child: child,
    ).animate().fadeIn(duration: 600.ms).slideY(begin: 0.1, curve: Curves.easeOutBack);
  }
}

class _ThinkingIndicator extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Container(
          width: 8,
          height: 8,
          decoration: const BoxDecoration(color: Color(0xFFA7B1C2), shape: BoxShape.circle),
        ).animate(onPlay: (controller) => controller.repeat()).scale(
          duration: 600.ms,
          begin: const Offset(0.8, 0.8),
          end: const Offset(1.2, 1.2),
          curve: Curves.easeInOut,
        ).then().scale(begin: const Offset(1.2, 1.2), end: const Offset(0.8, 0.8)),
        const SizedBox(width: 4),
        Container(
          width: 8,
          height: 8,
          decoration: const BoxDecoration(color: Color(0xFFA7B1C2), shape: BoxShape.circle),
        ).animate(onPlay: (controller) => controller.repeat()).scale(
          delay: 200.ms,
          duration: 600.ms,
          begin: const Offset(0.8, 0.8),
          end: const Offset(1.2, 1.2),
          curve: Curves.easeInOut,
        ).then().scale(begin: const Offset(1.2, 1.2), end: const Offset(0.8, 0.8)),
        const SizedBox(width: 4),
        Container(
          width: 8,
          height: 8,
          decoration: const BoxDecoration(color: Color(0xFFA7B1C2), shape: BoxShape.circle),
        ).animate(onPlay: (controller) => controller.repeat()).scale(
          delay: 400.ms,
          duration: 600.ms,
          begin: const Offset(0.8, 0.8),
          end: const Offset(1.2, 1.2),
          curve: Curves.easeInOut,
        ).then().scale(begin: const Offset(1.2, 1.2), end: const Offset(0.8, 0.8)),
      ],
    );
  }
}
