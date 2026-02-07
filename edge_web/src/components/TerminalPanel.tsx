import { cn } from '@/lib/utils';
import { Copy, Check, Terminal, XCircle, CheckCircle } from 'lucide-react';
import { useState } from 'react';

interface TerminalPanelProps {
  output: string;
  title?: string;
  exitCode?: number;
  className?: string;
}

export const TerminalPanel = ({ output, title = 'Output', exitCode, className }: TerminalPanelProps) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(output);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className={cn('rounded-lg overflow-hidden border border-outline', className)}>
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2 bg-surface-2 border-b border-outline">
        <div className="flex items-center gap-2">
          <Terminal className="w-4 h-4 text-muted-foreground" />
          <span className="text-sm font-medium text-muted-foreground">{title}</span>
        </div>
        <div className="flex items-center gap-2">
          {exitCode !== undefined && (
            <div className={cn(
              'flex items-center gap-1 px-2 py-0.5 rounded text-xs font-mono',
              exitCode === 0 
                ? 'bg-safe-green/20 text-safe-green' 
                : 'bg-danger-pink/20 text-danger-pink'
            )}>
              {exitCode === 0 ? (
                <CheckCircle className="w-3 h-3" />
              ) : (
                <XCircle className="w-3 h-3" />
              )}
              exit {exitCode}
            </div>
          )}
          <button
            onClick={handleCopy}
            className="p-1.5 rounded-md hover:bg-surface-variant transition-colors"
          >
            {copied ? (
              <Check className="w-4 h-4 text-safe-green" />
            ) : (
              <Copy className="w-4 h-4 text-muted-foreground" />
            )}
          </button>
        </div>
      </div>
      
      {/* Content */}
      <div className="bg-surface-1 p-4 max-h-64 overflow-auto">
        <pre className="text-sm font-mono text-safe-green whitespace-pre-wrap">
          {output}
        </pre>
      </div>
    </div>
  );
};
