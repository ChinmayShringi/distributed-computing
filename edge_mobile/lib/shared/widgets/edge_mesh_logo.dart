import 'package:flutter/material.dart';
import '../../theme/app_colors.dart';

class EdgeMeshLogo extends StatelessWidget {
  final double size;

  const EdgeMeshLogo({super.key, this.size = 64});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(size * 0.28),
        color: Colors.white.withOpacity(0.06),
        border: Border.all(color: AppColors.outline.withOpacity(0.5)),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.45),
            blurRadius: 22,
            offset: const Offset(0, 14),
          ),
        ],
      ),
      child: Center(
        child: Padding(
          padding: EdgeInsets.all(size * 0.2),
          child: CustomPaint(
            size: Size(size * 0.6, size * 0.6),
            painter: _ConnectedEPainter(),
          ),
        ),
      ),
    );
  }
}

class _ConnectedEPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final paintNodes = Paint()
      ..shader = AppColors.meshGradient.createShader(Offset.zero & size)
      ..style = PaintingStyle.fill;

    final paintLines = Paint()
      ..color = const Color(0xFFE6EDF6).withOpacity(0.3)
      ..strokeWidth = 1.2
      ..style = PaintingStyle.stroke;

    // Define nodes for 'E' shape
    final nodes = [
      Offset(size.width * 0.8, size.height * 0.1), // Top right
      Offset(size.width * 0.2, size.height * 0.1), // Top left
      Offset(size.width * 0.2, size.height * 0.5), // Center left
      Offset(size.width * 0.6, size.height * 0.5), // Center right (stub)
      Offset(size.width * 0.2, size.height * 0.9), // Bottom left
      Offset(size.width * 0.8, size.height * 0.9), // Bottom right
    ];

    // Draw connecting lines for 'E'
    canvas.drawLine(nodes[0], nodes[1], paintLines);
    canvas.drawLine(nodes[1], nodes[2], paintLines);
    canvas.drawLine(nodes[2], nodes[3], paintLines);
    canvas.drawLine(nodes[2], nodes[4], paintLines);
    canvas.drawLine(nodes[4], nodes[5], paintLines);

    // Draw nodes
    for (int i = 0; i < nodes.length; i++) {
      final isHub = i == 2; // Center left is the hub
      canvas.drawCircle(nodes[i], isHub ? 3.5 : 2.5, paintNodes);
    }
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}
