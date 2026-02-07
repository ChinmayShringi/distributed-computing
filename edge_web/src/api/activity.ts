import { apiGet } from './client';
import type { ActivityResponse } from './types';

export async function getActivity(includeMetricsHistory = false): Promise<ActivityResponse> {
  const params = new URLSearchParams();
  if (includeMetricsHistory) {
    params.set('include_metrics_history', 'true');
  }
  const query = params.toString();
  const path = query ? `/api/activity?${query}` : '/api/activity';
  return apiGet<ActivityResponse>(path);
}
