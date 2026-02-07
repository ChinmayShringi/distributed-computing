import { Smartphone, Laptop, Server, ChevronRight } from 'lucide-react';
import { GlassCard } from '@/components/GlassCard';
import { cn } from '@/lib/utils';
import type { DeviceInfo } from '../../types';

interface DeviceListCardProps {
  devices: DeviceInfo[];
  onDeviceClick?: (device: DeviceInfo) => void;
}

function getDeviceIcon(type: string) {
  switch (type) {
    case 'mobile':
      return Smartphone;
    case 'server':
      return Server;
    default:
      return Laptop;
  }
}

export function DeviceListCard({ devices, onDeviceClick }: DeviceListCardProps) {
  return (
    <div className="space-y-3">
      {devices.map((device) => {
        const Icon = getDeviceIcon(device.type);
        const isOnline = device.status === 'online';

        return (
          <GlassCard
            key={device.device_id}
            className="p-4 rounded-2xl cursor-pointer hover:border-primary/30 transition-colors"
            onClick={() => onDeviceClick?.(device)}
          >
            <div className="flex items-center gap-4">
              {/* Icon */}
              <div
                className={cn(
                  'w-10 h-10 rounded-xl flex items-center justify-center',
                  isOnline ? 'bg-safe-green/20' : 'bg-surface-2'
                )}
              >
                <Icon
                  className={cn(
                    'w-5 h-5',
                    isOnline ? 'text-safe-green' : 'text-muted-foreground'
                  )}
                />
              </div>

              {/* Info */}
              <div className="flex-1 min-w-0">
                <p className="font-bold text-[13px] text-foreground truncate">
                  {device.name}
                </p>
                <p className="text-[10px] text-muted-foreground">
                  {device.os} • {device.status.toUpperCase()}
                </p>
              </div>

              {/* Chevron */}
              <ChevronRight className="w-4 h-4 text-muted-foreground" />
            </div>
          </GlassCard>
        );
      })}
    </div>
  );
}
