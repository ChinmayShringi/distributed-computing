// Download API endpoints
import { post } from './client';
import type { DownloadRequest, DownloadResponse } from './types';

// Request a file download ticket
export async function requestDownload(
  deviceId: string,
  path: string
): Promise<DownloadResponse> {
  const request: DownloadRequest = {
    device_id: deviceId,
    path,
  };

  return post<DownloadResponse>('/api/request-download', request);
}
