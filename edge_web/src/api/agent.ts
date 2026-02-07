// LLM Agent API endpoints
import { get, post } from './client';
import type {
  AgentHealthResponse,
  AgentRequest,
  AgentResponse,
  ChatMemoryResponse,
} from './types';

// Check agent health
export async function getAgentHealth(): Promise<AgentHealthResponse> {
  return get<AgentHealthResponse>('/api/agent/health');
}

// Send a message to the LLM agent with tool calling
export async function sendAgentMessage(message: string): Promise<AgentResponse> {
  const request: AgentRequest = { message };
  return post<AgentResponse>('/api/agent', request);
}

// Get chat history from server
export async function getChatMemory(): Promise<ChatMemoryResponse> {
  return get<ChatMemoryResponse>('/api/chat/memory');
}
