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
import '../../data/mock_data.dart';

class DevicesScreen extends StatelessWidget {
  const DevicesScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.backgroundDark,
      body: SafeArea(
        child: Column(
          children: [
            
            const StatusStrip(
              isConnected: true,
              serverAddress: '192.168.1.10:50051',
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
                      onPressed: () {},
                      icon: const Icon(LucideIcons.sliders, size: 16, color: AppColors.mutedIcon),
                    ),
                  ),
                ],
              ),
            ),
            
            // Device List
            Expanded(
              child: ListView.separated(
                padding: const EdgeInsets.symmetric(horizontal: 20),
                itemCount: MockData.devices.length,
                separatorBuilder: (_, __) => const SizedBox(height: 12),
                physics: const BouncingScrollPhysics(),
                itemBuilder: (context, index) {
                  final device = MockData.devices[index];
                  return _DeviceCard(device: device)
                      .animate()
                      .fadeIn(delay: (40 * index).ms)
                      .slideY(begin: 0.1, end: 0);
                },
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
