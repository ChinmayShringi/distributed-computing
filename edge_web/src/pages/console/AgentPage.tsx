import { useState, useRef, useEffect } from 'react';
import { motion } from 'framer-motion';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import {
  getAgentHealth,
  sendAgentMessage,
  type AgentHealthResponse,
  type AgentResponse,
  type ToolCall,
} from '@/api';
import {
  Bot,
  Send,
  Loader2,
  RefreshCw,
  CheckCircle,
  XCircle,
  AlertCircle,
  Wrench,
  User,
  Sparkles,
} from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  toolCalls?: ToolCall[];
  iterations?: number;
  timestamp: Date;
}

export const AgentPage = () => {
  const { toast } = useToast();
  const chatEndRef = useRef<HTMLDivElement>(null);

  // Health state
  const [healthLoading, setHealthLoading] = useState(false);
  const [health, setHealth] = useState<AgentHealthResponse | null>(null);

  // Chat state
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);

  // Scroll to bottom
  const scrollToBottom = () => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // Check health on mount
  useEffect(() => {
    handleHealthCheck();
  }, []);

  // Health check
  const handleHealthCheck = async () => {
    setHealthLoading(true);
    try {
      const result = await getAgentHealth();
      setHealth(result);
    } catch (err) {
      console.error('Health check failed:', err);
      setHealth({
        ok: false,
        provider: 'unknown',
        base_url: '',
        model: '',
        error: err instanceof Error ? err.message : 'Health check failed',
      });
    } finally {
      setHealthLoading(false);
    }
  };

  // Send message
  const handleSend = async () => {
    if (!input.trim() || sending) return;

    const userMessage: ChatMessage = {
      id: `msg-${Date.now()}`,
      role: 'user',
      content: input,
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInput('');
    setSending(true);

    try {
      const response = await sendAgentMessage(input);

      const assistantMessage: ChatMessage = {
        id: `msg-${Date.now() + 1}`,
        role: 'assistant',
        content: response.reply,
        toolCalls: response.tool_calls,
        iterations: response.iterations,
        timestamp: new Date(),
      };

      setMessages((prev) => [...prev, assistantMessage]);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to send message';
      toast({
        title: 'Error',
        description: message,
        variant: 'destructive',
      });

      // Add error message
      const errorMessage: ChatMessage = {
        id: `msg-${Date.now() + 1}`,
        role: 'assistant',
        content: `Error: ${message}`,
        timestamp: new Date(),
      };
      setMessages((prev) => [...prev, errorMessage]);
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
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Bot className="w-6 h-6 text-primary" />
            LLM Agent
          </h1>
          <p className="text-muted-foreground mt-1">
            Chat with an AI agent that can use tools.
          </p>
        </div>
        <Button
          variant="outline"
          size="icon"
          onClick={handleHealthCheck}
          disabled={healthLoading}
        >
          <RefreshCw className={`w-4 h-4 ${healthLoading ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      {/* Health Status */}
      <GlassCard className="p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            {health?.ok ? (
              <CheckCircle className="w-6 h-6 text-safe-green" />
            ) : (
              <XCircle className="w-6 h-6 text-danger-pink" />
            )}
            <div>
              <p className="font-medium">
                {health?.ok ? 'Agent Available' : 'Agent Unavailable'}
              </p>
              {health && (
                <p className="text-sm text-muted-foreground">
                  {health.provider} / {health.model}
                </p>
              )}
            </div>
          </div>
          {health && (
            <div className="flex gap-2">
              <Badge variant="outline">{health.provider}</Badge>
              <Badge variant="secondary">{health.model}</Badge>
            </div>
          )}
        </div>
        {health?.error && (
          <div className="mt-3 flex items-center gap-2 text-sm text-danger-pink">
            <AlertCircle className="w-4 h-4" />
            {health.error}
          </div>
        )}
      </GlassCard>

      {/* Chat Container */}
      <GlassContainer className="h-[500px] flex flex-col p-0">
        {/* Chat Header */}
        <div className="p-4 border-b border-outline flex items-center gap-3">
          <div className="p-2 rounded-lg bg-primary/20">
            <Sparkles className="w-5 h-5 text-primary" />
          </div>
          <div>
            <h3 className="font-semibold">Tool-Enabled Agent</h3>
            <p className="text-xs text-muted-foreground">
              {health?.model || 'Connecting...'}
            </p>
          </div>
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {messages.length === 0 && (
            <div className="text-center py-12 text-muted-foreground">
              <Bot className="w-12 h-12 mx-auto mb-3 opacity-50" />
              <p>Start a conversation with the agent</p>
              <p className="text-sm">The agent can call tools to help you</p>
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
                className={`max-w-[80%] ${
                  message.role === 'user'
                    ? 'bg-primary text-primary-foreground rounded-2xl rounded-br-md'
                    : 'bg-surface-2 rounded-2xl rounded-bl-md'
                } px-4 py-3`}
              >
                {/* Role Icon */}
                <div className="flex items-center gap-2 mb-1">
                  {message.role === 'user' ? (
                    <User className="w-4 h-4" />
                  ) : (
                    <Bot className="w-4 h-4" />
                  )}
                  <span className="text-xs opacity-70">
                    {message.role === 'user' ? 'You' : 'Agent'}
                  </span>
                  {message.iterations !== undefined && message.iterations > 0 && (
                    <Badge variant="outline" className="text-xs py-0 h-5">
                      {message.iterations} iteration{message.iterations > 1 ? 's' : ''}
                    </Badge>
                  )}
                </div>

                {/* Content */}
                <p className="text-sm whitespace-pre-wrap">{message.content}</p>

                {/* Tool Calls */}
                {message.toolCalls && message.toolCalls.length > 0 && (
                  <div className="mt-3 pt-3 border-t border-outline/30 space-y-2">
                    <p className="text-xs opacity-70 flex items-center gap-1">
                      <Wrench className="w-3 h-3" />
                      Tools used:
                    </p>
                    <div className="flex flex-wrap gap-2">
                      {message.toolCalls.map((tool, i) => (
                        <Badge key={i} variant="secondary" className="text-xs gap-1">
                          <Wrench className="w-3 h-3" />
                          {tool.name}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}

                {/* Timestamp */}
                <p
                  className={`text-xs mt-2 ${
                    message.role === 'user' ? 'text-primary-foreground/70' : 'text-muted-foreground'
                  }`}
                >
                  {message.timestamp.toLocaleTimeString([], {
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </p>
              </div>
            </motion.div>
          ))}

          {/* Typing Indicator */}
          {sending && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              className="flex justify-start"
            >
              <div className="bg-surface-2 rounded-2xl rounded-bl-md px-4 py-3">
                <div className="flex items-center gap-2">
                  <Loader2 className="w-4 h-4 animate-spin" />
                  <span className="text-sm text-muted-foreground">Thinking...</span>
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
              placeholder="Ask the agent anything..."
              className="bg-surface-2 border-outline"
              disabled={sending || !health?.ok}
            />
            <Button
              onClick={handleSend}
              disabled={!input.trim() || sending || !health?.ok}
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

      {/* Info */}
      <GlassCard className="p-4">
        <div className="flex items-start gap-3">
          <AlertCircle className="w-5 h-5 text-info-blue flex-shrink-0 mt-0.5" />
          <div className="text-sm text-muted-foreground">
            <p className="font-medium text-foreground mb-1">About LLM Agent</p>
            <ul className="list-disc list-inside space-y-1">
              <li>The agent can call tools to perform actions on your behalf</li>
              <li>Available tools include file reading, command execution, and more</li>
              <li>Uses Ollama or OpenAI-compatible API as the backend</li>
              <li>Configure via CHAT_* environment variables on the server</li>
            </ul>
          </div>
        </div>
      </GlassCard>
    </div>
  );
};
