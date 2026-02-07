import { cn } from '@/lib/utils';

interface CircularProgressProps {
  value: number; // 0-100
  size?: number; // Size in pixels
  strokeWidth?: number;
  label?: string;
  variant?: 'cpu' | 'memory' | 'gpu' | 'npu' | 'default';
  showValue?: boolean;
  className?: string;
}

const variantColors = {
  cpu: {
    stroke: 'stroke-info-blue',
    text: 'text-info-blue',
    track: 'stroke-info-blue/20',
  },
  memory: {
    stroke: 'stroke-primary',
    text: 'text-primary',
    track: 'stroke-primary/20',
  },
  gpu: {
    stroke: 'stroke-safe-green',
    text: 'text-safe-green',
    track: 'stroke-safe-green/20',
  },
  npu: {
    stroke: 'stroke-purple-500',
    text: 'text-purple-400',
    track: 'stroke-purple-500/20',
  },
  default: {
    stroke: 'stroke-muted-foreground',
    text: 'text-muted-foreground',
    track: 'stroke-muted-foreground/20',
  },
};

export const CircularProgress = ({
  value,
  size = 80,
  strokeWidth = 8,
  label,
  variant = 'default',
  showValue = true,
  className,
}: CircularProgressProps) => {
  const normalizedValue = Math.min(100, Math.max(0, value));
  const radius = (size - strokeWidth) / 2;
  const circumference = radius * 2 * Math.PI;
  const offset = circumference - (normalizedValue / 100) * circumference;

  const colors = variantColors[variant];

  return (
    <div className={cn('flex flex-col items-center gap-1', className)}>
      <div className="relative" style={{ width: size, height: size }}>
        <svg
          width={size}
          height={size}
          className="transform -rotate-90"
        >
          {/* Background track */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            strokeWidth={strokeWidth}
            className={colors.track}
          />
          {/* Progress arc */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            strokeWidth={strokeWidth}
            strokeLinecap="round"
            className={cn(colors.stroke, 'transition-all duration-500 ease-out')}
            style={{
              strokeDasharray: circumference,
              strokeDashoffset: offset,
            }}
          />
        </svg>
        {showValue && (
          <div
            className={cn(
              'absolute inset-0 flex items-center justify-center font-semibold',
              colors.text
            )}
            style={{ fontSize: size * 0.2 }}
          >
            {Math.round(normalizedValue)}%
          </div>
        )}
      </div>
      {label && (
        <span className="text-xs text-muted-foreground font-medium">{label}</span>
      )}
    </div>
  );
};
