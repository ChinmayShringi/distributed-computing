import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
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

// ShellRoute wrapper for bottom navigation
class ScaffoldWithNavBar extends StatelessWidget {
  const ScaffoldWithNavBar({
    required this.navigationShell,
    super.key,
  });

  final StatefulNavigationShell navigationShell;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: navigationShell,
      bottomNavigationBar: NavigationBar(
        selectedIndex: navigationShell.currentIndex,
        onDestinationSelected: (index) => navigationShell.goBranch(
          index,
          initialLocation: index == navigationShell.currentIndex,
        ),
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.dashboard_outlined),
            selectedIcon: Icon(Icons.dashboard),
            label: 'Dashboard',
          ),
          NavigationDestination(
            icon: Icon(Icons.devices_outlined),
            selectedIcon: Icon(Icons.devices),
            label: 'Devices',
          ),
          NavigationDestination(
            icon: Icon(Icons.terminal_outlined),
            selectedIcon: Icon(Icons.terminal),
            label: 'Run',
          ),
          NavigationDestination(
            icon: Icon(Icons.rule_outlined),
            selectedIcon: Icon(Icons.rule),
            label: 'Jobs',
          ),
          NavigationDestination(
            icon: Icon(Icons.settings_outlined),
            selectedIcon: Icon(Icons.settings),
            label: 'More',
          ),
        ],
      ),
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
        return ScaffoldWithNavBar(navigationShell: navigationShell);
      },
      branches: [
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
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/settings',
              builder: (context, state) => const SettingsScreen(),
            ),
          ],
        ),
      ],
    ),
  ],
);
