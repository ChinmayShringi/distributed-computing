import { useParams, useNavigate } from 'react-router-dom';
import { useState, useEffect, useCallback } from 'react';
import { motion } from 'framer-motion';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { CircularProgress } from '@/components/CircularProgress';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { listDevices, getDeviceMetrics, hasCapability, type Device, type DeviceMetricsHistory } from '@/api';
import {
  ArrowLeft,
  Monitor,
  RefreshCw,
  Cpu,
  Zap,
  Bot,
  Eye,
  AlertCircle,
  Loader2,
} from 'lucide-react';

export const DeviceDetailPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const [device, setDevice] = useState<Device | null>(null);
  const [metrics, setMetrics] = useState<DeviceMetricsHistory | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    if (!id) return;

    try {
      setLoading(true);
      setError(null);

      // Fetch devices and find the one we want
      const devices = await listDevices();
      const found = devices.find((d) => d.device_id === id);

      if (!found) {
        setError('Device not found');
        setDevice(null);
        return;
      }

      setDevice(found);

      // Fetch metrics for this device
      try {
        const metricsData = await getDeviceMetrics(id);
        setMetrics(metricsData);
      } catch {
        // Metrics might not be available, that's ok
        setMetrics(null);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch device';
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Get latest metrics sample
  const latestSample = metrics?.samples?.[metrics.samples.length - 1];
  const cpuPercent = latestSample?.cpu_load !== undefined && latestSample.cpu_load >= 0
    ? latestSample.cpu_load * 100
    : 0;
  const memPercent = latestSample?.mem_total_mb && latestSample?.mem_used_mb
    ? (latestSample.mem_used_mb / latestSample.mem_total_mb) * 100
    : 0;

  if (loading) {
    return (
      <div className="p-6 flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  if (error || !device) {
    return (
      <div className="p-6 space-y-4">
        <Button variant="ghost" onClick={() => navigate('/console/devices')}>
          <ArrowLeft className="w-4 h-4 mr-2" />
          Back to Devices
        </Button>
        <div className="flex items-center gap-3 p-4 rounded-lg bg-danger-pink/10 border border-danger-pink/20">
          <AlertCircle className="w-5 h-5 text-danger-pink" />
          <span className="text-danger-pink">{error || 'Device not found'}</span>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Back button */}
      <div className="flex items-center justify-between">
        <Button variant="ghost" onClick={() => navigate('/console/devices')}>
          <ArrowLeft className="w-4 h-4 mr-2" />
          Back to Devices
        </Button>
        <Button variant="outline" size="icon" onClick={fetchData} disabled={loading}>
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      {/* Device Header */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
      >
        <GlassCard className="p-6">
          <div className="flex flex-col md:flex-row md:items-center gap-6">
            <div className="p-4 rounded-2xl bg-primary/20">
              <Monitor className="w-10 h-10 text-primary" />
            </div>

            <div className="flex-1">
              <div className="flex items-center gap-3 mb-2">
                <h1 className="text-2xl font-bold">{device.device_name}</h1>
                <span className="px-3 py-1 rounded-full text-sm font-medium bg-safe-green/20 text-safe-green">
                  Online
                </span>
              </div>
              <p className="text-muted-foreground">
                {device.platform} / {device.arch}
              </p>
              <p className="text-sm font-mono text-muted-foreground mt-1">
                {device.grpc_addr}
              </p>
            </div>
          </div>
        </GlassCard>
      </motion.div>

      {/* Capabilities */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.1 }}
      >
        <GlassContainer>
          <h2 className="text-lg font-semibold mb-4">Capabilities</h2>
          <div className="flex flex-wrap gap-2">
            {hasCapability(device, 'gpu') && (
              <Badge variant="outline" className="gap-1 text-safe-green border-safe-green">
                <Cpu className="w-3 h-3" />
                GPU
              </Badge>
            )}
            {hasCapability(device, 'npu') && (
              <Badge variant="outline" className="gap-1 text-primary border-primary">
                <Zap className="w-3 h-3" />
                NPU
              </Badge>
            )}
            {device.has_local_model && (
              <Badge variant="outline" className="gap-1 text-info-blue border-info-blue">
                <Bot className="w-3 h-3" />
                LLM
              </Badge>
            )}
            {device.can_screen_capture && (
              <Badge variant="outline" className="gap-1 text-warning-amber border-warning-amber">
                <Eye className="w-3 h-3" />
                Screen Capture
              </Badge>
            )}
          </div>
        </GlassContainer>
      </motion.div>

      {/* Telemetry */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.2 }}
      >
        <GlassContainer>
          <h2 className="text-lg font-semibold mb-6">Telemetry</h2>
          <div className="flex items-center justify-center gap-12">
            <CircularProgress value={cpuPercent} label="CPU" variant="cpu" size={100} strokeWidth={8} />
            <CircularProgress value={memPercent} label="Memory" variant="memory" size={100} strokeWidth={8} />
          </div>
          {!latestSample && (
            <p className="text-center text-sm text-muted-foreground mt-4">
              No metrics data available yet
            </p>
          )}
        </GlassContainer>
      </motion.div>

      {/* Device Info */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.3 }}
      >
        <GlassContainer>
          <h2 className="text-lg font-semibold mb-4">Device Info</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="flex justify-between py-2 border-b border-outline">
              <span className="text-muted-foreground">Device ID</span>
              <span className="font-mono text-xs">{device.device_id}</span>
            </div>
            <div className="flex justify-between py-2 border-b border-outline">
              <span className="text-muted-foreground">Platform</span>
              <span className="font-medium">{device.platform}</span>
            </div>
            <div className="flex justify-between py-2 border-b border-outline">
              <span className="text-muted-foreground">Architecture</span>
              <span className="font-medium">{device.arch}</span>
            </div>
            <div className="flex justify-between py-2 border-b border-outline">
              <span className="text-muted-foreground">gRPC Address</span>
              <span className="font-mono text-xs">{device.grpc_addr}</span>
            </div>
          </div>
        </GlassContainer>
      </motion.div>
    </div>
  );
};
