import { useState } from 'react';
import { motion } from 'framer-motion';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Progress } from '@/components/ui/progress';
import { mockDevices } from '@/lib/mock-data';
import { Download, Ticket, Clock, HardDrive, Check, Loader2 } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

interface Transfer {
  id: string;
  fileName: string;
  device: string;
  progress: number;
  size: string;
  status: 'downloading' | 'completed' | 'queued';
}

const mockTransfers: Transfer[] = [
  { id: '1', fileName: 'model_weights.pt', device: 'Workstation Alpha', progress: 78, size: '2.4 GB', status: 'downloading' },
  { id: '2', fileName: 'dataset_v2.tar.gz', device: 'Edge Node Beta', progress: 100, size: '845 MB', status: 'completed' },
  { id: '3', fileName: 'logs_2024.zip', device: 'MacBook Pro M3', progress: 0, size: '156 MB', status: 'queued' },
];

export const DownloadsPage = () => {
  const { toast } = useToast();
  const [selectedDevice, setSelectedDevice] = useState<string>('');
  const [filePath, setFilePath] = useState('/home/user/data/');
  const [ttl, setTtl] = useState([24]);
  const [generating, setGenerating] = useState(false);
  const [transfers] = useState<Transfer[]>(mockTransfers);

  const onlineDevices = mockDevices.filter(d => d.status === 'online');

  const handleGenerateTicket = async () => {
    setGenerating(true);
    await new Promise(resolve => setTimeout(resolve, 1500));
    setGenerating(false);
    toast({
      title: 'Ticket Generated',
      description: 'Download ticket created successfully.',
    });
  };

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Downloads</h1>
        <p className="text-muted-foreground mt-1">
          Generate download tickets and manage file transfers.
        </p>
      </div>

      {/* Generate Ticket */}
      <GlassContainer>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Ticket className="w-5 h-5 text-primary" />
          Generate Download Ticket
        </h2>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Source Device</Label>
              <Select value={selectedDevice} onValueChange={setSelectedDevice}>
                <SelectTrigger className="bg-surface-2 border-outline">
                  <SelectValue placeholder="Select device" />
                </SelectTrigger>
                <SelectContent className="bg-surface-2 border-outline">
                  {onlineDevices.map((device) => (
                    <SelectItem key={device.id} value={device.id}>
                      {device.name}
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
              />
            </div>
          </div>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label className="flex items-center justify-between">
                <span>Time to Live (TTL)</span>
                <span className="text-sm text-muted-foreground">{ttl[0]} hours</span>
              </Label>
              <Slider
                value={ttl}
                onValueChange={setTtl}
                min={1}
                max={72}
                step={1}
                className="py-4"
              />
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>1h</span>
                <span>72h</span>
              </div>
            </div>

            <Button
              onClick={handleGenerateTicket}
              disabled={!selectedDevice || !filePath || generating}
              className="w-full bg-primary hover:bg-primary/90"
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
      </GlassContainer>

      {/* Transfers */}
      <GlassContainer>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Download className="w-5 h-5 text-primary" />
          Transfers
        </h2>

        <div className="space-y-3">
          {transfers.map((transfer, index) => (
            <motion.div
              key={transfer.id}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.1 }}
            >
              <GlassCard className="p-4">
                <div className="flex items-center gap-4">
                  <div className="p-2 rounded-lg bg-surface-variant">
                    {transfer.status === 'completed' ? (
                      <Check className="w-5 h-5 text-safe-green" />
                    ) : transfer.status === 'downloading' ? (
                      <Loader2 className="w-5 h-5 text-primary animate-spin" />
                    ) : (
                      <Clock className="w-5 h-5 text-muted-foreground" />
                    )}
                  </div>

                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between mb-1">
                      <p className="font-mono text-sm truncate">{transfer.fileName}</p>
                      <span className="text-xs text-muted-foreground ml-2">
                        {transfer.size}
                      </span>
                    </div>
                    <div className="flex items-center gap-2 text-xs text-muted-foreground mb-2">
                      <HardDrive className="w-3 h-3" />
                      {transfer.device}
                    </div>
                    <Progress 
                      value={transfer.progress} 
                      className="h-1.5"
                    />
                  </div>

                  <span className="text-sm font-medium">
                    {transfer.progress}%
                  </span>
                </div>
              </GlassCard>
            </motion.div>
          ))}
        </div>
      </GlassContainer>
    </div>
  );
};
