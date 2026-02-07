import { getConnectionState } from '@/lib/connection';
import type { AssistantResponse, JobResult, DeviceInfo } from './types';

// Get the server address from connection state
function getServerAddress(): string {
  const state = getConnectionState();
  return state.serverAddress || 'http://localhost:8080';
}

// Send a message to the assistant
export async function sendAssistantMessage(text: string): Promise<AssistantResponse> {
  const serverAddress = getServerAddress();

  const response = await fetch(`${serverAddress}/api/assistant`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ text }),
  });

  if (!response.ok) {
    throw new Error(`Assistant request failed: ${response.statusText}`);
  }

  return response.json();
}

// Get job status and results
export async function getJob(jobId: string): Promise<JobResult> {
  const serverAddress = getServerAddress();

  const response = await fetch(`${serverAddress}/api/job?id=${encodeURIComponent(jobId)}`);

  if (!response.ok) {
    throw new Error(`Job request failed: ${response.statusText}`);
  }

  return response.json();
}

// List all devices
export async function listDevices(): Promise<DeviceInfo[]> {
  const serverAddress = getServerAddress();

  const response = await fetch(`${serverAddress}/api/devices`);

  if (!response.ok) {
    throw new Error(`Devices request failed: ${response.statusText}`);
  }

  const data = await response.json();
  return data.devices || [];
}

// Submit a job for execution
export async function submitJob(plan: Record<string, unknown>): Promise<{ job_id: string }> {
  const serverAddress = getServerAddress();

  const response = await fetch(`${serverAddress}/api/submit-job`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(plan),
  });

  if (!response.ok) {
    throw new Error(`Submit job failed: ${response.statusText}`);
  }

  return response.json();
}
