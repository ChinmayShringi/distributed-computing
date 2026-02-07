import { useState, useRef, useEffect, useCallback } from 'react';
import { motion } from 'framer-motion';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';
import {
  sendAssistantMessage,
  getJobStatus,
  type AssistantResponse,
  type JobStatusResponse,
} from '@/api';
import {
  Bot,
  Send,
  Loader2,
  ChevronDown,
  FileText,
  AlertCircle,
  Shield,
  ShieldAlert,
} from 'lucide-react';

interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: string;
  response?: AssistantResponse;
  jobStatus?: JobStatusResponse;
}

export const ChatPage = () => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pollingJobId, setPollingJobId] = useState<string | null>(null);

  const chatEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // Poll job status
  const pollJob = useCallback(async (jobId: string, messageId: string) => {
    try {
      const status = await getJobStatus(jobId);
      setMessages((prev) =>
        prev.map((msg) =>
          msg.id === messageId ? { ...msg, jobStatus: status } : msg
        )
      );

      if (status.state !== 'DONE' && status.state !== 'FAILED') {
        // Continue polling
        setTimeout(() => pollJob(jobId, messageId), 500);
      } else {
        setPollingJobId(null);
      }
    } catch {
      setPollingJobId(null);
    }
  }, []);

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
      const response = await sendAssistantMessage(userMessage.content);

      const assistantMessage: ChatMessage = {
        id: `assistant-${Date.now()}`,
        role: 'assistant',
        content: response.reply,
        timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
        response,
      };

      setMessages((prev) => [...prev, assistantMessage]);

      // If a job was created, start polling
      if (response.job_id) {
        setPollingJobId(response.job_id);
        pollJob(response.job_id, assistantMessage.id);
      }
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
      <div className="mb-4">
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <Bot className="w-6 h-6 text-primary" />
          Assistant
        </h1>
        <p className="text-muted-foreground mt-1">
          Natural language interface for EdgeCLI commands
        </p>
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
              <p>Start a conversation with the assistant.</p>
              <p className="text-sm mt-2">
                Try: "List all devices" or "Run hostname on best device"
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

                {/* Assistant response details */}
                {message.response && (
                  <div className="mt-3 pt-3 border-t border-outline/50 space-y-2">
                    {/* Mode badge */}
                    <div className="flex items-center gap-2">
                      {message.response.mode === 'safe' ? (
                        <Badge variant="outline" className="gap-1 text-safe-green border-safe-green">
                          <Shield className="w-3 h-3" />
                          Safe Mode
                        </Badge>
                      ) : (
                        <Badge variant="outline" className="gap-1 text-danger-pink border-danger-pink">
                          <ShieldAlert className="w-3 h-3" />
                          Dangerous Mode
                        </Badge>
                      )}

                      {message.response.job_id && (
                        <Badge variant="secondary" className="font-mono text-xs">
                          Job: {message.response.job_id.slice(0, 8)}...
                        </Badge>
                      )}
                    </div>

                    {/* Job status */}
                    {message.jobStatus && (
                      <div className="text-xs">
                        <span className="text-muted-foreground">Status: </span>
                        <Badge
                          variant={message.jobStatus.state === 'DONE' ? 'default' : 'outline'}
                          className={
                            message.jobStatus.state === 'FAILED'
                              ? 'bg-danger-pink'
                              : message.jobStatus.state === 'DONE'
                              ? 'bg-safe-green'
                              : ''
                          }
                        >
                          {message.jobStatus.state}
                          {pollingJobId === message.response?.job_id && (
                            <Loader2 className="w-3 h-3 ml-1 animate-spin" />
                          )}
                        </Badge>
                        {message.jobStatus.final_result && (
                          <pre className="mt-2 p-2 rounded bg-background text-xs font-mono overflow-auto max-h-[200px]">
                            {message.jobStatus.final_result}
                          </pre>
                        )}
                        {message.jobStatus.error && (
                          <p className="mt-2 text-danger-pink">{message.jobStatus.error}</p>
                        )}
                      </div>
                    )}

                    {/* Plan preview */}
                    {message.response.plan && (
                      <Collapsible>
                        <CollapsibleTrigger asChild>
                          <Button variant="ghost" size="sm" className="w-full justify-between p-2 h-auto">
                            <span className="flex items-center gap-1 text-xs">
                              <FileText className="w-3 h-3" />
                              View Execution Plan
                            </span>
                            <ChevronDown className="w-3 h-3" />
                          </Button>
                        </CollapsibleTrigger>
                        <CollapsibleContent>
                          <pre className="mt-2 p-2 rounded bg-background text-xs font-mono overflow-auto max-h-[200px]">
                            {JSON.stringify(message.response.plan, null, 2)}
                          </pre>
                        </CollapsibleContent>
                      </Collapsible>
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
              placeholder="Ask the assistant..."
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
