import { Smartphone, Cpu } from 'lucide-react';
import { GlassCard } from '@/components/GlassCard';
import { TerminalPanel } from '@/components/TerminalPanel';
import { cn } from '@/lib/utils';
import type { ResultPayload } from '../../types';

interface ResultCardProps {
  payload: ResultPayload;
}

function MetricChip({ icon: Icon, label }: { icon: typeof Smartphone; label: string }) {
  return (
    <div className="flex items-center gap-1.5 px-2 py-1.5 bg-surface-2 rounded-lg">
      <Icon className="w-3 h-3 text-muted-foreground" />
      <span className="text-[10px] font-bold text-muted-foreground">{label}</span>
    </div>
  );
}

export function ResultCard({ payload }: ResultCardProps) {
  const { device, host_compute, time, exit_code, output } = payload;
  const isSuccess = exit_code === 0;

  return (
    <GlassCard className="p-5 rounded-[20px]">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <span
          className={cn(
            'text-[10px] font-black tracking-[1.5px]',
            isSuccess ? 'text-safe-green' : 'text-danger-pink'
          )}
        >
          {isSuccess ? 'EXECUTION SUCCESS' : 'EXECUTION FAILED'}
        </span>
        <span className="font-mono text-[10px] text-muted-foreground">{time}</span>
      </div>

      {/* Metrics */}
      <div className="flex gap-2 mb-5">
        <MetricChip icon={Smartphone} label={device} />
        <MetricChip icon={Cpu} label={host_compute} />
      </div>

      {/* Terminal Output */}
      <TerminalPanel output={output} exitCode={exit_code} />
    </GlassCard>
  );
}
