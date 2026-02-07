// Message types matching Flutter's MessageType enum
export type MessageType =
  | 'text'
  | 'plan'
  | 'devices'
  | 'result'
  | 'approval'
  | 'job'
  | 'stream'
  | 'download';

export type MessageSender = 'user' | 'assistant';

export interface ChatMessage {
  id: string;
  type: MessageType;
  sender: MessageSender;
  text?: string;
  payload?: Record<string, unknown>;
  createdAt: Date;
}

export interface PlanPayload {
  steps: string[];
  job_id?: string;
  device: string;
  policy: string;
  risk: string;
  cmd?: string;
}

export interface DeviceInfo {
  name: string;
  type: 'mobile' | 'desktop' | 'server';
  status: 'online' | 'offline';
  os: string;
  device_id: string;
  capabilities?: string[];
}

export interface DevicePayload {
  devices: DeviceInfo[];
}

export interface ResultPayload {
  cmd: string;
  device: string;
  host_compute: string;
  time: string;
  exit_code: number;
  output: string;
}

export interface AssistantResponse {
  reply: string;
  raw?: unknown;
  mode?: string;
  job_id?: string;
  plan?: Record<string, unknown>;
}

export interface JobResult {
  id: string;
  state: 'PENDING' | 'RUNNING' | 'DONE' | 'FAILED';
  results?: Array<{
    device_id: string;
    stdout: string;
    stderr: string;
    exit_code: number;
    elapsed_ms: number;
  }>;
  error?: string;
}

// Helper to create a new message
export function createMessage(
  sender: MessageSender,
  type: MessageType,
  text?: string,
  payload?: Record<string, unknown>
): ChatMessage {
  return {
    id: crypto.randomUUID(),
    type,
    sender,
    text,
    payload,
    createdAt: new Date(),
  };
}
