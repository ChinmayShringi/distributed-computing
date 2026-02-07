import { apiPost } from './client';
import type {
  RoutingPolicy,
  StreamStartRequest,
  StreamStartResponse,
  StreamAnswerRequest,
  StreamAnswerResponse,
  StreamStopRequest,
  StreamStopResponse,
} from './types';

export interface StreamOptions {
  forceDeviceId?: string;
  fps?: number;
  quality?: number;
  monitorIndex?: number;
}

export async function startStream(
  policy: RoutingPolicy,
  options: StreamOptions = {}
): Promise<StreamStartResponse> {
  const request: StreamStartRequest = {
    policy,
    force_device_id: options.forceDeviceId,
    fps: options.fps,
    quality: options.quality,
    monitor_index: options.monitorIndex,
  };
  return apiPost<StreamStartResponse>('/api/stream/start', request);
}

export async function sendStreamAnswer(
  streamId: string,
  answerSdp: string,
  selectedDeviceAddr: string
): Promise<StreamAnswerResponse> {
  const request: StreamAnswerRequest = {
    stream_id: streamId,
    answer_sdp: answerSdp,
    selected_device_addr: selectedDeviceAddr,
  };
  return apiPost<StreamAnswerResponse>('/api/stream/answer', request);
}

export async function stopStream(streamId: string): Promise<StreamStopResponse> {
  const request: StreamStopRequest = {
    stream_id: streamId,
  };
  return apiPost<StreamStopResponse>('/api/stream/stop', request);
}
