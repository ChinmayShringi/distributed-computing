import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { ModePill } from '@/components/ModePill';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog';
import { getConnectionState, toggleAdvancedMode, disconnect } from '@/lib/connection';
import { Shield, FolderKey, Fingerprint, Bug, LogOut, AlertTriangle } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

export const SettingsPage = () => {
  const navigate = useNavigate();
  const { toast } = useToast();
  const [state, setState] = useState(getConnectionState());
  const [modalOpen, setModalOpen] = useState(false);
  const [confirmText, setConfirmText] = useState('');

  const handleEnableAdvanced = () => {
    if (confirmText === 'ENABLE') {
      toggleAdvancedMode(true);
      setState(getConnectionState());
      window.dispatchEvent(new Event('storage'));
      setModalOpen(false);
      setConfirmText('');
      toast({ title: 'Advanced Mode Enabled', description: 'Additional features are now available.' });
    }
  };

  const handleDisableAdvanced = () => {
    toggleAdvancedMode(false);
    setState(getConnectionState());
    window.dispatchEvent(new Event('storage'));
    toast({ title: 'Safe Mode Enabled' });
  };

  const handleEndSession = () => {
    disconnect();
    navigate('/console/connect');
  };

  const configTiles = [
    { icon: FolderKey, label: 'Paths', description: 'Configure file paths' },
    { icon: Fingerprint, label: 'Identity', description: 'Manage credentials' },
    { icon: Bug, label: 'Debug', description: 'Debug settings' },
  ];

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>

      <GlassContainer>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Shield className="w-5 h-5" /> Security & Modes
        </h2>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <ModePill mode={state.advancedMode ? 'advanced' : 'safe'} />
            <span className="text-sm text-muted-foreground">
              {state.advancedMode ? 'Advanced features enabled' : 'Running in safe mode'}
            </span>
          </div>
          {state.advancedMode ? (
            <Button variant="outline" onClick={handleDisableAdvanced}>Switch to Safe Mode</Button>
          ) : (
            <Button variant="outline" onClick={() => setModalOpen(true)} className="border-danger-pink/50 text-danger-pink">
              Enable Advanced Mode
            </Button>
          )}
        </div>
      </GlassContainer>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {configTiles.map(tile => (
          <GlassCard hover key={tile.label} className="p-4 cursor-pointer">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-surface-variant"><tile.icon className="w-5 h-5 text-primary" /></div>
              <div>
                <p className="font-medium">{tile.label}</p>
                <p className="text-xs text-muted-foreground">{tile.description}</p>
              </div>
            </div>
          </GlassCard>
        ))}
      </div>

      <Button variant="destructive" onClick={handleEndSession} className="bg-danger-pink hover:bg-danger-pink/90">
        <LogOut className="w-4 h-4 mr-2" /> End Session
      </Button>

      <Dialog open={modalOpen} onOpenChange={setModalOpen}>
        <DialogContent className="bg-surface-1 border-outline">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertTriangle className="w-5 h-5 text-warning-amber" /> Enable Advanced Mode
            </DialogTitle>
            <DialogDescription>
              Advanced mode unlocks additional tools and capabilities. Some operations may have elevated risk.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 pt-4">
            <p className="text-sm">Type <span className="font-mono text-primary">ENABLE</span> to confirm:</p>
            <Input value={confirmText} onChange={e => setConfirmText(e.target.value)} className="bg-surface-2 font-mono" />
            <Button onClick={handleEnableAdvanced} disabled={confirmText !== 'ENABLE'} className="w-full bg-danger-pink hover:bg-danger-pink/90">
              Confirm
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
};
