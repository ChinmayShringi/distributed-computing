import { motion } from 'framer-motion';
import { cn } from '@/lib/utils';

interface TextBubbleProps {
  text: string;
  isAssistant: boolean;
}

export function TextBubble({ text, isAssistant }: TextBubbleProps) {
  return (
    <motion.div
      initial={{ opacity: 0, x: isAssistant ? -10 : 10 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.4 }}
      className={cn(
        'max-w-[80%]',
        isAssistant
          ? 'text-foreground font-medium'
          : 'bg-surface-2 rounded-2xl rounded-br-md px-4 py-3'
      )}
    >
      <p className="text-[15px] leading-relaxed whitespace-pre-wrap">{text}</p>
    </motion.div>
  );
}
