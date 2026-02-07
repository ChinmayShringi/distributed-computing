import { useEffect, useState } from 'react';
import { getConnectionState } from '@/lib/connection';
import { ModePill } from './ModePill';
import { Server, Wifi, WifiOff } from 'lucide-react';

export const StatusStrip = () => {
  const [state, setState] = useState({ connected: false, serverAddress: '', advancedMode: false });

  useEffect(() => {
    setState(getConnectionState());
    
    const handleStorageChange = () => {
      setState(getConnectionState());
    };
    
    window.addEventListener('storage', handleStorageChange);
    return () => window.removeEventListener('storage', handleStorageChange);
  }, []);

  return (
    <div className="flex items-center gap-4 px-4 py-2 bg-surface-1 border-b border-outline">
      <div className="flex items-center gap-2">
        {state.connected ? (
          <>
            <div className="status-dot status-dot-online" />
            <Wifi className="w-4 h-4 text-safe-green" />
            <span className="text-sm text-safe-green font-medium">Connected</span>
          </>
        ) : (
          <>
            <div className="status-dot status-dot-offline" />
            <WifiOff className="w-4 h-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">Offline</span>
          </>
        )}
      </div>
      
      {state.connected && state.serverAddress && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Server className="w-4 h-4" />
          <span className="font-mono">{state.serverAddress}</span>
        </div>
      )}
      
      <div className="ml-auto">
        <ModePill mode={state.advancedMode ? 'advanced' : 'safe'} size="sm" />
      </div>
    </div>
  );
};
