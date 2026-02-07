import { useState, useEffect, useCallback, useRef } from 'react';
import { getActivity, type ActivityResponse } from '@/api';

interface UseActivityPollingOptions {
  interval?: number; // Polling interval in ms (default 2000)
  enabled?: boolean; // Whether polling is enabled
  includeMetricsHistory?: boolean; // Include 120s metrics history
}

interface UseActivityPollingResult {
  data: ActivityResponse | null;
  error: string | null;
  loading: boolean;
  isPolling: boolean;
  startPolling: () => void;
  stopPolling: () => void;
  togglePolling: () => void;
  refresh: () => Promise<void>;
}

export function useActivityPolling(
  options: UseActivityPollingOptions = {}
): UseActivityPollingResult {
  const { interval = 2000, enabled = false, includeMetricsHistory = true } = options;

  const [data, setData] = useState<ActivityResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [isPolling, setIsPolling] = useState(enabled);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchActivity = useCallback(async () => {
    try {
      const response = await getActivity(includeMetricsHistory);
      setData(response);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch activity');
    } finally {
      setLoading(false);
    }
  }, [includeMetricsHistory]);

  const startPolling = useCallback(() => {
    if (intervalRef.current) return;
    setIsPolling(true);
    setLoading(true);
    fetchActivity();
    intervalRef.current = setInterval(fetchActivity, interval);
  }, [fetchActivity, interval]);

  const stopPolling = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
    setIsPolling(false);
  }, []);

  const togglePolling = useCallback(() => {
    if (isPolling) {
      stopPolling();
    } else {
      startPolling();
    }
  }, [isPolling, startPolling, stopPolling]);

  const refresh = useCallback(async () => {
    setLoading(true);
    await fetchActivity();
  }, [fetchActivity]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, []);

  // Start polling if enabled on mount
  useEffect(() => {
    if (enabled && !isPolling) {
      startPolling();
    }
  }, [enabled, isPolling, startPolling]);

  return {
    data,
    error,
    loading,
    isPolling,
    startPolling,
    stopPolling,
    togglePolling,
    refresh,
  };
}
