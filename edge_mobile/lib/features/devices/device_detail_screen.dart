import 'package:flutter/material.dart';
import 'package:lucide_icons/lucide_icons.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:go_router/go_router.dart';
import 'package:google_fonts/google_fonts.dart';
import '../../theme/app_colors.dart';
import '../../shared/widgets/edge_mesh_wordmark.dart';
import '../../shared/widgets/glass_container.dart';
import '../../shared/widgets/capability_chip.dart';
import '../../shared/widgets/three_d_badge_icon.dart';
import '../../services/grpc_service.dart';
import 'dart:async';

class DeviceDetailScreen extends StatefulWidget {
  final String deviceId;

  const DeviceDetailScreen({super.key, required this.deviceId});

  @override
  State<DeviceDetailScreen> createState() => _DeviceDetailScreenState();
}

class _DeviceDetailScreenState extends State<DeviceDetailScreen> {
  final _grpcService = GrpcService();
  Map<String, dynamic>? _device;
  Map<String, dynamic>? _deviceStatus;
  bool _isLoading = true;
  String? _errorMessage;
  Timer? _refreshTimer;

  @override
  void initState() {
    super.initState();
    _loadDeviceData();
    _startPeriodicRefresh();
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    super.dispose();
  }

  void _startPeriodicRefresh() {
    _refreshTimer = Timer.periodic(const Duration(seconds: 5), (_) {
      if (mounted) _loadDeviceStatus();
    });
  }

  Future<void> _loadDeviceData() async {
    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });

    try {
      final devices = await _grpcService.listDevices();
      final device = devices.firstWhere(
        (d) => d['device_id'] == widget.deviceId,
        orElse: () => throw Exception('Device not found'),
      );

      final status = await _grpcService.getDeviceStatus(widget.deviceId);

      if (mounted) {
        setState(() {
          _device = device;
          _deviceStatus = status;
          _isLoading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _isLoading = false;
          _errorMessage = 'Failed to load device: $e';
        });
      }
    }
  }

  String _truncateMeshId(dynamic val) {
    final s = val?.toString() ?? 'N/A';
    return s.length >= 8 ? s.substring(0, 8).toUpperCase() : s.toUpperCase();
  }

  Future<void> _loadDeviceStatus() async {
    if (_device == null) return;

    try {
      final status = await _grpcService.getDeviceStatus(widget.deviceId);
      if (mounted) {
        setState(() {
          _deviceStatus = status;
        });
      }
    } catch (e) {
      // Silent fail for periodic refresh
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_isLoading) {
      return Scaffold(
        backgroundColor: AppColors.backgroundDark,
        body: const Center(
          child: CircularProgressIndicator(color: AppColors.safeGreen),
        ),
      );
    }

    if (_errorMessage != null || _device == null) {
      return Scaffold(
        backgroundColor: AppColors.backgroundDark,
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(32),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Icon(LucideIcons.alertCircle, size: 48, color: AppColors.primaryRed),
                const SizedBox(height: 16),
                Text(
                  _errorMessage ?? 'Device not found',
                  style: GoogleFonts.inter(color: AppColors.textSecondary, fontSize: 14),
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 24),
                ElevatedButton.icon(
                  onPressed: () => context.pop(),
                  icon: const Icon(LucideIcons.arrowLeft, size: 16),
                  label: const Text('GO BACK'),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: AppColors.safeGreen,
                    foregroundColor: Colors.black,
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    }

    final device = _device!;
    final isOnline = _deviceStatus != null;
    
    // Calculate stats from device status
    final cpuLoad = isOnline ? ((_deviceStatus!['cpu_load'] as num).toDouble() * 100).clamp(0, 100) : 0.0;
    final memUsedMb = isOnline ? (_deviceStatus!['mem_used_mb'] as num).toDouble() : 0.0;
    final memTotalMb = isOnline ? (_deviceStatus!['mem_total_mb'] as num).toDouble() : 1.0;
    final memoryPercent = isOnline ? ((memUsedMb / memTotalMb) * 100).clamp(0, 100) : 0.0;

    return Scaffold(
      backgroundColor: AppColors.backgroundDark,
      body: SafeArea(
        child: Column(
          children: [
            // Standardized Header
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              child: Row(
                children: [
                  IconButton(
                    icon: const Icon(LucideIcons.chevronLeft, color: AppColors.mutedIcon, size: 20),
                    onPressed: () => context.pop(),
                  ),
                  const Expanded(child: Center(child: EdgeMeshWordmark(fontSize: 18))),
                  const SizedBox(width: 48), 
                ],
              ),
            ),

            Expanded(
              child: SingleChildScrollView(
                physics: const BouncingScrollPhysics(),
                padding: const EdgeInsets.symmetric(horizontal: 20),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const SizedBox(height: 16),
                    _buildDeviceHeader(device, isOnline),
                    const SizedBox(height: 32),

                    // Remote Viewport (only if device supports screen capture)
                    if (isOnline && device['can_screen_capture'] == true) ...[
                      _SectionHeader(label: 'ENCRYPTED REMOTE VIEWPORT'),
                      const SizedBox(height: 12),
                      _RemoteViewport(device: device),
                      const SizedBox(height: 32),
                    ],

                    // Telemetry
                    _SectionHeader(label: 'REAL-TIME NODE TELEMETRY'),
                    const SizedBox(height: 16),
                    if (isOnline)
                      Row(
                        children: [
                          Expanded(
                            child: _LargeDonut(
                              label: 'CPU LOAD',
                              value: cpuLoad.toDouble(),
                              color: AppColors.safeGreen,
                            ),
                          ),
                          const SizedBox(width: 16),
                          Expanded(
                            child: _LargeDonut(
                              label: 'MEM USAGE',
                              value: memoryPercent.toDouble(),
                              color: AppColors.infoBlue,
                            ),
                          ),
                        ],
                      ),
                    
                    if (!isOnline)
                      Container(
                        width: double.infinity,
                        padding: const EdgeInsets.all(24),
                        decoration: BoxDecoration(
                          color: AppColors.surface2,
                          borderRadius: BorderRadius.circular(16),
                        ),
                        child: Column(
                          children: [
                            const Icon(LucideIcons.cloudOff, size: 32, color: AppColors.mutedIcon),
                            const SizedBox(height: 12),
                            Text(
                              'TELEMETRY LINK SEVERED',
                              style: GoogleFonts.inter(fontWeight: FontWeight.w900, fontSize: 10, color: AppColors.mutedIcon, letterSpacing: 2),
                            ),
                          ],
                        ),
                      ),

                    const SizedBox(height: 32),
                    _SectionHeader(label: 'SYSTEM CAPABILITIES'),
                    const SizedBox(height: 16),
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children: (device['capabilities'] as List)
                          .map((c) => CapabilityChip(
                                capability: DeviceCapability.fromString(c.toString()),
                              ))
                          .toList(),
                    ),

                    const SizedBox(height: 32),
                    _SectionHeader(label: 'ORCHESTRATION ACTIONS'),
                    const SizedBox(height: 16),
                    GridView.count(
                      shrinkWrap: true,
                      physics: const NeverScrollableScrollPhysics(),
                      crossAxisCount: 2,
                      mainAxisSpacing: 12,
                      crossAxisSpacing: 12,
                      childAspectRatio: 1.4,
                      children: [
                        _ActionCard(icon: LucideIcons.terminal, label: 'REMOTE SHELL', color: AppColors.primaryRed, onTap: () => context.push('/run')),
                        _ActionCard(icon: LucideIcons.zap, label: 'MANUAL SYNC', color: AppColors.infoBlue, onTap: () {
                          ScaffoldMessenger.of(context).showSnackBar(
                            const SnackBar(content: Text('Manual sync triggered'), backgroundColor: AppColors.infoBlue),
                          );
                        }),
                        _ActionCard(icon: LucideIcons.refreshCw, label: 'REFRESH', color: AppColors.warningAmber, onTap: _loadDeviceData),
                        _ActionCard(icon: LucideIcons.activity, label: 'TELEMETRY', color: AppColors.safeGreen, onTap: _loadDeviceStatus),
                      ],
                    ),
                    
                    const SizedBox(height: 32),
                    _SectionHeader(label: 'HARDWARE MANIFEST'),
                    const SizedBox(height: 16),
                    GlassContainer(
                      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
                      child: Column(
                        children: [
                           _ManifestRow(label: 'ARCHITECTURE', value: device['arch']?.toString().toUpperCase() ?? 'N/A'),
                           _ManifestRow(label: 'PLATFORM', value: device['platform']?.toString().toUpperCase() ?? 'N/A'),
                           _ManifestRow(label: 'MESH ID', value: _truncateMeshId(device['device_id'])),
                           _ManifestRow(label: 'gRPC ADDRESS', value: device['grpc_addr']?.toString() ?? 'N/A'),
                        ],
                      ),
                    ),
                    const SizedBox(height: 60),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildDeviceHeader(Map<String, dynamic> device, bool isOnline) {
    IconData icon = LucideIcons.laptop;
    final platform = device['platform']?.toString().toLowerCase() ?? '';
    if (platform.contains('android')) icon = LucideIcons.smartphone;
    if (platform.contains('mac')) icon = LucideIcons.command;
    if (platform.contains('linux')) icon = LucideIcons.terminal;
    if (platform.contains('windows')) icon = LucideIcons.monitor;

    return GlassContainer(
      padding: const EdgeInsets.all(20),
      child: Row(
        children: [
          ThreeDBadgeIcon(
            icon: icon,
            accentColor: isOnline ? AppColors.safeGreen : AppColors.mutedIcon,
            size: 24,
          ),
          const SizedBox(width: 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  device['device_name']?.toString().toUpperCase() ?? 'UNKNOWN',
                  style: GoogleFonts.inter(fontSize: 18, fontWeight: FontWeight.w900, letterSpacing: 0.5),
                ),
                Text(
                  '${device['platform']?.toString().toUpperCase() ?? 'N/A'} ${device['arch']?.toString().toUpperCase() ?? ''}',
                  style: GoogleFonts.inter(color: AppColors.textSecondary, fontSize: 10, fontWeight: FontWeight.w700, letterSpacing: 1),
                ),
              ],
            ),
          ),
          _StatusBadge(isOnline: isOnline),
        ],
      ),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  final String label;
  const _SectionHeader({required this.label});

  @override
  Widget build(BuildContext context) {
    return Text(
      label,
      style: GoogleFonts.inter(
        fontSize: 9,
        fontWeight: FontWeight.w900,
        color: AppColors.mutedIcon,
        letterSpacing: 1.5,
      ),
    );
  }
}

class _RemoteViewport extends StatelessWidget {
  final Map<String, dynamic> device;

  const _RemoteViewport({required this.device});

  @override
  Widget build(BuildContext context) {
    final isMobile = device['type'] == 'mobile';
    final aspectRatio = isMobile ? 9 / 16 : 16 / 10;
    
    return GlassContainer(
      padding: EdgeInsets.zero,
      borderRadius: 24,
      child: Column(
        children: [
          AspectRatio(
            aspectRatio: aspectRatio,
            child: Container(
              margin: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                color: Colors.black,
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: AppColors.outline),
              ),
              clipBehavior: Clip.antiAlias,
              child: Stack(
                fit: StackFit.expand,
                children: [
                  ColorFiltered(
                    colorFilter: ColorFilter.mode(
                      Colors.white.withOpacity(0.8),
                      BlendMode.dstATop,
                    ),
                    child: Image.network(
                      isMobile 
                        ? 'https://images.unsplash.com/photo-1616348436168-de43ad0db179?auto=format&fit=crop&q=80&w=1000'
                        : 'https://images.unsplash.com/photo-1517694712202-14dd9538aa97?auto=format&fit=crop&q=80&w=2670',
                      fit: BoxFit.cover,
                    ),
                  ).animate().fadeIn(duration: 800.ms),
                  
                  // HUD Elements
                  Positioned(
                    top: 12,
                    right: 12,
                    child: Column(
                      children: [
                        _HudButton(icon: LucideIcons.maximize2),
                        const SizedBox(height: 8),
                        _HudButton(icon: LucideIcons.refreshCw),
                      ],
                    ),
                  ),

                  Positioned(
                    top: 12,
                    left: 12,
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                      decoration: BoxDecoration(
                        color: Colors.black.withOpacity(0.7),
                        borderRadius: BorderRadius.circular(4),
                        border: Border.all(color: AppColors.safeGreen.withOpacity(0.3)),
                      ),
                      child: Row(
                        children: [
                          const Icon(LucideIcons.wifi, size: 8, color: AppColors.safeGreen),
                          const SizedBox(width: 6),
                          Text(
                            'AES-256 STREAM • 24MS',
                            style: GoogleFonts.jetBrainsMono(color: AppColors.safeGreen, fontSize: 7, fontWeight: FontWeight.w800),
                          ),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 16),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceAround,
              children: [
                _ToolIcon(icon: LucideIcons.mousePointer2, label: 'TOUCH', active: true),
                _ToolIcon(icon: LucideIcons.keyboard, label: 'KEYS'),
                _ToolIcon(icon: LucideIcons.share2, label: 'CAST'),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _HudButton extends StatelessWidget {
  final IconData icon;
  const _HudButton({required this.icon});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        color: Colors.black.withOpacity(0.6),
        shape: BoxShape.circle,
        border: Border.all(color: AppColors.outline),
      ),
      child: Icon(icon, size: 14, color: Colors.white),
    );
  }
}

class _ToolIcon extends StatelessWidget {
  final IconData icon;
  final String label;
  final bool active;
  const _ToolIcon({required this.icon, required this.label, this.active = false});

  @override
  Widget build(BuildContext context) {
    final color = active ? AppColors.safeGreen : AppColors.mutedIcon;
    return Column(
      children: [
        Icon(icon, size: 18, color: color),
        const SizedBox(height: 4),
        Text(
          label,
          style: GoogleFonts.inter(fontSize: 8, fontWeight: FontWeight.w900, color: color, letterSpacing: 0.5),
        ),
      ],
    );
  }
}

class _StatusBadge extends StatelessWidget {
  final bool isOnline;
  const _StatusBadge({required this.isOnline});

  @override
  Widget build(BuildContext context) {
    final color = isOnline ? AppColors.safeGreen : AppColors.mutedIcon;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: color.withOpacity(0.05),
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: color.withOpacity(0.2)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(width: 5, height: 5, decoration: BoxDecoration(color: color, shape: BoxShape.circle)),
          const SizedBox(width: 8),
          Text(
            isOnline ? 'ONLINE' : 'OFFLINE',
            style: GoogleFonts.inter(fontSize: 9, fontWeight: FontWeight.w900, color: color, letterSpacing: 1),
          ),
        ],
      ),
    );
  }
}

class _LargeDonut extends StatelessWidget {
  final String label;
  final double value;
  final Color color;

  const _LargeDonut({required this.label, required this.value, required this.color});

  @override
  Widget build(BuildContext context) {
    return GlassContainer(
      padding: const EdgeInsets.all(24),
      borderRadius: 16,
      child: Column(
        children: [
          SizedBox(
            height: 80,
            width: 80,
            child: Stack(
              children: [
                PieChart(
                  PieChartData(
                    sectionsSpace: 0,
                    centerSpaceRadius: 30,
                    startDegreeOffset: -90,
                    sections: [
                      PieChartSectionData(color: color, value: value, title: '', radius: 6),
                      PieChartSectionData(color: AppColors.outline.withOpacity(0.2), value: 100 - value, title: '', radius: 6),
                    ],
                  ),
                ),
                Center(
                  child: Text(
                    '${value.toInt()}%',
                    style: GoogleFonts.jetBrainsMono(fontSize: 16, fontWeight: FontWeight.w800, color: AppColors.textPrimary),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),
          Text(
            label,
            style: GoogleFonts.inter(fontSize: 8, fontWeight: FontWeight.w900, color: AppColors.mutedIcon, letterSpacing: 1),
          ),
        ],
      ),
    ).animate().scale(duration: 400.ms, curve: Curves.easeOutBack);
  }
}

class _ActionCard extends StatelessWidget {
  final IconData icon;
  final String label;
  final Color color;
  final VoidCallback onTap;

  const _ActionCard({required this.icon, required this.label, required this.color, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return GlassContainer(
      padding: EdgeInsets.zero,
      borderRadius: 12,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            ThreeDBadgeIcon(
              icon: icon,
              accentColor: color,
              size: 14,
              isDanger: color == AppColors.dangerPink || color == AppColors.primaryRed,
              useRotation: icon == LucideIcons.refreshCw || icon == LucideIcons.zap,
            ),
            const SizedBox(height: 12),
            Text(
              label,
              style: GoogleFonts.inter(fontSize: 9, fontWeight: FontWeight.w900, color: AppColors.textPrimary, letterSpacing: 0.5),
            ),
          ],
        ),
      ),
    );
  }
}

class _ManifestRow extends StatelessWidget {
  final String label;
  final String value;
  const _ManifestRow({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 10),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(label, style: GoogleFonts.inter(color: AppColors.mutedIcon, fontSize: 9, fontWeight: FontWeight.w800, letterSpacing: 0.5)),
          Text(value, style: GoogleFonts.jetBrainsMono(fontSize: 10, fontWeight: FontWeight.w700, color: AppColors.textSecondary)),
        ],
      ),
    );
  }
}
