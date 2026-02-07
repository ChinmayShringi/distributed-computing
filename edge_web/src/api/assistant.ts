import { apiPost } from './client';
import type { AssistantRequest, AssistantResponse } from './types';

export async function sendAssistantMessage(message: string, deviceId?: string): Promise<AssistantResponse> {
  const request: AssistantRequest = { message, device_id: deviceId };
  return apiPost<AssistantResponse>('/api/assistant', request);
}
