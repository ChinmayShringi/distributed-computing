import { motion } from 'framer-motion';
import { cn } from '@/lib/utils';
import type { ChatMessage, PlanPayload, DevicePayload, ResultPayload } from '../types';
import { TextBubble } from './TextBubble';
import { PlanCard } from './cards/PlanCard';
import { DeviceListCard } from './cards/DeviceListCard';
import { ResultCard } from './cards/ResultCard';

interface MessageItemProps {
  message: ChatMessage;
  onRunPlan?: (payload: PlanPayload) => void;
}

export function MessageItem({ message, onRunPlan }: MessageItemProps) {
  const isAssistant = message.sender === 'assistant';

  const renderContent = () => {
    switch (message.type) {
      case 'text':
        return <TextBubble text={message.text || ''} isAssistant={isAssistant} />;

      case 'plan':
        return (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, ease: [0.34, 1.56, 0.64, 1] }}
            className="max-w-[90%]"
          >
            <PlanCard
              payload={message.payload as PlanPayload}
              onRun={() => onRunPlan?.(message.payload as PlanPayload)}
            />
          </motion.div>
        );

      case 'devices':
        return (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, ease: [0.34, 1.56, 0.64, 1] }}
            className="max-w-[90%]"
          >
            <DeviceListCard
              devices={(message.payload as DevicePayload)?.devices || []}
            />
          </motion.div>
        );

      case 'result':
        return (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, ease: [0.34, 1.56, 0.64, 1] }}
            className="max-w-[90%]"
          >
            <ResultCard payload={message.payload as ResultPayload} />
          </motion.div>
        );

      case 'job':
      case 'stream':
      case 'download':
      case 'approval':
        // For these types, render as text with the message
        return <TextBubble text={message.text || ''} isAssistant={isAssistant} />;

      default:
        return null;
    }
  };

  return (
    <div
      className={cn(
        'mb-6',
        isAssistant ? 'flex justify-start' : 'flex justify-end'
      )}
    >
      {renderContent()}
    </div>
  );
}
