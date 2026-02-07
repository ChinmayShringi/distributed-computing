// Routed Command API endpoints
import { post } from './client';
import type { RoutedCommandRequest, RoutedCommandResponse } from './types';

// Execute a routed command on a device
export async function executeRoutedCommand(
  cmd: string,
  args: string[],
  policy: RoutedCommandRequest['policy'],
  forceDeviceId?: string
): Promise<RoutedCommandResponse> {
  const request: RoutedCommandRequest = {
    cmd,
    args,
    policy,
  };

  if (forceDeviceId) {
    request.force_device_id = forceDeviceId;
  }

  return post<RoutedCommandResponse>('/api/routed-cmd', request);
}
