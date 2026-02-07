import 'package:flutter/material.dart';
import '../../theme/app_colors.dart';

class EdgeMeshWordmark extends StatelessWidget {
  final double? size;
  final double? fontSize;
  
  const EdgeMeshWordmark({
    super.key, 
    this.size,
    this.fontSize,
  });

  @override
  Widget build(BuildContext context) {
    final effectiveSize = size ?? fontSize ?? 18;
    
    return RichText(
      text: TextSpan(
        style: TextStyle(
          fontSize: effectiveSize,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.3,
          color: const Color(0xFFE6EDF6),
        ),
        children: const [
          TextSpan(
            text: 'EDGE',
            style: TextStyle(color: AppColors.safeGreen),
          ),
          TextSpan(text: ' '),
          TextSpan(
            text: 'MESH',
            style: TextStyle(color: AppColors.primaryRed),
          ),
        ],
      ),
    );
  }
}
