import 'package:flutter/material.dart';

class AppColors {
  // --- ENTERPRISE TECH PALETTE ---
  
  // Base Architecture
  static const Color backgroundDark = Color(0xFF0A0D12); // Deep Charcoal
  static const Color surface1 = Color(0xFF10141B);      // Primary Surface
  static const Color surface2 = Color(0xFF161B22);      // Secondary Surface
  static const Color surfaceVariant = Color(0xFF1F242C); // Tertiary Surface
  
  // Orchestration Tokens (Accents)
  static const Color safeGreen = Color(0xFF25F29B);      // Safe / Authorized
  static const Color primaryRed = Color(0xFFFF3B6B);     // Edge Mesh Pink/Red
  static const Color warningAmber = Color(0xFFFFB800);   // Caution
  static const Color infoBlue = Color(0xFF3B82F6);       // Informational
  static const Color dangerPink = Color(0xFFFF2D55);     // High Alert (Alternative)
  
  // Typography & Icons
  static const Color textPrimary = Color(0xFFF9FAFB);    // High Emphasis
  static const Color textSecondary = Color(0xFF94A3B8);  // Medium Emphasis
  static const Color mutedIcon = Color(0xFF64748B);     // Low Emphasis
  
  // UI Framework
  static const Color outline = Color(0xFF30363D);        // Surgical Borders
  static const Color glassSurface = Color(0x1A64748B);   // 10% Slate
  static const Color glassBorder = Color(0x3394A3B8);    // 20% Slate
  
  // Legacy Shorthands (for compatibility)
  static const Color surface = surface1;
  static const Color white = Colors.white;

  // Global Gradients
  static const LinearGradient meshGradient = LinearGradient(
    colors: [primaryRed, Color(0xFF8B5CF6)],
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
  );
}
