import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:google_fonts/google_fonts.dart';
import '../../theme/app_colors.dart';

class TerminalPanel extends StatelessWidget {
  final String output;
  final int? exitCode;

  const TerminalPanel({
    super.key,
    required this.output,
    this.exitCode,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: const Color(0xFF000000),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppColors.outline),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Header
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: const BoxDecoration(
              border: Border(bottom: BorderSide(color: AppColors.outline)),
            ),
            child: Row(
              children: [
                const Icon(Icons.terminal, size: 16, color: AppColors.textSecondary),
                const SizedBox(width: 8),
                const Text(
                  'OUTPUT',
                  style: TextStyle(
                    color: AppColors.textSecondary,
                    fontSize: 10,
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const Spacer(),
                if (exitCode != null)
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                    decoration: BoxDecoration(
                      color: (exitCode == 0 ? AppColors.safeGreen : AppColors.dangerPink)
                          .withOpacity(0.1),
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Text(
                      'EXIT $exitCode',
                      style: TextStyle(
                        color: exitCode == 0 ? AppColors.safeGreen : AppColors.dangerPink,
                        fontSize: 10,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ),
                const SizedBox(width: 8),
                InkWell(
                  onTap: () {
                    Clipboard.setData(ClipboardData(text: output));
                  },
                  child: const Icon(Icons.copy, size: 14, color: AppColors.textSecondary),
                ),
              ],
            ),
          ),
          // Content
          Padding(
            padding: const EdgeInsets.all(12),
            child: SelectableText(
              output,
              style: GoogleFonts.jetBrainsMono(
                color: AppColors.textPrimary,
                fontSize: 12,
                height: 1.5,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
