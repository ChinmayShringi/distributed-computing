import { cn } from '@/lib/utils';
import { LucideIcon } from 'lucide-react';
import { GlassCard } from './GlassCard';
import { motion } from 'framer-motion';

interface MetricCardProps {
  title: string;
  value: string | number;
  icon: LucideIcon;
  trend?: 'up' | 'down' | 'neutral';
  trendValue?: string;
  variant?: 'primary' | 'safe' | 'warning' | 'info';
}

const variantStyles = {
  primary: 'text-primary',
  safe: 'text-safe-green',
  warning: 'text-warning-amber',
  info: 'text-info-blue',
};

export const MetricCard = ({ 
  title, 
  value, 
  icon: Icon, 
  trend, 
  trendValue,
  variant = 'primary' 
}: MetricCardProps) => {
  return (
    <GlassCard 
      hover
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
    >
      <div className="flex items-start justify-between">
        <div className="space-y-1">
          <p className="text-sm text-muted-foreground">{title}</p>
          <motion.p 
            className={cn('text-3xl font-bold', variantStyles[variant])}
            initial={{ scale: 0.5 }}
            animate={{ scale: 1 }}
            transition={{ duration: 0.3, delay: 0.1 }}
          >
            {value}
          </motion.p>
          {trend && trendValue && (
            <p className={cn(
              'text-xs',
              trend === 'up' && 'text-safe-green',
              trend === 'down' && 'text-danger-pink',
              trend === 'neutral' && 'text-muted-foreground'
            )}>
              {trend === 'up' ? '↑' : trend === 'down' ? '↓' : '→'} {trendValue}
            </p>
          )}
        </div>
        <div className={cn(
          'p-3 rounded-xl bg-surface-variant',
          variantStyles[variant]
        )}>
          <Icon className="w-6 h-6" />
        </div>
      </div>
    </GlassCard>
  );
};
