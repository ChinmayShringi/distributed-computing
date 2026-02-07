import { apiPost } from './client';
import type { AssistantRequest, AssistantResponse } from './types';

export async function sendAssistantMessage(message: string): Promise<AssistantResponse> {
  const request: AssistantRequest = { message };
  return apiPost<AssistantResponse>('/api/assistant', request);
}
