import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../../theme/app_colors.dart';
import '../../../shared/widgets/glass_container.dart';
import '../../../shared/widgets/three_d_badge_icon.dart';
import '../../../shared/widgets/terminal_panel.dart';

class PlanCard extends StatelessWidget {
  final Map<String, dynamic> payload;
  final VoidCallback onRun;
  final VoidCallback onEdit;

  const PlanCard({
    super.key,
    required this.payload,
    required this.onRun,
    required this.onEdit,
  });

  @override
  Widget build(BuildContext context) {
    final steps = payload['steps'] as List;
    final risk = payload['risk'] ?? 'SAFE';
    final cmd = payload['cmd'] ?? '';
    final device = payload['device'] ?? '';

    return GlassContainer(
      padding: const EdgeInsets.all(20),
      borderRadius: 20,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                'PROPOSED PLAN',
                style: GoogleFonts.inter(
                  fontSize: 10,
                  fontWeight: FontWeight.w900,
                  color: AppColors.mutedIcon,
                  letterSpacing: 1.5,
                ),
              ),
              _RiskBadge(label: risk.toUpperCase()),
            ],
          ),
          const SizedBox(height: 16),
          Text(
            cmd,
            style: GoogleFonts.jetBrainsMono(
              fontSize: 14,
              fontWeight: FontWeight.bold,
              color: AppColors.textPrimary,
            ),
          ),
          const SizedBox(height: 12),
          Text(
            'Target: $device',
            style: GoogleFonts.inter(
              fontSize: 11,
              color: AppColors.textSecondary,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 20),
          ...steps.asMap().entries.map((entry) => Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      '${entry.key + 1}.',
                      style: GoogleFonts.jetBrainsMono(
                        fontSize: 11,
                        color: AppColors.safeGreen,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        entry.value,
                        style: GoogleFonts.inter(
                          fontSize: 12,
                          color: AppColors.textPrimary,
                        ),
                      ),
                    ),
                  ],
                ),
              )),
          const SizedBox(height: 20),
          Row(
            children: [
              Expanded(
                child: FilledButton(
                  onPressed: onRun,
                  style: FilledButton.styleFrom(
                    backgroundColor: AppColors.safeGreen,
                    foregroundColor: AppColors.backgroundDark,
                    padding: const EdgeInsets.symmetric(vertical: 16),
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                  ),
                  child: Text('RUN', style: GoogleFonts.inter(fontWeight: FontWeight.w900, fontSize: 12, letterSpacing: 1)),
                ),
              ),
              const SizedBox(width: 12),
              IconButton(
                onPressed: onEdit,
                icon: const Icon(LucideIcons.settings2, size: 20),
                style: IconButton.styleFrom(
                  backgroundColor: AppColors.surface2,
                  padding: const EdgeInsets.all(16),
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class DeviceListCard extends StatelessWidget {
  final Map<String, dynamic> payload;

  const DeviceListCard({super.key, required this.payload});

  @override
  Widget build(BuildContext context) {
    final devices = payload['devices'] as List;

    return Column(
      children: devices.map((device) => Padding(
        padding: const EdgeInsets.only(bottom: 12),
        child: GlassContainer(
          padding: const EdgeInsets.all(16),
          borderRadius: 16,
          child: Row(
            children: [
              ThreeDBadgeIcon(
                icon: device['type'] == 'mobile' ? LucideIcons.smartphone : LucideIcons.laptop,
                accentColor: device['status'] == 'online' ? AppColors.safeGreen : AppColors.mutedIcon,
                size: 16,
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      device['name'],
                      style: GoogleFonts.inter(fontWeight: FontWeight.bold, fontSize: 13),
                    ),
                    Text(
                      '${device['os']} • ${device['status'].toString().toUpperCase()}',
                      style: GoogleFonts.inter(fontSize: 10, color: AppColors.textSecondary),
                    ),
                  ],
                ),
              ),
              const Icon(LucideIcons.chevronRight, size: 16, color: AppColors.mutedIcon),
            ],
          ),
        ),
      )).toList(),
    );
  }
}

class ResultCard extends StatelessWidget {
  final Map<String, dynamic> payload;

  const ResultCard({super.key, required this.payload});

  @override
  Widget build(BuildContext context) {
    return GlassContainer(
      padding: const EdgeInsets.all(20),
      borderRadius: 20,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                'EXECUTION SUCCESS',
                style: GoogleFonts.inter(
                  fontSize: 10,
                  fontWeight: FontWeight.w900,
                  color: AppColors.safeGreen,
                  letterSpacing: 1.5,
                ),
              ),
              Text(
                payload['time'],
                style: GoogleFonts.jetBrainsMono(fontSize: 10, color: AppColors.mutedIcon),
              ),
            ],
          ),
          const SizedBox(height: 16),
          _MetricRow(
            device: payload['device'],
            compute: payload['host_compute'],
          ),
          const SizedBox(height: 20),
          TerminalPanel(
            output: payload['output'],
            exitCode: payload['exit_code'],
          ),
        ],
      ),
    );
  }
}

class _RiskBadge extends StatelessWidget {
  final String label;
  const _RiskBadge({required this.label});

  @override
  Widget build(BuildContext context) {
    final isSafe = label == 'SAFE' || label == 'LOW';
    final color = isSafe ? AppColors.safeGreen : AppColors.warningAmber;
    
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.1),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: color.withOpacity(0.3)),
      ),
      child: Text(
        label,
        style: GoogleFonts.inter(
          fontSize: 8,
          fontWeight: FontWeight.w900,
          color: color,
        ),
      ),
    );
  }
}

class _MetricRow extends StatelessWidget {
  final String device;
  final String compute;

  const _MetricRow({required this.device, required this.compute});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        _MetricChip(icon: LucideIcons.smartphone, label: device),
        const SizedBox(width: 8),
        _MetricChip(icon: LucideIcons.cpu, label: compute),
      ],
    );
  }
}

class _MetricChip extends StatelessWidget {
  final IconData icon;
  final String label;

  const _MetricChip({required this.icon, required this.label});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
      decoration: BoxDecoration(
        color: AppColors.surface2,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 12, color: AppColors.mutedIcon),
          const SizedBox(width: 6),
          Text(
            label,
            style: GoogleFonts.inter(fontSize: 10, fontWeight: FontWeight.bold, color: AppColors.textSecondary),
          ),
        ],
      ),
    );
  }
}
