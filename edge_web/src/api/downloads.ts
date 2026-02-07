import { apiPost } from './client';
import type { DownloadRequest, DownloadTicketResponse } from './types';

export async function requestDownload(
  deviceId: string,
  filePath: string
): Promise<DownloadTicketResponse> {
  const request: DownloadRequest = {
    device_id: deviceId,
    file_path: filePath,
  };
  return apiPost<DownloadTicketResponse>('/api/request-download', request);
}

export function triggerDownload(downloadUrl: string, filename?: string): void {
  const link = document.createElement('a');
  link.href = downloadUrl;
  if (filename) {
    link.download = filename;
  }
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}
