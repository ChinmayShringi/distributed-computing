import 'package:flutter/material.dart';
import '../../theme/app_colors.dart';
import 'mode_pill.dart';

class StatusStrip extends StatelessWidget {
  final bool isConnected;
  final String? serverAddress;
  final bool isDangerous;

  const StatusStrip({
    super.key,
    required this.isConnected,
    this.serverAddress,
    required this.isDangerous,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
      decoration: const BoxDecoration(
        color: AppColors.backgroundDark,
        border: Border(
          bottom: BorderSide(color: AppColors.outline),
        ),
      ),
      child: Row(
        children: [
          // Connection Status
          Container(
            width: 8,
            height: 8,
            decoration: BoxDecoration(
              color: isConnected ? AppColors.safeGreen : AppColors.outline,
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: 8),
          Text(
            isConnected ? 'CONNECTED' : 'OFFLINE',
            style: TextStyle(
              color: isConnected ? AppColors.safeGreen : AppColors.textSecondary,
              fontSize: 10,
              fontWeight: FontWeight.bold,
            ),
          ),
          if (serverAddress != null) ...[
            const SizedBox(width: 8),
            Text(
              '•  $serverAddress',
              style: const TextStyle(
                color: AppColors.textSecondary,
                fontSize: 10,
                fontFamily: 'Roboto Mono',
              ),
            ),
          ],
          const Spacer(),
          // Mode Indicator
          if (isConnected)
            ModePill(isDangerous: isDangerous),
        ],
      ),
    );
  }
}
