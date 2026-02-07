import { useState, useRef, useEffect, useCallback } from 'react';
import { motion } from 'framer-motion';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { CapabilityChip } from '@/components/CapabilityChip';
import { TerminalPanel } from '@/components/TerminalPanel';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import {
  listDevices,
  executeRoutedCommand,
  type Device,
  type RoutingPolicy,
  type RoutedCommandResponse,
} from '@/api';
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
  RefreshCw,
  Clock,
  Server,
  CheckCircle,
  XCircle,
} from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

const routingPolicies: { value: RoutingPolicy; label: string; description: string }[] = [
  { value: 'BEST_AVAILABLE', label: 'Best Available', description: 'NPU > GPU > CPU preference' },
  { value: 'PREFER_REMOTE', label: 'Prefer Remote', description: 'Prefer non-local devices' },
  { value: 'REQUIRE_NPU', label: 'Require NPU', description: 'Fail if no NPU device' },
  { value: 'PREFER_LOCAL_MODEL', label: 'Prefer Local Model', description: 'Prefer device with Ollama' },
  { value: 'REQUIRE_LOCAL_MODEL', label: 'Require Local Model', description: 'Fail if no LLM device' },
  { value: 'FORCE_DEVICE_ID', label: 'Force Device', description: 'Target specific device' },
];

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

interface CommandResult {
  response: RoutedCommandResponse;
  command: string;
  args: string[];
}

export const RunPage = () => {
  const { toast } = useToast();

  // Devices state
  const [devices, setDevices] = useState<Device[]>([]);
  const [loadingDevices, setLoadingDevices] = useState(true);

  // Command form state
  const [command, setCommand] = useState('pwd');
  const [args, setArgs] = useState('');
  const [policy, setPolicy] = useState<RoutingPolicy>('BEST_AVAILABLE');
  const [forceDeviceId, setForceDeviceId] = useState<string>('');

  // Execution state
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<CommandResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Fetch devices
  const fetchDevices = useCallback(async () => {
    try {
      setLoadingDevices(true);
      const data = await listDevices();
      setDevices(data);
      // Set default force device if available
      if (data.length > 0 && !forceDeviceId) {
        setForceDeviceId(data[0].device_id);
      }
    } catch (err) {
      console.error('Failed to fetch devices:', err);
    } finally {
      setLoadingDevices(false);
    }
  }, [forceDeviceId]);

  useEffect(() => {
    fetchDevices();
  }, [fetchDevices]);

  const handleRun = async () => {
    if (!command.trim()) {
      toast({
        title: 'Error',
        description: 'Please enter a command',
        variant: 'destructive',
      });
      return;
    }

    setRunning(true);
    setResult(null);
    setError(null);

    try {
      // Parse args: split by comma, trim whitespace
      const argsArray = args
        .split(',')
        .map((a) => a.trim())
        .filter((a) => a.length > 0);

      const response = await executeRoutedCommand(
        command,
        argsArray,
        policy,
        policy === 'FORCE_DEVICE_ID' ? forceDeviceId : undefined
      );

      setResult({
        response,
        command,
        args: argsArray,
      });

      toast({
        title: 'Command Executed',
        description: `Ran on ${response.selected_device_name} in ${response.total_time_ms}ms`,
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Command execution failed';
      setError(message);
      toast({
        title: 'Execution Failed',
        description: message,
        variant: 'destructive',
      });
    } finally {
      setRunning(false);
    }
  };

  const handleToolRun = (toolName: string) => {
    toast({
      title: 'Tool Started',
      description: `${toolName} is now running...`,
    });
  };

  // Format output for terminal display
  const formatOutput = () => {
    if (!result) return '';

    const { response, command, args } = result;
    const fullCommand = args.length > 0 ? `${command} ${args.join(' ')}` : command;

    let output = `$ ${fullCommand}\n\n`;

    if (response.stdout) {
      output += response.stdout;
    }

    if (response.stderr) {
      output += `\n\n[stderr]\n${response.stderr}`;
    }

    return output;
  };

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Run</h1>
          <p className="text-muted-foreground mt-1">
            Execute commands on distributed devices with routing policies.
          </p>
        </div>
        <Button
          variant="outline"
          size="icon"
          onClick={fetchDevices}
          disabled={loadingDevices}
        >
          <RefreshCw className={`w-4 h-4 ${loadingDevices ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      <Tabs defaultValue="script" className="space-y-6">
        <TabsList className="bg-surface-2">
          <TabsTrigger value="script" className="data-[state=active]:bg-primary data-[state=active]:text-primary-foreground">
            <Terminal className="w-4 h-4 mr-2" />
            Routed Command
          </TabsTrigger>
          <TabsTrigger value="tools" className="data-[state=active]:bg-primary data-[state=active]:text-primary-foreground">
            <Wrench className="w-4 h-4 mr-2" />
            Tools
          </TabsTrigger>
        </TabsList>

        {/* Routed Command Tab */}
        <TabsContent value="script" className="space-y-4">
          <GlassContainer>
            <div className="space-y-6">
              {/* Command Input */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="command">Command</Label>
                  <Input
                    id="command"
                    value={command}
                    onChange={(e) => setCommand(e.target.value)}
                    placeholder="e.g., pwd, ls, echo"
                    className="font-mono bg-surface-2 border-outline"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="args">Arguments (comma-separated)</Label>
                  <Input
                    id="args"
                    value={args}
                    onChange={(e) => setArgs(e.target.value)}
                    placeholder="e.g., -la, /path/to/dir"
                    className="font-mono bg-surface-2 border-outline"
                  />
                </div>
              </div>

              {/* Routing Options */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Routing Policy</Label>
                  <Select value={policy} onValueChange={(v) => setPolicy(v as RoutingPolicy)}>
                    <SelectTrigger className="bg-surface-2 border-outline">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="bg-surface-2 border-outline">
                      {routingPolicies.map((p) => (
                        <SelectItem key={p.value} value={p.value}>
                          <div className="flex flex-col">
                            <span>{p.label}</span>
                            <span className="text-xs text-muted-foreground">{p.description}</span>
                          </div>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                {policy === 'FORCE_DEVICE_ID' && (
                  <div className="space-y-2">
                    <Label>Target Device</Label>
                    <Select value={forceDeviceId} onValueChange={setForceDeviceId}>
                      <SelectTrigger className="bg-surface-2 border-outline">
                        <SelectValue placeholder="Select device" />
                      </SelectTrigger>
                      <SelectContent className="bg-surface-2 border-outline">
                        {devices.map((device) => (
                          <SelectItem key={device.device_id} value={device.device_id}>
                            {device.device_name} ({device.platform})
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                )}
              </div>

              {/* Run Button */}
              <Button
                onClick={handleRun}
                disabled={!command.trim() || running}
                className="bg-safe-green hover:bg-safe-green/90 text-background w-full md:w-auto"
              >
                {running ? (
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                ) : (
                  <Play className="w-4 h-4 mr-2" />
                )}
                Run Command
              </Button>

              {/* Error Display */}
              {error && (
                <motion.div
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  className="p-4 rounded-lg bg-danger-pink/10 border border-danger-pink/20"
                >
                  <div className="flex items-center gap-2 text-danger-pink">
                    <XCircle className="w-5 h-5" />
                    <span>{error}</span>
                  </div>
                </motion.div>
              )}

              {/* Result Display */}
              {result && (
                <motion.div
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  className="space-y-4"
                >
                  {/* Execution Metadata */}
                  <div className="flex flex-wrap gap-4 p-4 rounded-lg bg-surface-2 border border-outline">
                    <div className="flex items-center gap-2">
                      <Server className="w-4 h-4 text-muted-foreground" />
                      <span className="text-sm">
                        <span className="text-muted-foreground">Device:</span>{' '}
                        <span className="font-medium">{result.response.selected_device_name}</span>
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Clock className="w-4 h-4 text-muted-foreground" />
                      <span className="text-sm">
                        <span className="text-muted-foreground">Time:</span>{' '}
                        <span className="font-mono text-warning-amber">{result.response.total_time_ms}ms</span>
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      {result.response.exit_code === 0 ? (
                        <CheckCircle className="w-4 h-4 text-safe-green" />
                      ) : (
                        <XCircle className="w-4 h-4 text-danger-pink" />
                      )}
                      <span className="text-sm">
                        <span className="text-muted-foreground">Exit:</span>{' '}
                        <span className={`font-mono ${result.response.exit_code === 0 ? 'text-safe-green' : 'text-danger-pink'}`}>
                          {result.response.exit_code}
                        </span>
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-sm">
                        <span className="text-muted-foreground">Context:</span>{' '}
                        <span className="font-medium">
                          {result.response.executed_locally ? 'Local' : 'Remote'}
                        </span>
                      </span>
                    </div>
                  </div>

                  {/* Terminal Output */}
                  <TerminalPanel
                    output={formatOutput()}
                    exitCode={result.response.exit_code}
                  />
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
      </Tabs>
    </div>
  );
};
