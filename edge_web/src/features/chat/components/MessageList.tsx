import { useRef, useEffect } from 'react';
import type { ChatMessage, PlanPayload } from '../types';
import { MessageItem } from './MessageItem';
import { ThinkingIndicator } from './ThinkingIndicator';

interface MessageListProps {
  messages: ChatMessage[];
  isThinking: boolean;
  onRunPlan?: (payload: PlanPayload) => void;
}

export function MessageList({ messages, isThinking, onRunPlan }: MessageListProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when messages change or thinking state changes
  useEffect(() => {
    if (bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages, isThinking]);

  return (
    <div ref={scrollRef} className="flex-1 overflow-y-auto px-5 py-3">
      {messages.map((message) => (
        <MessageItem key={message.id} message={message} onRunPlan={onRunPlan} />
      ))}

      {isThinking && (
        <div className="flex justify-start mb-6">
          <ThinkingIndicator />
        </div>
      )}

      <div ref={bottomRef} />
    </div>
  );
}
