import { cn } from '@/lib/utils';

interface RiskBadgeProps {
  level: 'low' | 'medium' | 'high';
}

export const RiskBadge = ({ level }: RiskBadgeProps) => {
  return (
    <span
      className={cn(
        'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium uppercase tracking-wide',
        level === 'low' && 'bg-safe-green/20 text-safe-green',
        level === 'medium' && 'bg-warning-amber/20 text-warning-amber',
        level === 'high' && 'bg-danger-pink/20 text-danger-pink'
      )}
    >
      {level}
    </span>
  );
};
