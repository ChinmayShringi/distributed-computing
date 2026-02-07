// Job API endpoints
import { get, post } from './client';
import type {
  JobSubmitRequest,
  JobSubmitResponse,
  JobStatusResponse,
  JobDetailResponse,
  PlanPreviewRequest,
  PlanPreviewResponse,
  PlanCostRequest,
  PlanCostResponse,
  ExecutionPlan,
} from './types';

// Submit a distributed job
export async function submitJob(
  text: string,
  maxWorkers?: number,
  plan?: ExecutionPlan
): Promise<JobSubmitResponse> {
  const request: JobSubmitRequest = { text };

  if (maxWorkers !== undefined && maxWorkers > 0) {
    request.max_workers = maxWorkers;
  }

  if (plan) {
    request.plan = plan;
  }

  return post<JobSubmitResponse>('/api/submit-job', request);
}

// Get job status
export async function getJobStatus(jobId: string): Promise<JobStatusResponse> {
  return get<JobStatusResponse>('/api/job', { id: jobId });
}

// Get enhanced job details with task timing
export async function getJobDetail(jobId: string): Promise<JobDetailResponse> {
  return get<JobDetailResponse>('/api/job-detail', { id: jobId });
}

// Preview execution plan without running
export async function previewPlan(
  text: string,
  maxWorkers?: number
): Promise<PlanPreviewResponse> {
  const request: PlanPreviewRequest = { text };

  if (maxWorkers !== undefined && maxWorkers > 0) {
    request.max_workers = maxWorkers;
  }

  return post<PlanPreviewResponse>('/api/plan', request);
}

// Estimate execution cost for a plan
export async function previewPlanCost(
  text: string,
  maxWorkers?: number
): Promise<PlanCostResponse> {
  const request: PlanCostRequest = { text };

  if (maxWorkers !== undefined && maxWorkers > 0) {
    request.max_workers = maxWorkers;
  }

  return post<PlanCostResponse>('/api/plan-cost', request);
}
