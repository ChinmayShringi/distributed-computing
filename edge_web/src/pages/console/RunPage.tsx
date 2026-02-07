import { useState, useEffect, useCallback } from 'react';
import { motion } from 'framer-motion';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { TerminalPanel } from '@/components/TerminalPanel';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
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
  Loader2,
  AlertCircle,
  RefreshCw,
  Clock,
  CheckCircle,
  XCircle,
} from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

const routingPolicies: { value: RoutingPolicy; label: string; description: string }[] = [
  { value: 'BEST_AVAILABLE', label: 'Best Available', description: 'NPU > GPU > CPU preference' },
  { value: 'PREFER_REMOTE', label: 'Prefer Remote', description: 'Prefer non-local devices' },
  { value: 'REQUIRE_NPU', label: 'Require NPU', description: 'Fail if no NPU device' },
  { value: 'PREFER_LOCAL_MODEL', label: 'Prefer Local Model', description: 'Prefer device with LLM' },
  { value: 'REQUIRE_LOCAL_MODEL', label: 'Require Local Model', description: 'Fail if no LLM device' },
  { value: 'FORCE_DEVICE_ID', label: 'Force Device', description: 'Target specific device' },
];

export const RunPage = () => {
  const { toast } = useToast();

  // Device state
  const [devices, setDevices] = useState<Device[]>([]);
  const [loadingDevices, setLoadingDevices] = useState(true);

  // Form state
  const [command, setCommand] = useState('');
  const [args, setArgs] = useState('');
  const [policy, setPolicy] = useState<RoutingPolicy>('BEST_AVAILABLE');
  const [forceDeviceId, setForceDeviceId] = useState('');

  // Execution state
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<RoutedCommandResponse | null>(null);
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

  // Handle command execution
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
      const argsArray = args.trim() ? args.split(',').map((a) => a.trim()) : [];
      const response = await executeRoutedCommand(
        command.trim(),
        argsArray,
        policy,
        policy === 'FORCE_DEVICE_ID' ? forceDeviceId : undefined
      );

      setResult(response);

      toast({
        title: response.exit_code === 0 ? 'Command Succeeded' : 'Command Failed',
        description: `Executed on ${response.device_name} in ${response.elapsed_ms}ms`,
        variant: response.exit_code === 0 ? 'default' : 'destructive',
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Command execution failed';
      setError(message);
      toast({
        title: 'Error',
        description: message,
        variant: 'destructive',
      });
    } finally {
      setRunning(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !running) {
      handleRun();
    }
  };

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Terminal className="w-6 h-6 text-primary" />
            Run Command
          </h1>
          <p className="text-muted-foreground mt-1">
            Execute commands on remote devices with intelligent routing
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

      {/* Command Form */}
      <GlassContainer>
        <div className="space-y-4">
          {/* Command Input */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="command">Command</Label>
              <Input
                id="command"
                value={command}
                onChange={(e) => setCommand(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="e.g., hostname, whoami, uname"
                className="font-mono bg-surface-2 border-outline"
                disabled={running}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="args">Arguments (comma-separated)</Label>
              <Input
                id="args"
                value={args}
                onChange={(e) => setArgs(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="e.g., -a, -v, --help"
                className="font-mono bg-surface-2 border-outline"
                disabled={running}
              />
            </div>
          </div>

          {/* Routing Policy */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Routing Policy</Label>
              <Select
                value={policy}
                onValueChange={(v) => setPolicy(v as RoutingPolicy)}
                disabled={running}
              >
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
                <Select
                  value={forceDeviceId}
                  onValueChange={setForceDeviceId}
                  disabled={running}
                >
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
            className="bg-safe-green hover:bg-safe-green/90 text-background"
          >
            {running ? (
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            ) : (
              <Play className="w-4 h-4 mr-2" />
            )}
            Execute Command
          </Button>
        </div>
      </GlassContainer>

      {/* Error Display */}
      {error && (
        <div className="flex items-center gap-3 p-4 rounded-lg bg-danger-pink/10 border border-danger-pink/20">
          <AlertCircle className="w-5 h-5 text-danger-pink" />
          <span className="text-danger-pink">{error}</span>
        </div>
      )}

      {/* Result Display */}
      {result && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
        >
          <GlassCard className="p-5 space-y-4">
            {/* Result Header */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                {result.exit_code === 0 ? (
                  <CheckCircle className="w-6 h-6 text-safe-green" />
                ) : (
                  <XCircle className="w-6 h-6 text-danger-pink" />
                )}
                <div>
                  <h3 className="font-semibold">Command Result</h3>
                  <p className="text-sm text-muted-foreground">
                    Executed on {result.device_name}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <Badge
                  variant={result.exit_code === 0 ? 'default' : 'destructive'}
                  className={result.exit_code === 0 ? 'bg-safe-green' : ''}
                >
                  Exit: {result.exit_code}
                </Badge>
                <Badge variant="outline" className="gap-1">
                  <Clock className="w-3 h-3" />
                  {result.elapsed_ms}ms
                </Badge>
              </div>
            </div>

            {/* Output */}
            {result.stdout && (
              <TerminalPanel
                output={result.stdout}
                exitCode={result.exit_code}
              />
            )}

            {/* Stderr */}
            {result.stderr && (
              <div className="space-y-2">
                <Label className="text-danger-pink">Standard Error</Label>
                <pre className="p-3 rounded-lg bg-danger-pink/10 border border-danger-pink/20 text-sm font-mono text-danger-pink overflow-auto max-h-[200px]">
                  {result.stderr}
                </pre>
              </div>
            )}
          </GlassCard>
        </motion.div>
      )}
    </div>
  );
};
