// QAI Hub API endpoints
import { get, post } from './client';
import type {
  QAIHubDoctorResponse,
  QAIHubCompileRequest,
  QAIHubCompileResponse,
} from './types';

// Check QAI Hub CLI health
export async function runQAIHubDoctor(): Promise<QAIHubDoctorResponse> {
  return get<QAIHubDoctorResponse>('/api/qaihub/doctor');
}

// Compile a model with QAI Hub
export async function compileModel(
  onnxPath: string,
  target: string,
  runtime?: string
): Promise<QAIHubCompileResponse> {
  const request: QAIHubCompileRequest = {
    onnx_path: onnxPath,
    target,
  };

  if (runtime) {
    request.runtime = runtime;
  }

  return post<QAIHubCompileResponse>('/api/qaihub/compile', request);
}
