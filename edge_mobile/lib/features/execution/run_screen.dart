import 'package:flutter/material.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../theme/app_colors.dart';
import '../../shared/widgets/edge_mesh_wordmark.dart';
import '../../shared/widgets/terminal_panel.dart';
import '../../shared/widgets/status_strip.dart';
import '../../shared/widgets/glass_container.dart';
import '../../shared/widgets/three_d_badge_icon.dart';

class RunScreen extends StatelessWidget {
  const RunScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 3,
      child: Scaffold(
        body: Container(
          decoration: const BoxDecoration(
            color: AppColors.backgroundDark,
            image: DecorationImage(
              image: NetworkImage('https://images.unsplash.com/photo-1550751827-4bd374c3f58b?auto=format&fit=crop&q=80&w=2670&ixlib=rb-4.0.3'),
              fit: BoxFit.cover,
              opacity: 0.03,
            ),
          ),
          child: SafeArea(
            child: Column(
              children: [
                // Custom AppBar
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const EdgeMeshWordmark(fontSize: 20),
                      IconButton(
                        icon: const Icon(LucideIcons.history, size: 20),
                        onPressed: () {},
                      ),
                    ],
                  ),
                ),
                
                const StatusStrip(
                  isConnected: true,
                  serverAddress: '192.168.1.10:50051',
                  isDangerous: false,
                ),

                const TabBar(
                  indicatorColor: AppColors.primaryRed,
                  labelColor: AppColors.primaryRed,
                  unselectedLabelColor: AppColors.textSecondary,
                  dividerColor: Colors.transparent,
                  indicatorSize: TabBarIndicatorSize.label,
                  tabs: [
                    Tab(icon: Icon(LucideIcons.terminal, size: 18), text: 'Script'),
                    Tab(icon: Icon(LucideIcons.hammer, size: 18), text: 'Tools'),
                    Tab(icon: Icon(LucideIcons.bot, size: 18), text: 'AI Hub'),
                  ],
                ),
                
                const Expanded(
                  child: TabBarView(
                    children: [
                      _CommandTab(),
                      _ToolsTab(),
                      _AssistantTab(),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _CommandTab extends StatelessWidget {
  const _CommandTab();

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      physics: const BouncingScrollPhysics(),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Target Selector
          GlassContainer(
            padding: EdgeInsets.zero,
            child: ListTile(
              leading: const ThreeDBadgeIcon(icon: LucideIcons.laptop, accentColor: AppColors.primaryRed, size: 14),
              title: const Text('Target Device', style: TextStyle(fontSize: 14, fontWeight: FontWeight.bold)),
              subtitle: const Text('Samsung Galaxy S24 • Online', style: TextStyle(fontSize: 12)),
              trailing: const Icon(LucideIcons.chevronDown, size: 16),
              onTap: () {},
            ),
          ),
          const SizedBox(height: 16),
          
          // Execution Plan Preview
          GlassContainer(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text(
                  'EXECUTION PLAN',
                  style: TextStyle(
                    fontSize: 10,
                    fontWeight: FontWeight.bold,
                    color: AppColors.textSecondary,
                    letterSpacing: 1.2,
                  ),
                ),
                const SizedBox(height: 16),
                _buildStep(1, 'Identify target node fingerprint', true),
                _buildStep(2, 'Validate runtime permissions', true),
                _buildStep(3, 'Inject execution payload', false, isCurrent: true),
              ],
            ),
          ),
          
          const SizedBox(height: 24),
          
          // Terminal Input
          GlassContainer(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            borderRadius: 12,
            child: TextField(
              style: const TextStyle(fontFamily: 'JetBrains Mono', fontSize: 13),
              decoration: InputDecoration(
                hintText: 'shell@edge:~\$ enter command...',
                hintStyle: TextStyle(color: AppColors.textSecondary.withOpacity(0.5)),
                border: InputBorder.none,
                prefixIcon: const Icon(LucideIcons.chevronRight, color: AppColors.primaryRed, size: 16),
                contentPadding: const EdgeInsets.symmetric(vertical: 14),
              ),
            ),
          ),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: () {},
            icon: const Icon(LucideIcons.zap, size: 16),
            label: const Text('EXECUTE RUNTIME'),
            style: FilledButton.styleFrom(
              backgroundColor: AppColors.primaryRed,
              foregroundColor: Colors.white,
              minimumSize: const Size(double.infinity, 54),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
              elevation: 4,
              shadowColor: AppColors.primaryRed.withOpacity(0.5),
            ),
          ),
          
          const SizedBox(height: 24),
          
          // Result
          const TerminalPanel(
            output: 'STDOUT: Listening for packets on eth0...\nSYSTEM: Permission node verified.',
            exitCode: null,
          ),
        ],
      ),
    );
  }

  Widget _buildStep(int result, String label, bool isDone, {bool isCurrent = false}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Row(
        children: [
          Container(
            width: 20,
            height: 20,
            decoration: BoxDecoration(
              color: isDone 
                  ? AppColors.primaryRed 
                  : (isCurrent ? AppColors.infoBlue : AppColors.surfaceVariant.withOpacity(0.5)),
              shape: BoxShape.circle,
            ),
            child: isDone 
                ? const Icon(LucideIcons.check, size: 12, color: Colors.white)
                : (isCurrent 
                    ? const Center(child: SizedBox(width: 8, height: 8, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white)))
                    : Center(child: Text('$result', style: const TextStyle(fontSize: 10, color: AppColors.textSecondary)))),
          ),
          const SizedBox(width: 12),
          Text(
            label,
            style: TextStyle(
              fontSize: 13,
              color: isDone || isCurrent ? AppColors.textPrimary : AppColors.textSecondary,
              decoration: isDone ? TextDecoration.lineThrough : null,
              decorationColor: AppColors.textSecondary,
            ),
          ),
        ],
      ),
    );
  }
}

class _ToolsTab extends StatelessWidget {
  const _ToolsTab();

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      physics: const BouncingScrollPhysics(),
      children: [
        _ToolCard(
          name: 'Network Mapper',
          description: 'Identify all sibling devices',
          isDangerous: false,
        ),
        const SizedBox(height: 12),
        _ToolCard(
          name: 'Kernel Debug',
          description: 'Direct memory injection',
          isDangerous: true,
        ),
        const SizedBox(height: 12),
        _ToolCard(
          name: 'Remote Wipe',
          description: 'Sanitize target node storage',
          isDangerous: true,
        ),
      ],
    );
  }
}

class _ToolCard extends StatelessWidget {
  final String name;
  final String description;
  final bool isDangerous;

  const _ToolCard({
    required this.name,
    required this.description,
    required this.isDangerous,
  });

  @override
  Widget build(BuildContext context) {
    return GlassContainer(
      padding: EdgeInsets.zero,
      child: ListTile(
        leading: ThreeDBadgeIcon(
          icon: isDangerous ? LucideIcons.skull : LucideIcons.package,
          accentColor: isDangerous ? AppColors.primaryRed : AppColors.infoBlue,
          size: 14,
          isDanger: isDangerous,
        ),
        title: Text(name, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 14)),
        subtitle: Text(description, style: const TextStyle(fontSize: 12, color: AppColors.textSecondary)),
        trailing: const Icon(LucideIcons.chevronRight, size: 16, color: AppColors.textSecondary),
      ),
    );
  }
}

class _AssistantTab extends StatelessWidget {
  const _AssistantTab();

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Expanded(
          child: Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const ThreeDBadgeIcon(
                  icon: LucideIcons.bot,
                  accentColor: AppColors.primaryRed,
                  size: 32,
                  useRotation: true,
                ),
                const SizedBox(height: 24),
                const Text(
                  'How can I help you orchestrate today?',
                  style: TextStyle(color: AppColors.textSecondary, fontSize: 14),
                ),
              ],
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.all(16),
          child: GlassContainer(
            borderRadius: 30,
            padding: const EdgeInsets.fromLTRB(20, 4, 8, 4),
            child: Row(
              children: [
                const Expanded(
                  child: TextField(
                    decoration: InputDecoration(
                      hintText: 'Ask the Mesh Assistant...',
                      hintStyle: TextStyle(fontSize: 14, color: AppColors.textSecondary),
                      border: InputBorder.none,
                    ),
                  ),
                ),
                CircleAvatar(
                  backgroundColor: AppColors.primaryRed,
                  radius: 18,
                  child: IconButton(
                    icon: const Icon(LucideIcons.send, color: Colors.white, size: 14),
                    onPressed: () {},
                  ),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}
