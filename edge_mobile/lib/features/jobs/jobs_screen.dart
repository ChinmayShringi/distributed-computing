import 'dart:async';
import 'package:flutter/material.dart';
import 'package:lucide_icons/lucide_icons.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:google_fonts/google_fonts.dart';
import '../../theme/app_colors.dart';
import '../../shared/widgets/status_strip.dart';
import '../../shared/widgets/glass_container.dart';
import '../../shared/widgets/three_d_badge_icon.dart';
import '../../services/grpc_service.dart';

class JobsScreen extends StatefulWidget {
  const JobsScreen({super.key});

  @override
  State<JobsScreen> createState() => _JobsScreenState();
}

class _JobsScreenState extends State<JobsScreen> {
  final _grpcService = GrpcService();
  List<Map<String, dynamic>> _runningTasks = [];
  List<Map<String, dynamic>> _deviceActivities = [];
  bool _isLoading = true;
  String? _errorMessage;
  Timer? _refreshTimer;

  // Connection status
  bool _isConnected = false;
  String _serverAddress = '';

  @override
  void initState() {
    super.initState();
    _loadActivityData();
    _startPeriodicRefresh();
    _loadConnectionStatus();
    _grpcService.connectionStatus.addListener(_onConnectionStatusChanged);
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    _grpcService.connectionStatus.removeListener(_onConnectionStatusChanged);
    super.dispose();
  }

  void _startPeriodicRefresh() {
    _refreshTimer = Timer.periodic(const Duration(seconds: 10), (_) {
      if (mounted) _loadActivityData(silent: true);
    });
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

  Future<void> _loadActivityData({bool silent = false}) async {
    if (!silent) {
      setState(() {
        _isLoading = true;
        _errorMessage = null;
      });
    }

    try {
      final activityData = await _grpcService.getActivity(includeMetrics: false);

      List<Map<String, dynamic>> runningTasks = [];
      List<Map<String, dynamic>> deviceActivities = [];

      if (activityData['running_tasks'] is List) {
        runningTasks = (activityData['running_tasks'] as List)
            .map((t) => Map<String, dynamic>.from(t as Map))
            .toList();
      }

      if (activityData['device_activities'] is List) {
        deviceActivities = (activityData['device_activities'] as List)
            .map((d) => Map<String, dynamic>.from(d as Map))
            .toList();
      }

      if (mounted) {
        setState(() {
          _runningTasks = runningTasks;
          _deviceActivities = deviceActivities;
          _isLoading = false;
          _errorMessage = null;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _isLoading = false;
          _errorMessage = 'Failed to load tasks: $e';
        });
      }
    }
  }

  String _formatElapsedMs(int elapsedMs) {
    if (elapsedMs < 1000) return '${elapsedMs}ms';
    if (elapsedMs < 60000) return '${(elapsedMs / 1000).toStringAsFixed(1)}s';
    return '${(elapsedMs / 60000).toStringAsFixed(1)}m';
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.backgroundDark,
      body: SafeArea(
        child: _isLoading
            ? const Center(
                child: CircularProgressIndicator(color: AppColors.safeGreen),
              )
            : _errorMessage != null
                ? _buildErrorState()
                : RefreshIndicator(
                    onRefresh: _loadActivityData,
                    color: AppColors.safeGreen,
                    child: CustomScrollView(
                      physics: const AlwaysScrollableScrollPhysics(),
                      slivers: [
                        // Status Strip
                        SliverToBoxAdapter(
                          child: StatusStrip(
                            isConnected: _isConnected,
                            serverAddress: _isConnected ? _serverAddress : null,
                            isDangerous: false,
                          ),
                        ),

                        // Header
                        SliverToBoxAdapter(child: _buildHeader()),

                        // Device Summary
                        if (_deviceActivities.any((d) => (d['running_task_count'] ?? 0) > 0))
                          SliverToBoxAdapter(child: _buildDeviceSummary()),

                        // Section Header
                        SliverToBoxAdapter(
                          child: Padding(
                            padding: const EdgeInsets.fromLTRB(20, 16, 20, 12),
                            child: _SectionHeader(
                              label: _runningTasks.isNotEmpty
                                  ? 'RUNNING TASKS (${_runningTasks.length})'
                                  : 'RUNNING TASKS',
                            ),
                          ),
                        ),

                        // Task List or Empty State
                        if (_runningTasks.isEmpty)
                          SliverFillRemaining(
                            hasScrollBody: false,
                            child: _buildEmptyState(),
                          )
                        else
                          SliverPadding(
                            padding: const EdgeInsets.symmetric(horizontal: 20),
                            sliver: SliverList(
                              delegate: SliverChildBuilderDelegate(
                                (context, index) {
                                  final task = _runningTasks[index];
                                  return _RunningTaskCard(
                                    task: task,
                                    formatElapsedMs: _formatElapsedMs,
                                  ).animate().fadeIn(delay: (40 * index).ms);
                                },
                                childCount: _runningTasks.length,
                              ),
                            ),
                          ),

                        // Bottom padding
                        const SliverPadding(padding: EdgeInsets.only(bottom: 80)),
                      ],
                    ),
                  ),
      ),
    );
  }

  Widget _buildHeader() {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 20, 20, 8),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Row(
            children: [
              ThreeDBadgeIcon(
                icon: LucideIcons.layers,
                size: 14,
                accentColor: _runningTasks.isNotEmpty
                    ? AppColors.warningAmber
                    : AppColors.mutedIcon,
              ),
              const SizedBox(width: 12),
              Text(
                'TASK MONITOR',
                style: GoogleFonts.inter(
                  fontSize: 14,
                  fontWeight: FontWeight.w900,
                  color: AppColors.textPrimary,
                  letterSpacing: 1,
                ),
              ),
            ],
          ),
          Row(
            children: [
              // Live indicator
              if (_runningTasks.isNotEmpty)
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  decoration: BoxDecoration(
                    color: AppColors.safeGreen.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(4),
                    border: Border.all(color: AppColors.safeGreen.withOpacity(0.3)),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Container(
                        width: 6,
                        height: 6,
                        decoration: const BoxDecoration(
                          color: AppColors.safeGreen,
                          shape: BoxShape.circle,
                        ),
                      )
                          .animate(onPlay: (c) => c.repeat(reverse: true))
                          .fade(begin: 0.4, end: 1.0, duration: 1.seconds),
                      const SizedBox(width: 6),
                      Text(
                        'LIVE',
                        style: GoogleFonts.jetBrainsMono(
                          fontSize: 9,
                          fontWeight: FontWeight.w800,
                          color: AppColors.safeGreen,
                        ),
                      ),
                    ],
                  ),
                ),
              const SizedBox(width: 8),
              IconButton(
                onPressed: _loadActivityData,
                icon: const Icon(LucideIcons.refreshCw, size: 16),
                color: AppColors.mutedIcon,
                tooltip: 'Refresh',
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildDeviceSummary() {
    final activeDevices = _deviceActivities
        .where((d) => (d['running_task_count'] ?? 0) > 0)
        .toList();

    if (activeDevices.isEmpty) return const SizedBox.shrink();

    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 8),
      child: GlassContainer(
        padding: const EdgeInsets.all(16),
        borderRadius: 16,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'ACTIVE DEVICES',
              style: GoogleFonts.inter(
                fontSize: 9,
                fontWeight: FontWeight.w800,
                color: AppColors.mutedIcon,
                letterSpacing: 1.5,
              ),
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: activeDevices.map((device) {
                final taskCount = device['running_task_count'] as int? ?? 0;
                return Container(
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                  decoration: BoxDecoration(
                    color: AppColors.surface2,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: AppColors.outline),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Icon(LucideIcons.server, size: 12, color: AppColors.safeGreen),
                      const SizedBox(width: 8),
                      Text(
                        device['device_name']?.toString().toUpperCase() ?? 'UNKNOWN',
                        style: GoogleFonts.inter(
                          fontSize: 10,
                          fontWeight: FontWeight.w700,
                          color: AppColors.textPrimary,
                        ),
                      ),
                      const SizedBox(width: 8),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                        decoration: BoxDecoration(
                          color: AppColors.warningAmber.withOpacity(0.1),
                          borderRadius: BorderRadius.circular(4),
                        ),
                        child: Text(
                          '$taskCount',
                          style: GoogleFonts.jetBrainsMono(
                            fontSize: 10,
                            fontWeight: FontWeight.w800,
                            color: AppColors.warningAmber,
                          ),
                        ),
                      ),
                    ],
                  ),
                );
              }).toList(),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildEmptyState() {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(48),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ThreeDBadgeIcon(
              icon: LucideIcons.coffee,
              size: 32,
              accentColor: AppColors.mutedIcon,
            ),
            const SizedBox(height: 24),
            Text(
              'ALL QUIET',
              style: GoogleFonts.inter(
                fontSize: 16,
                fontWeight: FontWeight.w900,
                color: AppColors.textSecondary,
                letterSpacing: 1.5,
              ),
            ),
            const SizedBox(height: 12),
            Text(
              'No tasks are currently running.\nSubmit a job from Chat or Run screen.',
              style: GoogleFonts.inter(
                fontSize: 12,
                color: AppColors.mutedIcon,
                fontWeight: FontWeight.w600,
                height: 1.5,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 24),
            TextButton.icon(
              onPressed: _loadActivityData,
              icon: const Icon(LucideIcons.refreshCw, size: 14),
              label: Text(
                'REFRESH',
                style: GoogleFonts.inter(
                  fontSize: 11,
                  fontWeight: FontWeight.w800,
                  letterSpacing: 0.5,
                ),
              ),
              style: TextButton.styleFrom(
                foregroundColor: AppColors.safeGreen,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildErrorState() {
    return Center(
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
              onPressed: _loadActivityData,
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

class _RunningTaskCard extends StatelessWidget {
  final Map<String, dynamic> task;
  final String Function(int) formatElapsedMs;

  const _RunningTaskCard({
    required this.task,
    required this.formatElapsedMs,
  });

  IconData _getTaskIcon(String kind) {
    switch (kind.toUpperCase()) {
      case 'LLM_GENERATE':
        return LucideIcons.brain;
      case 'SYSINFO':
        return LucideIcons.activity;
      case 'ECHO':
        return LucideIcons.messageSquare;
      case 'SHELL':
        return LucideIcons.terminal;
      default:
        return LucideIcons.play;
    }
  }

  @override
  Widget build(BuildContext context) {
    final kind = task['kind'] as String? ?? 'TASK';
    final elapsedMs = task['elapsed_ms'] as int? ?? 0;
    final deviceName = task['device_name'] as String? ?? 'Unknown';
    final jobId = task['job_id'] as String? ?? '';
    final taskId = task['task_id'] as String? ?? '';
    final input = task['input'] as String? ?? '';

    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      child: GlassContainer(
        padding: EdgeInsets.zero,
        borderRadius: 16,
        child: ExpansionTile(
          tilePadding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
          iconColor: AppColors.mutedIcon,
          collapsedIconColor: AppColors.mutedIcon,
          leading: ThreeDBadgeIcon(
            icon: _getTaskIcon(kind),
            size: 14,
            accentColor: AppColors.warningAmber,
            useRotation: true,
          ),
          title: Text(
            kind,
            style: GoogleFonts.jetBrainsMono(
              fontSize: 14,
              fontWeight: FontWeight.w700,
              color: AppColors.textPrimary,
            ),
          ),
          subtitle: Text(
            '${deviceName.toUpperCase()} • ${formatElapsedMs(elapsedMs)}',
            style: GoogleFonts.inter(
              fontSize: 10,
              color: AppColors.textSecondary,
              fontWeight: FontWeight.w700,
              letterSpacing: 0.5,
            ),
          ),
          trailing: Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            decoration: BoxDecoration(
              color: AppColors.warningAmber.withOpacity(0.1),
              borderRadius: BorderRadius.circular(4),
              border: Border.all(color: AppColors.warningAmber.withOpacity(0.3)),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                SizedBox(
                  width: 8,
                  height: 8,
                  child: CircularProgressIndicator(
                    strokeWidth: 1.5,
                    color: AppColors.warningAmber,
                  ),
                ),
                const SizedBox(width: 6),
                Text(
                  'RUNNING',
                  style: GoogleFonts.jetBrainsMono(
                    color: AppColors.warningAmber,
                    fontSize: 9,
                    fontWeight: FontWeight.w800,
                  ),
                ),
              ],
            ),
          ),
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _buildDetailRow('JOB ID', jobId.length > 12 ? '${jobId.substring(0, 12)}...' : jobId),
                  _buildDetailRow('TASK ID', taskId.length > 12 ? '${taskId.substring(0, 12)}...' : taskId),
                  if (input.isNotEmpty)
                    _buildDetailRow('INPUT', input.length > 50 ? '${input.substring(0, 50)}...' : input),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildDetailRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 80,
            child: Text(
              label,
              style: GoogleFonts.inter(
                fontSize: 9,
                color: AppColors.mutedIcon,
                fontWeight: FontWeight.w800,
              ),
            ),
          ),
          Expanded(
            child: Text(
              value,
              style: GoogleFonts.jetBrainsMono(
                fontSize: 11,
                color: AppColors.textSecondary,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
