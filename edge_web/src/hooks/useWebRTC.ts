import { useState, useRef, useCallback } from 'react';
import {
  startStream,
  sendStreamAnswer,
  stopStream,
  type RoutingPolicy,
  type StreamStartResponse,
} from '@/api';

type WebRTCState = 'idle' | 'connecting' | 'connected' | 'disconnected' | 'error';

interface UseWebRTCOptions {
  onFrame?: (frameUrl: string) => void;
  onError?: (error: string) => void;
}

interface StreamOptions {
  forceDeviceId?: string;
  fps?: number;
  quality?: number;
  monitorIndex?: number;
}

interface UseWebRTCResult {
  state: WebRTCState;
  error: string | null;
  streamInfo: StreamStartResponse | null;
  iceConnectionState: RTCIceConnectionState | null;
  start: (policy: RoutingPolicy, options?: StreamOptions) => Promise<void>;
  stop: () => Promise<void>;
}

export function useWebRTC(options: UseWebRTCOptions = {}): UseWebRTCResult {
  const { onFrame, onError } = options;

  const [state, setState] = useState<WebRTCState>('idle');
  const [error, setError] = useState<string | null>(null);
  const [streamInfo, setStreamInfo] = useState<StreamStartResponse | null>(null);
  const [iceConnectionState, setIceConnectionState] = useState<RTCIceConnectionState | null>(null);

  const pcRef = useRef<RTCPeerConnection | null>(null);
  const dcRef = useRef<RTCDataChannel | null>(null);
  const frameUrlRef = useRef<string | null>(null);

  const cleanup = useCallback(() => {
    // Revoke old frame URL to prevent memory leaks
    if (frameUrlRef.current) {
      URL.revokeObjectURL(frameUrlRef.current);
      frameUrlRef.current = null;
    }

    // Close data channel
    if (dcRef.current) {
      dcRef.current.close();
      dcRef.current = null;
    }

    // Close peer connection
    if (pcRef.current) {
      pcRef.current.close();
      pcRef.current = null;
    }

    setIceConnectionState(null);
  }, []);

  const start = useCallback(
    async (policy: RoutingPolicy, streamOptions: StreamOptions = {}) => {
      try {
        setState('connecting');
        setError(null);

        // Step 1: Get offer SDP from server
        const startResponse = await startStream(policy, {
          forceDeviceId: streamOptions.forceDeviceId,
          fps: streamOptions.fps,
          quality: streamOptions.quality,
          monitorIndex: streamOptions.monitorIndex,
        });
        setStreamInfo(startResponse);

        // Step 2: Create RTCPeerConnection (no ICE servers for local/direct)
        const pc = new RTCPeerConnection();
        pcRef.current = pc;

        // Monitor ICE connection state
        pc.oniceconnectionstatechange = () => {
          setIceConnectionState(pc.iceConnectionState);
          if (pc.iceConnectionState === 'failed') {
            const errorMsg = 'ICE connection failed - UDP may be blocked';
            setError(errorMsg);
            onError?.(errorMsg);
          } else if (pc.iceConnectionState === 'disconnected') {
            setState('disconnected');
          }
        };

        // Handle incoming data channel for screen frames
        pc.ondatachannel = (event) => {
          const dc = event.channel;
          dcRef.current = dc;

          dc.binaryType = 'arraybuffer';

          dc.onopen = () => {
            setState('connected');
          };

          dc.onmessage = (msgEvent) => {
            // Binary JPEG frame
            const blob = new Blob([msgEvent.data], { type: 'image/jpeg' });

            // Revoke old URL to prevent memory leaks
            if (frameUrlRef.current) {
              URL.revokeObjectURL(frameUrlRef.current);
            }

            const url = URL.createObjectURL(blob);
            frameUrlRef.current = url;
            onFrame?.(url);
          };

          dc.onclose = () => {
            setState('disconnected');
          };
        };

        // Step 3: Set remote description (server's offer)
        await pc.setRemoteDescription({
          type: 'offer',
          sdp: startResponse.offer_sdp,
        });

        // Step 4: Create and set local answer
        const answer = await pc.createAnswer();
        await pc.setLocalDescription(answer);

        // Step 5: Wait for ICE gathering to complete (with timeout)
        await new Promise<void>((resolve, reject) => {
          const timeout = setTimeout(() => {
            resolve(); // Proceed even if not all candidates gathered
          }, 5000);

          pc.onicegatheringstatechange = () => {
            if (pc.iceGatheringState === 'complete') {
              clearTimeout(timeout);
              resolve();
            }
          };

          // If already complete, resolve immediately
          if (pc.iceGatheringState === 'complete') {
            clearTimeout(timeout);
            resolve();
          }
        });

        // Step 6: Send answer SDP to server
        const localDesc = pc.localDescription;
        if (!localDesc) {
          throw new Error('No local description');
        }

        await sendStreamAnswer(startResponse.stream_id, localDesc.sdp);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to start stream';
        setError(message);
        setState('error');
        onError?.(message);
        cleanup();
      }
    },
    [onFrame, onError, cleanup]
  );

  const stop = useCallback(async () => {
    try {
      if (streamInfo) {
        await stopStream(streamInfo.stream_id);
      }
    } catch {
      // Ignore errors during stop
    } finally {
      cleanup();
      setStreamInfo(null);
      setState('idle');
      setError(null);
    }
  }, [streamInfo, cleanup]);

  return {
    state,
    error,
    streamInfo,
    iceConnectionState,
    start,
    stop,
  };
}
