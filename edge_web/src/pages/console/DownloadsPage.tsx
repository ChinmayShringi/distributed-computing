import { useState, useEffect, useCallback } from 'react';
import { motion } from 'framer-motion';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import {
  listDevices,
  requestDownload,
  type Device,
  type DownloadResponse,
} from '@/api';
import {
  Download,
  Ticket,
  Clock,
  HardDrive,
  Loader2,
  RefreshCw,
  AlertCircle,
  FileDown,
  Timer,
  ExternalLink,
} from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

interface DownloadTicket {
  id: string;
  response: DownloadResponse;
  deviceId: string;
  deviceName: string;
  path: string;
  createdAt: number;
}

export const DownloadsPage = () => {
  const { toast } = useToast();

  // Devices state
  const [devices, setDevices] = useState<Device[]>([]);
  const [loadingDevices, setLoadingDevices] = useState(true);

  // Form state
  const [selectedDeviceId, setSelectedDeviceId] = useState<string>('');
  const [filePath, setFilePath] = useState('test.txt');
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Tickets state
  const [tickets, setTickets] = useState<DownloadTicket[]>([]);

  // Countdown state - update every second
  const [, setTick] = useState(0);

  // Fetch devices
  const fetchDevices = useCallback(async () => {
    try {
      setLoadingDevices(true);
      const data = await listDevices();
      setDevices(data);
      // Set default device if available
      if (data.length > 0 && !selectedDeviceId) {
        setSelectedDeviceId(data[0].device_id);
      }
    } catch (err) {
      console.error('Failed to fetch devices:', err);
    } finally {
      setLoadingDevices(false);
    }
  }, [selectedDeviceId]);

  useEffect(() => {
    fetchDevices();
  }, [fetchDevices]);

  // Countdown timer
  useEffect(() => {
    const interval = setInterval(() => {
      setTick((t) => t + 1);
      // Clean up expired tickets
      setTickets((prev) =>
        prev.filter((ticket) => ticket.response.expires_unix_ms > Date.now())
      );
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  // Get remaining time for a ticket
  const getRemainingTime = (expiresMs: number): string => {
    const remaining = Math.max(0, expiresMs - Date.now());
    const seconds = Math.floor(remaining / 1000);
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
  };

  // Format file size
  const formatSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
  };

  // Handle generate ticket
  const handleGenerateTicket = async () => {
    if (!selectedDeviceId || !filePath.trim()) {
      toast({
        title: 'Error',
        description: 'Please select a device and enter a file path',
        variant: 'destructive',
      });
      return;
    }

    setGenerating(true);
    setError(null);

    try {
      const response = await requestDownload(selectedDeviceId, filePath);
      const device = devices.find((d) => d.device_id === selectedDeviceId);

      const ticket: DownloadTicket = {
        id: `ticket-${Date.now()}`,
        response,
        deviceId: selectedDeviceId,
        deviceName: device?.device_name || selectedDeviceId,
        path: filePath,
        createdAt: Date.now(),
      };

      setTickets((prev) => [ticket, ...prev]);

      toast({
        title: 'Ticket Generated',
        description: `Download ticket for ${response.filename} created`,
      });

      // Auto-download
      window.open(response.download_url, '_blank');
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to generate ticket';
      setError(message);
      toast({
        title: 'Error',
        description: message,
        variant: 'destructive',
      });
    } finally {
      setGenerating(false);
    }
  };

  // Handle download click
  const handleDownload = (url: string) => {
    window.open(url, '_blank');
  };

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Downloads</h1>
          <p className="text-muted-foreground mt-1">
            Request file downloads from connected devices with one-time tokens.
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

      {/* Generate Ticket Form */}
      <GlassContainer>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Ticket className="w-5 h-5 text-primary" />
          Request Download
        </h2>

        <div className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Source Device</Label>
              <Select value={selectedDeviceId} onValueChange={setSelectedDeviceId}>
                <SelectTrigger className="bg-surface-2 border-outline">
                  <SelectValue placeholder="Select device" />
                </SelectTrigger>
                <SelectContent className="bg-surface-2 border-outline">
                  {devices.map((device) => (
                    <SelectItem key={device.device_id} value={device.device_id}>
                      <div className="flex items-center gap-2">
                        <HardDrive className="w-4 h-4" />
                        {device.device_name}
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>File Path (relative to shared/)</Label>
              <Input
                value={filePath}
                onChange={(e) => setFilePath(e.target.value)}
                placeholder="e.g., test.txt, logs/app.log"
                className="font-mono bg-surface-2 border-outline"
              />
            </div>
          </div>

          <Button
            onClick={handleGenerateTicket}
            disabled={!selectedDeviceId || !filePath.trim() || generating}
            className="bg-primary hover:bg-primary/90"
          >
            {generating ? (
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            ) : (
              <FileDown className="w-4 h-4 mr-2" />
            )}
            Request Download
          </Button>

          {error && (
            <div className="flex items-center gap-2 p-3 rounded-lg bg-danger-pink/10 border border-danger-pink/20 text-danger-pink text-sm">
              <AlertCircle className="w-4 h-4" />
              {error}
            </div>
          )}
        </div>
      </GlassContainer>

      {/* Active Tickets */}
      <GlassContainer>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Download className="w-5 h-5 text-primary" />
          Download Tickets
          {tickets.length > 0 && (
            <Badge variant="outline">{tickets.length}</Badge>
          )}
        </h2>

        {tickets.length > 0 ? (
          <div className="space-y-3">
            {tickets.map((ticket, index) => {
              const remaining = ticket.response.expires_unix_ms - Date.now();
              const isExpired = remaining <= 0;
              const isExpiringSoon = remaining > 0 && remaining < 30000; // 30 seconds

              return (
                <motion.div
                  key={ticket.id}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: index * 0.05 }}
                >
                  <GlassCard className={`p-4 ${isExpired ? 'opacity-50' : ''}`}>
                    <div className="flex items-center gap-4">
                      <div
                        className={`p-2 rounded-lg ${
                          isExpired
                            ? 'bg-surface-variant'
                            : isExpiringSoon
                            ? 'bg-warning-amber/20'
                            : 'bg-safe-green/20'
                        }`}
                      >
                        {isExpired ? (
                          <Clock className="w-5 h-5 text-muted-foreground" />
                        ) : (
                          <FileDown
                            className={`w-5 h-5 ${
                              isExpiringSoon ? 'text-warning-amber' : 'text-safe-green'
                            }`}
                          />
                        )}
                      </div>

                      <div className="flex-1 min-w-0">
                        <div className="flex items-center justify-between mb-1">
                          <p className="font-mono text-sm truncate">
                            {ticket.response.filename}
                          </p>
                          <span className="text-xs text-muted-foreground ml-2">
                            {formatSize(ticket.response.size_bytes)}
                          </span>
                        </div>
                        <div className="flex items-center gap-4 text-xs text-muted-foreground">
                          <span className="flex items-center gap-1">
                            <HardDrive className="w-3 h-3" />
                            {ticket.deviceName}
                          </span>
                          <span className="flex items-center gap-1">
                            <Timer className="w-3 h-3" />
                            {isExpired ? (
                              <span className="text-danger-pink">Expired</span>
                            ) : (
                              <span
                                className={
                                  isExpiringSoon ? 'text-warning-amber' : ''
                                }
                              >
                                {getRemainingTime(ticket.response.expires_unix_ms)}
                              </span>
                            )}
                          </span>
                        </div>
                      </div>

                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleDownload(ticket.response.download_url)}
                        disabled={isExpired}
                        className="gap-1"
                      >
                        <ExternalLink className="w-4 h-4" />
                        Download
                      </Button>
                    </div>
                  </GlassCard>
                </motion.div>
              );
            })}
          </div>
        ) : (
          <div className="text-center py-8 text-muted-foreground">
            <Ticket className="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>No active download tickets</p>
            <p className="text-sm">Request a file download to create a ticket</p>
          </div>
        )}
      </GlassContainer>

      {/* Info Card */}
      <GlassCard className="p-4">
        <div className="flex items-start gap-3">
          <AlertCircle className="w-5 h-5 text-info-blue flex-shrink-0 mt-0.5" />
          <div className="text-sm text-muted-foreground">
            <p className="font-medium text-foreground mb-1">About Download Tickets</p>
            <ul className="list-disc list-inside space-y-1">
              <li>Each ticket is a one-time use token</li>
              <li>Tickets expire after 60 seconds by default</li>
              <li>File paths are relative to the device's shared/ directory</li>
              <li>Large files may take time to transfer</li>
            </ul>
          </div>
        </div>
      </GlassCard>
    </div>
  );
};
