// WebRTC Streaming API endpoints
import { post } from './client';
import type {
  StreamStartRequest,
  StreamStartResponse,
  StreamAnswerRequest,
  StreamStopRequest,
  RoutingPolicy,
} from './types';

// Start a WebRTC stream
export async function startStream(
  policy: RoutingPolicy,
  options?: {
    forceDeviceId?: string;
    fps?: number;
    quality?: number;
    monitorIndex?: number;
  }
): Promise<StreamStartResponse> {
  const request: StreamStartRequest = { policy };

  if (options?.forceDeviceId) {
    request.force_device_id = options.forceDeviceId;
  }
  if (options?.fps !== undefined) {
    request.fps = options.fps;
  }
  if (options?.quality !== undefined) {
    request.quality = options.quality;
  }
  if (options?.monitorIndex !== undefined) {
    request.monitor_index = options.monitorIndex;
  }

  return post<StreamStartResponse>('/api/stream/start', request);
}

// Complete WebRTC handshake with answer SDP
export async function sendStreamAnswer(
  selectedDeviceAddr: string,
  streamId: string,
  answerSdp: string
): Promise<void> {
  const request: StreamAnswerRequest = {
    selected_device_addr: selectedDeviceAddr,
    stream_id: streamId,
    answer_sdp: answerSdp,
  };

  await post<void>('/api/stream/answer', request);
}

// Stop an active stream
export async function stopStream(
  selectedDeviceAddr: string,
  streamId: string
): Promise<void> {
  const request: StreamStopRequest = {
    selected_device_addr: selectedDeviceAddr,
    stream_id: streamId,
  };

  await post<void>('/api/stream/stop', request);
}
