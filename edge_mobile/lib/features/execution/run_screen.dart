import 'package:flutter/material.dart';
import 'package:lucide_icons/lucide_icons.dart';
import 'package:go_router/go_router.dart';
import '../../theme/app_colors.dart';
import '../../shared/widgets/edge_mesh_wordmark.dart';
import '../../shared/widgets/terminal_panel.dart';
import '../../shared/widgets/status_strip.dart';
import '../../shared/widgets/glass_container.dart';
import '../../shared/widgets/three_d_badge_icon.dart';
import '../../services/grpc_service.dart';

class RunScreen extends StatefulWidget {
  const RunScreen({super.key});

  @override
  State<RunScreen> createState() => _RunScreenState();
}

class _RunScreenState extends State<RunScreen> {
  final _grpcService = GrpcService();
  final _commandController = TextEditingController();
  List<Map<String, dynamic>> _devices = [];
  Map<String, dynamic>? _selectedDevice;
  String? _output;
  int? _exitCode;
  bool _isExecuting = false;
  bool _devicesLoaded = false;

  @override
  void initState() {
    super.initState();
    _loadDevices();
  }

  @override
  void dispose() {
    _commandController.dispose();
    super.dispose();
  }

  Future<void> _loadDevices() async {
    try {
      final devices = await _grpcService.listDevices();
      if (mounted) {
        setState(() {
          _devices = devices;
          _devicesLoaded = true;
          if (_selectedDevice == null && devices.isNotEmpty) {
            _selectedDevice = devices.first;
          }
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _devices = [];
          _devicesLoaded = true;
        });
      }
    }
  }

  Future<void> _executeCommand() async {
    final cmd = _commandController.text.trim();
    if (cmd.isEmpty) return;
    setState(() {
      _isExecuting = true;
      _output = null;
      _exitCode = null;
    });
    try {
      final parts = cmd.split(RegExp(r'\s+'));
      final command = parts.isNotEmpty ? parts.first : cmd;
      final args = parts.length > 1 ? parts.sublist(1) : <String>[];
      final result = await _grpcService.executeRoutedCommand(
        command: command,
        args: args,
        policy: 'BEST_AVAILABLE',
      );
      if (mounted) {
        setState(() {
          _output = result['stdout']?.toString() ?? result['output']?.toString() ?? result.toString();
          _exitCode = result['exit_code'] as int? ?? 0;
          _isExecuting = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _output = 'Error: $e';
          _exitCode = -1;
          _isExecuting = false;
        });
      }
    }
  }

  void _showDevicePicker() {
    if (_devices.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('No devices available. Ensure orchestrator is running.')),
      );
      return;
    }
    showModalBottomSheet(
      context: context,
      backgroundColor: AppColors.surface2,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => ListView.builder(
        shrinkWrap: true,
        itemCount: _devices.length,
        itemBuilder: (_, i) {
          final d = _devices[i];
          final isSelected = _selectedDevice?['device_id'] == d['device_id'];
          return ListTile(
            leading: Icon(
              (d['platform']?.toString().toLowerCase().contains('android') ?? false) ? LucideIcons.smartphone : LucideIcons.laptop,
              color: isSelected ? AppColors.safeGreen : AppColors.textSecondary,
            ),
            title: Text(d['device_name']?.toString() ?? 'Unknown'),
            subtitle: Text('${d['platform']} ${d['arch']}'),
            trailing: isSelected ? const Icon(LucideIcons.check, color: AppColors.safeGreen) : null,
            onTap: () {
              setState(() => _selectedDevice = d);
              Navigator.pop(ctx);
            },
          );
        },
      ),
    );
  }

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
                
                Expanded(
                  child: TabBarView(
                    children: [
                      _CommandTab(
                        commandController: _commandController,
                        selectedDevice: _selectedDevice,
                        output: _output,
                        exitCode: _exitCode,
                        isExecuting: _isExecuting,
                        onSelectDevice: _showDevicePicker,
                        onExecute: _executeCommand,
                      ),
                      const _ToolsTab(),
                      _AssistantTab(onAsk: () => context.go('/chat')),
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
  final TextEditingController commandController;
  final Map<String, dynamic>? selectedDevice;
  final String? output;
  final int? exitCode;
  final bool isExecuting;
  final VoidCallback onSelectDevice;
  final VoidCallback onExecute;

  const _CommandTab({
    required this.commandController,
    required this.selectedDevice,
    required this.output,
    required this.exitCode,
    required this.isExecuting,
    required this.onSelectDevice,
    required this.onExecute,
  });

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
              subtitle: Text(
                selectedDevice != null
                    ? '${selectedDevice!['device_name']} • ${selectedDevice!['platform']}'
                    : 'Select a device...',
                style: const TextStyle(fontSize: 12),
              ),
              trailing: const Icon(LucideIcons.chevronDown, size: 16),
              onTap: onSelectDevice,
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
              controller: commandController,
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
            onPressed: isExecuting ? null : onExecute,
            icon: isExecuting ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white)) : const Icon(LucideIcons.zap, size: 16),
            label: Text(isExecuting ? 'EXECUTING...' : 'EXECUTE RUNTIME'),
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
          if (output != null)
            TerminalPanel(output: output!, exitCode: exitCode)
          else
            const TerminalPanel(
              output: 'Output will appear here after execution.',
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
  final VoidCallback onAsk;

  const _AssistantTab({required this.onAsk});

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
                const SizedBox(height: 16),
                FilledButton.icon(
                  onPressed: onAsk,
                  icon: const Icon(LucideIcons.messageSquare, size: 16),
                  label: const Text('Open Chat Assistant'),
                  style: FilledButton.styleFrom(
                    backgroundColor: AppColors.primaryRed,
                    foregroundColor: Colors.white,
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
