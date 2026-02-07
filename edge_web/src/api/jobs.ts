import { apiGet, apiPost } from './client';
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

export async function submitJob(
  text: string,
  maxWorkers?: number
): Promise<JobSubmitResponse> {
  const request: JobSubmitRequest = {
    text,
    max_workers: maxWorkers,
  };
  return apiPost<JobSubmitResponse>('/api/submit-job', request);
}

export async function getJobStatus(jobId: string): Promise<JobStatusResponse> {
  return apiGet<JobStatusResponse>(`/api/job?id=${encodeURIComponent(jobId)}`);
}

export async function getJobDetail(jobId: string): Promise<JobDetailResponse> {
  return apiGet<JobDetailResponse>(`/api/job-detail?id=${encodeURIComponent(jobId)}`);
}

export async function previewPlan(
  text: string,
  maxWorkers?: number
): Promise<PlanPreviewResponse> {
  const request: PlanPreviewRequest = {
    text,
    max_workers: maxWorkers,
  };
  return apiPost<PlanPreviewResponse>('/api/plan', request);
}

export async function previewPlanCost(plan: ExecutionPlan): Promise<PlanCostResponse> {
  const request: PlanCostRequest = { plan };
  return apiPost<PlanCostResponse>('/api/plan-cost', request);
}
