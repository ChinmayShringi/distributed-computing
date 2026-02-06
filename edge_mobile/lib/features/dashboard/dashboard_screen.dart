import 'package:flutter/material.dart';
import 'package:lucide_icons/lucide_icons.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:google_fonts/google_fonts.dart';
import '../../theme/app_colors.dart';
import '../../shared/widgets/edge_mesh_wordmark.dart';
import '../../shared/widgets/status_strip.dart';
import '../../shared/widgets/terminal_panel.dart';
import '../../shared/widgets/glass_container.dart';
import '../../shared/widgets/three_d_badge_icon.dart';
import '../../shared/widgets/execution_host_panel.dart';
import '../../data/mock_data.dart';

class DashboardScreen extends StatelessWidget {
  const DashboardScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.backgroundDark,
      body: SafeArea(
        child: CustomScrollView(
          physics: const BouncingScrollPhysics(),
          slivers: [
            // Standardized Header
            SliverToBoxAdapter(
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    const EdgeMeshWordmark(fontSize: 18),
                    Row(
                      children: [
                        IconButton(icon: const Icon(LucideIcons.bell, size: 18, color: AppColors.mutedIcon), onPressed: () {}),
                        IconButton(icon: const Icon(LucideIcons.user, size: 18, color: AppColors.mutedIcon), onPressed: () {}),
                      ],
                    ),
                  ],
                ),
              ),
            ),

            const SliverToBoxAdapter(
              child: StatusStrip(
                isConnected: true,
                serverAddress: '192.168.1.10:50051',
                isDangerous: false,
              ),
            ),

            const SliverToBoxAdapter(child: SizedBox(height: 16)),

            // KPI Cards
            SliverToBoxAdapter(
              child: SizedBox(
                height: 140,
                child: ListView(
                  scrollDirection: Axis.horizontal,
                  padding: const EdgeInsets.symmetric(horizontal: 20),
                  physics: const BouncingScrollPhysics(),
                  children: const [
                    _KpiCard(
                      title: 'CONNECTED NODES',
                      value: '12',
                      icon: LucideIcons.laptop,
                      color: AppColors.safeGreen,
                      subtitle: '4 ACTIVE SESSIONS',
                    ),
                    SizedBox(width: 12),
                    _KpiCard(
                      title: 'SECURE TOOLS',
                      value: '24',
                      icon: LucideIcons.shieldCheck,
                      color: AppColors.infoBlue,
                      subtitle: '8 ELEVATED',
                    ),
                    SizedBox(width: 12),
                    _KpiCard(
                      title: 'ACTIVE JOBS',
                      value: '03',
                      icon: LucideIcons.activity,
                      color: AppColors.warningAmber,
                      subtitle: '1 SYNCHRONIZING',
                    ),
                  ],
                ),
              ),
            ),

            // Execution Host + Utilization Panel
            SliverToBoxAdapter(
              child: Padding(
                padding: const EdgeInsets.all(20),
                child: ExecutionHostPanel(executions: MockData.executions),
              ),
            ),

            SliverToBoxAdapter(
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 20),
                child: _SectionHeader(label: 'RECENT ORCHESTRATION EVENTS'),
              ),
            ),

            const SliverToBoxAdapter(child: SizedBox(height: 12)),

            // Recent Activity List
            SliverPadding(
              padding: const EdgeInsets.symmetric(horizontal: 20),
              sliver: SliverList(
                delegate: SliverChildBuilderDelegate(
                  (context, index) {
                    final item = MockData.recentActivity[index];
                    return _ActivityItem(item: item)
                        .animate()
                        .fadeIn(delay: (40 * index).ms);
                  },
                  childCount: MockData.recentActivity.length,
                ),
              ),
            ),

            const SliverPadding(padding: EdgeInsets.only(bottom: 60)),
          ],
        ),
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
        fontWeight: FontWeight.w800,
        color: AppColors.mutedIcon,
        letterSpacing: 1.5,
      ),
    );
  }
}

class _KpiCard extends StatelessWidget {
  final String title;
  final String value;
  final String subtitle;
  final IconData icon;
  final Color color;

  const _KpiCard({
    required this.title,
    required this.value,
    required this.subtitle,
    required this.icon,
    required this.color,
  });


  @override
  Widget build(BuildContext context) {
    return GlassContainer(
      padding: const EdgeInsets.all(18),
      borderRadius: 16,
      child: SizedBox(
        width: 130,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            ThreeDBadgeIcon(icon: icon, accentColor: color, size: 16),
            const SizedBox(height: 16),
            Text(
              value,
              style: GoogleFonts.jetBrainsMono(
                fontSize: 24,
                fontWeight: FontWeight.w800,
                color: AppColors.textPrimary,
                height: 1.0,
              ),
            ),
            const SizedBox(height: 6),
            Text(
              title,
              style: GoogleFonts.inter(
                fontSize: 8,
                color: AppColors.textSecondary,
                fontWeight: FontWeight.w800,
                letterSpacing: 0.5,
                height: 1.2,
              ),
            ),
            const SizedBox(height: 3),
            Text(
              subtitle,
              style: GoogleFonts.inter(
                fontSize: 8,
                color: AppColors.mutedIcon,
                fontWeight: FontWeight.w600,
                height: 1.2,
              ),
            ),
          ],
        ),
      ),
    );
  }

}

class _ActivityItem extends StatelessWidget {
  final Map<String, dynamic> item;

  const _ActivityItem({required this.item});

  @override
  Widget build(BuildContext context) {
    final exitCode = item['exit_code'] as int? ?? -1;
    final color = exitCode == 0 ? AppColors.safeGreen : AppColors.primaryRed;

    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      child: GlassContainer(
        padding: EdgeInsets.zero,
        borderRadius: 16,
        child: ExpansionTile(
          tilePadding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
          iconColor: AppColors.mutedIcon,
          collapsedIconColor: AppColors.mutedIcon,
          leading: ThreeDBadgeIcon(
            icon: LucideIcons.terminal,
            size: 14,
            accentColor: AppColors.mutedIcon,
            useRotation: false,
          ),
          title: Text(
            item['cmd']!,
            style: GoogleFonts.jetBrainsMono(
              fontSize: 13,
              fontWeight: FontWeight.w600,
              color: AppColors.textPrimary,
            ),
          ),
          subtitle: Text(
            '${item['device'].toUpperCase()} • ${item['time'].toUpperCase()}',
            style: GoogleFonts.inter(fontSize: 9, color: AppColors.textSecondary, fontWeight: FontWeight.w700, letterSpacing: 0.5),
          ),
          trailing: Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            decoration: BoxDecoration(
              color: color.withOpacity(0.05),
              borderRadius: BorderRadius.circular(4),
              border: Border.all(color: color.withOpacity(0.2)),
            ),
            child: Text(
              'EXIT $exitCode',
              style: GoogleFonts.jetBrainsMono(
                color: color,
                fontSize: 9,
                fontWeight: FontWeight.w800,
              ),
            ),
          ),
          children: [
            Padding(
              padding: const EdgeInsets.all(20),
              child: TerminalPanel(
                output: item['output']!,
                exitCode: exitCode,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
