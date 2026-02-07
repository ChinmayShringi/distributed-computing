import { useState, useRef, useEffect, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';
import {
  sendAssistantMessage,
  getJobStatus,
  getChatMemory,
  listDevices,
  type AssistantResponse,
  type JobStatusResponse,
  type Device,
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
  Users,
  Monitor,
  Smartphone,
  Check,
  Search,
} from 'lucide-react';

interface ChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
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
  const [lastUpdated, setLastUpdated] = useState<number>(0);
  const [devices, setDevices] = useState<Device[]>([]);
  const [selectedDeviceId, setSelectedDeviceId] = useState<string>('self');
  const [deviceSearch, setDeviceSearch] = useState('');

  const chatEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  // Load devices
  useEffect(() => {
    const fetchDevices = async () => {
      try {
        const resp = await listDevices();
        setDevices(resp);
      } catch (err) {
        console.error('Failed to fetch devices:', err);
      }
    };
    fetchDevices();
    const interval = setInterval(fetchDevices, 10000);
    return () => clearInterval(interval);
  }, []);

  // Poll chat history
  const pollHistory = useCallback(async () => {
    if (sending) return; // Don't poll while sending to avoid race conditions
    try {
      const data = await getChatMemory(selectedDeviceId);
      if (data.last_updated_ms > lastUpdated) {
        setLastUpdated(data.last_updated_ms);

        const transformedMgs: ChatMessage[] = data.messages.map((m, idx) => ({
          id: `msg-${m.role}-${m.timestamp_ms || idx}`,
          role: m.role === 'system' ? 'assistant' : m.role as 'user' | 'assistant',
          content: m.content,
          timestamp: m.timestamp_ms
            ? new Date(m.timestamp_ms).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
            : '',
        }));

        setMessages(transformedMgs);
      }
    } catch (err) {
      console.error('Failed to poll chat history:', err);
    }
  }, [lastUpdated, selectedDeviceId, sending]);

  useEffect(() => {
    setMessages([]);
    setLastUpdated(0);
    pollHistory();
  }, [selectedDeviceId]);

  useEffect(() => {
    const interval = setInterval(() => {
      pollHistory();
    }, 3000);
    return () => clearInterval(interval);
  }, [pollHistory]);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const pollJob = useCallback(async (jobId: string, messageId: string) => {
    try {
      const status = await getJobStatus(jobId);
      setMessages((prev) =>
        prev.map((msg) =>
          msg.id === messageId ? { ...msg, jobStatus: status } : msg
        )
      );

      if (status.state !== 'DONE' && status.state !== 'FAILED') {
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
      const response = await sendAssistantMessage(userMessage.content, selectedDeviceId === 'self' ? undefined : selectedDeviceId);

      const assistantMessage: ChatMessage = {
        id: `assistant-${Date.now()}`,
        role: 'assistant',
        content: response.reply,
        timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
        response,
      };

      setMessages((prev) => [...prev, assistantMessage]);

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

  const filteredDevices = devices.filter(d =>
    d.device_name.toLowerCase().includes(deviceSearch.toLowerCase()) ||
    d.device_id.toLowerCase().includes(deviceSearch.toLowerCase())
  );

  const selectedDeviceName = selectedDeviceId === 'self'
    ? 'Local Device'
    : devices.find(d => d.device_id === selectedDeviceId)?.device_name || 'Selected Device';

  return (
    <div className="p-6 h-[calc(100vh-4rem)] flex gap-6">
      {/* Sidebar */}
      <div className="w-80 flex flex-col gap-4">
        <GlassCard className="flex-1 flex flex-col p-4 overflow-hidden">
          <div className="flex items-center gap-2 mb-4">
            <Users className="w-5 h-5 text-primary" />
            <h2 className="font-bold">Active Chats</h2>
          </div>

          <div className="relative mb-4">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="Search devices..."
              value={deviceSearch}
              onChange={(e) => setDeviceSearch(e.target.value)}
              className="pl-9 bg-surface-2 border-outline h-9"
            />
          </div>

          <ScrollArea className="flex-1 -mx-2 px-2">
            <div className="space-y-1">
              {/* Local Device Option */}
              <button
                onClick={() => setSelectedDeviceId('self')}
                className={`w-full flex items-center justify-between p-3 rounded-xl transition-all ${selectedDeviceId === 'self'
                  ? 'bg-primary/20 text-primary border border-primary/50'
                  : 'hover:bg-surface-2 border border-transparent'
                  }`}
              >
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full bg-primary/20 flex items-center justify-center shrink-0">
                    <Monitor className="w-5 h-5" />
                  </div>
                  <div className="text-left">
                    <p className="font-semibold text-sm">Local Device</p>
                    <p className="text-xs opacity-70">Main instance</p>
                  </div>
                </div>
                {selectedDeviceId === 'self' && <Check className="w-4 h-4" />}
              </button>

              <div className="my-2 border-t border-outline/50" />

              {/* Other Devices */}
              {filteredDevices.map((device) => (
                <button
                  key={device.device_id}
                  onClick={() => setSelectedDeviceId(device.device_id)}
                  className={`w-full flex items-center justify-between p-3 rounded-xl transition-all ${selectedDeviceId === device.device_id
                    ? 'bg-primary/20 text-primary border border-primary/50'
                    : 'hover:bg-surface-2 border border-transparent'
                    }`}
                >
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-full bg-surface-3 flex items-center justify-center shrink-0">
                      {device.platform === 'android' ? <Smartphone className="w-5 h-5" /> : <Monitor className="w-5 h-5" />}
                    </div>
                    <div className="text-left max-w-[140px]">
                      <p className="font-semibold text-sm truncate">{device.device_name}</p>
                      <p className="text-xs opacity-70 truncate">{device.platform}</p>
                    </div>
                  </div>
                  {selectedDeviceId === device.device_id && <Check className="w-4 h-4" />}
                </button>
              ))}

              {filteredDevices.length === 0 && deviceSearch && (
                <p className="text-center py-4 text-sm text-muted-foreground">No devices found</p>
              )}
            </div>
          </ScrollArea>
        </GlassCard>
      </div>

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col min-w-0">
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold flex items-center gap-2">
              <Bot className="w-6 h-6 text-primary" />
              Assistant
              <Badge variant="outline" className="ml-2 font-normal text-muted-foreground">
                {selectedDeviceName}
              </Badge>
            </h1>
          </div>
        </div>

        {error && (
          <div className="flex items-center gap-3 p-4 rounded-lg bg-danger-pink/10 border border-danger-pink/20 mb-4">
            <AlertCircle className="w-5 h-5 text-danger-pink" />
            <span className="text-danger-pink">{error}</span>
          </div>
        )}

        <GlassContainer className="flex-1 flex flex-col p-0 overflow-hidden">
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {messages.length === 0 && (
              <div className="text-center py-12 text-muted-foreground">
                <Bot className="w-12 h-12 mx-auto mb-4 opacity-50" />
                <p>Start a conversation with the assistant on <strong>{selectedDeviceName}</strong>.</p>
                <p className="text-sm mt-2">
                  Try: "List all devices" or "Run hostname on best device"
                </p>
              </div>
            )}

            <AnimatePresence initial={false}>
              {messages.map((message) => (
                <motion.div
                  key={message.id}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
                >
                  <div
                    className={`max-w-[85%] rounded-2xl px-4 py-3 ${message.role === 'user'
                      ? 'bg-primary text-primary-foreground rounded-br-md'
                      : 'bg-surface-2 rounded-bl-md'
                      }`}
                  >
                    <p className="text-sm whitespace-pre-wrap">{message.content}</p>

                    {message.response && (
                      <div className="mt-3 pt-3 border-t border-outline/50 space-y-2">
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
                      className={`text-xs mt-2 ${message.role === 'user' ? 'text-primary-foreground/70' : 'text-muted-foreground'
                        }`}
                    >
                      {message.timestamp}
                    </p>
                  </div>
                </motion.div>
              ))}
            </AnimatePresence>

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

          <div className="p-4 border-t border-outline">
            <div className="flex gap-2">
              <Input
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder={`Message ${selectedDeviceName}...`}
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
    </div>
  );
};
