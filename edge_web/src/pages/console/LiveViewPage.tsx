import { useState, useEffect, useCallback } from 'react';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { listDevices, type Device, type RoutingPolicy } from '@/api';
import { useWebRTC } from '@/hooks/useWebRTC';
import {
  Eye,
  Play,
  Square,
  Monitor,
  RefreshCw,
  AlertCircle,
  Wifi,
  WifiOff,
  Settings,
} from 'lucide-react';

const streamPolicies: { value: RoutingPolicy; label: string }[] = [
  { value: 'BEST_AVAILABLE', label: 'Best Available' },
  { value: 'PREFER_REMOTE', label: 'Prefer Remote' },
  { value: 'FORCE_DEVICE_ID', label: 'Force Device' },
];

const iceStateColors: Record<string, string> = {
  new: 'text-muted-foreground',
  checking: 'text-warning-amber',
  connected: 'text-safe-green',
  completed: 'text-safe-green',
  disconnected: 'text-warning-amber',
  failed: 'text-danger-pink',
  closed: 'text-muted-foreground',
};

export const LiveViewPage = () => {
  // Devices state
  const [devices, setDevices] = useState<Device[]>([]);
  const [loadingDevices, setLoadingDevices] = useState(true);

  // Stream settings
  const [policy, setPolicy] = useState<RoutingPolicy>('BEST_AVAILABLE');
  const [forceDeviceId, setForceDeviceId] = useState<string>('');
  const [fps, setFps] = useState(8);
  const [quality, setQuality] = useState(60);
  const [monitorIndex, setMonitorIndex] = useState(0);
  const [showSettings, setShowSettings] = useState(false);

  // Frame state
  const [frameUrl, setFrameUrl] = useState<string | null>(null);

  // WebRTC hook
  const { state, error, streamInfo, iceConnectionState, start, stop } = useWebRTC({
    onFrame: setFrameUrl,
    onError: (err) => console.error('WebRTC error:', err),
  });

  // Filter devices that can screen capture
  const screenCapableDevices = devices.filter((d) => d.can_screen_capture);

  // Fetch devices
  const fetchDevices = useCallback(async () => {
    try {
      setLoadingDevices(true);
      const data = await listDevices();
      setDevices(data);
      // Set default force device if available
      const capable = data.filter((d) => d.can_screen_capture);
      if (capable.length > 0 && !forceDeviceId) {
        setForceDeviceId(capable[0].device_id);
      }
    } catch (err) {
      console.error('Failed to fetch devices:', err);
    } finally {
      setLoadingDevices(false);
    }
  }, [forceDeviceId]);

  useEffect(() => {
    fetchDevices();
  }, [fetchDevices]);

  // Handle start stream
  const handleStart = () => {
    start(policy, {
      forceDeviceId: policy === 'FORCE_DEVICE_ID' ? forceDeviceId : undefined,
      fps,
      quality,
      monitorIndex,
    });
  };

  // Handle stop stream
  const handleStop = () => {
    stop();
    setFrameUrl(null);
  };

  const isStreaming = state === 'connecting' || state === 'connected';

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Live View</h1>
          <p className="text-muted-foreground mt-1">
            Stream real-time screen capture from connected devices via WebRTC.
          </p>
        </div>
        <Button
          variant="outline"
          size="icon"
          onClick={fetchDevices}
          disabled={loadingDevices}
        >
          <RefreshCw className={`w-4 h-4 ${loadingDevices ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      {/* Viewport */}
      <GlassContainer className="aspect-video flex items-center justify-center relative overflow-hidden">
        {frameUrl ? (
          <img
            src={frameUrl}
            alt="Live stream"
            className="w-full h-full object-contain"
          />
        ) : isStreaming ? (
          <div className="text-center space-y-4">
            <div className="w-16 h-16 mx-auto rounded-full bg-primary/20 flex items-center justify-center animate-pulse">
              <Monitor className="w-8 h-8 text-primary" />
            </div>
            <div>
              <p className="font-semibold">
                {state === 'connecting' ? 'Connecting...' : 'Waiting for frames...'}
              </p>
              {streamInfo && (
                <p className="text-sm text-muted-foreground">
                  {streamInfo.selected_device_name}
                </p>
              )}
            </div>
          </div>
        ) : (
          <div className="text-center space-y-4">
            <div className="w-20 h-20 mx-auto rounded-full bg-surface-variant flex items-center justify-center">
              <Eye className="w-10 h-10 text-muted-foreground" />
            </div>
            <div>
              <p className="text-lg font-semibold">No Active Stream</p>
              <p className="text-sm text-muted-foreground">
                Configure settings and start streaming
              </p>
            </div>
          </div>
        )}

        {/* Stream Status Overlay */}
        {isStreaming && (
          <div className="absolute top-4 left-4 flex items-center gap-2">
            <Badge
              variant="outline"
              className={`${
                state === 'connected' ? 'bg-safe-green/20 border-safe-green' : 'bg-warning-amber/20 border-warning-amber'
              }`}
            >
              <span className="relative flex h-2 w-2 mr-2">
                <span
                  className={`animate-ping absolute inline-flex h-full w-full rounded-full opacity-75 ${
                    state === 'connected' ? 'bg-safe-green' : 'bg-warning-amber'
                  }`}
                />
                <span
                  className={`relative inline-flex rounded-full h-2 w-2 ${
                    state === 'connected' ? 'bg-safe-green' : 'bg-warning-amber'
                  }`}
                />
              </span>
              {state === 'connected' ? 'LIVE' : 'CONNECTING'}
            </Badge>
          </div>
        )}

        {/* ICE State Overlay */}
        {iceConnectionState && (
          <div className="absolute top-4 right-4">
            <Badge variant="outline" className="gap-1">
              {iceConnectionState === 'connected' || iceConnectionState === 'completed' ? (
                <Wifi className="w-3 h-3 text-safe-green" />
              ) : iceConnectionState === 'failed' ? (
                <WifiOff className="w-3 h-3 text-danger-pink" />
              ) : (
                <Wifi className="w-3 h-3 text-warning-amber" />
              )}
              <span className={iceStateColors[iceConnectionState]}>
                ICE: {iceConnectionState}
              </span>
            </Badge>
          </div>
        )}
      </GlassContainer>

      {/* Error Display */}
      {error && (
        <div className="flex items-center gap-3 p-4 rounded-lg bg-danger-pink/10 border border-danger-pink/20">
          <AlertCircle className="w-5 h-5 text-danger-pink" />
          <span className="text-danger-pink">{error}</span>
        </div>
      )}

      {/* Controls */}
      <GlassCard className="p-4">
        <div className="space-y-4">
          {/* Main Controls */}
          <div className="flex flex-col sm:flex-row items-center gap-4">
            <div className="flex-1 w-full sm:w-auto">
              <Select value={policy} onValueChange={(v) => setPolicy(v as RoutingPolicy)}>
                <SelectTrigger className="bg-surface-2 border-outline">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent className="bg-surface-2 border-outline">
                  {streamPolicies.map((p) => (
                    <SelectItem key={p.value} value={p.value}>
                      {p.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {policy === 'FORCE_DEVICE_ID' && (
              <div className="flex-1 w-full sm:w-auto">
                <Select value={forceDeviceId} onValueChange={setForceDeviceId}>
                  <SelectTrigger className="bg-surface-2 border-outline">
                    <SelectValue placeholder="Select device" />
                  </SelectTrigger>
                  <SelectContent className="bg-surface-2 border-outline">
                    {screenCapableDevices.map((device) => (
                      <SelectItem key={device.device_id} value={device.device_id}>
                        <div className="flex items-center gap-2">
                          <Monitor className="w-4 h-4" />
                          {device.device_name}
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            <Button
              variant="outline"
              size="icon"
              onClick={() => setShowSettings(!showSettings)}
            >
              <Settings className="w-4 h-4" />
            </Button>

            <Button
              onClick={isStreaming ? handleStop : handleStart}
              disabled={state === 'connecting'}
              className={
                isStreaming
                  ? 'bg-danger-pink hover:bg-danger-pink/90'
                  : 'bg-primary hover:bg-primary/90'
              }
            >
              {isStreaming ? (
                <>
                  <Square className="w-4 h-4 mr-2" />
                  Stop Stream
                </>
              ) : (
                <>
                  <Play className="w-4 h-4 mr-2" />
                  Start Stream
                </>
              )}
            </Button>
          </div>

          {/* Advanced Settings */}
          {showSettings && (
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 pt-4 border-t border-outline">
              <div className="space-y-2">
                <Label htmlFor="fps">FPS (1-30)</Label>
                <Input
                  id="fps"
                  type="number"
                  min={1}
                  max={30}
                  value={fps}
                  onChange={(e) => setFps(parseInt(e.target.value) || 8)}
                  className="bg-surface-2 border-outline"
                  disabled={isStreaming}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="quality">Quality (10-100)</Label>
                <Input
                  id="quality"
                  type="number"
                  min={10}
                  max={100}
                  value={quality}
                  onChange={(e) => setQuality(parseInt(e.target.value) || 60)}
                  className="bg-surface-2 border-outline"
                  disabled={isStreaming}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="monitor">Monitor Index (0-3)</Label>
                <Input
                  id="monitor"
                  type="number"
                  min={0}
                  max={3}
                  value={monitorIndex}
                  onChange={(e) => setMonitorIndex(parseInt(e.target.value) || 0)}
                  className="bg-surface-2 border-outline"
                  disabled={isStreaming}
                />
              </div>
            </div>
          )}
        </div>
      </GlassCard>

      {/* Stream Info */}
      {streamInfo && (
        <GlassCard className="p-4">
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm">
            <div>
              <span className="text-muted-foreground">Device:</span>
              <p className="font-medium">{streamInfo.selected_device_name}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Stream ID:</span>
              <p className="font-mono text-xs">{streamInfo.stream_id.slice(0, 12)}...</p>
            </div>
            <div>
              <span className="text-muted-foreground">Address:</span>
              <p className="font-mono text-xs">{streamInfo.selected_device_addr}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Settings:</span>
              <p className="font-mono text-xs">
                {fps}fps / {quality}% / Mon {monitorIndex}
              </p>
            </div>
          </div>
        </GlassCard>
      )}
    </div>
  );
};
