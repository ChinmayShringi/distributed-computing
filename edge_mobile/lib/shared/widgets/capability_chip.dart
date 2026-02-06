import 'package:flutter/material.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../theme/app_colors.dart';
import 'package:google_fonts/google_fonts.dart';

enum DeviceCapability {
  cpu('CPU', LucideIcons.cpu),
  gpu('GPU', LucideIcons.circuitBoard),
  npu('NPU', LucideIcons.brain),
  screen('VIEW', LucideIcons.monitor),
  memory('RAM', LucideIcons.memoryStick),
  storage('SSD', LucideIcons.hardDrive),
  network('NET', LucideIcons.network),
  sensors('SNS', LucideIcons.gauge),
  unknown('?', LucideIcons.helpCircle);

  final String label;
  final IconData icon;
  const DeviceCapability(this.label, this.icon);

  static DeviceCapability fromString(String value) {
    return DeviceCapability.values.firstWhere(
      (e) => e.name.toLowerCase() == value.toLowerCase(),
      orElse: () => DeviceCapability.unknown,
    );
  }
}

class CapabilityChip extends StatelessWidget {
  final DeviceCapability capability;

  const CapabilityChip({
    super.key,
    required this.capability,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: AppColors.surface2,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppColors.outline.withOpacity(0.5)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            capability.icon,
            size: 10,
            color: AppColors.safeGreen,
          ),
          const SizedBox(width: 6),
          Text(
            capability.label,
            style: GoogleFonts.inter(
              fontSize: 8,
              fontWeight: FontWeight.w800,
              color: AppColors.textSecondary,
              letterSpacing: 0.5,
            ),
          ),
        ],
      ),
    );
  }
}
