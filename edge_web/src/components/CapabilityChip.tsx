import { cn } from '@/lib/utils';
import { 
  Cpu, 
  Container, 
  Terminal, 
  Wifi, 
  Monitor, 
  Fingerprint,
  LucideIcon 
} from 'lucide-react';

interface CapabilityChipProps {
  capability: string;
  size?: 'sm' | 'md';
}

const capabilityIcons: Record<string, LucideIcon> = {
  'GPU Compute': Cpu,
  'NPU': Cpu,
  'Docker': Container,
  'SSH': Terminal,
  'Kubernetes': Container,
  'GPIO': Wifi,
  'ADB': Terminal,
  'Screen Mirror': Monitor,
  'default': Fingerprint,
};

export const CapabilityChip = ({ capability, size = 'sm' }: CapabilityChipProps) => {
  const Icon = capabilityIcons[capability] || capabilityIcons.default;
  
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-md bg-surface-variant border border-outline',
        size === 'sm' ? 'px-2 py-0.5 text-xs' : 'px-3 py-1 text-sm',
        'text-muted-foreground'
      )}
    >
      <Icon className={size === 'sm' ? 'w-3 h-3' : 'w-4 h-4'} />
      {capability}
    </span>
  );
};
