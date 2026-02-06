import 'package:flutter/material.dart';
import '../../theme/app_colors.dart';
import 'three_d_badge_icon.dart';
import 'package:lucide_icons/lucide_icons.dart';

enum RiskLevel { low, medium, high }

class RiskBadge extends StatelessWidget {
  final RiskLevel level;

  const RiskBadge({super.key, required this.level});

  @override
  Widget build(BuildContext context) {
    final color = _getColor();
    final label = level.name.toUpperCase();

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        ThreeDBadgeIcon(
          icon: level == RiskLevel.high ? LucideIcons.shieldAlert : LucideIcons.shieldCheck,
          accentColor: color,
          size: 10,
          isDanger: level == RiskLevel.high,
        ),
        const SizedBox(width: 8),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
          decoration: BoxDecoration(
            color: color.withOpacity(0.1),
            borderRadius: BorderRadius.circular(4),
            border: Border.all(color: color.withOpacity(0.5)),
          ),
          child: Text(
            label,
            style: TextStyle(
              color: color,
              fontSize: 10,
              fontWeight: FontWeight.bold,
            ),
          ),
        ),
      ],
    );
  }

  Color _getColor() {
    switch (level) {
      case RiskLevel.low:
        return AppColors.safeGreen;
      case RiskLevel.medium:
        return AppColors.warningAmber;
      case RiskLevel.high:
        return AppColors.dangerPink;
    }
  }
}
