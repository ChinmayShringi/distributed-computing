import { cn } from '@/lib/utils';

interface MiniGaugeProps {
  value: number;
  max?: number;
  label: string;
  unit?: string;
  variant?: 'primary' | 'safe' | 'warning' | 'info';
}

export const MiniGauge = ({ 
  value, 
  max = 100, 
  label, 
  unit = '%',
  variant = 'primary' 
}: MiniGaugeProps) => {
  const percentage = (value / max) * 100;
  const circumference = 2 * Math.PI * 28;
  const offset = circumference - (percentage / 100) * circumference;

  const variantColors = {
    primary: 'stroke-primary',
    safe: 'stroke-safe-green',
    warning: 'stroke-warning-amber',
    info: 'stroke-info-blue',
  };

  return (
    <div className="flex flex-col items-center gap-2">
      <div className="relative w-20 h-20">
        <svg className="w-20 h-20 -rotate-90" viewBox="0 0 64 64">
          {/* Background circle */}
          <circle
            cx="32"
            cy="32"
            r="28"
            fill="none"
            stroke="hsl(var(--outline))"
            strokeWidth="4"
          />
          {/* Progress circle */}
          <circle
            cx="32"
            cy="32"
            r="28"
            fill="none"
            className={cn(variantColors[variant])}
            strokeWidth="4"
            strokeLinecap="round"
            strokeDasharray={circumference}
            strokeDashoffset={offset}
            style={{ transition: 'stroke-dashoffset 0.5s ease' }}
          />
        </svg>
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="text-lg font-bold text-foreground">
            {value}{unit}
          </span>
        </div>
      </div>
      <span className="text-xs text-muted-foreground">{label}</span>
    </div>
  );
};
