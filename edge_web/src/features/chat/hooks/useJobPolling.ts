import { useCallback, useRef } from 'react';
import { getJob } from '../api';
import type { JobResult } from '../types';

interface UseJobPollingOptions {
  interval?: number;
  maxAttempts?: number;
  onResult?: (result: JobResult) => void;
  onError?: (error: Error) => void;
}

export function useJobPolling(options: UseJobPollingOptions = {}) {
  const {
    interval = 1000,
    maxAttempts = 30,
    onResult,
    onError,
  } = options;

  const pollingRef = useRef<NodeJS.Timeout | null>(null);
  const attemptCountRef = useRef(0);

  const stopPolling = useCallback(() => {
    if (pollingRef.current) {
      clearInterval(pollingRef.current);
      pollingRef.current = null;
    }
    attemptCountRef.current = 0;
  }, []);

  const startPolling = useCallback(
    (jobId: string): Promise<JobResult> => {
      return new Promise((resolve, reject) => {
        attemptCountRef.current = 0;

        const poll = async () => {
          try {
            attemptCountRef.current++;
            const result = await getJob(jobId);

            if (result.state === 'DONE' || result.state === 'FAILED') {
              stopPolling();
              onResult?.(result);
              resolve(result);
              return;
            }

            if (attemptCountRef.current >= maxAttempts) {
              stopPolling();
              const error = new Error('Job polling timed out');
              onError?.(error);
              reject(error);
              return;
            }
          } catch (err) {
            stopPolling();
            const error = err instanceof Error ? err : new Error('Polling failed');
            onError?.(error);
            reject(error);
          }
        };

        // Initial poll
        poll();

        // Set up interval
        pollingRef.current = setInterval(poll, interval);
      });
    },
    [interval, maxAttempts, onResult, onError, stopPolling]
  );

  return {
    startPolling,
    stopPolling,
  };
}
