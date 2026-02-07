import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { Server, Terminal, PlayCircle, Download } from 'lucide-react';

interface ActionSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAction: (message: string) => void;
}

const actions = [
  { icon: Server, label: 'List Devices', message: 'Show my devices' },
  { icon: Terminal, label: 'Run Command', message: 'Run ls -la' },
  { icon: PlayCircle, label: 'Start Stream', message: 'Start stream' },
  { icon: Download, label: 'Download File', message: 'Download shared file' },
];

export function ActionSheet({ open, onOpenChange, onAction }: ActionSheetProps) {
  const handleAction = (message: string) => {
    onAction(message);
    onOpenChange(false);
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="bottom" className="bg-[#0F1623] border-t border-outline rounded-t-[22px]">
        <SheetHeader className="sr-only">
          <SheetTitle>Quick Actions</SheetTitle>
        </SheetHeader>
        <div className="py-4">
          {actions.map(({ icon: Icon, label, message }) => (
            <button
              key={label}
              onClick={() => handleAction(message)}
              className="w-full flex items-center gap-4 px-4 py-3 text-foreground hover:bg-surface-variant rounded-lg transition-colors"
            >
              <Icon className="w-5 h-5" />
              <span className="text-base">{label}</span>
            </button>
          ))}
        </div>
      </SheetContent>
    </Sheet>
  );
}
