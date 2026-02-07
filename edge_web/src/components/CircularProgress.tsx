import { cn } from '@/lib/utils';

export type CircularProgressVariant = 'cpu' | 'memory' | 'gpu' | 'default';

interface CircularProgressProps {
  value: number;
  label?: string;
  variant?: CircularProgressVariant;
  size?: number;
  strokeWidth?: number;
  className?: string;
}

const variantColors: Record<CircularProgressVariant, { stroke: string; bg: string }> = {
  cpu: { stroke: 'stroke-info-blue', bg: 'stroke-info-blue/20' },
  memory: { stroke: 'stroke-primary', bg: 'stroke-primary/20' },
  gpu: { stroke: 'stroke-safe-green', bg: 'stroke-safe-green/20' },
  default: { stroke: 'stroke-primary', bg: 'stroke-primary/20' },
};

export function CircularProgress({
  value,
  label,
  variant = 'default',
  size = 80,
  strokeWidth = 8,
  className,
}: CircularProgressProps) {
  const clampedValue = Math.min(100, Math.max(0, value));
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const strokeDashoffset = circumference - (clampedValue / 100) * circumference;
  const center = size / 2;
  const colors = variantColors[variant];

  return (
    <div className={cn('flex flex-col items-center', className)}>
      <svg width={size} height={size} className="transform -rotate-90">
        {/* Background circle */}
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          strokeWidth={strokeWidth}
          className={colors.bg}
        />
        {/* Progress circle */}
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={strokeDashoffset}
          className={cn(colors.stroke, 'transition-all duration-300')}
        />
      </svg>
      <div
        className="absolute flex flex-col items-center justify-center"
        style={{ width: size, height: size }}
      >
        <span className="text-lg font-semibold">{Math.round(clampedValue)}%</span>
      </div>
      {label && (
        <span className="mt-1 text-xs text-muted-foreground">{label}</span>
      )}
    </div>
  );
}
