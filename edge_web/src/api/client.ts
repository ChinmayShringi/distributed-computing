// API Client for EdgeCLI Web Console
// Centralized HTTP client with error handling

import { getConnectionState } from '@/lib/connection';

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public statusText: string
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

function getBaseUrl(): string {
  const state = getConnectionState();
  return state.serverAddress || 'http://localhost:8080';
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const errorText = await response.text().catch(() => response.statusText);
    throw new ApiError(
      `API request failed: ${errorText}`,
      response.status,
      response.statusText
    );
  }
  return response.json();
}

export async function get<T>(endpoint: string, params?: Record<string, string>): Promise<T> {
  const baseUrl = getBaseUrl();
  const url = new URL(endpoint, baseUrl);

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      url.searchParams.append(key, value);
    });
  }

  const response = await fetch(url.toString(), {
    method: 'GET',
    headers: {
      'Accept': 'application/json',
    },
  });

  return handleResponse<T>(response);
}

export async function post<T, B = unknown>(endpoint: string, body?: B): Promise<T> {
  const baseUrl = getBaseUrl();
  const url = new URL(endpoint, baseUrl);

  const response = await fetch(url.toString(), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  return handleResponse<T>(response);
}

// Export base URL getter for components that need it
export { getBaseUrl };
