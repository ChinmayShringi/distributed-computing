import { apiGet, apiPost } from './client';
import type { AgentRequest, AgentResponse, AgentHealthResponse, ChatMemoryResponse } from './types';

export async function sendAgentMessage(message: string): Promise<AgentResponse> {
  const request: AgentRequest = { message };
  return apiPost<AgentResponse>('/api/agent', request);
}

export async function getAgentHealth(): Promise<AgentHealthResponse> {
  return apiGet<AgentHealthResponse>('/api/agent/health');
}

export async function getChatMemory(): Promise<ChatMemoryResponse> {
  return apiGet<ChatMemoryResponse>('/api/chat/memory');
}
