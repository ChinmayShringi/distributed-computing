// Activity API endpoints
import { get } from './client';
import type { ActivityResponse } from './types';

// Get running tasks and device activity
export async function getActivity(includeMetricsHistory = true): Promise<ActivityResponse> {
  return get<ActivityResponse>('/api/activity', {
    include_metrics_history: includeMetricsHistory.toString(),
  });
}
