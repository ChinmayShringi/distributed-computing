import { cn } from '@/lib/utils';
import { LucideIcon } from 'lucide-react';

interface IconBadgeProps {
  icon: LucideIcon;
  variant?: 'primary' | 'safe' | 'warning' | 'info' | 'muted';
  size?: 'sm' | 'md' | 'lg';
}

const variantStyles = {
  primary: 'bg-primary/20 text-primary border-primary/30',
  safe: 'bg-safe-green/20 text-safe-green border-safe-green/30',
  warning: 'bg-warning-amber/20 text-warning-amber border-warning-amber/30',
  info: 'bg-info-blue/20 text-info-blue border-info-blue/30',
  muted: 'bg-muted/50 text-muted-foreground border-outline',
};

const sizeStyles = {
  sm: 'p-2',
  md: 'p-3',
  lg: 'p-4',
};

const iconSizes = {
  sm: 'w-4 h-4',
  md: 'w-5 h-5',
  lg: 'w-6 h-6',
};

export const IconBadge = ({ icon: Icon, variant = 'primary', size = 'md' }: IconBadgeProps) => {
  return (
    <div
      className={cn(
        'inline-flex items-center justify-center rounded-xl border',
        'shadow-lg shadow-black/20',
        variantStyles[variant],
        sizeStyles[size]
      )}
    >
      <Icon className={iconSizes[size]} />
    </div>
  );
};
