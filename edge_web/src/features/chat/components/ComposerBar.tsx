import { useState, KeyboardEvent } from 'react';
import { Plus, Mic, ArrowUp } from 'lucide-react';
import { cn } from '@/lib/utils';

interface ComposerBarProps {
  onSend: (text: string) => void;
  onOpenActions: () => void;
  disabled?: boolean;
}

export function ComposerBar({ onSend, onOpenActions, disabled }: ComposerBarProps) {
  const [value, setValue] = useState('');

  const handleSend = () => {
    const text = value.trim();
    if (text && !disabled) {
      onSend(text);
      setValue('');
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="px-6 py-3 pb-4">
      <div className="flex items-center gap-2">
        {/* Input Container */}
        <div className="flex-1 flex items-center min-h-[48px] bg-[#1E2636] rounded-3xl">
          {/* Add Button */}
          <button
            onClick={onOpenActions}
            className="ml-2 p-2 text-muted-foreground hover:text-foreground transition-colors"
          >
            <Plus className="w-5 h-5" />
          </button>

          {/* Input Field */}
          <input
            type="text"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Message Edge Mesh..."
            disabled={disabled}
            className="flex-1 bg-transparent text-foreground text-sm placeholder:text-muted-foreground
                       focus:outline-none py-3 px-2"
          />

          {/* Mic Button */}
          <button className="mr-3 p-2 text-muted-foreground hover:text-foreground transition-colors">
            <Mic className="w-5 h-5" />
          </button>
        </div>

        {/* Send Button */}
        <button
          onClick={handleSend}
          disabled={!value.trim() || disabled}
          className={cn(
            'w-9 h-9 rounded-full flex items-center justify-center transition-colors',
            value.trim() && !disabled
              ? 'bg-foreground text-background hover:bg-foreground/90'
              : 'bg-surface-2 text-muted-foreground cursor-not-allowed'
          )}
        >
          <ArrowUp className="w-5 h-5" />
        </button>
      </div>
    </div>
  );
}
