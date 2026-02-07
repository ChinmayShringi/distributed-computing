import 'package:flutter/material.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:lucide_icons/lucide_icons.dart';
import 'package:google_fonts/google_fonts.dart';
import '../../theme/app_colors.dart';
import 'glass_container.dart';

class ExecutionHostPanel extends StatefulWidget {
  final List<Map<String, dynamic>> executions;

  const ExecutionHostPanel({super.key, required this.executions});

  @override
  State<ExecutionHostPanel> createState() => _ExecutionHostPanelState();
}

class _ExecutionHostPanelState extends State<ExecutionHostPanel> {
  String? _selectedDevice;

  @override
  Widget build(BuildContext context) {
    if (widget.executions.isEmpty) {
      return GlassContainer(
        padding: const EdgeInsets.all(32),
        borderRadius: 16,
        child: Center(
          child: Text(
            'No executions yet',
            style: GoogleFonts.inter(fontSize: 12, color: AppColors.mutedIcon),
          ),
        ),
      );
    }

    final filteredExecutions = _selectedDevice == null
        ? widget.executions
        : widget.executions.where((e) => e['selected_device_id'] == _selectedDevice).toList();

    // Calculate device distribution
    final deviceCounts = <String, int>{};
    for (var exec in widget.executions) {
      final deviceName = exec['selected_device_name'] as String? ?? 'Unknown';
      deviceCounts[deviceName] = (deviceCounts[deviceName] ?? 0) + 1;
    }

    // Calculate compute distribution
    final computeCounts = <String, int>{};
    for (var exec in widget.executions) {
      final compute = ((exec['host_compute'] as String?) ?? 'unknown').toUpperCase();
      computeCounts[compute] = (computeCounts[compute] ?? 0) + 1;
    }

    // Calculate aggregate metrics with null safety
    final totalMemory = filteredExecutions.fold<double>(
      0,
      (sum, e) {
        final usage = e['resource_usage'] as Map?;
        if (usage == null) return sum;
        final memUsed = usage['memory_used_mb'];
        return sum + (memUsed is num ? memUsed.toDouble() : 0.0);
      },
    );
    final avgMemory = filteredExecutions.isEmpty ? 0.0 : totalMemory / filteredExecutions.length;

    final avgCpu = filteredExecutions.isEmpty
        ? 0.0
        : filteredExecutions.fold<double>(
              0,
              (sum, e) {
                final usage = e['resource_usage'] as Map?;
                if (usage == null) return sum;
                final cpuPercent = usage['cpu_percent'];
                return sum + (cpuPercent is num ? cpuPercent.toDouble() : 0.0);
              },
            ) /
            filteredExecutions.length;

    final avgTime = filteredExecutions.isEmpty
        ? 0.0
        : filteredExecutions.fold<double>(
              0,
              (sum, e) {
                final time = e['total_time_ms'];
                return sum + (time is num ? time.toDouble() : 0.0);
              },
            ) /
            filteredExecutions.length;

    return GlassContainer(
      padding: const EdgeInsets.all(20),
      borderRadius: 16,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Minimal Header
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                'Execution Telemetry',
                style: GoogleFonts.inter(
                  fontSize: 13,
                  fontWeight: FontWeight.w700,
                  color: AppColors.textPrimary,
                  letterSpacing: 0.3,
                ),
              ),
              if (_selectedDevice != null)
                TextButton(
                  onPressed: () => setState(() => _selectedDevice = null),
                  style: TextButton.styleFrom(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                    minimumSize: Size.zero,
                    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  ),
                  child: Text(
                    'Clear filter',
                    style: GoogleFonts.inter(
                      fontSize: 10,
                      fontWeight: FontWeight.w600,
                      color: AppColors.safeGreen,
                    ),
                  ),
                ),
            ],
          ),

          const SizedBox(height: 20),

          // Metrics Row - Simplified
          Row(
            children: [
              Expanded(child: _MetricCard(label: 'Avg Memory', value: '${avgMemory.toStringAsFixed(0)} MB')),
              const SizedBox(width: 12),
              Expanded(child: _MetricCard(label: 'Avg CPU', value: '${avgCpu.toStringAsFixed(1)}%')),
              const SizedBox(width: 12),
              Expanded(child: _MetricCard(label: 'Avg Time', value: '${avgTime.toStringAsFixed(1)} ms')),
            ],
          ),

          const SizedBox(height: 20),

          // Compute Distribution - Minimal Bars
          Text(
            'Compute Distribution',
            style: GoogleFonts.inter(fontSize: 10, fontWeight: FontWeight.w600, color: AppColors.textSecondary),
          ),
          const SizedBox(height: 8),
          ...computeCounts.entries.map((entry) => Padding(
                padding: const EdgeInsets.only(bottom: 6),
                child: _ComputeBar(
                  label: entry.key,
                  count: entry.value,
                  total: widget.executions.length,
                ),
              )),

          const SizedBox(height: 20),

          // Recent Executions - Clean List
          Text(
            'Recent Executions',
            style: GoogleFonts.inter(fontSize: 10, fontWeight: FontWeight.w600, color: AppColors.textSecondary),
          ),
          const SizedBox(height: 8),

          ...filteredExecutions.take(4).map((exec) => Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: _ExecutionRow(execution: exec),
              )),
        ],
      ),
    );
  }
}

class _MetricCard extends StatelessWidget {
  final String label;
  final String value;

  const _MetricCard({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppColors.surface2.withOpacity(0.5),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppColors.outline.withOpacity(0.3)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: GoogleFonts.inter(fontSize: 9, color: AppColors.mutedIcon, fontWeight: FontWeight.w500),
          ),
          const SizedBox(height: 4),
          Text(
            value,
            style: GoogleFonts.jetBrainsMono(fontSize: 14, fontWeight: FontWeight.w700, color: AppColors.textPrimary),
          ),
        ],
      ),
    );
  }
}

class _ComputeBar extends StatelessWidget {
  final String label;
  final int count;
  final int total;

  const _ComputeBar({required this.label, required this.count, required this.total});

  @override
  Widget build(BuildContext context) {
    final percentage = total > 0 ? (count / total * 100) : 0.0;
    final color = {
      'CPU': AppColors.infoBlue,
      'GPU': AppColors.safeGreen,
      'NPU': AppColors.warningAmber,
    }[label] ?? AppColors.mutedIcon;

    return Row(
      children: [
        SizedBox(
          width: 40,
          child: Text(
            label,
            style: GoogleFonts.inter(fontSize: 10, color: AppColors.textSecondary, fontWeight: FontWeight.w600),
          ),
        ),
        const SizedBox(width: 8),
        Expanded(
          child: Stack(
            children: [
              Container(
                height: 6,
                decoration: BoxDecoration(
                  color: AppColors.outline.withOpacity(0.2),
                  borderRadius: BorderRadius.circular(3),
                ),
              ),
              FractionallySizedBox(
                widthFactor: percentage / 100,
                child: Container(
                  height: 6,
                  decoration: BoxDecoration(
                    color: color,
                    borderRadius: BorderRadius.circular(3),
                  ),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(width: 8),
        SizedBox(
          width: 35,
          child: Text(
            '$count',
            textAlign: TextAlign.right,
            style: GoogleFonts.jetBrainsMono(fontSize: 10, fontWeight: FontWeight.w700, color: color),
          ),
        ),
      ],
    );
  }
}

class _ExecutionRow extends StatelessWidget {
  final Map<String, dynamic> execution;

  const _ExecutionRow({required this.execution});

  @override
  Widget build(BuildContext context) {
    final exitCode = execution['exit_code'] as int? ?? -1;
    final isSuccess = exitCode == 0;
    final compute = ((execution['host_compute'] as String?) ?? 'unknown').toUpperCase();
    final deviceName = execution['selected_device_name'] as String? ?? 'Unknown Device';
    final cmd = execution['cmd'] as String? ?? 'Unknown Command';
    
    final usage = execution['resource_usage'] as Map?;
    final memoryUsed = usage?['memory_used_mb'];
    final memoryStr = memoryUsed is num ? '${memoryUsed.toStringAsFixed(0)} MB' : 'N/A';
    
    final time = execution['total_time_ms'];
    final timeStr = time is num ? '${time.toStringAsFixed(1)} ms' : 'N/A';

    final computeColor = {
      'CPU': AppColors.infoBlue,
      'GPU': AppColors.safeGreen,
      'NPU': AppColors.warningAmber,
    }[compute] ?? AppColors.mutedIcon;

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppColors.surface2.withOpacity(0.3),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(
          color: isSuccess ? AppColors.safeGreen.withOpacity(0.2) : AppColors.primaryRed.withOpacity(0.2),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  deviceName,
                  style: GoogleFonts.inter(fontSize: 11, fontWeight: FontWeight.w700, color: AppColors.textPrimary),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(
                  color: isSuccess ? AppColors.safeGreen.withOpacity(0.15) : AppColors.primaryRed.withOpacity(0.15),
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Text(
                  isSuccess ? 'SUCCESS' : 'FAILED',
                  style: GoogleFonts.inter(
                    fontSize: 8,
                    fontWeight: FontWeight.w800,
                    color: isSuccess ? AppColors.safeGreen : AppColors.primaryRed,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 6),
          Text(
            cmd,
            style: GoogleFonts.jetBrainsMono(fontSize: 9, color: AppColors.textSecondary),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              _InfoChip(icon: LucideIcons.cpu, label: compute, color: computeColor),
              const SizedBox(width: 6),
              _InfoChip(icon: LucideIcons.database, label: memoryStr, color: AppColors.textSecondary),
              const SizedBox(width: 6),
              _InfoChip(icon: LucideIcons.clock, label: timeStr, color: AppColors.textSecondary),
            ],
          ),
        ],
      ),
    );
  }
}

class _InfoChip extends StatelessWidget {
  final IconData icon;
  final String label;
  final Color color;

  const _InfoChip({required this.icon, required this.label, required this.color});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 9, color: color),
        const SizedBox(width: 3),
        Text(
          label,
          style: GoogleFonts.inter(fontSize: 8, fontWeight: FontWeight.w600, color: color),
        ),
      ],
    );
  }
}
