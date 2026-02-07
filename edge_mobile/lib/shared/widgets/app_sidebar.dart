import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:lucide_icons/lucide_icons.dart';
import 'package:go_router/go_router.dart';
import '../../theme/app_colors.dart';
import '../../shared/widgets/edge_mesh_wordmark.dart';
import '../../data/mock_data.dart';

class AppSidebar extends ConsumerWidget {
  final StatefulNavigationShell navigationShell;

  const AppSidebar({
    super.key,
    required this.navigationShell,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Drawer(
      backgroundColor: AppColors.backgroundDark,
      child: Column(
        children: [
          _SidebarHeader(),
          Expanded(
            child: ListView(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 24),
              children: [
                _SectionHeader(title: 'NAVIGATION'),
                _SidebarItem(
                  icon: LucideIcons.messageSquare,
                  label: 'Chat',
                  isActive: navigationShell.currentIndex == 0,
                  onTap: () => _navigate(context, 0),
                ),
                _SidebarItem(
                  icon: LucideIcons.layoutDashboard,
                  label: 'Dashboard',
                  isActive: navigationShell.currentIndex == 1,
                  onTap: () => _navigate(context, 1),
                ),
                _SidebarItem(
                  icon: LucideIcons.server,
                  label: 'Devices',
                  isActive: navigationShell.currentIndex == 2,
                  onTap: () => _navigate(context, 2),
                ),
                _SidebarItem(
                  icon: LucideIcons.terminal,
                  label: 'Run',
                  isActive: navigationShell.currentIndex == 3,
                  onTap: () => _navigate(context, 3),
                ),
                _SidebarItem(
                  icon: LucideIcons.layers,
                  label: 'Jobs',
                  isActive: navigationShell.currentIndex == 4,
                  badgeCount: MockData.jobs.where((j) => j['status'] == 'running').length,
                  onTap: () => _navigate(context, 4),
                ),
                
                const SizedBox(height: 24),
                _SectionHeader(title: 'SERVICES'),
                _SidebarItem(
                  icon: LucideIcons.checkCircle,
                  label: 'Approvals',
                  badgeCount: 2, // Mocked pending count
                  onTap: () {
                    context.pop();
                    context.go('/dashboard/approvals');
                  },
                ),
                _SidebarItem(
                  icon: LucideIcons.playCircle,
                  label: 'Stream',
                  onTap: () {
                    context.pop();
                    context.go('/dashboard/stream');
                  },
                ),
                _SidebarItem(
                  icon: LucideIcons.download,
                  label: 'Downloads',
                  onTap: () {
                    context.pop();
                    context.go('/dashboard/download');
                  },
                ),

                const SizedBox(height: 24),
                _SectionHeader(title: 'QUICK ACTIONS'),
                _QuickActionItem(
                  icon: LucideIcons.command,
                  label: 'Run a command',
                  onTap: () => _navigate(context, 3),
                ),
                _QuickActionItem(
                  icon: LucideIcons.list,
                  label: 'List devices',
                  onTap: () => _navigate(context, 2),
                ),
                _QuickActionItem(
                  icon: LucideIcons.playCircle,
                  label: 'Start stream',
                  onTap: () {
                    context.pop();
                    context.go('/dashboard/stream');
                  },
                ),
                _QuickActionItem(
                  icon: LucideIcons.download,
                  label: 'Download file',
                  onTap: () {
                    context.pop();
                    context.go('/dashboard/download');
                  },
                ),
              ],
            ),
          ),
          _SidebarFooter(),
        ],
      ),
    );
  }

  void _navigate(BuildContext context, int index) {
    navigationShell.goBranch(
      index,
      initialLocation: index == navigationShell.currentIndex,
    );
    context.pop(); // Close drawer
  }
}

class _SidebarHeader extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.only(top: 60, left: 24, right: 24, bottom: 20),
      decoration: const BoxDecoration(
        border: Border(bottom: BorderSide(color: AppColors.outline, width: 0.5)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 8,
                height: 8,
                decoration: const BoxDecoration(
                  color: AppColors.safeGreen,
                  shape: BoxShape.circle,
                  boxShadow: [
                    BoxShadow(color: AppColors.safeGreen, blurRadius: 4),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              Text(
                'CONNECTED',
                style: GoogleFonts.inter(
                  fontSize: 10,
                  fontWeight: FontWeight.w900,
                  color: AppColors.safeGreen,
                  letterSpacing: 1.0,
                ),
              ),
            ],
          ),
          const SizedBox(height: 4),
          Text(
            'mesh-node-alpha.edge',
            style: GoogleFonts.jetBrainsMono(
              fontSize: 12,
              color: AppColors.textSecondary,
            ),
          ),
          const SizedBox(height: 12),
          _ModePill(isSafe: true),
        ],
      ),
    );
  }
}

class _ModePill extends StatelessWidget {
  final bool isSafe;
  const _ModePill({required this.isSafe});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: (isSafe ? AppColors.safeGreen : AppColors.primaryRed).withOpacity(0.1),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: (isSafe ? AppColors.safeGreen : AppColors.primaryRed).withOpacity(0.2)),
      ),
      child: Text(
        isSafe ? 'SAFE MODE' : 'DANGEROUS',
        style: GoogleFonts.inter(
          fontSize: 9,
          fontWeight: FontWeight.w900,
          color: isSafe ? AppColors.safeGreen : AppColors.primaryRed,
          letterSpacing: 0.5,
        ),
      ),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  final String title;
  const _SectionHeader({required this.title});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(left: 12, bottom: 8),
      child: Text(
        title,
        style: GoogleFonts.inter(
          fontSize: 10,
          fontWeight: FontWeight.w900,
          color: AppColors.mutedIcon,
          letterSpacing: 1.2,
        ),
      ),
    );
  }
}

class _SidebarItem extends StatelessWidget {
  final IconData icon;
  final String label;
  final bool isActive;
  final VoidCallback onTap;
  final int? badgeCount;

  const _SidebarItem({
    required this.icon,
    required this.label,
    this.isActive = false,
    required this.onTap,
    this.badgeCount,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 4),
      decoration: BoxDecoration(
        color: isActive ? AppColors.surface2 : Colors.transparent,
        borderRadius: BorderRadius.circular(12),
      ),
      child: ListTile(
        onTap: onTap,
        dense: true,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
        leading: Icon(
          icon,
          size: 20,
          color: isActive ? AppColors.textPrimary : AppColors.textSecondary,
        ),
        title: Text(
          label,
          style: GoogleFonts.inter(
            fontSize: 14,
            fontWeight: isActive ? FontWeight.w600 : FontWeight.w500,
            color: isActive ? AppColors.textPrimary : AppColors.textSecondary,
          ),
        ),
        trailing: badgeCount != null && badgeCount! > 0
            ? Container(
                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(
                  color: AppColors.primaryRed,
                  borderRadius: BorderRadius.circular(100),
                ),
                child: Text(
                  badgeCount.toString(),
                  style: GoogleFonts.inter(
                    fontSize: 10,
                    fontWeight: FontWeight.bold,
                    color: Colors.white,
                  ),
                ),
              )
            : null,
      ),
    );
  }
}

class _QuickActionItem extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  const _QuickActionItem({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return ListTile(
      onTap: onTap,
      dense: true,
      leading: Icon(icon, size: 18, color: AppColors.textSecondary),
      title: Text(
        label,
        style: GoogleFonts.inter(
          fontSize: 13,
          color: AppColors.textSecondary,
        ),
      ),
    );
  }
}

class _SidebarFooter extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: const BoxDecoration(
        border: Border(top: BorderSide(color: AppColors.outline, width: 0.5)),
      ),
      child: Row(
        children: [
          CircleAvatar(
            radius: 16,
            backgroundColor: AppColors.surfaceVariant,
            child: const Icon(LucideIcons.user, size: 16, color: AppColors.textPrimary),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  'Sariya Rizwan',
                  style: GoogleFonts.inter(fontSize: 13, fontWeight: FontWeight.w600, color: AppColors.textPrimary),
                ),
                Text(
                  'Admin Mode',
                  style: GoogleFonts.inter(fontSize: 11, color: AppColors.mutedIcon),
                ),
              ],
            ),
          ),
          IconButton(
            onPressed: () {},
            icon: const Icon(LucideIcons.settings, size: 18, color: AppColors.mutedIcon),
          ),
        ],
      ),
    );
  }
}
