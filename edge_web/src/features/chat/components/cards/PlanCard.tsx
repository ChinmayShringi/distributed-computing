import { Settings2 } from 'lucide-react';
import { GlassCard } from '@/components/GlassCard';
import { cn } from '@/lib/utils';
import type { PlanPayload } from '../../types';

interface PlanCardProps {
  payload: PlanPayload;
  onRun: () => void;
  onEdit?: () => void;
}

function RiskBadge({ risk }: { risk: string }) {
  const isSafe = risk === 'SAFE' || risk === 'LOW';
  return (
    <span
      className={cn(
        'px-2 py-1 rounded text-[8px] font-black tracking-wide border',
        isSafe
          ? 'bg-safe-green/10 text-safe-green border-safe-green/30'
          : 'bg-warning-amber/10 text-warning-amber border-warning-amber/30'
      )}
    >
      {risk.toUpperCase()}
    </span>
  );
}

export function PlanCard({ payload, onRun, onEdit }: PlanCardProps) {
  const { steps = [], risk = 'SAFE', cmd = '', device = '' } = payload;

  return (
    <GlassCard className="p-5 rounded-[20px]">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <span className="text-[10px] font-black text-muted-foreground tracking-[1.5px]">
          PROPOSED PLAN
        </span>
        <RiskBadge risk={risk} />
      </div>

      {/* Command */}
      <pre className="font-mono text-sm font-bold text-foreground mb-3">{cmd}</pre>

      {/* Target */}
      <p className="text-[11px] font-semibold text-muted-foreground mb-5">
        Target: {device}
      </p>

      {/* Steps */}
      <div className="space-y-3 mb-5">
        {steps.map((step, index) => (
          <div key={index} className="flex gap-2">
            <span className="font-mono text-[11px] font-bold text-safe-green">
              {index + 1}.
            </span>
            <span className="text-xs text-foreground">{step}</span>
          </div>
        ))}
      </div>

      {/* Actions */}
      <div className="flex gap-3">
        <button
          onClick={onRun}
          className="flex-1 bg-safe-green text-background font-black text-xs tracking-wide
                     py-4 rounded-xl hover:bg-safe-green/90 transition-colors"
        >
          RUN
        </button>
        {onEdit && (
          <button
            onClick={onEdit}
            className="p-4 bg-surface-2 rounded-xl hover:bg-surface-variant transition-colors"
          >
            <Settings2 className="w-5 h-5 text-muted-foreground" />
          </button>
        )}
      </div>
    </GlassCard>
  );
}
