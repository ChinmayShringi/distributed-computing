import { cn } from '@/lib/utils';
import {
  Cpu,
  Container,
  Terminal,
  Wifi,
  Monitor,
  Fingerprint,
  Zap,
  Brain,
  LucideIcon,
} from 'lucide-react';

interface CapabilityChipProps {
  capability: string;
  size?: 'sm' | 'md';
}

const capabilityConfig: Record<string, { icon: LucideIcon; color: string }> = {
  // Hardware capabilities
  GPU: { icon: Zap, color: 'bg-safe-green/20 text-safe-green border-safe-green/30' },
  NPU: { icon: Brain, color: 'bg-purple-500/20 text-purple-400 border-purple-500/30' },
  'GPU Compute': { icon: Zap, color: 'bg-safe-green/20 text-safe-green border-safe-green/30' },

  // Screen capabilities
  'Screen Capture': { icon: Monitor, color: 'bg-info-blue/20 text-info-blue border-info-blue/30' },
  'Screen Mirror': { icon: Monitor, color: 'bg-info-blue/20 text-info-blue border-info-blue/30' },

  // LLM capabilities
  LLM: { icon: Brain, color: 'bg-safe-green/20 text-safe-green border-safe-green/30' },

  // Infrastructure
  Docker: { icon: Container, color: 'bg-info-blue/20 text-info-blue border-info-blue/30' },
  Kubernetes: { icon: Container, color: 'bg-info-blue/20 text-info-blue border-info-blue/30' },
  SSH: { icon: Terminal, color: 'bg-surface-variant text-muted-foreground border-outline' },
  ADB: { icon: Terminal, color: 'bg-surface-variant text-muted-foreground border-outline' },
  GPIO: { icon: Wifi, color: 'bg-warning-amber/20 text-warning-amber border-warning-amber/30' },

  // Default
  default: { icon: Fingerprint, color: 'bg-surface-variant text-muted-foreground border-outline' },
};

export const CapabilityChip = ({ capability, size = 'sm' }: CapabilityChipProps) => {
  // Check for LLM with model name
  const isLLMWithModel = capability.startsWith('LLM:');
  const displayText = isLLMWithModel ? capability : capability;

  // Get config - check for LLM prefix or exact match
  let config = capabilityConfig[capability];
  if (!config && isLLMWithModel) {
    config = capabilityConfig.LLM;
  }
  if (!config) {
    config = capabilityConfig.default;
  }

  const { icon: Icon, color } = config;

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-md border',
        color,
        size === 'sm' ? 'px-2 py-0.5 text-xs' : 'px-3 py-1 text-sm'
      )}
    >
      <Icon className={size === 'sm' ? 'w-3 h-3' : 'w-4 h-4'} />
      {displayText}
    </span>
  );
};
