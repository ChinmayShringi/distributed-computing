// Connection state management using localStorage

export interface ConnectionState {
  connected: boolean;
  serverAddress: string;
  advancedMode: boolean;
}

const STORAGE_KEY = 'edgemesh_connection';

export const getConnectionState = (): ConnectionState => {
  if (typeof window === 'undefined') {
    return { connected: false, serverAddress: '', advancedMode: false };
  }
  
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored) {
    try {
      return JSON.parse(stored);
    } catch {
      return { connected: false, serverAddress: '', advancedMode: false };
    }
  }
  return { connected: false, serverAddress: '', advancedMode: false };
};

export const setConnectionState = (state: Partial<ConnectionState>) => {
  const current = getConnectionState();
  const newState = { ...current, ...state };
  localStorage.setItem(STORAGE_KEY, JSON.stringify(newState));
  return newState;
};

export const connect = (serverAddress: string): ConnectionState => {
  return setConnectionState({ connected: true, serverAddress });
};

export const disconnect = (): ConnectionState => {
  localStorage.removeItem(STORAGE_KEY);
  return { connected: false, serverAddress: '', advancedMode: false };
};

export const toggleAdvancedMode = (enabled: boolean): ConnectionState => {
  return setConnectionState({ advancedMode: enabled });
};
