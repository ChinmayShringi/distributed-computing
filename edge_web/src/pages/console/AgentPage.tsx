import { useState, useRef, useEffect, useCallback } from 'react';
import { motion } from 'framer-motion';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import {
  sendAgentMessage,
  getAgentHealth,
  getChatMemory,
  type AgentResponse,
  type AgentHealthResponse,
  type ChatMessage as ApiChatMessage,
} from '@/api';
import {
  Bot,
  Send,
  Loader2,
  AlertCircle,
  Stethoscope,
  Wrench,
  RefreshCw,
  CheckCircle,
  XCircle,
} from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: string;
  toolName?: string;
  iterations?: number;
}

export const AgentPage = () => {
  const { toast } = useToast();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [health, setHealth] = useState<AgentHealthResponse | null>(null);
  const [healthLoading, setHealthLoading] = useState(false);

  const chatEndRef = useRef<HTMLDivElement>(null);
  const lastMessageCountRef = useRef(0);
  const pollingRef = useRef<number | null>(null);

  const scrollToBottom = () => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // Check agent health on mount
  useEffect(() => {
    checkHealth();
  }, []);

  const checkHealth = async () => {
    setHealthLoading(true);
    try {
      const result = await getAgentHealth();
      setHealth(result);
      toast({
        title: result.available ? 'Agent Available' : 'Agent Unavailable',
        description: result.available
          ? `Provider: ${result.provider}, Model: ${result.model}`
          : result.error,
        variant: result.available ? 'default' : 'destructive',
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to check health';
      setHealth({ provider: '', model: '', available: false, error: message });
    } finally {
      setHealthLoading(false);
    }
  };

  // Poll chat memory
  const pollChatMemory = useCallback(async () => {
    if (sending) return; // Don't poll while sending

    try {
      const memory = await getChatMemory();
      if (memory.messages.length > lastMessageCountRef.current) {
        // New messages from server
        const newApiMessages = memory.messages.slice(lastMessageCountRef.current);
        const newMessages: ChatMessage[] = newApiMessages.map((msg, idx) => ({
          id: `memory-${Date.now()}-${idx}`,
          role: msg.role,
          content: msg.content,
          timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
        }));

        setMessages((prev) => [...prev, ...newMessages]);
        lastMessageCountRef.current = memory.messages.length;
      }
    } catch {
      // Ignore polling errors
    }
  }, [sending]);

  // Start/stop polling
  useEffect(() => {
    pollingRef.current = window.setInterval(pollChatMemory, 2000);
    return () => {
      if (pollingRef.current) {
        window.clearInterval(pollingRef.current);
      }
    };
  }, [pollChatMemory]);

  const handleSend = async () => {
    if (!input.trim() || sending) return;

    const userMessage: ChatMessage = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: input.trim(),
      timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInput('');
    setSending(true);
    setError(null);

    try {
      const response = await sendAgentMessage(userMessage.content);

      const assistantMessage: ChatMessage = {
        id: `assistant-${Date.now()}`,
        role: 'assistant',
        content: response.reply,
        timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
        toolName: response.tool_name,
        iterations: response.iterations,
      };

      setMessages((prev) => [...prev, assistantMessage]);
      lastMessageCountRef.current += 2; // Account for user + assistant message
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to send message';
      setError(message);
    } finally {
      setSending(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="p-6 h-[calc(100vh-4rem)] flex flex-col">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Wrench className="w-6 h-6 text-primary" />
            Agent
          </h1>
          <p className="text-muted-foreground mt-1">
            LLM agent with tool calling capabilities
          </p>
        </div>

        <div className="flex items-center gap-2">
          {health && (
            <Badge
              variant="outline"
              className={health.available ? 'text-safe-green border-safe-green' : 'text-danger-pink border-danger-pink'}
            >
              {health.available ? (
                <CheckCircle className="w-3 h-3 mr-1" />
              ) : (
                <XCircle className="w-3 h-3 mr-1" />
              )}
              {health.available ? health.model : 'Unavailable'}
            </Badge>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={checkHealth}
            disabled={healthLoading}
          >
            {healthLoading ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Stethoscope className="w-4 h-4" />
            )}
          </Button>
        </div>
      </div>

      {/* Error Display */}
      {error && (
        <div className="flex items-center gap-3 p-4 rounded-lg bg-danger-pink/10 border border-danger-pink/20 mb-4">
          <AlertCircle className="w-5 h-5 text-danger-pink" />
          <span className="text-danger-pink">{error}</span>
        </div>
      )}

      {/* Chat Container */}
      <GlassContainer className="flex-1 flex flex-col p-0 overflow-hidden">
        {/* Messages */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {messages.length === 0 && (
            <div className="text-center py-12 text-muted-foreground">
              <Bot className="w-12 h-12 mx-auto mb-4 opacity-50" />
              <p>Start a conversation with the agent.</p>
              <p className="text-sm mt-2">
                The agent can use tools to help complete tasks.
              </p>
            </div>
          )}

          {messages.map((message) => (
            <motion.div
              key={message.id}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
            >
              <div
                className={`max-w-[85%] rounded-2xl px-4 py-3 ${
                  message.role === 'user'
                    ? 'bg-primary text-primary-foreground rounded-br-md'
                    : 'bg-surface-2 rounded-bl-md'
                }`}
              >
                <p className="text-sm whitespace-pre-wrap">{message.content}</p>

                {/* Tool usage info */}
                {message.toolName && (
                  <div className="mt-2 pt-2 border-t border-outline/50 flex items-center gap-2">
                    <Badge variant="secondary" className="gap-1 text-xs">
                      <Wrench className="w-3 h-3" />
                      {message.toolName}
                    </Badge>
                    {message.iterations !== undefined && message.iterations > 0 && (
                      <Badge variant="outline" className="text-xs">
                        {message.iterations} iteration{message.iterations > 1 ? 's' : ''}
                      </Badge>
                    )}
                  </div>
                )}

                <p
                  className={`text-xs mt-2 ${
                    message.role === 'user' ? 'text-primary-foreground/70' : 'text-muted-foreground'
                  }`}
                >
                  {message.timestamp}
                </p>
              </div>
            </motion.div>
          ))}

          {/* Typing indicator */}
          {sending && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              className="flex justify-start"
            >
              <div className="bg-surface-2 rounded-2xl rounded-bl-md px-4 py-3">
                <div className="flex items-center gap-1">
                  <div className="w-2 h-2 rounded-full bg-muted-foreground animate-bounce" style={{ animationDelay: '0ms' }} />
                  <div className="w-2 h-2 rounded-full bg-muted-foreground animate-bounce" style={{ animationDelay: '150ms' }} />
                  <div className="w-2 h-2 rounded-full bg-muted-foreground animate-bounce" style={{ animationDelay: '300ms' }} />
                </div>
              </div>
            </motion.div>
          )}

          <div ref={chatEndRef} />
        </div>

        {/* Input */}
        <div className="p-4 border-t border-outline">
          <div className="flex gap-2">
            <Input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Ask the agent..."
              className="bg-surface-2 border-outline"
              disabled={sending}
            />
            <Button
              onClick={handleSend}
              disabled={!input.trim() || sending}
              className="bg-primary hover:bg-primary/90"
            >
              {sending ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Send className="w-4 h-4" />
              )}
            </Button>
          </div>
        </div>
      </GlassContainer>
    </div>
  );
};
