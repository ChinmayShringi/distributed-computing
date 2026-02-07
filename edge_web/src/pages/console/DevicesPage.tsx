import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { GlassCard } from '@/components/GlassCard';
import { CapabilityChip } from '@/components/CapabilityChip';
import { MiniGauge } from '@/components/MiniGauge';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { listDevices, type Device } from '@/api';
import {
  Search,
  Filter,
  Monitor,
  Laptop,
  Server,
  Smartphone,
  Cpu,
  WifiOff,
  ChevronRight,
  RefreshCw,
  AlertCircle,
} from 'lucide-react';

// Device type icons based on platform
const getDeviceIcon = (platform: string) => {
  const lower = platform.toLowerCase();
  if (lower.includes('darwin') || lower.includes('macos')) return Laptop;
  if (lower.includes('windows')) return Monitor;
  if (lower.includes('linux')) return Server;
  if (lower.includes('android') || lower.includes('ios')) return Smartphone;
  return Cpu;
};

// Determine device type from platform/arch
const getDeviceType = (platform: string): 'desktop' | 'laptop' | 'server' | 'mobile' | 'iot' => {
  const lower = platform.toLowerCase();
  if (lower.includes('darwin') || lower.includes('macos')) return 'laptop';
  if (lower.includes('windows')) return 'desktop';
  if (lower.includes('linux')) return 'server';
  if (lower.includes('android') || lower.includes('ios')) return 'mobile';
  return 'iot';
};

export const DevicesPage = () => {
  const [searchQuery, setSearchQuery] = useState('');
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const fetchDevices = useCallback(async () => {
    try {
      const data = await listDevices();
      setDevices(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch devices');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    fetchDevices();
  }, [fetchDevices]);

  const handleRefresh = () => {
    setRefreshing(true);
    fetchDevices();
  };

  // Build capabilities array from device properties
  const getCapabilities = (device: Device): string[] => {
    const caps: string[] = [];
    if (device.has_gpu) caps.push('GPU');
    if (device.has_npu) caps.push('NPU');
    if (device.can_screen_capture) caps.push('Screen Capture');
    if (device.has_local_model && device.local_model_name) {
      caps.push(`LLM: ${device.local_model_name}`);
    } else if (device.has_local_model) {
      caps.push('LLM');
    }
    // Add legacy capabilities if present
    if (device.capabilities) {
      device.capabilities.forEach(cap => {
        if (!caps.includes(cap)) caps.push(cap);
      });
    }
    return caps;
  };

  const filteredDevices = devices.filter(
    (device) =>
      device.device_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      device.platform.toLowerCase().includes(searchQuery.toLowerCase()) ||
      device.device_id.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const onlineCount = devices.length; // All registered devices are considered online

  if (loading) {
    return (
      <div className="p-6 flex items-center justify-center min-h-[400px]">
        <div className="flex flex-col items-center gap-3">
          <RefreshCw className="w-8 h-8 animate-spin text-primary" />
          <p className="text-muted-foreground">Loading devices...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">Devices</h1>
          <p className="text-muted-foreground mt-1">
            {onlineCount} device{onlineCount !== 1 ? 's' : ''} registered
          </p>
        </div>

        <div className="flex items-center gap-3">
          <div className="relative flex-1 md:w-64">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search devices..."
              className="pl-10 bg-surface-2 border-outline"
            />
          </div>
          <Button
            variant="outline"
            size="icon"
            className="border-outline"
            onClick={handleRefresh}
            disabled={refreshing}
          >
            <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
          </Button>
          <Button variant="outline" size="icon" className="border-outline">
            <Filter className="w-4 h-4" />
          </Button>
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-3 p-4 rounded-lg bg-danger-pink/10 border border-danger-pink/20">
          <AlertCircle className="w-5 h-5 text-danger-pink" />
          <span className="text-danger-pink">{error}</span>
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefresh}
            className="ml-auto"
          >
            Retry
          </Button>
        </div>
      )}

      {/* Device Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {filteredDevices.map((device, index) => {
          const DeviceIcon = getDeviceIcon(device.platform);
          const capabilities = getCapabilities(device);
          const isOnline = true; // Registered devices are online

          return (
            <motion.div
              key={device.device_id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.05 }}
            >
              <Link to={`/console/devices/${device.device_id}`}>
                <GlassCard hover className="p-5">
                  <div className="flex items-start gap-4">
                    {/* Icon Badge */}
                    <div
                      className={`p-3 rounded-xl ${
                        isOnline ? 'bg-primary/20' : 'bg-surface-variant'
                      }`}
                    >
                      <DeviceIcon
                        className={`w-6 h-6 ${
                          isOnline ? 'text-primary' : 'text-muted-foreground'
                        }`}
                      />
                    </div>

                    {/* Info */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <h3 className="font-semibold truncate">{device.device_name}</h3>
                        <div
                          className={`w-2 h-2 rounded-full ${
                            isOnline ? 'status-dot-online' : 'status-dot-offline'
                          }`}
                        />
                      </div>
                      <p className="text-sm text-muted-foreground mb-1">
                        {device.platform} / {device.arch}
                      </p>
                      <p className="text-xs text-muted-foreground/60 mb-3 font-mono">
                        {device.grpc_addr} | {device.device_id.slice(0, 12)}...
                      </p>

                      {/* Capabilities */}
                      <div className="flex flex-wrap gap-2 mb-4">
                        {capabilities.slice(0, 4).map((cap) => (
                          <CapabilityChip key={cap} capability={cap} />
                        ))}
                        {capabilities.length > 4 && (
                          <span className="text-xs text-muted-foreground">
                            +{capabilities.length - 4} more
                          </span>
                        )}
                      </div>

                      {/* Status - shows placeholder gauges for now */}
                      {isOnline ? (
                        <div className="flex items-center gap-6">
                          <MiniGauge
                            value={0}
                            label="CPU"
                            variant="info"
                          />
                          <MiniGauge
                            value={0}
                            label="Memory"
                            variant="primary"
                          />
                        </div>
                      ) : (
                        <div className="flex items-center gap-2 text-sm text-muted-foreground py-4">
                          <WifiOff className="w-4 h-4" />
                          <span>Offline</span>
                        </div>
                      )}
                    </div>

                    <ChevronRight className="w-5 h-5 text-muted-foreground" />
                  </div>
                </GlassCard>
              </Link>
            </motion.div>
          );
        })}
      </div>

      {filteredDevices.length === 0 && !error && (
        <div className="text-center py-12">
          <p className="text-muted-foreground">
            {searchQuery
              ? 'No devices found matching your search.'
              : 'No devices registered. Start the server and register devices.'}
          </p>
        </div>
      )}
    </div>
  );
};
