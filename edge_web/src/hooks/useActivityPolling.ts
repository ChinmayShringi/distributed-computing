import { useState, useEffect, useCallback, useRef } from 'react';
import { getActivity, type ActivityResponse } from '@/api';

interface UseActivityPollingOptions {
  enabled?: boolean;
  interval?: number;
  includeMetricsHistory?: boolean;
}

interface UseActivityPollingResult {
  data: ActivityResponse | null;
  error: string | null;
  loading: boolean;
  isPolling: boolean;
  togglePolling: () => void;
  startPolling: () => void;
  stopPolling: () => void;
  refresh: () => Promise<void>;
}

export function useActivityPolling(
  options: UseActivityPollingOptions = {}
): UseActivityPollingResult {
  const { enabled = false, interval = 2000, includeMetricsHistory = false } = options;

  const [data, setData] = useState<ActivityResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [isPolling, setIsPolling] = useState(enabled);

  const intervalRef = useRef<number | null>(null);

  const fetchActivity = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await getActivity(includeMetricsHistory);
      setData(response);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch activity';
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [includeMetricsHistory]);

  const startPolling = useCallback(() => {
    setIsPolling(true);
  }, []);

  const stopPolling = useCallback(() => {
    setIsPolling(false);
  }, []);

  const togglePolling = useCallback(() => {
    setIsPolling((prev) => !prev);
  }, []);

  const refresh = useCallback(async () => {
    await fetchActivity();
  }, [fetchActivity]);

  // Set up polling interval
  useEffect(() => {
    if (isPolling) {
      // Fetch immediately when polling starts
      fetchActivity();

      // Set up interval
      intervalRef.current = window.setInterval(fetchActivity, interval);

      return () => {
        if (intervalRef.current !== null) {
          window.clearInterval(intervalRef.current);
          intervalRef.current = null;
        }
      };
    } else {
      // Clear interval when polling stops
      if (intervalRef.current !== null) {
        window.clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    }
  }, [isPolling, interval, fetchActivity]);

  return {
    data,
    error,
    loading,
    isPolling,
    togglePolling,
    startPolling,
    stopPolling,
    refresh,
  };
}
