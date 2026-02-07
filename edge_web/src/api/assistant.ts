// Assistant API endpoints
import { post } from './client';
import type { AssistantRequest, AssistantResponse } from './types';

// Send a message to the natural language assistant
export async function sendAssistantMessage(text: string): Promise<AssistantResponse> {
  const request: AssistantRequest = { text };
  return post<AssistantResponse>('/api/assistant', request);
}
