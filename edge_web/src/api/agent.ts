import { apiGet, apiPost } from './client';
import type { AgentRequest, AgentResponse, AgentHealthResponse, ChatMemoryResponse } from './types';

export async function sendAgentMessage(message: string, deviceId?: string, senderDeviceId?: string): Promise<AgentResponse> {
  const request: AgentRequest = {
    message,
    device_id: deviceId,
    sender_device_id: senderDeviceId
  };
  return apiPost<AgentResponse>('/api/agent', request);
}

export async function getAgentHealth(): Promise<AgentHealthResponse> {
  return apiGet<AgentHealthResponse>('/api/agent/health');
}

export async function getChatMemory(deviceId?: string): Promise<ChatMemoryResponse> {
  const params = deviceId ? { device_id: deviceId } : {};
  return apiGet<ChatMemoryResponse>('/api/chat/memory', params);
}
