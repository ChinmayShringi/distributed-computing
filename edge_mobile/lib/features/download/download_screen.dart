import 'package:flutter/material.dart';
import 'package:lucide_icons/lucide_icons.dart';
import 'package:flutter_animate/flutter_animate.dart';
import '../../theme/app_colors.dart';
import '../../shared/widgets/edge_mesh_wordmark.dart';
import '../../shared/widgets/status_strip.dart';
import '../../shared/widgets/glass_container.dart';
import '../../shared/widgets/three_d_badge_icon.dart';
import '../../services/grpc_service.dart';

class DownloadScreen extends StatefulWidget {
  const DownloadScreen({super.key});

  @override
  State<DownloadScreen> createState() => _DownloadScreenState();
}

class _DownloadScreenState extends State<DownloadScreen> {
  final _grpcService = GrpcService();
  final _pathController = TextEditingController(text: '/shared/artifacts/data.pkg');
  List<Map<String, dynamic>> _devices = [];
  Map<String, dynamic>? _selectedDevice;
  double _ttlHours = 2;
  bool _isLoading = true;
  bool _isGenerating = false;

  @override
  void initState() {
    super.initState();
    _loadDevices();
  }

  @override
  void dispose() {
    _pathController.dispose();
    super.dispose();
  }

  Future<void> _loadDevices() async {
    try {
      final devices = await _grpcService.listDevices();
      if (mounted) {
        setState(() {
          _devices = devices;
          _isLoading = false;
          if (_selectedDevice == null && devices.isNotEmpty) {
            _selectedDevice = devices.first;
          }
        });
      }
    } catch (e) {
      if (mounted) setState(() { _devices = []; _isLoading = false; });
    }
  }

  void _showDevicePicker() {
    if (_devices.isEmpty) return;
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
            leading: ThreeDBadgeIcon(
              icon: (d['platform']?.toString().toLowerCase().contains('android') ?? false) ? LucideIcons.smartphone : LucideIcons.laptop,
              accentColor: isSelected ? AppColors.safeGreen : AppColors.primaryRed,
              size: 14,
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

  void _generateTicket() {
    if (_selectedDevice == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Select a source device first')),
      );
      return;
    }
    setState(() => _isGenerating = true);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text('Ticket generated for ${_pathController.text} from ${_selectedDevice!['device_name']} (${_ttlHours.toInt()}h TTL). Use orchestrator CLI for full transfer.'),
        backgroundColor: AppColors.safeGreen,
      ),
    );
    setState(() => _isGenerating = false);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
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
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(16),
            physics: const BouncingScrollPhysics(),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    const EdgeMeshWordmark(fontSize: 20),
                    const Icon(LucideIcons.download, color: AppColors.primaryRed, size: 20),
                  ],
                ),
                const SizedBox(height: 8),
                const StatusStrip(
                  isConnected: true,
                  serverAddress: '192.168.1.10:50051',
                  isDangerous: false,
                ),
                const SizedBox(height: 24),
                const Text(
                  'SECURE FILE TICKET',
                  style: TextStyle(
                    fontSize: 10,
                    fontWeight: FontWeight.bold,
                    color: AppColors.textSecondary,
                    letterSpacing: 1.2,
                  ),
                ),
                const SizedBox(height: 16),
                GlassContainer(
                  child: Column(
                    children: [
                      ListTile(
                        contentPadding: EdgeInsets.zero,
                        leading: const ThreeDBadgeIcon(icon: LucideIcons.laptop, accentColor: AppColors.primaryRed, size: 14),
                        title: const Text('Source Node', style: TextStyle(fontWeight: FontWeight.bold)),
                        subtitle: const Text('Samsung Galaxy S24 (R3CXC)'),
                        trailing: const Icon(LucideIcons.chevronDown, size: 16),
                        onTap: () {},
                      ),
                      const SizedBox(height: 12),
                      GlassContainer(
                        padding: const EdgeInsets.symmetric(horizontal: 16),
                        borderRadius: 12,
                        opacity: 0.05,
                        child: TextField(
                          decoration: InputDecoration(
                            hintText: '/home/mesh/artifacts/data.pkg',
                            hintStyle: TextStyle(color: AppColors.textSecondary.withOpacity(0.3), fontSize: 13),
                            prefixIcon: const Icon(LucideIcons.file, size: 16, color: AppColors.primaryRed),
                            border: InputBorder.none,
                            contentPadding: const EdgeInsets.symmetric(vertical: 14),
                          ),
                        ),
                      ),
                      const SizedBox(height: 24),
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          const Text('Ticket TTL', style: TextStyle(fontSize: 12)),
                          _OverlayPill(label: '2 HOURS', color: AppColors.warningAmber, icon: LucideIcons.clock),
                        ],
                      ),
                      const SizedBox(height: 12),
                      Slider(
                        value: 2,
                        min: 1,
                        max: 24,
                        onChanged: (v) {},
                        activeColor: AppColors.primaryRed,
                        inactiveColor: Colors.white10,
                      ),
                      const SizedBox(height: 24),
                      FilledButton.icon(
                        onPressed: _isGenerating ? null : _generateTicket,
                        icon: const Icon(LucideIcons.key, size: 16),
                        label: const Text('GENERATE ENCRYPTED TICKET'),
                        style: FilledButton.styleFrom(
                          minimumSize: const Size(double.infinity, 54),
                          backgroundColor: AppColors.primaryRed,
                          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                        ),
                      ),
                    ],
                  ),
                ),
                
                const SizedBox(height: 32),
                const Text(
                  'DOWNLOADS IN FLIGHT',
                  style: TextStyle(
                    fontSize: 10,
                    fontWeight: FontWeight.bold,
                    color: AppColors.textSecondary,
                    letterSpacing: 1.2,
                  ),
                ),
                const SizedBox(height: 16),
                _buildHistoryItem('edge_node_v1.0.2.bin', '45 MB', 'In Progress', progress: 0.65),
                _buildHistoryItem('dataset_training_shard_1.zip', '1.2 GB', 'Completed', progress: 1.0),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildHistoryItem(String name, String size, String status, {double progress = 0}) {
    final isDone = progress >= 1.0;
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: GlassContainer(
        padding: const EdgeInsets.all(16),
        child: Column(
          children: [
            Row(
              children: [
                 ThreeDBadgeIcon(
                  icon: isDone ? LucideIcons.fileCheck : LucideIcons.fileClock,
                  accentColor: isDone ? AppColors.safeGreen : AppColors.primaryRed,
                  size: 14,
                  isDanger: !isDone,
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(name, style: const TextStyle(fontSize: 14, fontWeight: FontWeight.bold)),
                      Text('$size • $status', style: const TextStyle(fontSize: 12, color: AppColors.textSecondary)),
                    ],
                  ),
                ),
                if (isDone) const Icon(LucideIcons.share2, size: 16, color: AppColors.textSecondary),
              ],
            ),
            if (!isDone) ...[
              const SizedBox(height: 16),
              ClipRRect(
                borderRadius: BorderRadius.circular(2),
                child: LinearProgressIndicator(
                  value: progress,
                  backgroundColor: Colors.white.withOpacity(0.05),
                  color: AppColors.primaryRed,
                  minHeight: 2,
                ),
              ),
            ],
          ],
        ),
      ),
    ).animate().fadeIn().slideX(begin: -0.1, end: 0);
  }
}

class _OverlayPill extends StatelessWidget {
  final String label;
  final Color color;
  final IconData icon;

  const _OverlayPill({required this.label, required this.color, required this.icon});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: Colors.black.withOpacity(0.4),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: color.withOpacity(0.2)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 12, color: color),
          const SizedBox(width: 8),
          Text(
            label,
            style: TextStyle(color: color, fontSize: 10, fontWeight: FontWeight.bold, letterSpacing: 0.5),
          ),
        ],
      ),
    );
  }
}
