// HTTP client for API calls
// Uses same-origin (relative paths) since web app is served from Go server

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

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let message = `${response.status} ${response.statusText}`;
    try {
      const body = await response.text();
      if (body) {
        message = body;
      }
    } catch {
      // Ignore parse errors
    }
    throw new ApiError(message, response.status, response.statusText);
  }
  return response.json();
}

export async function apiGet<T>(path: string): Promise<T> {
  const response = await fetch(path, {
    method: 'GET',
    headers: {
      'Accept': 'application/json',
    },
  });
  return handleResponse<T>(response);
}

export async function apiPost<T>(path: string, body?: unknown): Promise<T> {
  const response = await fetch(path, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  return handleResponse<T>(response);
}
