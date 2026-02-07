import { cn } from '@/lib/utils';
import { Shield, AlertTriangle } from 'lucide-react';

interface ModePillProps {
  mode: 'safe' | 'advanced';
  size?: 'sm' | 'md';
}

export const ModePill = ({ mode, size = 'md' }: ModePillProps) => {
  const isSafe = mode === 'safe';
  
  return (
    <div
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full font-medium transition-all',
        size === 'sm' ? 'px-2 py-0.5 text-xs' : 'px-3 py-1 text-sm',
        isSafe 
          ? 'bg-safe-green/20 text-safe-green border border-safe-green/30' 
          : 'bg-danger-pink/20 text-danger-pink border border-danger-pink/30'
      )}
    >
      {isSafe ? (
        <Shield className={size === 'sm' ? 'w-3 h-3' : 'w-4 h-4'} />
      ) : (
        <AlertTriangle className={size === 'sm' ? 'w-3 h-3' : 'w-4 h-4'} />
      )}
      <span>{isSafe ? 'SAFE' : 'ADVANCED'}</span>
    </div>
  );
};
