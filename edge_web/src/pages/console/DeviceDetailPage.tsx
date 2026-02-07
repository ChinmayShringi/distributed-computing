import { useParams, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { CapabilityChip } from '@/components/CapabilityChip';
import { MiniGauge } from '@/components/MiniGauge';
import { Button } from '@/components/ui/button';
import { mockDevices } from '@/lib/mock-data';
import {
  ArrowLeft,
  Monitor,
  Laptop,
  Server,
  Smartphone,
  Cpu,
  Terminal,
  RefreshCw,
  Power,
  FolderSync,
  Eye,
  WifiOff,
} from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

const deviceIcons = {
  desktop: Monitor,
  laptop: Laptop,
  server: Server,
  mobile: Smartphone,
  iot: Cpu,
};

export const DeviceDetailPage = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { toast } = useToast();
  
  const device = mockDevices.find((d) => d.id === id);

  if (!device) {
    return (
      <div className="p-6 text-center">
        <p className="text-muted-foreground">Device not found.</p>
        <Button variant="ghost" onClick={() => navigate('/console/devices')} className="mt-4">
          <ArrowLeft className="w-4 h-4 mr-2" />
          Back to Devices
        </Button>
      </div>
    );
  }

  const DeviceIcon = deviceIcons[device.type];
  const isOnline = device.status === 'online';

  const actions = [
    { icon: Terminal, label: 'Remote Shell', action: 'shell', requiresOnline: true },
    { icon: FolderSync, label: 'Sync', action: 'sync', requiresOnline: true },
    { icon: RefreshCw, label: 'Restart', action: 'restart', requiresOnline: true },
    { icon: Power, label: 'Disconnect Node', action: 'disconnect', requiresOnline: false },
  ];

  const handleAction = (action: string) => {
    toast({
      title: `Action: ${action}`,
      description: `${action} initiated on ${device.name}`,
    });
  };

  return (
    <div className="p-6 space-y-6">
      {/* Back button */}
      <Button variant="ghost" onClick={() => navigate('/console/devices')} className="mb-4">
        <ArrowLeft className="w-4 h-4 mr-2" />
        Back to Devices
      </Button>

      {/* Device Header */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
      >
        <GlassCard className="p-6">
          <div className="flex flex-col md:flex-row md:items-center gap-6">
            <div
              className={`p-4 rounded-2xl ${
                isOnline ? 'bg-primary/20' : 'bg-surface-variant'
              }`}
            >
              <DeviceIcon
                className={`w-10 h-10 ${
                  isOnline ? 'text-primary' : 'text-muted-foreground'
                }`}
              />
            </div>

            <div className="flex-1">
              <div className="flex items-center gap-3 mb-2">
                <h1 className="text-2xl font-bold">{device.name}</h1>
                <span
                  className={`px-3 py-1 rounded-full text-sm font-medium ${
                    isOnline
                      ? 'bg-safe-green/20 text-safe-green'
                      : 'bg-muted text-muted-foreground'
                  }`}
                >
                  {isOnline ? 'Online' : 'Offline'}
                </span>
              </div>
              <p className="text-muted-foreground">{device.os}</p>
              {device.ip && (
                <p className="text-sm font-mono text-muted-foreground mt-1">
                  {device.ip}
                </p>
              )}
            </div>
          </div>
        </GlassCard>
      </motion.div>

      {/* Remote View Placeholder */}
      {isOnline && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
        >
          <GlassContainer className="aspect-video flex items-center justify-center">
            <div className="text-center space-y-4">
              <div className="w-16 h-16 mx-auto rounded-full bg-surface-variant flex items-center justify-center">
                <Eye className="w-8 h-8 text-muted-foreground" />
              </div>
              <div>
                <p className="font-semibold">Remote View</p>
                <p className="text-sm text-muted-foreground">
                  Click to start remote viewing session
                </p>
              </div>
            </div>
          </GlassContainer>
        </motion.div>
      )}

      {!isOnline && (
        <GlassCard className="p-6 text-center">
          <div className="flex items-center justify-center gap-2 text-muted-foreground">
            <WifiOff className="w-5 h-5" />
            <span>Device is offline. Last seen: {device.lastSeen}</span>
          </div>
        </GlassCard>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Telemetry */}
        {isOnline && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
          >
            <GlassContainer>
              <h2 className="text-lg font-semibold mb-6">Telemetry</h2>
              <div className="flex items-center justify-center gap-12">
                <MiniGauge value={device.cpuPct} label="CPU" variant="info" />
                <MiniGauge value={device.memPct} label="Memory" variant="primary" />
              </div>
            </GlassContainer>
          </motion.div>
        )}

        {/* Capabilities */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.25 }}
        >
          <GlassContainer>
            <h2 className="text-lg font-semibold mb-4">Capabilities</h2>
            <div className="flex flex-wrap gap-2">
              {device.capabilities.map((cap) => (
                <CapabilityChip key={cap} capability={cap} size="md" />
              ))}
            </div>
          </GlassContainer>
        </motion.div>
      </div>

      {/* Actions */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.3 }}
      >
        <GlassContainer>
          <h2 className="text-lg font-semibold mb-4">Actions</h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {actions.map((action) => (
              <Button
                key={action.action}
                variant="outline"
                className="flex-col h-auto py-4 gap-2 border-outline hover:bg-surface-variant"
                disabled={action.requiresOnline && !isOnline}
                onClick={() => handleAction(action.label)}
              >
                <action.icon className="w-5 h-5" />
                <span className="text-xs">{action.label}</span>
              </Button>
            ))}
          </div>
        </GlassContainer>
      </motion.div>

      {/* Hardware Info */}
      {device.hardware && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.35 }}
        >
          <GlassContainer>
            <h2 className="text-lg font-semibold mb-4">Hardware</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {Object.entries(device.hardware).map(([key, value]) => (
                <div key={key} className="flex justify-between py-2 border-b border-outline last:border-0">
                  <span className="text-muted-foreground capitalize">{key}</span>
                  <span className="font-medium">{value}</span>
                </div>
              ))}
            </div>
          </GlassContainer>
        </motion.div>
      )}
    </div>
  );
};
