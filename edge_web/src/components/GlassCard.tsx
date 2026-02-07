import { cn } from '@/lib/utils';
import { motion, HTMLMotionProps } from 'framer-motion';
import { ReactNode } from 'react';

interface GlassCardProps extends Omit<HTMLMotionProps<'div'>, 'children'> {
  children: ReactNode;
  className?: string;
  hover?: boolean;
  glow?: 'primary' | 'safe' | 'none';
}

export const GlassCard = ({ 
  children, 
  className, 
  hover = false, 
  glow = 'none',
  ...props 
}: GlassCardProps) => {
  return (
    <motion.div
      className={cn(
        'glass-card p-4',
        hover && 'transition-all duration-300 hover:border-primary/50 hover:shadow-lg hover:shadow-primary/10',
        glow === 'primary' && 'glow-primary',
        glow === 'safe' && 'glow-safe',
        className
      )}
      {...props}
    >
      {children}
    </motion.div>
  );
};

interface GlassContainerProps {
  children: ReactNode;
  className?: string;
}

export const GlassContainer = ({ children, className }: GlassContainerProps) => {
  return (
    <div className={cn('glass-panel', className)}>
      {children}
    </div>
  );
};
