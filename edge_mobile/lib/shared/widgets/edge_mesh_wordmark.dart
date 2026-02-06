import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import '../../theme/app_colors.dart';

class EdgeMeshWordmark extends StatelessWidget {
  final double fontSize;

  const EdgeMeshWordmark({super.key, this.fontSize = 24});

  @override
  Widget build(BuildContext context) {
    return RichText(
      text: TextSpan(
        style: GoogleFonts.orbitron(
          fontSize: fontSize,
          fontWeight: FontWeight.w900,
          letterSpacing: 2.0,
        ),
        children: [
          TextSpan(
            text: 'EDGE',
            style: TextStyle(
              color: AppColors.primaryRed,
              shadows: [
                Shadow(
                  color: AppColors.primaryRed.withOpacity(0.5),
                  blurRadius: 10,
                ),
              ],
            ),
          ),
          const TextSpan(text: ' '),
          const TextSpan(
            text: 'MESH',
            style: TextStyle(color: AppColors.textPrimary),
          ),
        ],
      ),
    );
  }
}
