import { useState } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { GlassCard } from '@/components/GlassCard';
import { CapabilityChip } from '@/components/CapabilityChip';
import { MiniGauge } from '@/components/MiniGauge';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { mockDevices, Device } from '@/lib/mock-data';
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
} from 'lucide-react';

const deviceIcons: Record<Device['type'], typeof Monitor> = {
  desktop: Monitor,
  laptop: Laptop,
  server: Server,
  mobile: Smartphone,
  iot: Cpu,
};

export const DevicesPage = () => {
  const [searchQuery, setSearchQuery] = useState('');
  const [devices] = useState<Device[]>(mockDevices);

  const filteredDevices = devices.filter(
    (device) =>
      device.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      device.os.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="p-6 space-y-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">Devices</h1>
          <p className="text-muted-foreground mt-1">
            {devices.filter(d => d.status === 'online').length} of {devices.length} devices online
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
          <Button variant="outline" size="icon" className="border-outline">
            <Filter className="w-4 h-4" />
          </Button>
        </div>
      </div>

      {/* Device Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {filteredDevices.map((device, index) => {
          const DeviceIcon = deviceIcons[device.type];
          const isOnline = device.status === 'online';

          return (
            <motion.div
              key={device.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.05 }}
            >
              <Link to={`/console/devices/${device.id}`}>
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
                        <h3 className="font-semibold truncate">{device.name}</h3>
                        <div
                          className={`w-2 h-2 rounded-full ${
                            isOnline ? 'status-dot-online' : 'status-dot-offline'
                          }`}
                        />
                      </div>
                      <p className="text-sm text-muted-foreground mb-3">
                        {device.os}
                      </p>

                      {/* Capabilities */}
                      <div className="flex flex-wrap gap-2 mb-4">
                        {device.capabilities.slice(0, 3).map((cap) => (
                          <CapabilityChip key={cap} capability={cap} />
                        ))}
                        {device.capabilities.length > 3 && (
                          <span className="text-xs text-muted-foreground">
                            +{device.capabilities.length - 3} more
                          </span>
                        )}
                      </div>

                      {/* Status */}
                      {isOnline ? (
                        <div className="flex items-center gap-6">
                          <MiniGauge
                            value={device.cpuPct}
                            label="CPU"
                            variant="info"
                          />
                          <MiniGauge
                            value={device.memPct}
                            label="Memory"
                            variant="primary"
                          />
                        </div>
                      ) : (
                        <div className="flex items-center gap-2 text-sm text-muted-foreground py-4">
                          <WifiOff className="w-4 h-4" />
                          <span>Last seen: {device.lastSeen}</span>
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

      {filteredDevices.length === 0 && (
        <div className="text-center py-12">
          <p className="text-muted-foreground">No devices found matching your search.</p>
        </div>
      )}
    </div>
  );
};
