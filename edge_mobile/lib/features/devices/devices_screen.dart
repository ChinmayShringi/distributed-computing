import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:lucide_icons/lucide_icons.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:go_router/go_router.dart';
import 'package:google_fonts/google_fonts.dart';
import '../../theme/app_colors.dart';
import '../../shared/widgets/edge_mesh_wordmark.dart';
import '../../shared/widgets/status_strip.dart';
import '../../shared/widgets/capability_chip.dart';
import '../../shared/widgets/glass_container.dart';
import '../../shared/widgets/three_d_badge_icon.dart';
import '../../services/grpc_service.dart';
import 'dart:async';

class DevicesScreen extends StatefulWidget {
  const DevicesScreen({super.key});

  @override
  State<DevicesScreen> createState() => _DevicesScreenState();
}

class _DevicesScreenState extends State<DevicesScreen> {
  final _grpcService = GrpcService();
  List<Map<String, dynamic>> _devices = [];
  List<Map<String, dynamic>> _filteredDevices = [];
  bool _isLoading = true;
  String? _errorMessage;
  Timer? _refreshTimer;
  String _searchQuery = '';

  @override
  void initState() {
    super.initState();
    _loadDevices();
    _startPeriodicRefresh();
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    super.dispose();
  }

  void _startPeriodicRefresh() {
    _refreshTimer = Timer.periodic(const Duration(seconds: 10), (_) {
      if (mounted) _loadDevices(silent: true);
    });
  }

  Future<void> _loadDevices({bool silent = false}) async {
    if (!silent) {
      setState(() {
        _isLoading = true;
        _errorMessage = null;
      });
    }

    try {
      final devices = await _grpcService.listDevices();
      
      // Enrich devices with status info
      final enrichedDevices = await Future.wait(
        devices.map((device) async {
          try {
            final status = await _grpcService.getDeviceStatus(device['device_id'] as String);
            
            // Map backend data to UI format
            final cpuLoad = (status['cpu_load'] as num).toDouble();
            final memUsedMb = (status['mem_used_mb'] as num).toDouble();
            final memTotalMb = (status['mem_total_mb'] as num).toDouble();
            
            return {
              'id': device['device_id'],
              'name': device['device_name'],
              'os': '${device['platform']} ${device['arch']}',
              'type': _deriveDeviceType(device['platform'] as String),
              'status': 'online', // If we got status, it's online
              'cpu': (cpuLoad * 100).clamp(0, 100).toInt(),
              'memory': ((memUsedMb / memTotalMb) * 100).clamp(0, 100).toInt(),
              'isLocal': device['grpc_addr'].toString().contains('127.0.0.1') || 
                         device['grpc_addr'].toString().contains('localhost'),
              'capabilities': device['capabilities'] as List,
              'grpc_addr': device['grpc_addr'],
            };
          } catch (e) {
            // Device is offline or unreachable
            return {
              'id': device['device_id'],
              'name': device['device_name'],
              'os': '${device['platform']} ${device['arch']}',
              'type': _deriveDeviceType(device['platform'] as String),
              'status': 'offline',
              'cpu': 0,
              'memory': 0,
              'isLocal': false,
              'capabilities': device['capabilities'] as List,
              'grpc_addr': device['grpc_addr'],
            };
          }
        })
      );

      if (mounted) {
        setState(() {
          _devices = enrichedDevices;
          _filteredDevices = enrichedDevices;
          _isLoading = false;
          _errorMessage = null;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _isLoading = false;
          _errorMessage = 'Failed to load devices: $e';
        });
      }
    }
  }

  String _deriveDeviceType(String platform) {
    switch (platform.toLowerCase()) {
      case 'android':
      case 'ios':
        return 'mobile';
      case 'linux':
        return 'server';
      default:
        return 'desktop';
    }
  }

  void _filterDevices(String query) {
    setState(() {
      _searchQuery = query;
      if (query.isEmpty) {
        _filteredDevices = _devices;
      } else {
        _filteredDevices = _devices.where((device) {
          final name = device['name'].toString().toLowerCase();
          final os = device['os'].toString().toLowerCase();
          final searchLower = query.toLowerCase();
          return name.contains(searchLower) || os.contains(searchLower);
        }).toList();
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.backgroundDark,
      body: SafeArea(
        child: Column(
          children: [
            
            const StatusStrip(
              isConnected: true,
              serverAddress: '192.168.1.195:50051',
              isDangerous: false,
            ),

            // Search Bar
            Padding(
              padding: const EdgeInsets.all(20),
              child: Row(
                children: [
                  Expanded(
                    child: GlassContainer(
                      padding: const EdgeInsets.symmetric(horizontal: 16),
                      borderRadius: 12,
                      child: TextField(
                        style: GoogleFonts.inter(fontSize: 14),
                        onChanged: _filterDevices,
                        decoration: InputDecoration(
                          hintText: 'Search secure nodes...',
                          hintStyle: GoogleFonts.inter(color: AppColors.mutedIcon, fontSize: 13),
                          prefixIcon: const Icon(LucideIcons.search, size: 16, color: AppColors.mutedIcon),
                          border: InputBorder.none,
                          enabledBorder: InputBorder.none,
                          focusedBorder: InputBorder.none,
                          contentPadding: const EdgeInsets.symmetric(vertical: 14),
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(width: 12),
                  GlassContainer(
                    padding: EdgeInsets.zero,
                    borderRadius: 12,
                    child: IconButton(
                      onPressed: _loadDevices,
                      icon: const Icon(LucideIcons.refreshCw, size: 16, color: AppColors.mutedIcon),
                    ),
                  ),
                ],
              ),
            ),
            
            // Device List
            Expanded(
              child: _isLoading
                  ? const Center(
                      child: CircularProgressIndicator(
                        color: AppColors.safeGreen,
                      ),
                    )
                  : _errorMessage != null
                      ? Center(
                          child: Padding(
                            padding: const EdgeInsets.all(32),
                            child: Column(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                const Icon(LucideIcons.alertCircle, size: 48, color: AppColors.primaryRed),
                                const SizedBox(height: 16),
                                Text(
                                  _errorMessage!,
                                  style: GoogleFonts.inter(color: AppColors.textSecondary, fontSize: 14),
                                  textAlign: TextAlign.center,
                                ),
                                const SizedBox(height: 24),
                                ElevatedButton.icon(
                                  onPressed: _loadDevices,
                                  icon: const Icon(LucideIcons.refreshCw, size: 16),
                                  label: const Text('RETRY'),
                                  style: ElevatedButton.styleFrom(
                                    backgroundColor: AppColors.safeGreen,
                                    foregroundColor: Colors.black,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        )
                      : _filteredDevices.isEmpty
                          ? Center(
                              child: Padding(
                                padding: const EdgeInsets.all(32),
                                child: Column(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    const Icon(LucideIcons.server, size: 48, color: AppColors.mutedIcon),
                                    const SizedBox(height: 16),
                                    Text(
                                      _searchQuery.isEmpty
                                          ? 'No devices registered'
                                          : 'No devices found',
                                      style: GoogleFonts.inter(color: AppColors.textSecondary, fontSize: 14),
                                    ),
                                  ],
                                ),
                              ),
                            )
                          : RefreshIndicator(
                              onRefresh: _loadDevices,
                              color: AppColors.safeGreen,
                              child: ListView.separated(
                                padding: const EdgeInsets.symmetric(horizontal: 20),
                                itemCount: _filteredDevices.length,
                                separatorBuilder: (_, __) => const SizedBox(height: 12),
                                physics: const AlwaysScrollableScrollPhysics(),
                                itemBuilder: (context, index) {
                                  final device = _filteredDevices[index];
                                  return _DeviceCard(device: device)
                                      .animate()
                                      .fadeIn(delay: (40 * index).ms)
                                      .slideY(begin: 0.1, end: 0);
                                },
                              ),
                            ),
            ),
          ],
        ),
      ),
    );
  }
}

class _DeviceCard extends StatelessWidget {
  final Map<String, dynamic> device;

  const _DeviceCard({required this.device});

  @override
  Widget build(BuildContext context) {
    final isOnline = device['status'] == 'online';
    
    return InkWell(
      onTap: () => context.push('/devices/${device['id']}'),
      borderRadius: BorderRadius.circular(16),
      child: GlassContainer(
        padding: const EdgeInsets.all(20),
        borderRadius: 16,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                _buildOsIcon(device['os'], isOnline),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        device['name'].toString().toUpperCase(),
                        style: GoogleFonts.inter(
                          fontWeight: FontWeight.w800,
                          fontSize: 14,
                          letterSpacing: 0.5,
                        ),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        '${device['os'].toString().toUpperCase()} • ${device['type'].toString().toUpperCase()}',
                        style: GoogleFonts.inter(
                          color: AppColors.textSecondary,
                          fontSize: 9,
                          fontWeight: FontWeight.w700,
                          letterSpacing: 0.5,
                        ),
                      ),
                    ],
                  ),
                ),
                Container(
                  width: 6,
                  height: 6,
                  decoration: BoxDecoration(
                    color: isOnline ? AppColors.safeGreen : AppColors.mutedIcon,
                    shape: BoxShape.circle,
                  ),
                ).animate(onPlay: (c) => isOnline ? c.repeat(reverse: true) : null).fade(begin: 0.4, end: 1.0, duration: 1.seconds),
              ],
            ),
            const SizedBox(height: 20),
            
            // Capabilities (Safe Mapping)
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: (device['capabilities'] as List)
                  .map((c) => CapabilityChip(
                        capability: DeviceCapability.fromString(c.toString()),
                      ))
                  .toList(),
            ),
            
            const SizedBox(height: 24),
            
            // Stats (Professional Donut Charts)
            if (isOnline)
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceAround,
                children: [
                  _buildDonutChart('CPU LOAD', (device['cpu'] as num).toDouble(), AppColors.safeGreen),
                  _buildDonutChart('MEM USAGE', (device['memory'] as num).toDouble(), AppColors.infoBlue),
                ],
              ),
            
            if (!isOnline)
              Container(
                width: double.infinity,
                padding: const EdgeInsets.symmetric(vertical: 12),
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  color: AppColors.surface2,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(
                  'NODE OFFLINE • LAST HANDSHAKE 2D AGO',
                  style: GoogleFonts.inter(color: AppColors.mutedIcon, fontSize: 8, fontWeight: FontWeight.w900, letterSpacing: 1),
                ),
              ),
          ],
        ),
      ),
    );
  }
  
  Widget _buildOsIcon(String os, bool isOnline) {
    IconData icon = LucideIcons.smartphone;
    if (os.toLowerCase().contains('mac')) icon = LucideIcons.laptop;
    if (os.toLowerCase().contains('windows')) icon = LucideIcons.monitor;
    if (os.toLowerCase().contains('linux') || os.toLowerCase().contains('ubuntu')) icon = LucideIcons.terminal;
    
    return ThreeDBadgeIcon(
      icon: icon,
      accentColor: isOnline ? AppColors.safeGreen : AppColors.mutedIcon,
      size: 16,
    );
  }
  
  Widget _buildDonutChart(String label, double percentage, Color color) {
    return Column(
      children: [
        SizedBox(
          height: 64,
          width: 64,
          child: Stack(
            children: [
              PieChart(
                PieChartData(
                  sectionsSpace: 0,
                  centerSpaceRadius: 24,
                  startDegreeOffset: -90,
                  sections: [
                    PieChartSectionData(
                      color: color,
                      value: percentage,
                      title: '',
                      radius: 5,
                    ),
                    PieChartSectionData(
                      color: AppColors.outline.withOpacity(0.2),
                      value: 100 - percentage,
                      title: '',
                      radius: 5,
                    ),
                  ],
                ),
              ),
              Center(
                child: Text(
                  '${percentage.toInt()}%',
                  style: GoogleFonts.jetBrainsMono(
                    fontSize: 11,
                    fontWeight: FontWeight.w800,
                    color: AppColors.textPrimary,
                  ),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 10),
        Text(
          label,
          style: GoogleFonts.inter(
            color: AppColors.mutedIcon,
            fontSize: 8,
            fontWeight: FontWeight.w900,
            letterSpacing: 1,
          ),
        ),
      ],
    );
  }
}
