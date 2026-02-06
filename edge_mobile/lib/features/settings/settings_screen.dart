import 'package:flutter/material.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../theme/app_colors.dart';
import '../../shared/widgets/edge_mesh_wordmark.dart';
import '../../shared/widgets/status_strip.dart';
import '../../shared/widgets/mode_pill.dart';
import '../../shared/widgets/glass_container.dart';
import '../../shared/widgets/three_d_badge_icon.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(
          color: AppColors.backgroundDark,
          image: DecorationImage(
            image: NetworkImage('https://images.unsplash.com/photo-1550751827-4bd374c3f58b?auto=format&fit=crop&q=80&w=2670&ixlib=rb-4.0.3'),
            fit: BoxFit.cover,
            opacity: 0.03,
          ),
        ),
        child: SafeArea(
          child: ListView(
            padding: const EdgeInsets.all(16),
            physics: const BouncingScrollPhysics(),
            children: [
              Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    const EdgeMeshWordmark(fontSize: 20),
                    const ThreeDBadgeIcon(
                      icon: LucideIcons.settings,
                      accentColor: AppColors.primaryRed,
                      size: 14,
                      useRotation: true,
                    ),
                  ],
                ),
              ),
              const StatusStrip(
                isConnected: true,
                serverAddress: '192.168.1.10:50051',
                isDangerous: false,
              ),
              const SizedBox(height: 24),
              _buildSectionHeader('NODE SECURITY'),
              GlassContainer(
                padding: const EdgeInsets.all(16),
                child: Column(
                  children: [
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        const Text('Execution Mode', style: TextStyle(fontWeight: FontWeight.bold)),
                        const ModePill(isDangerous: false),
                      ],
                    ),
                    const SizedBox(height: 16),
                    Text(
                      'Safe Mode limits runtime toolsets to read-only or low-impact operations.',
                      style: TextStyle(color: AppColors.textSecondary.withOpacity(0.7), fontSize: 12),
                    ),
                    const SizedBox(height: 20),
                    SizedBox(
                      width: double.infinity,
                      child: OutlinedButton.icon(
                        onPressed: () => _showDangerousModeDialog(context),
                        style: OutlinedButton.styleFrom(
                          foregroundColor: AppColors.primaryRed,
                          side: const BorderSide(color: AppColors.primaryRed),
                          padding: const EdgeInsets.symmetric(vertical: 12),
                          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                        ),
                        icon: const Icon(LucideIcons.skull, size: 16),
                        label: const Text('ACTIVATE DANGEROUS MODE', style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, letterSpacing: 1)),
                      ),
                    ),
                  ],
                ),
              ),
              
              const SizedBox(height: 32),
              _buildSectionHeader('CORE CONFIG'),
              
              _GlassSettingsTile(
                icon: LucideIcons.fileCode,
                title: 'Manifest Paths',
                subtitle: '/etc/edge-mesh/registry.yaml',
                onTap: () {},
              ),
              _GlassSettingsTile(
                icon: LucideIcons.shield,
                title: 'Identity Store',
                subtitle: 'Manage node public keys',
                onTap: () {},
              ),
              _GlassSettingsTile(
                icon: LucideIcons.bug,
                title: 'Debugger',
                subtitle: 'Verbose trace collection',
                onTap: () {},
              ),
              
              const SizedBox(height: 32),
              _buildSectionHeader('APPLICATION'),
              _GlassSettingsTile(
                icon: LucideIcons.info,
                title: 'About Edge Mesh',
                subtitle: 'v1.1.0-alpha • Build 2026.02',
                onTap: () {},
              ),
               const SizedBox(height: 24),
               SizedBox(
                 width: double.infinity,
                 child: FilledButton(
                   onPressed: () {},
                   style: FilledButton.styleFrom(
                     backgroundColor: Colors.white.withOpacity(0.05),
                     foregroundColor: Colors.white,
                     padding: const EdgeInsets.symmetric(vertical: 16),
                     shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                   ),
                   child: const Text('TERMINATE SESSION', style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, letterSpacing: 1.5)),
                 ),
               ),
            ],
          ),
        ),
      ),
    );
  }

  void _showDangerousModeDialog(BuildContext context) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        backgroundColor: Colors.transparent,
        insetPadding: const EdgeInsets.all(20),
        content: GlassContainer(
          borderRadius: 24,
          padding: const EdgeInsets.all(24),
          border: Border.all(color: AppColors.primaryRed.withOpacity(0.5), width: 2),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const ThreeDBadgeIcon(
                icon: LucideIcons.alertTriangle,
                accentColor: AppColors.primaryRed,
                size: 32,
                isDanger: true,
              ),
              const SizedBox(height: 20),
              const Text(
                'UNRESTRICTED ACCESS',
                style: TextStyle(color: AppColors.primaryRed, fontSize: 18, fontWeight: FontWeight.bold, letterSpacing: 1),
              ),
              const SizedBox(height: 16),
              const Text(
                'Dangerous mode bypasses all security checks for tools. Actions may be irreversible.',
                style: TextStyle(color: AppColors.textPrimary, fontSize: 13),
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 24),
              GlassContainer(
                padding: const EdgeInsets.symmetric(horizontal: 16),
                borderRadius: 12,
                opacity: 0.05,
                child: TextField(
                  decoration: InputDecoration(
                    hintText: 'Type "ACTIVATE"',
                    hintStyle: TextStyle(color: AppColors.textSecondary.withOpacity(0.3)),
                    border: InputBorder.none,
                  ),
                ),
              ),
              const SizedBox(height: 24),
              Row(
                children: [
                   Expanded(
                    child: TextButton(
                      onPressed: () => Navigator.pop(context),
                      child: const Text('ABORT', style: TextStyle(color: AppColors.textSecondary)),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: FilledButton(
                      onPressed: () => Navigator.pop(context),
                      style: FilledButton.styleFrom(backgroundColor: AppColors.primaryRed),
                      child: const Text('CONFIRM'),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildSectionHeader(String title) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12, left: 4),
      child: Text(
        title,
        style: const TextStyle(
          color: AppColors.textSecondary,
          fontSize: 10,
          fontWeight: FontWeight.bold,
          letterSpacing: 1.5,
        ),
      ),
    );
  }
}

class _GlassSettingsTile extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;

  const _GlassSettingsTile({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: GlassContainer(
        padding: EdgeInsets.zero,
        borderRadius: 16,
        child: ListTile(
          leading: ThreeDBadgeIcon(
            icon: icon,
            accentColor: AppColors.textSecondary,
            size: 14,
          ),
          title: Text(title, style: const TextStyle(fontSize: 14, fontWeight: FontWeight.bold)),
          subtitle: Text(
            subtitle,
            style: const TextStyle(fontSize: 11, color: AppColors.textSecondary),
          ),
          trailing: const Icon(LucideIcons.chevronRight, size: 16, color: AppColors.textSecondary),
          onTap: onTap,
        ),
      ),
    );
  }
}
