import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:lucide_icons/lucide_icons.dart';
import '../../theme/app_colors.dart';
import '../../shared/widgets/edge_mesh_wordmark.dart';
import '../../shared/widgets/status_strip.dart';
import '../../shared/widgets/glass_container.dart';
import '../../services/grpc_service.dart';
import '../../services/rest_api_client.dart';
import '../../services/webrtc_stream_service.dart';
import 'models/stream_models.dart';

class StreamScreen extends StatefulWidget {
  const StreamScreen({super.key});

  @override
  State<StreamScreen> createState() => _StreamScreenState();
}

class _StreamScreenState extends State<StreamScreen> {
  final _grpcService = GrpcService();
  RestApiClient? _restApiClient;
  WebRTCStreamService? _webrtcService;

  List<Map<String, dynamic>> _devices = [];
  Map<String, dynamic>? _selectedDevice;
  bool _isLoading = true;
  String? _serverAddress;

  // Stream settings
  int _fps = 8;
  int _quality = 60;
  int _monitorIndex = 0;

  @override
  void initState() {
    super.initState();
    _initServices();
  }

  Future<void> _initServices() async {
    try {
      // Get connection status to determine server address
      final status = await _grpcService.getConnectionStatus();
      if (status.connected && status.host.isNotEmpty) {
        _serverAddress = 'http://${status.host}:${status.httpPort}';
        _restApiClient = RestApiClient(baseUrl: _serverAddress!);
        _webrtcService = WebRTCStreamService(apiClient: _restApiClient!);

        // Listen to state changes
        _webrtcService!.state.addListener(_onStreamStateChanged);
        _webrtcService!.errorMessage.addListener(_onErrorChanged);
        _webrtcService!.frameCount.addListener(_onFrameCountChanged);
      }
      await _loadDevices();
    } catch (e) {
      debugPrint('Error initializing services: $e');
      if (mounted) {
        setState(() => _isLoading = false);
      }
    }
  }

  void _onStreamStateChanged() {
    if (mounted) setState(() {});
  }

  void _onErrorChanged() {
    final error = _webrtcService?.errorMessage.value;
    if (error != null && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(error),
          backgroundColor: AppColors.primaryRed,
        ),
      );
    }
  }

  void _onFrameCountChanged() {
    if (mounted) setState(() {});
  }

  Future<void> _loadDevices() async {
    try {
      final devices = await _grpcService.listDevices();
      // Filter to devices that can screen capture
      final capableDevices = devices.where((d) {
        return d['can_screen_capture'] == true;
      }).toList();

      if (mounted) {
        setState(() {
          _devices = devices;
          _isLoading = false;
          // Select first capable device, or first device
          if (_selectedDevice == null) {
            if (capableDevices.isNotEmpty) {
              _selectedDevice = capableDevices.first;
            } else if (devices.isNotEmpty) {
              _selectedDevice = devices.first;
            }
          }
        });
      }
    } catch (e) {
      debugPrint('Error loading devices: $e');
      if (mounted) {
        setState(() {
          _devices = [];
          _isLoading = false;
        });
      }
    }
  }

  void _showDevicePicker() {
    if (_devices.isEmpty) return;
    showModalBottomSheet(
      context: context,
      backgroundColor: AppColors.surface2,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Padding(
            padding: const EdgeInsets.all(16),
            child: Text(
              'Select Source Device',
              style: TextStyle(
                color: Colors.white,
                fontSize: 16,
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
          const Divider(color: Colors.white10),
          Flexible(
            child: ListView.builder(
              shrinkWrap: true,
              itemCount: _devices.length,
              itemBuilder: (_, i) {
                final d = _devices[i];
                final isSelected = _selectedDevice?['device_id'] == d['device_id'];
                final canCapture = d['can_screen_capture'] == true;
                final isAndroid = d['platform']?.toString().toLowerCase().contains('android') ?? false;

                return ListTile(
                  leading: Icon(
                    isAndroid ? LucideIcons.smartphone : LucideIcons.laptop,
                    color: isSelected
                        ? AppColors.safeGreen
                        : canCapture
                            ? AppColors.textSecondary
                            : Colors.white24,
                  ),
                  title: Text(
                    d['device_name']?.toString() ?? 'Unknown',
                    style: TextStyle(
                      color: canCapture ? Colors.white : Colors.white38,
                    ),
                  ),
                  subtitle: Row(
                    children: [
                      Text(
                        '${d['platform']} ${d['arch']}',
                        style: TextStyle(color: Colors.white54),
                      ),
                      if (!canCapture) ...[
                        const SizedBox(width: 8),
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                          decoration: BoxDecoration(
                            color: Colors.orange.withOpacity(0.2),
                            borderRadius: BorderRadius.circular(4),
                          ),
                          child: Text(
                            'No Capture',
                            style: TextStyle(color: Colors.orange, fontSize: 10),
                          ),
                        ),
                      ],
                    ],
                  ),
                  trailing: isSelected
                      ? const Icon(LucideIcons.check, color: AppColors.safeGreen)
                      : null,
                  enabled: canCapture,
                  onTap: canCapture
                      ? () {
                          setState(() => _selectedDevice = d);
                          Navigator.pop(ctx);
                        }
                      : null,
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _startStream() async {
    if (_selectedDevice == null || _webrtcService == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Select a source device first')),
      );
      return;
    }

    try {
      await _webrtcService!.startStream(
        policy: 'FORCE_DEVICE_ID',
        forceDeviceId: _selectedDevice!['device_id'] as String,
        fps: _fps,
        quality: _quality,
        monitorIndex: _monitorIndex,
      );
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Failed to start stream: $e'),
            backgroundColor: AppColors.primaryRed,
          ),
        );
      }
    }
  }

  Future<void> _stopStream() async {
    try {
      await _webrtcService?.stopStream();
    } catch (e) {
      debugPrint('Error stopping stream: $e');
    }
  }

  @override
  void dispose() {
    _webrtcService?.state.removeListener(_onStreamStateChanged);
    _webrtcService?.errorMessage.removeListener(_onErrorChanged);
    _webrtcService?.frameCount.removeListener(_onFrameCountChanged);
    _webrtcService?.dispose();
    _restApiClient?.close();
    super.dispose();
  }

  WebRTCStreamState get _streamState =>
      _webrtcService?.state.value ?? WebRTCStreamState.idle;

  bool get _isConnecting => _streamState == WebRTCStreamState.connecting;
  bool get _isStreaming => _streamState == WebRTCStreamState.connected;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(
          color: AppColors.backgroundDark,
        ),
        child: SafeArea(
          child: Column(
            children: [
              // Header
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    const EdgeMeshWordmark(fontSize: 20),
                    Row(
                      children: [
                        if (_isStreaming)
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                            decoration: BoxDecoration(
                              color: AppColors.safeGreen.withOpacity(0.2),
                              borderRadius: BorderRadius.circular(8),
                            ),
                            child: Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                Container(
                                  width: 8,
                                  height: 8,
                                  decoration: BoxDecoration(
                                    color: AppColors.safeGreen,
                                    shape: BoxShape.circle,
                                  ),
                                ),
                                const SizedBox(width: 6),
                                Text(
                                  'LIVE',
                                  style: TextStyle(
                                    color: AppColors.safeGreen,
                                    fontSize: 10,
                                    fontWeight: FontWeight.bold,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        const SizedBox(width: 8),
                        Icon(
                          _isStreaming ? LucideIcons.video : LucideIcons.videoOff,
                          color: _isStreaming ? AppColors.safeGreen : AppColors.primaryRed,
                          size: 20,
                        ),
                      ],
                    ),
                  ],
                ),
              ),

              StatusStrip(
                isConnected: _serverAddress != null,
                serverAddress: _serverAddress ?? 'Not connected',
                isDangerous: false,
              ),

              // Stream Viewport
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: GlassContainer(
                    padding: EdgeInsets.zero,
                    borderRadius: 20,
                    child: Stack(
                      children: [
                        // Frame Display or Placeholder
                        if (_isStreaming && _webrtcService != null)
                          StreamBuilder<Uint8List>(
                            stream: _webrtcService!.frameStream,
                            builder: (context, snapshot) {
                              if (snapshot.hasData) {
                                return Center(
                                  child: Image.memory(
                                    snapshot.data!,
                                    gaplessPlayback: true,
                                    fit: BoxFit.contain,
                                  ),
                                );
                              }
                              return _buildConnectingIndicator();
                            },
                          )
                        else if (_isConnecting)
                          _buildConnectingIndicator()
                        else
                          _buildPlaceholder(),

                        // Status Overlay
                        Positioned(
                          top: 16,
                          left: 16,
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              _OverlayPill(
                                label: _isStreaming
                                    ? 'STREAMING'
                                    : _isConnecting
                                        ? 'CONNECTING'
                                        : 'OFFLINE',
                                color: _isStreaming
                                    ? AppColors.safeGreen
                                    : _isConnecting
                                        ? AppColors.warningAmber
                                        : Colors.white.withOpacity(0.5),
                                icon: _isStreaming
                                    ? LucideIcons.radio
                                    : _isConnecting
                                        ? LucideIcons.loader
                                        : LucideIcons.radioOff,
                              ),
                              const SizedBox(height: 8),
                              _OverlayPill(
                                label: '$_fps FPS | Q$_quality',
                                color: Colors.white.withOpacity(0.5),
                                icon: LucideIcons.activity,
                              ),
                            ],
                          ),
                        ),

                        // Frame counter
                        if (_isStreaming)
                          Positioned(
                            top: 16,
                            right: 16,
                            child: _OverlayPill(
                              label: '${_webrtcService?.frameCount.value ?? 0} frames',
                              color: Colors.white.withOpacity(0.5),
                              icon: LucideIcons.image,
                            ),
                          ),

                        // Device info
                        if (_webrtcService?.currentStream != null)
                          Positioned(
                            bottom: 16,
                            left: 16,
                            child: _OverlayPill(
                              label: _webrtcService!.currentStream!.deviceName,
                              color: AppColors.safeGreen,
                              icon: LucideIcons.laptop,
                            ),
                          ),
                      ],
                    ),
                  ),
                ),
              ),

              // Controls
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
                child: GlassContainer(
                  child: Column(
                    children: [
                      // Device Selector
                      ListTile(
                        contentPadding: EdgeInsets.zero,
                        leading: Icon(
                          (_selectedDevice?['platform']?.toString().toLowerCase().contains('android') ?? false)
                              ? LucideIcons.smartphone
                              : LucideIcons.laptop,
                          color: AppColors.primaryRed,
                        ),
                        title: const Text('Source Node', style: TextStyle(fontWeight: FontWeight.bold)),
                        subtitle: Text(
                          _selectedDevice?['device_name']?.toString() ?? 'Select a device',
                          style: TextStyle(color: Colors.white70),
                        ),
                        trailing: const Icon(LucideIcons.chevronDown, size: 16),
                        onTap: _isStreaming ? null : _showDevicePicker,
                      ),

                      const Divider(color: Colors.white10),

                      // Stream Settings
                      if (!_isStreaming) ...[
                        Padding(
                          padding: const EdgeInsets.symmetric(vertical: 8),
                          child: Row(
                            children: [
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text('FPS: $_fps', style: TextStyle(color: Colors.white70, fontSize: 12)),
                                    Slider(
                                      value: _fps.toDouble(),
                                      min: 1,
                                      max: 30,
                                      divisions: 29,
                                      activeColor: AppColors.primaryRed,
                                      onChanged: (v) => setState(() => _fps = v.toInt()),
                                    ),
                                  ],
                                ),
                              ),
                              const SizedBox(width: 16),
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text('Quality: $_quality', style: TextStyle(color: Colors.white70, fontSize: 12)),
                                    Slider(
                                      value: _quality.toDouble(),
                                      min: 10,
                                      max: 100,
                                      divisions: 9,
                                      activeColor: AppColors.primaryRed,
                                      onChanged: (v) => setState(() => _quality = v.toInt()),
                                    ),
                                  ],
                                ),
                              ),
                            ],
                          ),
                        ),
                        const Divider(color: Colors.white10),
                      ],

                      const SizedBox(height: 16),

                      // Start/Stop Button
                      if (_isStreaming)
                        FilledButton.icon(
                          onPressed: _stopStream,
                          icon: const Icon(LucideIcons.square, size: 16),
                          label: const Text('STOP STREAM'),
                          style: FilledButton.styleFrom(
                            backgroundColor: AppColors.primaryRed,
                            foregroundColor: Colors.white,
                            minimumSize: const Size(double.infinity, 54),
                            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                          ),
                        )
                      else
                        FilledButton.icon(
                          onPressed: _isConnecting || _serverAddress == null ? null : _startStream,
                          icon: _isConnecting
                              ? const SizedBox(
                                  width: 16,
                                  height: 16,
                                  child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                                )
                              : const Icon(LucideIcons.play, size: 16),
                          label: Text(_isConnecting ? 'CONNECTING...' : 'START STREAM'),
                          style: FilledButton.styleFrom(
                            backgroundColor: _serverAddress == null ? Colors.grey : AppColors.safeGreen,
                            foregroundColor: Colors.white,
                            minimumSize: const Size(double.infinity, 54),
                            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildPlaceholder() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(LucideIcons.monitorOff, size: 48, color: Colors.white.withOpacity(0.1)),
          const SizedBox(height: 16),
          Text(
            'NO ACTIVE STREAM',
            style: TextStyle(
              color: Colors.white.withOpacity(0.2),
              fontSize: 10,
              fontWeight: FontWeight.bold,
              letterSpacing: 2,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Select a device and tap Start Stream',
            style: TextStyle(
              color: Colors.white.withOpacity(0.15),
              fontSize: 12,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildConnectingIndicator() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const CircularProgressIndicator(
            color: AppColors.warningAmber,
          ),
          const SizedBox(height: 16),
          Text(
            'ESTABLISHING CONNECTION',
            style: TextStyle(
              color: AppColors.warningAmber,
              fontSize: 10,
              fontWeight: FontWeight.bold,
              letterSpacing: 2,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Negotiating WebRTC...',
            style: TextStyle(
              color: Colors.white.withOpacity(0.3),
              fontSize: 12,
            ),
          ),
        ],
      ),
    );
  }
}

class _OverlayPill extends StatelessWidget {
  final String label;
  final Color color;
  final IconData icon;

  const _OverlayPill({required this.label, required this.color, required this.icon});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: Colors.black.withOpacity(0.6),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: color.withOpacity(0.2)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 12, color: color),
          const SizedBox(width: 8),
          Text(
            label,
            style: TextStyle(color: color, fontSize: 10, fontWeight: FontWeight.bold, letterSpacing: 0.5),
          ),
        ],
      ),
    );
  }
}
