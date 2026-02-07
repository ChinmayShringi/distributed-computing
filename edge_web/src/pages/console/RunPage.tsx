import { useState, useRef, useEffect } from 'react';
import { motion } from 'framer-motion';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { CapabilityChip } from '@/components/CapabilityChip';
import { TerminalPanel } from '@/components/TerminalPanel';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { mockDevices, mockChatMessages, ChatMessage } from '@/lib/mock-data';
import {
  Play,
  Terminal,
  Wrench,
  Bot,
  Network,
  Stethoscope,
  RotateCcw,
  Send,
  Loader2,
} from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

const tools = [
  {
    id: 'network-scan',
    name: 'Network Scan',
    description: 'Scan local network for devices and open ports',
    icon: Network,
    mode: 'safe' as const,
  },
  {
    id: 'diagnostics',
    name: 'System Diagnostics',
    description: 'Run comprehensive system health checks',
    icon: Stethoscope,
    mode: 'advanced' as const,
  },
  {
    id: 'maintenance-reset',
    name: 'Maintenance Reset',
    description: 'Reset device to maintenance mode',
    icon: RotateCcw,
    mode: 'advanced' as const,
  },
];

export const RunPage = () => {
  const { toast } = useToast();
  const [selectedDevice, setSelectedDevice] = useState<string>('');
  const [command, setCommand] = useState('');
  const [output, setOutput] = useState('');
  const [running, setRunning] = useState(false);
  
  // Chat state
  const [messages, setMessages] = useState<ChatMessage[]>(mockChatMessages);
  const [chatInput, setChatInput] = useState('');
  const [isTyping, setIsTyping] = useState(false);
  const chatEndRef = useRef<HTMLDivElement>(null);

  const onlineDevices = mockDevices.filter((d) => d.status === 'online');

  const scrollToBottom = () => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages, isTyping]);

  const handleRun = async () => {
    if (!selectedDevice || !command) return;
    
    setRunning(true);
    setOutput('');
    
    await new Promise((resolve) => setTimeout(resolve, 1500));
    
    setOutput(`$ ${command}\n\n[Executing on ${onlineDevices.find(d => d.id === selectedDevice)?.name}...]\n\nCommand executed successfully.\nExit code: 0`);
    setRunning(false);
  };

  const handleToolRun = (toolName: string) => {
    toast({
      title: 'Tool Started',
      description: `${toolName} is now running...`,
    });
  };

  const handleSendMessage = async () => {
    if (!chatInput.trim()) return;
    
    const userMessage: ChatMessage = {
      id: `msg-${Date.now()}`,
      role: 'user',
      content: chatInput,
      timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    };
    
    setMessages((prev) => [...prev, userMessage]);
    setChatInput('');
    setIsTyping(true);
    
    // Simulate AI response
    await new Promise((resolve) => setTimeout(resolve, 2000));
    
    const responses = [
      "I've analyzed your request. Based on the current device status, I recommend running a diagnostic check first.",
      "I can help you with that. Would you like me to execute this command across all online devices or just a specific one?",
      "That operation completed successfully. The results show normal performance metrics across your mesh network.",
      "I've queued that job for execution. You can monitor its progress in the Jobs section.",
    ];
    
    const assistantMessage: ChatMessage = {
      id: `msg-${Date.now() + 1}`,
      role: 'assistant',
      content: responses[Math.floor(Math.random() * responses.length)],
      timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    };
    
    setIsTyping(false);
    setMessages((prev) => [...prev, assistantMessage]);
  };

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Run</h1>
        <p className="text-muted-foreground mt-1">
          Execute scripts, run tools, and interact with AI.
        </p>
      </div>

      <Tabs defaultValue="script" className="space-y-6">
        <TabsList className="bg-surface-2">
          <TabsTrigger value="script" className="data-[state=active]:bg-primary data-[state=active]:text-primary-foreground">
            <Terminal className="w-4 h-4 mr-2" />
            Script
          </TabsTrigger>
          <TabsTrigger value="tools" className="data-[state=active]:bg-primary data-[state=active]:text-primary-foreground">
            <Wrench className="w-4 h-4 mr-2" />
            Tools
          </TabsTrigger>
          <TabsTrigger value="ai" className="data-[state=active]:bg-primary data-[state=active]:text-primary-foreground">
            <Bot className="w-4 h-4 mr-2" />
            AI Hub
          </TabsTrigger>
        </TabsList>

        {/* Script Tab */}
        <TabsContent value="script" className="space-y-4">
          <GlassContainer>
            <div className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Select value={selectedDevice} onValueChange={setSelectedDevice}>
                  <SelectTrigger className="bg-surface-2 border-outline">
                    <SelectValue placeholder="Select target device" />
                  </SelectTrigger>
                  <SelectContent className="bg-surface-2 border-outline">
                    {onlineDevices.map((device) => (
                      <SelectItem key={device.id} value={device.id}>
                        {device.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>

                <div className="flex gap-2">
                  <Input
                    value={command}
                    onChange={(e) => setCommand(e.target.value)}
                    placeholder="Enter command..."
                    className="font-mono bg-surface-2 border-outline flex-1"
                    onKeyDown={(e) => e.key === 'Enter' && handleRun()}
                  />
                  <Button
                    onClick={handleRun}
                    disabled={!selectedDevice || !command || running}
                    className="bg-safe-green hover:bg-safe-green/90 text-background"
                  >
                    {running ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                      <Play className="w-4 h-4" />
                    )}
                  </Button>
                </div>
              </div>

              {output && (
                <motion.div
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                >
                  <TerminalPanel output={output} exitCode={0} />
                </motion.div>
              )}
            </div>
          </GlassContainer>
        </TabsContent>

        {/* Tools Tab */}
        <TabsContent value="tools">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {tools.map((tool, index) => (
              <motion.div
                key={tool.id}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: index * 0.1 }}
              >
                <GlassCard hover className="p-5">
                  <div className="flex items-start gap-4">
                    <div className="p-3 rounded-xl bg-surface-variant">
                      <tool.icon className="w-6 h-6 text-primary" />
                    </div>
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <h3 className="font-semibold">{tool.name}</h3>
                        <CapabilityChip
                          capability={tool.mode === 'safe' ? 'Safe' : 'Advanced'}
                          size="sm"
                        />
                      </div>
                      <p className="text-sm text-muted-foreground mb-4">
                        {tool.description}
                      </p>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleToolRun(tool.name)}
                        className="border-outline"
                      >
                        <Play className="w-4 h-4 mr-2" />
                        Run Tool
                      </Button>
                    </div>
                  </div>
                </GlassCard>
              </motion.div>
            ))}
          </div>
        </TabsContent>

        {/* AI Hub Tab */}
        <TabsContent value="ai">
          <GlassContainer className="h-[600px] flex flex-col p-0">
            {/* Chat Header */}
            <div className="p-4 border-b border-outline flex items-center gap-3">
              <div className="p-2 rounded-lg bg-primary/20">
                <Bot className="w-5 h-5 text-primary" />
              </div>
              <div>
                <h3 className="font-semibold">AI Assistant</h3>
                <p className="text-xs text-muted-foreground">
                  Ask anything about your mesh network
                </p>
              </div>
            </div>

            {/* Chat Messages */}
            <div className="flex-1 overflow-y-auto p-4 space-y-4">
              {messages.map((message) => (
                <motion.div
                  key={message.id}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
                >
                  <div
                    className={`max-w-[80%] rounded-2xl px-4 py-3 ${
                      message.role === 'user'
                        ? 'bg-primary text-primary-foreground rounded-br-md'
                        : 'bg-surface-2 rounded-bl-md'
                    }`}
                  >
                    <p className="text-sm whitespace-pre-wrap">{message.content}</p>
                    <p
                      className={`text-xs mt-1 ${
                        message.role === 'user' ? 'text-primary-foreground/70' : 'text-muted-foreground'
                      }`}
                    >
                      {message.timestamp}
                    </p>
                  </div>
                </motion.div>
              ))}

              {/* Typing Indicator */}
              {isTyping && (
                <motion.div
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  className="flex justify-start"
                >
                  <div className="bg-surface-2 rounded-2xl rounded-bl-md px-4 py-3">
                    <div className="flex items-center gap-1">
                      <div className="typing-dot" />
                      <div className="typing-dot" />
                      <div className="typing-dot" />
                    </div>
                  </div>
                </motion.div>
              )}

              <div ref={chatEndRef} />
            </div>

            {/* Chat Input */}
            <div className="p-4 border-t border-outline">
              <div className="flex gap-2">
                <Input
                  value={chatInput}
                  onChange={(e) => setChatInput(e.target.value)}
                  placeholder="Ask the AI assistant..."
                  className="bg-surface-2 border-outline"
                  onKeyDown={(e) => e.key === 'Enter' && !e.shiftKey && handleSendMessage()}
                />
                <Button
                  onClick={handleSendMessage}
                  disabled={!chatInput.trim() || isTyping}
                  className="bg-primary hover:bg-primary/90"
                >
                  <Send className="w-4 h-4" />
                </Button>
              </div>
            </div>
          </GlassContainer>
        </TabsContent>
      </Tabs>
    </div>
  );
};
