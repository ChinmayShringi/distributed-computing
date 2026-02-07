import 'package:flutter/material.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../theme/app_colors.dart';
import '../../shared/widgets/edge_mesh_wordmark.dart';
import '../../shared/widgets/status_strip.dart';
import '../../shared/widgets/glass_container.dart';
import '../../services/grpc_service.dart';

class StreamScreen extends StatefulWidget {
  const StreamScreen({super.key});

  @override
  State<StreamScreen> createState() => _StreamScreenState();
}

class _StreamScreenState extends State<StreamScreen> {
  final _grpcService = GrpcService();
  List<Map<String, dynamic>> _devices = [];
  Map<String, dynamic>? _selectedDevice;
  bool _isLoading = true;
  bool _isStreaming = false;

  @override
  void initState() {
    super.initState();
    _loadDevices();
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

  void _initializeStream() {
    if (_selectedDevice == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Select a source device first')),
      );
      return;
    }
    setState(() => _isStreaming = true);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text('Stream from ${_selectedDevice!['device_name']} - Start stream from the orchestrator on your laptop to view.'),
        backgroundColor: AppColors.infoBlue,
      ),
    );
    setState(() => _isStreaming = false);
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
          child: Column(
            children: [
               Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    const EdgeMeshWordmark(fontSize: 20),
                    const Icon(LucideIcons.activity, color: AppColors.primaryRed, size: 20),
                  ],
                ),
              ),
              const StatusStrip(
                isConnected: true,
                serverAddress: '192.168.1.10:50051',
                isDangerous: false,
              ),
              // Stream Viewport
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: GlassContainer(
                    padding: EdgeInsets.zero,
                    borderRadius: 20,
                    child: Stack(
                      children: [
                        // Simulated Feed
                        Center(
                          child: Column(
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              Icon(LucideIcons.monitorOff, size: 48, color: Colors.white.withOpacity(0.1)),
                              const SizedBox(height: 16),
                              Text(
                                'NO ACTIVE ENCRYPTED FEED',
                                style: TextStyle(
                                  color: Colors.white.withOpacity(0.2),
                                  fontSize: 10,
                                  fontWeight: FontWeight.bold,
                                  letterSpacing: 2,
                                ),
                              ),
                            ],
                          ),
                        ),
                        
                        // Status Overlay
                        Positioned(
                          top: 16,
                          left: 16,
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              _OverlayPill(label: 'ENCRYPTED', color: AppColors.primaryRed, icon: LucideIcons.lock),
                              const SizedBox(height: 8),
                              _OverlayPill(label: '4K • 60FPS', color: Colors.white.withOpacity(0.5), icon: LucideIcons.activity),
                            ],
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),

              // Controls
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
                child: GlassContainer(
                  child: Column(
                    children: [
                      ListTile(
                        contentPadding: EdgeInsets.zero,
                        leading: const Icon(LucideIcons.laptop, color: AppColors.primaryRed),
                        title: const Text('Source Node', style: TextStyle(fontWeight: FontWeight.bold)),
                        subtitle: const Text('Node-Alpha-7 (Samsung S24)'),
                        trailing: const Icon(LucideIcons.chevronDown, size: 16),
                        onTap: () {},
                      ),
                      const Divider(color: Colors.white10),
                      const SizedBox(height: 16),
                      FilledButton.icon(
                        onPressed: () {},
                        icon: _isStreaming ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white)) : const Icon(LucideIcons.play, size: 16),
                        label: Text(_isStreaming ? 'INITIALIZING...' : 'INITIALIZE STREAM'),
                        style: FilledButton.styleFrom(
                          backgroundColor: AppColors.primaryRed,
                          foregroundColor: Colors.white,
                          minimumSize: const Size(double.infinity, 54),
                          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
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
        color: Colors.black.withOpacity(0.6),
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
