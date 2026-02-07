import { apiPost } from './client';
import type { RoutingPolicy, RoutedCommandRequest, RoutedCommandResponse } from './types';

export async function executeRoutedCommand(
  cmd: string,
  args: string[],
  policy: RoutingPolicy = 'BEST_AVAILABLE',
  forceDeviceId?: string
): Promise<RoutedCommandResponse> {
  const request: RoutedCommandRequest = {
    cmd,
    args,
    policy,
    force_device_id: forceDeviceId,
  };
  return apiPost<RoutedCommandResponse>('/api/routed-cmd', request);
}
