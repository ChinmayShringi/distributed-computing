import { apiGet, apiPost } from './client';
import type { QAIHubDoctorResponse, QAIHubCompileRequest, QAIHubCompileResponse } from './types';

export async function runQAIHubDoctor(): Promise<QAIHubDoctorResponse> {
  return apiGet<QAIHubDoctorResponse>('/api/qaihub/doctor');
}

export async function compileModel(
  onnxPath: string,
  target: string,
  runtime?: string
): Promise<QAIHubCompileResponse> {
  const request: QAIHubCompileRequest = {
    onnx_path: onnxPath,
    target,
    runtime,
  };
  return apiPost<QAIHubCompileResponse>('/api/qaihub/compile', request);
}
