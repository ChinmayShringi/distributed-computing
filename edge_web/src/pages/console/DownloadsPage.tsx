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
  triggerDownload,
  type Device,
  type DownloadTicketResponse,
} from '@/api';
import {
  Download,
  Ticket,
  Clock,
  RefreshCw,
  Loader2,
  AlertCircle,
  CheckCircle,
  FileDown,
  HardDrive,
} from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

interface DownloadHistory {
  id: string;
  deviceName: string;
  filePath: string;
  ticket: DownloadTicketResponse;
  createdAt: Date;
}

export const DownloadsPage = () => {
  const { toast } = useToast();

  // Device state
  const [devices, setDevices] = useState<Device[]>([]);
  const [loadingDevices, setLoadingDevices] = useState(true);

  // Form state
  const [selectedDeviceId, setSelectedDeviceId] = useState('');
  const [filePath, setFilePath] = useState('/home/user/data/');

  // Download state
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [downloads, setDownloads] = useState<DownloadHistory[]>([]);

  // Fetch devices
  const fetchDevices = useCallback(async () => {
    try {
      setLoadingDevices(true);
      const data = await listDevices();
      setDevices(data);
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

  // Handle ticket generation
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
      const ticket = await requestDownload(selectedDeviceId, filePath.trim());
      const device = devices.find((d) => d.device_id === selectedDeviceId);

      const downloadEntry: DownloadHistory = {
        id: ticket.ticket_id,
        deviceName: device?.device_name || 'Unknown',
        filePath: filePath.trim(),
        ticket,
        createdAt: new Date(),
      };

      setDownloads((prev) => [downloadEntry, ...prev]);

      toast({
        title: 'Ticket Generated',
        description: `Download ticket created successfully. Size: ${formatFileSize(ticket.file_size)}`,
      });

      // Trigger download automatically
      triggerDownload(ticket.download_url, filePath.split('/').pop());
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

  // Format file size
  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
  };

  // Format time remaining
  const getTimeRemaining = (expiresAt: string): string => {
    const expires = new Date(expiresAt);
    const now = new Date();
    const diff = expires.getTime() - now.getTime();

    if (diff <= 0) return 'Expired';

    const minutes = Math.floor(diff / 60000);
    const seconds = Math.floor((diff % 60000) / 1000);

    if (minutes > 0) return `${minutes}m ${seconds}s`;
    return `${seconds}s`;
  };

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Download className="w-6 h-6 text-primary" />
            Downloads
          </h1>
          <p className="text-muted-foreground mt-1">
            Generate download tickets and transfer files from devices
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
          Generate Download Ticket
        </h2>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Source Device</Label>
              <Select
                value={selectedDeviceId}
                onValueChange={setSelectedDeviceId}
                disabled={generating}
              >
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
              <Label>File Path</Label>
              <Input
                value={filePath}
                onChange={(e) => setFilePath(e.target.value)}
                placeholder="/path/to/file"
                className="font-mono bg-surface-2 border-outline"
                disabled={generating}
              />
            </div>
          </div>

          <div className="flex flex-col justify-end">
            <Button
              onClick={handleGenerateTicket}
              disabled={!selectedDeviceId || !filePath.trim() || generating}
              className="bg-primary hover:bg-primary/90"
            >
              {generating ? (
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              ) : (
                <Ticket className="w-4 h-4 mr-2" />
              )}
              Generate Ticket
            </Button>
          </div>
        </div>

        {/* Error Display */}
        {error && (
          <div className="flex items-center gap-2 p-3 mt-4 rounded-lg bg-danger-pink/10 border border-danger-pink/20 text-danger-pink text-sm">
            <AlertCircle className="w-4 h-4" />
            {error}
          </div>
        )}
      </GlassContainer>

      {/* Download History */}
      <GlassContainer>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <FileDown className="w-5 h-5 text-primary" />
          Recent Downloads
          {downloads.length > 0 && (
            <Badge variant="outline">{downloads.length}</Badge>
          )}
        </h2>

        {downloads.length > 0 ? (
          <div className="space-y-3">
            {downloads.map((download, index) => (
              <motion.div
                key={download.id}
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: index * 0.05 }}
              >
                <GlassCard className="p-4">
                  <div className="flex items-center gap-4">
                    <div className="p-2 rounded-lg bg-safe-green/20">
                      <CheckCircle className="w-5 h-5 text-safe-green" />
                    </div>

                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between mb-1">
                        <p className="font-mono text-sm truncate">{download.filePath}</p>
                        <span className="text-xs text-muted-foreground ml-2">
                          {formatFileSize(download.ticket.file_size)}
                        </span>
                      </div>
                      <div className="flex items-center gap-4 text-xs text-muted-foreground">
                        <span className="flex items-center gap-1">
                          <HardDrive className="w-3 h-3" />
                          {download.deviceName}
                        </span>
                        <span className="flex items-center gap-1">
                          <Clock className="w-3 h-3" />
                          Expires: {getTimeRemaining(download.ticket.expires_at)}
                        </span>
                      </div>
                    </div>

                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => triggerDownload(download.ticket.download_url, download.filePath.split('/').pop())}
                    >
                      <Download className="w-4 h-4 mr-1" />
                      Download
                    </Button>
                  </div>
                </GlassCard>
              </motion.div>
            ))}
          </div>
        ) : (
          <p className="text-muted-foreground text-center py-6">
            No downloads yet. Generate a ticket to start.
          </p>
        )}
      </GlassContainer>
    </div>
  );
};
