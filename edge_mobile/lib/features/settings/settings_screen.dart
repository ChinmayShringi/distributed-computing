import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../theme/app_colors.dart';
import '../../shared/widgets/edge_mesh_wordmark.dart';
import '../../shared/widgets/status_strip.dart';
import '../../shared/widgets/mode_pill.dart';
import '../../shared/widgets/glass_container.dart';
import '../../shared/widgets/three_d_badge_icon.dart';
import '../../services/grpc_service.dart';

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  final _grpcService = GrpcService();
  bool _workerEnabled = false;
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _loadWorkerStatus();
  }

  Future<void> _loadWorkerStatus() async {
    try {
      final isRunning = await _grpcService.isWorkerRunning();
      setState(() {
        _workerEnabled = isRunning;
        _isLoading = false;
      });
    } catch (e) {
      setState(() {
        _isLoading = false;
      });
    }
  }

  Future<void> _toggleWorker() async {
    try {
      if (_workerEnabled) {
        await _grpcService.stopWorker();
        setState(() => _workerEnabled = false);
      } else {
        // When enabling worker, also request screen capture permission
        final granted = await _requestScreenCapturePermission();
        if (granted) {
          await _grpcService.startWorker();
          setState(() => _workerEnabled = true);
          
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(
                content: Text('Worker enabled with screen capture'),
                backgroundColor: AppColors.safeGreen,
              ),
            );
          }
        } else {
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(
                content: Text('Screen capture permission denied. Worker started without streaming.'),
                backgroundColor: AppColors.warningAmber,
              ),
            );
          }
          // Start worker anyway, just without screen capture
          await _grpcService.startWorker();
          setState(() => _workerEnabled = true);
        }
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to toggle worker: $e')),
        );
      }
    }
  }

  Future<bool> _requestScreenCapturePermission() async {
    try {
      // Show dialog explaining the permission
      final shouldRequest = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          backgroundColor: AppColors.surface2,
          title: const Text(
            'Enable Screen Streaming',
            style: TextStyle(color: AppColors.textPrimary),
          ),
          content: const Text(
            'To allow remote viewing of this device, we need permission to capture the screen. '
            'This enables WebRTC streaming when requested by the orchestrator.',
            style: TextStyle(color: AppColors.textSecondary),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('SKIP', style: TextStyle(color: AppColors.mutedIcon)),
            ),
            ElevatedButton(
              onPressed: () => Navigator.pop(context, true),
              style: ElevatedButton.styleFrom(
                backgroundColor: AppColors.safeGreen,
                foregroundColor: Colors.black,
              ),
              child: const Text('GRANT PERMISSION'),
            ),
          ],
        ),
      );
      
      if (shouldRequest == true) {
        // Request the actual permission
        return await _grpcService.requestScreenCapture();
      }
      
      return false;
    } catch (e) {
      debugPrint("Failed to request screen capture: $e");
      return false;
    }
  }

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
              _buildSectionHeader('WORKER MODE'),
              GlassContainer(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        const Row(
                          children: [
                            ThreeDBadgeIcon(
                              icon: LucideIcons.cpu,
                              accentColor: AppColors.safeGreen,
                              size: 14,
                            ),
                            SizedBox(width: 12),
                            Text('Enable Worker', style: TextStyle(fontWeight: FontWeight.bold)),
                          ],
                        ),
                        _isLoading
                            ? const SizedBox(
                                width: 20,
                                height: 20,
                                child: CircularProgressIndicator(strokeWidth: 2),
                              )
                            : Switch(
                                value: _workerEnabled,
                                onChanged: (value) => _toggleWorker(),
                                activeColor: AppColors.safeGreen,
                              ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    Text(
                      _workerEnabled
                          ? 'This device is accepting and executing distributed tasks.'
                          : 'Allow this device to execute tasks from the orchestrator.',
                      style: TextStyle(color: AppColors.textSecondary.withOpacity(0.7), fontSize: 12),
                    ),
                    if (_workerEnabled) ...[
                      const SizedBox(height: 16),
                      Container(
                        padding: const EdgeInsets.all(12),
                        decoration: BoxDecoration(
                          color: AppColors.safeGreen.withOpacity(0.1),
                          borderRadius: BorderRadius.circular(8),
                          border: Border.all(color: AppColors.safeGreen.withOpacity(0.3)),
                        ),
                        child: const Row(
                          children: [
                            Icon(LucideIcons.checkCircle, size: 14, color: AppColors.safeGreen),
                            SizedBox(width: 8),
                            Expanded(
                              child: Text(
                                'Worker service active - Broadcasting to network',
                                style: TextStyle(color: AppColors.safeGreen, fontSize: 11, fontWeight: FontWeight.w600),
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
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
                   onPressed: () async {
                     try {
                       await _grpcService.closeConnection();
                       if (mounted) context.go('/connect');
                     } catch (e) {
                       if (mounted) {
                         ScaffoldMessenger.of(context).showSnackBar(
                           SnackBar(content: Text('Failed to disconnect: $e')),
                         );
                       }
                     }
                   },
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
