import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../features/dashboard/dashboard_screen.dart';
import '../features/devices/devices_screen.dart';
import '../features/devices/device_detail_screen.dart';
import '../features/execution/run_screen.dart';
import '../features/jobs/jobs_screen.dart';
import '../features/settings/settings_screen.dart';
import '../features/auth/connect_screen.dart';
import '../features/approvals/approvals_screen.dart';
import '../features/stream/stream_screen.dart';
import '../features/download/download_screen.dart';
import '../features/chat/chat_screen.dart';
import '../shared/widgets/app_sidebar.dart';
import '../shared/widgets/edge_mesh_wordmark.dart';
import '../theme/app_colors.dart';

// ShellRoute wrapper for sidebar navigation
class ScaffoldWithSidebar extends StatelessWidget {
  const ScaffoldWithSidebar({
    required this.navigationShell,
    super.key,
  });

  final StatefulNavigationShell navigationShell;

  @override
  Widget build(BuildContext context) {
    final width = MediaQuery.of(context).size.width;
    final isDesktop = width >= 900;

    return Scaffold(
      backgroundColor: AppColors.backgroundDark,
      drawer: isDesktop ? null : AppSidebar(navigationShell: navigationShell),
      body: SafeArea(
        child: Row(
          children: [
            if (isDesktop)
              _BuildDesktopRail(navigationShell: navigationShell),
            Expanded(
              child: Column(
                children: [
                  _TopBar(navigationShell: navigationShell),
                  Expanded(child: navigationShell),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _TopBar extends StatelessWidget {
  final StatefulNavigationShell navigationShell;
  const _TopBar({required this.navigationShell});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      child: Row(
        children: [
          _CircleIconButton(
            icon: Icons.menu_rounded,
            onTap: () => Scaffold.of(context).openDrawer(),
          ),
          const SizedBox(width: 10),

          // Center pill selector like ChatGPT
          Expanded(
            child: GestureDetector(
              onTap: () => _openModeSheet(context),
              child: Container(
                height: 40,
                decoration: BoxDecoration(
                  color: const Color(0xFF121B2B).withOpacity(0.55),
                  borderRadius: BorderRadius.circular(999),
                  border: Border.all(color: const Color(0xFF233043).withOpacity(0.9)),
                  boxShadow: [
                    BoxShadow(
                      color: Colors.black.withOpacity(0.35),
                      blurRadius: 18,
                      offset: const Offset(0, 10),
                    ),
                  ],
                ),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: const [
                    EdgeMeshWordmark(fontSize: 16),
                    SizedBox(width: 6),
                    Icon(Icons.expand_more_rounded, color: Color(0xFFA7B1C2), size: 18),
                  ],
                ),
              ),
            ),
          ),

          const SizedBox(width: 10),
          _CircleIconButton(
            icon: Icons.person_add_alt_1_rounded,
            onTap: () {
              // Custom action
            },
          ),
          const SizedBox(width: 8),
          _CircleIconButton(
            icon: Icons.chat_bubble_outline_rounded,
            onTap: () {
              // History action
            },
          ),
        ],
      ),
    );
  }

  void _openModeSheet(BuildContext context) {
    showModalBottomSheet(
      context: context,
      backgroundColor: const Color(0xFF0F1623),
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(22)),
      ),
      builder: (_) => _QuickSwitchSheet(navigationShell: navigationShell),
    );
  }
}

class _QuickSwitchSheet extends StatelessWidget {
  final StatefulNavigationShell navigationShell;
  const _QuickSwitchSheet({required this.navigationShell});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 24, horizontal: 16),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          _SheetItem(icon: Icons.chat_bubble_rounded, label: 'Chat', onTap: () => _closeAndGo(context, 0)),
          _SheetItem(icon: Icons.dashboard_rounded, label: 'Dashboard', onTap: () => _closeAndGo(context, 1)),
          _SheetItem(icon: Icons.router_rounded, label: 'Devices', onTap: () => _closeAndGo(context, 2)),
          _SheetItem(icon: Icons.layers_rounded, label: 'Jobs', onTap: () => _closeAndGo(context, 4)),
          _SheetItem(icon: Icons.check_circle_rounded, label: 'Approvals', onTap: () {
            Navigator.pop(context);
            context.go('/dashboard/approvals');
          }),
        ],
      ),
    );
  }

  void _closeAndGo(BuildContext context, int index) {
    Navigator.pop(context);
    navigationShell.goBranch(index);
  }
}

class _SheetItem extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  const _SheetItem({required this.icon, required this.label, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: Icon(icon, color: const Color(0xFFE6EDF6), size: 22),
      title: Text(label, style: const TextStyle(color: Color(0xFFE6EDF6), fontSize: 16, fontWeight: FontWeight.w500)),
      onTap: onTap,
    );
  }
}

class _CircleIconButton extends StatelessWidget {
  final IconData icon;
  final VoidCallback onTap;
  const _CircleIconButton({required this.icon, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return InkWell(
      borderRadius: BorderRadius.circular(999),
      onTap: onTap,
      child: Container(
        width: 44,
        height: 44,
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          color: const Color(0xFF121B2B).withOpacity(0.55),
          border: Border.all(color: const Color(0xFF233043).withOpacity(0.9)),
        ),
        child: Icon(icon, color: const Color(0xFFE6EDF6), size: 18),
      ),
    );
  }
}

class _BuildDesktopRail extends StatelessWidget {
  final StatefulNavigationShell navigationShell;
  const _BuildDesktopRail({required this.navigationShell});

  @override
  Widget build(BuildContext context) {
    return NavigationRail(
      backgroundColor: AppColors.surface1,
      selectedIndex: navigationShell.currentIndex,
      onDestinationSelected: (index) => navigationShell.goBranch(index),
      labelType: NavigationRailLabelType.all,
      selectedLabelTextStyle: GoogleFonts.inter(color: AppColors.textPrimary, fontSize: 11, fontWeight: FontWeight.bold),
      unselectedLabelTextStyle: GoogleFonts.inter(color: AppColors.textSecondary, fontSize: 11),
      destinations: const [
        NavigationRailDestination(
          icon: Icon(LucideIcons.messageSquare, size: 20),
          label: Text('Chat'),
        ),
        NavigationRailDestination(
          icon: Icon(LucideIcons.layoutDashboard, size: 20),
          label: Text('Dash'),
        ),
        NavigationRailDestination(
          icon: Icon(LucideIcons.server, size: 20),
          label: Text('Devices'),
        ),
        NavigationRailDestination(
          icon: Icon(LucideIcons.terminal, size: 20),
          label: Text('Run'),
        ),
        NavigationRailDestination(
          icon: Icon(LucideIcons.layers, size: 20),
          label: Text('Jobs'),
        ),
      ],
    );
  }
}

final appRouter = GoRouter(
  initialLocation: '/connect',
  routes: [
    GoRoute(
      path: '/connect',
      builder: (context, state) => const ConnectScreen(),
    ),
    StatefulShellRoute.indexedStack(
      builder: (context, state, navigationShell) {
        return ScaffoldWithSidebar(navigationShell: navigationShell);
      },
      branches: [
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/chat',
              builder: (context, state) => const ChatScreen(),
            ),
          ],
        ),
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/dashboard',
              builder: (context, state) => const DashboardScreen(),
              routes: [
                GoRoute(
                    path: 'approvals',
                    builder: (context, state) => const ApprovalsScreen()),
                GoRoute(
                    path: 'stream',
                    builder: (context, state) => const StreamScreen()),
                GoRoute(
                    path: 'download',
                    builder: (context, state) => const DownloadScreen()),
              ],
            ),
          ],
        ),
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/devices',
              builder: (context, state) => const DevicesScreen(),
              routes: [
                GoRoute(
                  path: ':id',
                  builder: (context, state) {
                    final id = state.pathParameters['id']!;
                    return DeviceDetailScreen(deviceId: id);
                  },
                ),
              ],
            ),
          ],
        ),
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/run',
              builder: (context, state) => const RunScreen(),
            ),
          ],
        ),
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/jobs',
              builder: (context, state) => const JobsScreen(),
            ),
          ],
        ),
      ],
    ),
  ],
);
