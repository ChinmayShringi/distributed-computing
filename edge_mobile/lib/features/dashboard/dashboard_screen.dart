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
import '../../services/grpc_service.dart';
import 'dart:async';

class DashboardScreen extends StatefulWidget {
  const DashboardScreen({super.key});

  @override
  State<DashboardScreen> createState() => _DashboardScreenState();
}

class _DashboardScreenState extends State<DashboardScreen> {
  final _grpcService = GrpcService();
  List<Map<String, dynamic>> _devices = [];
  List<Map<String, dynamic>> _recentActivity = [];
  List<Map<String, dynamic>> _runningTasks = [];
  bool _isLoading = true;
  String? _errorMessage;
  Timer? _refreshTimer;

  // Connection status
  bool _isConnected = false;
  String _serverAddress = '';

  @override
  void initState() {
    super.initState();
    _loadDashboardData();
    _startPeriodicRefresh();
    _loadConnectionStatus();

    // Listen for connection status changes
    _grpcService.connectionStatus.addListener(_onConnectionStatusChanged);
  }

  void _onConnectionStatusChanged() {
    final status = _grpcService.connectionStatus.value;
    if (mounted) {
      setState(() {
        _isConnected = status.connected;
        _serverAddress = status.connected
            ? '${status.host}:${status.grpcPort}'
            : '';
      });
    }
  }

  Future<void> _loadConnectionStatus() async {
    try {
      final status = await _grpcService.getConnectionStatus();
      if (mounted) {
        setState(() {
          _isConnected = status.connected;
          _serverAddress = status.connected
              ? '${status.host}:${status.grpcPort}'
              : '';
        });
      }
    } catch (e) {
      debugPrint('Failed to load connection status: $e');
    }
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    _grpcService.connectionStatus.removeListener(_onConnectionStatusChanged);
    super.dispose();
  }

  void _startPeriodicRefresh() {
    _refreshTimer = Timer.periodic(const Duration(seconds: 10), (_) {
      if (mounted) _loadDashboardData(silent: true);
    });
  }

  String _formatTime(DateTime dt) {
    final now = DateTime.now();
    final diff = now.difference(dt);
    if (diff.inMinutes < 1) return 'Just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    return '${diff.inDays}d ago';
  }

  String _formatElapsedMs(int elapsedMs) {
    if (elapsedMs < 1000) return '${elapsedMs}ms';
    if (elapsedMs < 60000) return '${(elapsedMs / 1000).toStringAsFixed(1)}s';
    return '${(elapsedMs / 60000).toStringAsFixed(1)}m';
  }

  Future<void> _loadDashboardData({bool silent = false}) async {
    if (!silent) {
      setState(() {
        _isLoading = true;
        _errorMessage = null;
      });
    }

    try {
      // Load devices with status
      final devices = await _grpcService.listDevices();
      final enrichedDevices = <Map<String, dynamic>>[];

      for (var device in devices) {
        try {
          final status = await _grpcService.getDeviceStatus(device['device_id'] as String);
          enrichedDevices.add({
            ...device,
            'status': 'online',
            'cpu_load': status['cpu_load'],
            'mem_used_mb': status['mem_used_mb'],
            'mem_total_mb': status['mem_total_mb'],
            'last_seen': status['last_seen'],
          });
        } catch (e) {
          enrichedDevices.add({
            ...device,
            'status': 'offline',
          });
        }
      }

      // Load activity data (running tasks and device activities)
      List<Map<String, dynamic>> runningTasks = [];
      List<Map<String, dynamic>> activity = [];

      try {
        final activityData = await _grpcService.getActivity(includeMetrics: false);

        // Extract running tasks
        if (activityData['running_tasks'] is List) {
          runningTasks = (activityData['running_tasks'] as List)
              .map((t) => Map<String, dynamic>.from(t as Map))
              .toList();
        }

        // Build activity items from running tasks
        for (var task in runningTasks) {
          final startedAtMs = task['started_at_ms'] as int? ?? 0;
          final ts = startedAtMs > 0
              ? DateTime.fromMillisecondsSinceEpoch(startedAtMs)
              : DateTime.now();
          final elapsedMs = task['elapsed_ms'] as int? ?? 0;

          activity.add({
            'timestamp': ts.toIso8601String(),
            'type': 'running_task',
            'message': '${task['kind']} running on ${task['device_name']}',
            'device': task['device_name'],
            'severity': 'info',
            'cmd': task['kind'] ?? 'Task',
            'selected_device_name': task['device_name'] ?? 'Unknown',
            'time': _formatElapsedMs(elapsedMs),
            'output': 'Job: ${task['job_id']} • Task: ${task['task_id']}',
            'exit_code': -1, // Still running
            'is_running': true,
          });
        }

        // Add device activity summaries
        if (activityData['device_activities'] is List) {
          for (var deviceActivity in (activityData['device_activities'] as List)) {
            final da = deviceActivity as Map;
            final taskCount = da['running_task_count'] as int? ?? 0;
            if (taskCount > 0) {
              activity.add({
                'timestamp': DateTime.now().toIso8601String(),
                'type': 'device_activity',
                'message': '${da['device_name']} has $taskCount running task(s)',
                'device': da['device_name'],
                'severity': 'info',
                'cmd': 'Device Activity',
                'selected_device_name': da['device_name'] ?? 'Unknown',
                'time': 'Active',
                'output': '$taskCount task(s) running',
                'exit_code': 0,
                'is_running': false,
              });
            }
          }
        }
      } catch (e) {
        // If activity fetch fails, fall back to device status activity
        debugPrint('Activity fetch failed: $e');
      }

      // If no activity from server, generate from device updates
      if (activity.isEmpty) {
        for (var device in enrichedDevices) {
          if (device['status'] == 'online') {
            final ts = DateTime.now().subtract(Duration(seconds: activity.length * 30));
            activity.add({
              'timestamp': ts.toIso8601String(),
              'type': 'device_update',
              'message': '${device['device_name']} reported status',
              'device': device['device_name'],
              'severity': 'info',
              'cmd': 'Status check',
              'selected_device_name': device['device_name'],
              'time': _formatTime(ts),
              'output': 'Device online • CPU: ${((device['cpu_load'] ?? 0) * 100).toStringAsFixed(1)}%',
              'exit_code': 0,
              'is_running': false,
            });
          }
        }
      }

      if (mounted) {
        setState(() {
          _devices = enrichedDevices;
          _recentActivity = activity;
          _runningTasks = runningTasks;
          _isLoading = false;
          _errorMessage = null;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _isLoading = false;
          _errorMessage = 'Failed to load dashboard: $e';
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final onlineDevices = _devices.where((d) => d['status'] == 'online').length;
    final activeJobs = _runningTasks.length;

    return Scaffold(
      backgroundColor: AppColors.backgroundDark,
      body: SafeArea(
        child: _isLoading
            ? const Center(
                child: CircularProgressIndicator(color: AppColors.safeGreen),
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
                            onPressed: _loadDashboardData,
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
                : RefreshIndicator(
                    onRefresh: _loadDashboardData,
                    color: AppColors.safeGreen,
                    child: CustomScrollView(
                      physics: const AlwaysScrollableScrollPhysics(),
                      slivers: [
                        SliverToBoxAdapter(
                          child: StatusStrip(
                            isConnected: _isConnected,
                            serverAddress: _isConnected ? _serverAddress : null,
                            isDangerous: false,
                          ),
                        ),

                        const SliverToBoxAdapter(child: SizedBox(height: 16)),

                        // KPI Cards
                        SliverPadding(
                          padding: const EdgeInsets.symmetric(horizontal: 20),
                          sliver: SliverGrid(
                            gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                              crossAxisCount: 3,
                              mainAxisSpacing: 12,
                              crossAxisSpacing: 12,
                              childAspectRatio: 0.85,
                            ),
                            delegate: SliverChildListDelegate([
                              _KpiCard(
                                title: 'NODES',
                                value: onlineDevices.toString().padLeft(2, '0'),
                                icon: LucideIcons.laptop,
                                color: AppColors.safeGreen,
                                subtitle: '${_devices.length} TOTAL',
                              ),
                              const _KpiCard(
                                title: 'TOOLS',
                                value: '24',
                                icon: LucideIcons.shieldCheck,
                                color: AppColors.infoBlue,
                                subtitle: '8 ELEVATED',
                              ),
                              _KpiCard(
                                title: 'JOBS',
                                value: activeJobs.toString().padLeft(2, '0'),
                                icon: LucideIcons.activity,
                                color: activeJobs > 0 ? AppColors.warningAmber : AppColors.mutedIcon,
                                subtitle: activeJobs > 0 ? 'RUNNING' : 'IDLE',
                              ),
                            ]),
                          ),
                        ),

                        const SliverToBoxAdapter(child: SizedBox(height: 24)),

                        SliverToBoxAdapter(
                          child: Padding(
                            padding: const EdgeInsets.symmetric(horizontal: 20),
                            child: _SectionHeader(
                              label: _runningTasks.isNotEmpty
                                  ? 'RUNNING TASKS (${_runningTasks.length})'
                                  : 'RECENT ORCHESTRATION EVENTS',
                            ),
                          ),
                        ),

                        const SliverToBoxAdapter(child: SizedBox(height: 12)),

                        // Recent Activity List
                        if (_recentActivity.isEmpty)
                          SliverPadding(
                            padding: const EdgeInsets.all(32),
                            sliver: SliverToBoxAdapter(
                              child: Center(
                                child: Column(
                                  children: [
                                    const Icon(LucideIcons.activity, size: 32, color: AppColors.mutedIcon),
                                    const SizedBox(height: 12),
                                    Text(
                                      'No recent activity',
                                      style: GoogleFonts.inter(color: AppColors.textSecondary, fontSize: 12),
                                    ),
                                  ],
                                ),
                              ),
                            ),
                          )
                        else
                          SliverPadding(
                            padding: const EdgeInsets.symmetric(horizontal: 20),
                            sliver: SliverList(
                              delegate: SliverChildBuilderDelegate(
                                (context, index) {
                                  final item = _recentActivity[index];
                                  return _ActivityItem(item: item)
                                      .animate()
                                      .fadeIn(delay: (40 * index).ms);
                                },
                                childCount: _recentActivity.length,
                              ),
                            ),
                          ),

                        const SliverPadding(padding: EdgeInsets.only(bottom: 60)),
                      ],
                    ),
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
      padding: const EdgeInsets.all(12),
      borderRadius: 16,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          ThreeDBadgeIcon(icon: icon, accentColor: color, size: 18),
          const SizedBox(height: 8),
          Text(
            value,
            style: GoogleFonts.jetBrainsMono(
              fontSize: 24,
              fontWeight: FontWeight.w800,
              color: AppColors.textPrimary,
              height: 1.0,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            title,
            style: GoogleFonts.inter(
              fontSize: 10,
              color: AppColors.textSecondary,
              fontWeight: FontWeight.w900,
              letterSpacing: 0.8,
              height: 1.2,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            subtitle,
            style: GoogleFonts.inter(
              fontSize: 9,
              color: AppColors.mutedIcon,
              fontWeight: FontWeight.w600,
              height: 1.2,
            ),
          ),
        ],
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
    final isRunning = item['is_running'] as bool? ?? false;

    Color color;
    String statusText;
    if (isRunning) {
      color = AppColors.warningAmber;
      statusText = 'RUNNING';
    } else if (exitCode == 0) {
      color = AppColors.safeGreen;
      statusText = 'EXIT 0';
    } else if (exitCode == -1) {
      color = AppColors.mutedIcon;
      statusText = 'N/A';
    } else {
      color = AppColors.primaryRed;
      statusText = 'EXIT $exitCode';
    }

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
            icon: isRunning ? LucideIcons.loader : LucideIcons.terminal,
            size: 14,
            accentColor: color,
            useRotation: false,
          ),
          title: Text(
            item['cmd'] ?? 'N/A',
            style: GoogleFonts.jetBrainsMono(
              fontSize: 13,
              fontWeight: FontWeight.w600,
              color: AppColors.textPrimary,
            ),
          ),
          subtitle: Text(
            '${(item['selected_device_name'] ?? 'UNKNOWN').toUpperCase()} • ${(item['time'] ?? 'UNKNOWN').toUpperCase()}',
            style: GoogleFonts.inter(fontSize: 10, color: AppColors.textSecondary, fontWeight: FontWeight.w700, letterSpacing: 0.5),
          ),
          trailing: Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            decoration: BoxDecoration(
              color: color.withValues(alpha: 0.05),
              borderRadius: BorderRadius.circular(4),
              border: Border.all(color: color.withValues(alpha: 0.2)),
            ),
            child: Text(
              statusText,
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
                output: item['output']?.toString() ?? '',
                exitCode: exitCode,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
