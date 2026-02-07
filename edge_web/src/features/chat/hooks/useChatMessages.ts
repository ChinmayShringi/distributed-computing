import { useState, useCallback } from 'react';
import type { ChatMessage, MessageType, MessageSender } from '../types';
import { createMessage } from '../types';

export function useChatMessages() {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isThinking, setIsThinking] = useState(false);

  const addMessage = useCallback(
    (
      sender: MessageSender,
      type: MessageType,
      text?: string,
      payload?: Record<string, unknown>
    ) => {
      const message = createMessage(sender, type, text, payload);
      setMessages((prev) => [...prev, message]);
      return message;
    },
    []
  );

  const addUserMessage = useCallback(
    (text: string) => {
      return addMessage('user', 'text', text);
    },
    [addMessage]
  );

  const addAssistantMessage = useCallback(
    (type: MessageType, text?: string, payload?: Record<string, unknown>) => {
      return addMessage('assistant', type, text, payload);
    },
    [addMessage]
  );

  const setThinking = useCallback((thinking: boolean) => {
    setIsThinking(thinking);
  }, []);

  const clearMessages = useCallback(() => {
    setMessages([]);
  }, []);

  const updateMessage = useCallback(
    (id: string, updates: Partial<ChatMessage>) => {
      setMessages((prev) =>
        prev.map((msg) => (msg.id === id ? { ...msg, ...updates } : msg))
      );
    },
    []
  );

  return {
    messages,
    isThinking,
    addUserMessage,
    addAssistantMessage,
    setThinking,
    clearMessages,
    updateMessage,
  };
}
