// API Types for EdgeCLI Web Console
// Comprehensive types for all 19 API endpoints

// ==================== Device Types ====================

export type RoutingPolicy =
  | 'BEST_AVAILABLE'
  | 'PREFER_REMOTE'
  | 'REQUIRE_NPU'
  | 'PREFER_LOCAL_MODEL'
  | 'REQUIRE_LOCAL_MODEL'
  | 'FORCE_DEVICE_ID';

export interface DeviceCapabilities {
  has_gpu: boolean;
  has_npu: boolean;
  can_screen_capture: boolean;
  has_local_model: boolean;
  local_model_name?: string;
}

export interface Device {
  device_id: string;
  device_name: string;
  platform: string;
  arch: string;
  grpc_addr: string;
  capabilities?: string[];
  has_gpu?: boolean;
  has_npu?: boolean;
  can_screen_capture?: boolean;
  has_local_model?: boolean;
  local_model_name?: string;
}

export interface DevicesResponse {
  devices: Device[];
}

// ==================== Activity Types ====================

export interface RunningTask {
  job_id: string;
  task_id: string;
  device_id: string;
  device_name: string;
  kind: string;
  input: string;
  started_at: string;
  elapsed_ms: number;
  state: 'QUEUED' | 'RUNNING' | 'DONE' | 'FAILED';
}

export interface DeviceMetrics {
  cpu_percent: number;
  memory_percent: number;
  gpu_percent: number;
  npu_percent: number;
  timestamp: string;
}

export interface DeviceActivity {
  device_id: string;
  device_name: string;
  running_count: number;
  last_metrics?: DeviceMetrics;
}

export interface MetricsSample {
  timestamp: string;
  cpu_percent: number;
  memory_percent: number;
  gpu_percent: number;
  npu_percent: number;
}

export interface DeviceMetricsHistory {
  device_id: string;
  device_name: string;
  samples: MetricsSample[];
}

export interface ActivityResponse {
  running_tasks: RunningTask[];
  device_activities: DeviceActivity[];
  metrics_history?: DeviceMetricsHistory[];
}

// ==================== Command Types ====================

export interface RoutedCommandRequest {
  cmd: string;
  args: string[];
  policy: RoutingPolicy;
  force_device_id?: string;
}

export interface RoutedCommandResponse {
  selected_device_id: string;
  selected_device_name: string;
  executed_locally: boolean;
  total_time_ms: number;
  exit_code: number;
  stdout: string;
  stderr: string;
}

// ==================== Job Types ====================

export interface Task {
  task_id: string;
  group_index: number;
  kind: string;
  input: string;
  assigned_device_id?: string;
  assigned_device_name?: string;
  state: 'PENDING' | 'QUEUED' | 'RUNNING' | 'DONE' | 'FAILED';
  result?: string;
  error?: string;
  started_at?: string;
  ended_at?: string;
  elapsed_ms?: number;
}

export interface JobSubmitRequest {
  text: string;
  max_workers?: number;
  plan?: ExecutionPlan;
}

export interface JobSubmitResponse {
  job_id: string;
  summary: string;
}

export interface JobStatusResponse {
  id: string;
  state: 'SUBMITTED' | 'QUEUED' | 'RUNNING' | 'DONE' | 'FAILED';
  final_result?: string;
  error?: string;
  tasks?: Task[];
}

export interface JobDetailResponse {
  id: string;
  state: 'SUBMITTED' | 'QUEUED' | 'RUNNING' | 'DONE' | 'FAILED';
  tasks: Task[];
  final_result?: string;
  error?: string;
}

// ==================== Plan Types ====================

export interface TaskPlan {
  kind: string;
  input: string;
  device_id?: string;
}

export interface GroupPlan {
  tasks: TaskPlan[];
}

export interface ExecutionPlan {
  groups: GroupPlan[];
  reduce?: string;
}

export interface PlanPreviewRequest {
  text: string;
  max_workers?: number;
}

export interface PlanPreviewResponse {
  used_ai: boolean;
  rationale: string;
  notes: string[];
  plan: ExecutionPlan;
}

export interface PlanCostRequest {
  text: string;
  max_workers?: number;
}

export interface PlanCostResponse {
  estimated_latency_ms: number;
  estimated_memory_mb: number;
  recommended_devices: string[];
  notes: string[];
}

// ==================== Assistant Types ====================

export interface AssistantRequest {
  text: string;
}

export interface AssistantResponse {
  reply: string;
  mode?: 'ROUTED_CMD' | 'JOB' | 'QUERY' | 'UNKNOWN';
  job_id?: string;
  plan?: ExecutionPlan;
  raw?: unknown;
}

// ==================== Streaming Types ====================

export interface StreamStartRequest {
  policy: RoutingPolicy;
  force_device_id?: string;
  fps?: number;
  quality?: number;
  monitor_index?: number;
}

export interface StreamStartResponse {
  offer_sdp: string;
  stream_id: string;
  selected_device_id: string;
  selected_device_name: string;
  selected_device_addr: string;
}

export interface StreamAnswerRequest {
  selected_device_addr: string;
  stream_id: string;
  answer_sdp: string;
}

export interface StreamStopRequest {
  selected_device_addr: string;
  stream_id: string;
}

// ==================== Download Types ====================

export interface DownloadRequest {
  device_id: string;
  path: string;
}

export interface DownloadResponse {
  filename: string;
  size_bytes: number;
  download_url: string;
  expires_unix_ms: number;
}

// ==================== QAI Hub Types ====================

export interface QAIHubDoctorResponse {
  qai_hub_found: boolean;
  qai_hub_version?: string;
  token_env_present: boolean;
  notes: string[];
}

export interface QAIHubCompileRequest {
  onnx_path: string;
  target: string;
  runtime?: string;
}

export interface QAIHubCompileResponse {
  submitted: boolean;
  job_id?: string;
  out_dir?: string;
  notes: string[];
  raw_output_path?: string;
  error?: string;
}

// ==================== Agent Types ====================

export interface AgentHealthResponse {
  ok: boolean;
  provider: string;
  base_url: string;
  model: string;
  error?: string;
}

export interface ToolCall {
  name: string;
  arguments: Record<string, unknown>;
  result?: string;
}

export interface AgentRequest {
  message: string;
}

export interface AgentResponse {
  reply: string;
  tool_calls: ToolCall[];
  iterations: number;
}

export interface ChatMemoryMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

export interface ChatMemoryResponse {
  messages: ChatMemoryMessage[];
}

// ==================== Device Metrics Types ====================

export interface DeviceMetricsRequest {
  device_id: string;
}

export interface DeviceMetricsResponse {
  device_id: string;
  device_name: string;
  samples: MetricsSample[];
}
