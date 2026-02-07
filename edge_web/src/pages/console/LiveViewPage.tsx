import { useState } from 'react';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { Button } from '@/components/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { mockDevices } from '@/lib/mock-data';
import { Eye, Play, Square, Monitor } from 'lucide-react';

export const LiveViewPage = () => {
  const [selectedDevice, setSelectedDevice] = useState<string>('');
  const [isStreaming, setIsStreaming] = useState(false);
  const onlineDevices = mockDevices.filter(d => d.status === 'online');

  const handleToggleStream = () => {
    setIsStreaming(!isStreaming);
  };

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Live View</h1>
        <p className="text-muted-foreground mt-1">
          Stream real-time view from connected devices.
        </p>
      </div>

      {/* Viewport */}
      <GlassContainer className="aspect-video flex items-center justify-center">
        {isStreaming && selectedDevice ? (
          <div className="w-full h-full bg-surface-2 rounded-lg flex items-center justify-center">
            <div className="text-center space-y-4">
              <div className="w-16 h-16 mx-auto rounded-full bg-primary/20 flex items-center justify-center animate-pulse">
                <Monitor className="w-8 h-8 text-primary" />
              </div>
              <div>
                <p className="font-semibold">
                  Streaming from {onlineDevices.find(d => d.id === selectedDevice)?.name}
                </p>
                <p className="text-sm text-muted-foreground">
                  Live view simulation active
                </p>
              </div>
            </div>
          </div>
        ) : (
          <div className="text-center space-y-4">
            <div className="w-20 h-20 mx-auto rounded-full bg-surface-variant flex items-center justify-center">
              <Eye className="w-10 h-10 text-muted-foreground" />
            </div>
            <div>
              <p className="text-lg font-semibold">No Active Stream</p>
              <p className="text-sm text-muted-foreground">
                Select a device and start streaming
              </p>
            </div>
          </div>
        )}
      </GlassContainer>

      {/* Controls */}
      <GlassCard className="p-4">
        <div className="flex flex-col sm:flex-row items-center gap-4">
          <div className="flex-1 w-full sm:w-auto">
            <Select value={selectedDevice} onValueChange={setSelectedDevice}>
              <SelectTrigger className="bg-surface-2 border-outline">
                <SelectValue placeholder="Select a device" />
              </SelectTrigger>
              <SelectContent className="bg-surface-2 border-outline">
                {onlineDevices.map((device) => (
                  <SelectItem key={device.id} value={device.id}>
                    <div className="flex items-center gap-2">
                      <div className="w-2 h-2 rounded-full bg-safe-green" />
                      {device.name}
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <Button
            onClick={handleToggleStream}
            disabled={!selectedDevice}
            className={
              isStreaming
                ? 'bg-danger-pink hover:bg-danger-pink/90'
                : 'bg-primary hover:bg-primary/90'
            }
          >
            {isStreaming ? (
              <>
                <Square className="w-4 h-4 mr-2" />
                Stop Stream
              </>
            ) : (
              <>
                <Play className="w-4 h-4 mr-2" />
                Start Live View
              </>
            )}
          </Button>
        </div>
      </GlassCard>
    </div>
  );
};
