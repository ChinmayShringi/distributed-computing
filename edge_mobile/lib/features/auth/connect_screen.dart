import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:lucide_icons/lucide_icons.dart';
import 'package:google_fonts/google_fonts.dart';
import '../../theme/app_colors.dart';
import '../../shared/widgets/edge_mesh_wordmark.dart';
import '../../shared/widgets/glass_container.dart';

class ConnectScreen extends StatefulWidget {
  const ConnectScreen({super.key});

  @override
  State<ConnectScreen> createState() => _ConnectScreenState();
}

class _ConnectScreenState extends State<ConnectScreen> {
  final _serverController = TextEditingController(text: '192.168.1.10:50051');
  final _keyController = TextEditingController(text: 'dev');
  bool _isObscured = true;
  bool _isConnecting = false;
  bool _isConnected = false;

  void _handleConnect() async {
    setState(() {
      _isConnecting = true;
    });

    // Simulate connection handshake
    await Future.delayed(const Duration(milliseconds: 1500));

    if (mounted) {
      setState(() {
        _isConnecting = false;
        _isConnected = true;
      });
    }

    // Brief delay to show success state before navigating
    await Future.delayed(const Duration(milliseconds: 800));

    if (mounted) {
      context.go('/chat');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.backgroundDark,
      body: Stack(
        children: [
          // Background Vignette
          Positioned.fill(
            child: Container(
              decoration: const BoxDecoration(
                gradient: RadialGradient(
                  center: Alignment.center,
                  radius: 1.2,
                  colors: [
                    Color(0xFF161B22), // Surface 2
                    AppColors.backgroundDark,
                  ],
                ),
              ),
            ),
          ),
          
          SafeArea(
            child: SingleChildScrollView(
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 40),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [
                  const SizedBox(height: 40),
                  
                  // Top Branding
                  const EdgeMeshWordmark(fontSize: 28),
                  const SizedBox(height: 8),
                  Text(
                    'SECURE ORCHESTRATOR CONSOLE',
                    style: GoogleFonts.inter(
                      fontSize: 10,
                      fontWeight: FontWeight.w900,
                      color: AppColors.mutedIcon,
                      letterSpacing: 2,
                    ),
                  ),
                  
                  const SizedBox(height: 60),

                  // Main Login Console
                  GlassContainer(
                    padding: const EdgeInsets.all(32),
                    borderRadius: 24,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Row(
                          children: [
                            Container(
                              padding: const EdgeInsets.all(10),
                              decoration: BoxDecoration(
                                color: AppColors.safeGreen.withOpacity(0.1),
                                shape: BoxShape.circle,
                              ),
                              child: const Icon(LucideIcons.shieldCheck, size: 20, color: AppColors.safeGreen),
                            ),
                            const SizedBox(width: 16),
                            Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  'AUTHENTICATION',
                                  style: GoogleFonts.inter(
                                    fontSize: 13,
                                    fontWeight: FontWeight.w900,
                                    letterSpacing: 0.5,
                                  ),
                                ),
                                Text(
                                  'AES-256 HANDSHAKE READY',
                                  style: GoogleFonts.jetBrainsMono(
                                    fontSize: 9,
                                    color: AppColors.safeGreen,
                                    fontWeight: FontWeight.bold,
                                  ),
                                ),
                              ],
                            ),
                          ],
                        ),
                        
                        const SizedBox(height: 40),
                        
                        // Form
                        _ConsoleInput(
                          controller: _serverController,
                          label: 'GATEWAY ADDRESS',
                          icon: LucideIcons.server,
                          hint: '0.0.0.0:0000',
                        ),
                        const SizedBox(height: 24),
                        _ConsoleInput(
                          controller: _keyController,
                          label: 'ACCESS SIGNATURE',
                          icon: LucideIcons.fileKey,
                          hint: 'RSA-IDENTITY-TOKEN',
                          isPassword: true,
                          isObscured: _isObscured,
                          onToggleVisibility: () => setState(() => _isObscured = !_isObscured),
                        ),
                        
                        const SizedBox(height: 40),
                        
                        // Primary Action
                        FilledButton(
                          onPressed: (_isConnecting || _isConnected) ? null : _handleConnect,
                          style: FilledButton.styleFrom(
                            backgroundColor: _isConnected ? AppColors.safeGreen : AppColors.safeGreen,
                            foregroundColor: AppColors.backgroundDark,
                            padding: const EdgeInsets.symmetric(vertical: 20),
                            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                            elevation: 0,
                            disabledBackgroundColor: _isConnected ? AppColors.safeGreen : AppColors.safeGreen.withOpacity(0.5),
                            disabledForegroundColor: AppColors.backgroundDark,
                          ),
                          child: AnimatedSwitcher(
                            duration: const Duration(milliseconds: 200),
                            child: _isConnected
                                ? Row(
                                    mainAxisAlignment: MainAxisAlignment.center,
                                    children: [
                                      const Icon(LucideIcons.check, size: 18),
                                      const SizedBox(width: 8),
                                      Text(
                                        'CONNECTION ESTABLISHED',
                                        style: GoogleFonts.inter(
                                          fontSize: 13,
                                          fontWeight: FontWeight.w900,
                                          letterSpacing: 1,
                                        ),
                                      ),
                                    ],
                                  )
                                : _isConnecting
                                    ? const SizedBox(
                                        height: 18,
                                        width: 18,
                                        child: CircularProgressIndicator(
                                          strokeWidth: 2.5,
                                          color: AppColors.backgroundDark,
                                        ),
                                      )
                                    : Text(
                                        'INITIALIZE CONNECTION',
                                        style: GoogleFonts.inter(
                                          fontSize: 13,
                                          fontWeight: FontWeight.w900,
                                          letterSpacing: 1,
                                        ),
                                      ),
                          ),
                        ),
                      ],
                    ),
                  ).animate().fadeIn(duration: 400.ms).slideY(begin: 0.05, end: 0),

                  const SizedBox(height: 40),
                  
                  // Security Attributes Section
                  Row(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      _SecurityBadge(icon: LucideIcons.check, label: 'APP CHECK'),
                      const SizedBox(width: 12),
                      _SecurityBadge(icon: LucideIcons.lock, label: 'RBAC ACTIVE'),
                      const SizedBox(width: 12),
                      _SecurityBadge(icon: LucideIcons.activity, label: 'mTLS'),
                    ],
                  ).animate().fadeIn(delay: 400.ms),

                  const SizedBox(height: 60),
                  
                  // Footer Technical Info
                  Text(
                    'SESSION ID: HX-992-004-X',
                    style: GoogleFonts.jetBrainsMono(
                      fontSize: 10,
                      color: AppColors.mutedIcon,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _ConsoleInput extends StatelessWidget {
  final TextEditingController controller;
  final String label;
  final IconData icon;
  final String hint;
  final bool isPassword;
  final bool isObscured;
  final VoidCallback? onToggleVisibility;

  const _ConsoleInput({
    required this.controller,
    required this.label,
    required this.icon,
    required this.hint,
    this.isPassword = false,
    this.isObscured = false,
    this.onToggleVisibility,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: GoogleFonts.inter(
            fontSize: 9,
            fontWeight: FontWeight.w900,
            color: AppColors.textSecondary,
            letterSpacing: 1.5,
          ),
        ),
        const SizedBox(height: 12),
        Container(
          decoration: BoxDecoration(
            color: AppColors.surface2,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: AppColors.outline),
          ),
          child: TextField(
            controller: controller,
            obscureText: isObscured,
            style: GoogleFonts.jetBrainsMono(fontSize: 14, color: AppColors.textPrimary),
            decoration: InputDecoration(
              prefixIcon: Icon(icon, size: 18, color: AppColors.mutedIcon),
              suffixIcon: isPassword
                  ? IconButton(
                      icon: Icon(isObscured ? LucideIcons.eye : LucideIcons.eyeOff, size: 18, color: AppColors.mutedIcon),
                      onPressed: onToggleVisibility,
                    )
                  : null,
              hintText: hint,
              hintStyle: GoogleFonts.jetBrainsMono(color: AppColors.mutedIcon.withOpacity(0.5), fontSize: 13),
              border: InputBorder.none,
              contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 18),
            ),
          ),
        ),
      ],
    );
  }
}

class _SecurityBadge extends StatelessWidget {
  final IconData icon;
  final String label;
  const _SecurityBadge({required this.icon, required this.label});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: AppColors.surface2,
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: AppColors.outline),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 10, color: AppColors.mutedIcon),
          const SizedBox(width: 6),
          Text(
            label,
            style: GoogleFonts.inter(
              fontSize: 8,
              fontWeight: FontWeight.w800,
              color: AppColors.mutedIcon,
              letterSpacing: 0.5,
            ),
          ),
        ],
      ),
    );
  }
}
