import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { GlassCard } from '@/components/GlassCard';
import { CapabilityChip } from '@/components/CapabilityChip';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { listDevices, hasCapability, type Device } from '@/api';
import {
  Search,
  RefreshCw,
  Monitor,
  ChevronRight,
  AlertCircle,
  Cpu,
  Zap,
  Bot,
  Eye,
} from 'lucide-react';

export const DevicesPage = () => {
  const [searchQuery, setSearchQuery] = useState('');
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchDevices = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await listDevices();
      setDevices(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch devices';
      setError(message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDevices();
  }, [fetchDevices]);

  const filteredDevices = devices.filter(
    (device) =>
      device.device_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      device.platform.toLowerCase().includes(searchQuery.toLowerCase()) ||
      device.arch.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="p-6 space-y-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">Devices</h1>
          <p className="text-muted-foreground mt-1">
            {devices.length} device{devices.length !== 1 ? 's' : ''} registered
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
            onClick={fetchDevices}
            disabled={loading}
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </Button>
        </div>
      </div>

      {/* Error State */}
      {error && (
        <div className="flex items-center gap-3 p-4 rounded-lg bg-danger-pink/10 border border-danger-pink/20">
          <AlertCircle className="w-5 h-5 text-danger-pink" />
          <span className="text-danger-pink">{error}</span>
          <Button variant="outline" size="sm" onClick={fetchDevices} className="ml-auto">
            Retry
          </Button>
        </div>
      )}

      {/* Loading State */}
      {loading && devices.length === 0 && (
        <div className="text-center py-12">
          <RefreshCw className="w-8 h-8 mx-auto mb-4 animate-spin text-primary" />
          <p className="text-muted-foreground">Loading devices...</p>
        </div>
      )}

      {/* Device Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {filteredDevices.map((device, index) => (
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
                  <div className="p-3 rounded-xl bg-primary/20">
                    <Monitor className="w-6 h-6 text-primary" />
                  </div>

                  {/* Info */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="font-semibold truncate">{device.device_name}</h3>
                      <div className="w-2 h-2 rounded-full status-dot-online" />
                    </div>
                    <p className="text-sm text-muted-foreground mb-3">
                      {device.platform} / {device.arch}
                    </p>

                    {/* Capabilities */}
                    <div className="flex flex-wrap gap-2 mb-3">
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
                          {device.local_model_name || 'LLM'}
                        </Badge>
                      )}
                      {device.can_screen_capture && (
                        <Badge variant="outline" className="gap-1 text-warning-amber border-warning-amber">
                          <Eye className="w-3 h-3" />
                          Screen
                        </Badge>
                      )}
                    </div>

                    {/* Address */}
                    <p className="text-xs text-muted-foreground font-mono">
                      {device.grpc_addr}
                    </p>
                  </div>

                  <ChevronRight className="w-5 h-5 text-muted-foreground" />
                </div>
              </GlassCard>
            </Link>
          </motion.div>
        ))}
      </div>

      {!loading && filteredDevices.length === 0 && (
        <div className="text-center py-12">
          <p className="text-muted-foreground">
            {searchQuery ? 'No devices found matching your search.' : 'No devices registered yet.'}
          </p>
        </div>
      )}
    </div>
  );
};
