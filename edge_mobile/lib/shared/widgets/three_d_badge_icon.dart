import 'dart:math' as math;
import 'package:flutter/material.dart';
import '../../theme/app_colors.dart';

class ThreeDBadgeIcon extends StatefulWidget {
  final IconData icon;
  final Color accentColor;
  final bool isAnimated;
  final bool isDanger;
  final double size;
  final bool useRotation;

  const ThreeDBadgeIcon({
    super.key,
    required this.icon,
    this.accentColor = AppColors.safeGreen,
    this.isAnimated = true,
    this.isDanger = false,
    this.size = 24,
    this.useRotation = false,
  });

  @override
  State<ThreeDBadgeIcon> createState() => _ThreeDBadgeIconState();
}

class _ThreeDBadgeIconState extends State<ThreeDBadgeIcon> with TickerProviderStateMixin {
  late AnimationController _pulseController;
  late AnimationController _rotateController;
  late AnimationController _shimmerController;
  Offset _tilt = Offset.zero;
  bool _isPressed = false;

  @override
  void initState() {
    super.initState();
    _pulseController = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 2),
    )..repeat(reverse: true);

    _rotateController = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 4),
    );
    if (widget.useRotation) _rotateController.repeat();

    _shimmerController = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 3),
    )..repeat();
  }

  @override
  void dispose() {
    _pulseController.dispose();
    _rotateController.dispose();
    _shimmerController.dispose();
    super.dispose();
  }

  void _handlePanUpdate(DragUpdateDetails details, BoxConstraints constraints) {
    if (!widget.isAnimated) return;
    setState(() {
      final x = (details.localPosition.dx / constraints.maxWidth) * 2 - 1;
      final y = (details.localPosition.dy / constraints.maxHeight) * 2 - 1;
      _tilt = Offset(x.clamp(-1, 1), y.clamp(-1, 1));
    });
  }

  void _handlePanEnd(DragEndDetails details) {
    setState(() {
      _tilt = Offset.zero;
      _isPressed = false;
    });
  }

  @override
  Widget build(BuildContext context) {
    final effectiveAccent = widget.isDanger ? AppColors.primaryRed : widget.accentColor;
    final badgeSize = widget.size * 2.2;

    return GestureDetector(
      onPanUpdate: (d) => _handlePanUpdate(d, BoxConstraints.tight(Size(badgeSize, badgeSize))),
      onPanEnd: _handlePanEnd,
      onTapDown: (_) => setState(() => _isPressed = true),
      onTapUp: (_) => setState(() => _isPressed = false),
      onTapCancel: () => setState(() => _isPressed = false),
      child: AnimatedScale(
        scale: _isPressed ? 0.96 : 1.0,
        duration: const Duration(milliseconds: 100),
        child: AnimatedBuilder(
          animation: Listenable.merge([_pulseController, _rotateController, _shimmerController]),
          builder: (context, child) {
            return Transform(
              transform: Matrix4.identity()
                ..setEntry(3, 2, 0.002) // Perspective
                ..rotateX(-_tilt.dy * 0.15)
                ..rotateY(_tilt.dx * 0.15),
              alignment: Alignment.center,
              child: Container(
                width: badgeSize,
                height: badgeSize,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  boxShadow: [
                    // Outer Shadow
                    BoxShadow(
                      color: Colors.black.withOpacity(0.4),
                      blurRadius: _isPressed ? 4 : 12,
                      offset: Offset(0, _isPressed ? 2 : 6),
                    ),
                    // Accent Glow (Pulse)
                    BoxShadow(
                      color: effectiveAccent.withValues(alpha: 0.15 + (_pulseController.value * 0.1)),
                      blurRadius: 10 + (_pulseController.value * 8),
                      spreadRadius: -2,
                    ),
                  ],
                ),
                child: Stack(
                  alignment: Alignment.center,
                  children: [
                    // 1. Core Medallion (Base)
                    Container(
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        gradient: LinearGradient(
                          begin: Alignment.topLeft,
                          end: Alignment.bottomRight,
                          colors: [
                            AppColors.surface2,
                            AppColors.surface1,
                          ],
                        ),
                        border: Border.all(
                          color: Colors.white.withOpacity(0.08),
                          width: 1,
                        ),
                      ),
                    ),

                    // 2. Dynamic Rim Light (Mode-Aware)
                    Container(
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        border: Border.all(
                          color: effectiveAccent.withValues(alpha: 0.2 + (_pulseController.value * 0.2)),
                          width: 0.5,
                        ),
                      ),
                    ),

                    // 3. Inner Bevel / Reflection Overlay
                    Container(
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        gradient: RadialGradient(
                          center: const Alignment(-0.3, -0.3),
                          radius: 0.8,
                          colors: [
                            Colors.white.withOpacity(0.05),
                            Colors.transparent,
                          ],
                        ),
                      ),
                    ),

                    // 4. Shimmer Sweep (Positioned for animation)
                    Positioned.fill(
                      child: ClipOval(
                        child: Transform.rotate(
                          angle: 0.5,
                          child: AnimatedBuilder(
                            animation: _shimmerController,
                            builder: (context, child) {
                              return Transform.translate(
                                offset: Offset(
                                  (_shimmerController.value * 200) - 100,
                                  (_shimmerController.value * 200) - 100,
                                ),
                                child: FractionallySizedBox(
                                  widthFactor: 2,
                                  heightFactor: 0.1,
                                  child: Container(
                                    decoration: BoxDecoration(
                                      gradient: LinearGradient(
                                        colors: [
                                          Colors.transparent,
                                          Colors.white.withOpacity(0.05),
                                          Colors.transparent,
                                        ],
                                      ),
                                    ),
                                  ),
                                ),
                              );
                            },
                          ),
                        ),
                      ),
                    ),

                    // 5. The Icon (3D Layered)
                    Transform.rotate(
                      angle: widget.useRotation ? _rotateController.value * 2 * math.pi : 0,
                      child: Icon(
                        widget.icon,
                        size: widget.size,
                        color: effectiveAccent,
                      ),
                    ),

                    // 6. Glass Lens Overlay (Material look)
                    Container(
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        gradient: LinearGradient(
                          begin: Alignment.topLeft,
                          end: Alignment.bottomRight,
                          colors: [
                            Colors.white.withOpacity(0.12),
                            Colors.transparent,
                            Colors.black.withOpacity(0.05),
                          ],
                          stops: const [0, 0.4, 1],
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            );
          },
        ),
      ),
    );
  }
}
