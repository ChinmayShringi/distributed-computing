import { useState, useCallback, useRef, useEffect } from 'react';
import {
  startStream,
  sendStreamAnswer,
  stopStream,
  type RoutingPolicy,
  type StreamStartResponse,
} from '@/api';

export type StreamState = 'idle' | 'connecting' | 'connected' | 'failed' | 'stopped';

interface UseWebRTCOptions {
  onFrame?: (imageUrl: string) => void;
  onError?: (error: string) => void;
  onStateChange?: (state: StreamState) => void;
}

interface UseWebRTCResult {
  state: StreamState;
  error: string | null;
  streamInfo: StreamStartResponse | null;
  iceConnectionState: RTCIceConnectionState | null;
  start: (
    policy: RoutingPolicy,
    options?: {
      forceDeviceId?: string;
      fps?: number;
      quality?: number;
      monitorIndex?: number;
    }
  ) => Promise<void>;
  stop: () => Promise<void>;
}

export function useWebRTC(options: UseWebRTCOptions = {}): UseWebRTCResult {
  const { onFrame, onError, onStateChange } = options;

  const [state, setState] = useState<StreamState>('idle');
  const [error, setError] = useState<string | null>(null);
  const [streamInfo, setStreamInfo] = useState<StreamStartResponse | null>(null);
  const [iceConnectionState, setIceConnectionState] = useState<RTCIceConnectionState | null>(null);

  const pcRef = useRef<RTCPeerConnection | null>(null);
  const frameUrlRef = useRef<string | null>(null);

  // Update state with callback
  const updateState = useCallback(
    (newState: StreamState) => {
      setState(newState);
      onStateChange?.(newState);
    },
    [onStateChange]
  );

  // Cleanup function
  const cleanup = useCallback(() => {
    // Revoke object URL to prevent memory leak
    if (frameUrlRef.current) {
      URL.revokeObjectURL(frameUrlRef.current);
      frameUrlRef.current = null;
    }

    // Close peer connection
    if (pcRef.current) {
      pcRef.current.close();
      pcRef.current = null;
    }

    setIceConnectionState(null);
  }, []);

  // Stop stream
  const stop = useCallback(async () => {
    if (streamInfo) {
      try {
        await stopStream(streamInfo.selected_device_addr, streamInfo.stream_id);
      } catch (err) {
        console.error('Error stopping stream:', err);
      }
    }

    cleanup();
    setStreamInfo(null);
    updateState('stopped');
  }, [streamInfo, cleanup, updateState]);

  // Start stream
  const start = useCallback(
    async (
      policy: RoutingPolicy,
      startOptions?: {
        forceDeviceId?: string;
        fps?: number;
        quality?: number;
        monitorIndex?: number;
      }
    ) => {
      try {
        updateState('connecting');
        setError(null);

        // 1. Request stream start and get offer SDP
        const response = await startStream(policy, startOptions);
        setStreamInfo(response);

        // 2. Create peer connection
        const pc = new RTCPeerConnection({
          iceServers: [], // No TURN servers, direct connection
        });
        pcRef.current = pc;

        // 3. Set up event handlers
        pc.oniceconnectionstatechange = () => {
          setIceConnectionState(pc.iceConnectionState);

          if (pc.iceConnectionState === 'failed') {
            setError('ICE connection failed. UDP might be blocked.');
            updateState('failed');
          } else if (pc.iceConnectionState === 'connected') {
            updateState('connected');
          } else if (pc.iceConnectionState === 'disconnected') {
            updateState('failed');
          }
        };

        pc.ondatachannel = (event) => {
          const channel = event.channel;
          channel.binaryType = 'arraybuffer';

          channel.onmessage = (msgEvent) => {
            // Receive JPEG frame data
            const data = msgEvent.data as ArrayBuffer;
            const blob = new Blob([data], { type: 'image/jpeg' });

            // Revoke previous URL to prevent memory leak
            if (frameUrlRef.current) {
              URL.revokeObjectURL(frameUrlRef.current);
            }

            const url = URL.createObjectURL(blob);
            frameUrlRef.current = url;
            onFrame?.(url);
          };

          channel.onerror = (err) => {
            console.error('DataChannel error:', err);
            setError('Data channel error');
          };
        };

        // 4. Set remote description (offer from server)
        await pc.setRemoteDescription({
          type: 'offer',
          sdp: response.offer_sdp,
        });

        // 5. Create answer
        const answer = await pc.createAnswer();
        await pc.setLocalDescription(answer);

        // 6. Wait for ICE gathering to complete (timeout 5s)
        await new Promise<void>((resolve) => {
          if (pc.iceGatheringState === 'complete') {
            resolve();
            return;
          }

          const timeout = setTimeout(() => {
            resolve();
          }, 5000);

          pc.onicegatheringstatechange = () => {
            if (pc.iceGatheringState === 'complete') {
              clearTimeout(timeout);
              resolve();
            }
          };
        });

        // 7. Send answer SDP to server
        const answerSdp = pc.localDescription?.sdp;
        if (!answerSdp) {
          throw new Error('Failed to create answer SDP');
        }

        await sendStreamAnswer(
          response.selected_device_addr,
          response.stream_id,
          answerSdp
        );
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to start stream';
        setError(message);
        onError?.(message);
        updateState('failed');
        cleanup();
      }
    },
    [cleanup, onError, onFrame, updateState]
  );

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      cleanup();
    };
  }, [cleanup]);

  return {
    state,
    error,
    streamInfo,
    iceConnectionState,
    start,
    stop,
  };
}
