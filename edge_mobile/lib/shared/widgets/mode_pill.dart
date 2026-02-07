import 'package:flutter/material.dart';
import '../../theme/app_colors.dart';

class ModePill extends StatelessWidget {
  final bool isDangerous;

  const ModePill({super.key, required this.isDangerous});

  @override
  Widget build(BuildContext context) {
    final color = isDangerous ? AppColors.dangerPink : AppColors.safeGreen;
    final label = isDangerous ? 'DANGEROUS' : 'SAFE MODE';

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: color.withOpacity(0.1),
        border: Border.all(color: color.withOpacity(0.5)),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 6,
            height: 6,
            decoration: BoxDecoration(
              color: color,
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: 6),
          Text(
            label,
            style: TextStyle(
              color: color,
              fontSize: 10,
              fontWeight: FontWeight.bold,
              letterSpacing: 0.5,
            ),
          ),
        ],
      ),
    );
  }
}
