// Routing policies for command execution
export type RoutingPolicy =
  | 'BEST_AVAILABLE'
  | 'PREFER_REMOTE'
  | 'REQUIRE_NPU'
  | 'PREFER_LOCAL_MODEL'
  | 'REQUIRE_LOCAL_MODEL'
  | 'FORCE_DEVICE_ID';

// Device types
export interface Device {
  device_id: string;
  device_name: string;
  platform: string;
  arch: string;
  self_addr: string;
  has_gpu: boolean;
  has_npu: boolean;
  can_screen_capture: boolean;
  has_local_model: boolean;
}

export interface DeviceMetrics {
  cpu_percent: number;
  memory_percent: number;
  gpu_percent: number;
  npu_percent: number;
}

export interface DeviceMetricsHistory {
  device_id: string;
  samples: Array<{
    timestamp: string;
    cpu_percent: number;
    memory_percent: number;
    gpu_percent: number;
    npu_percent: number;
  }>;
}

// Command execution types
export interface RoutedCommandRequest {
  cmd: string;
  args: string[];
  policy: RoutingPolicy;
  force_device_id?: string;
}

export interface RoutedCommandResponse {
  device_id: string;
  device_name: string;
  stdout: string;
  stderr: string;
  exit_code: number;
  elapsed_ms: number;
}

// Job types
export interface JobSubmitRequest {
  text: string;
  max_workers?: number;
}

export interface JobSubmitResponse {
  job_id: string;
}

export type JobState = 'SUBMITTED' | 'QUEUED' | 'RUNNING' | 'DONE' | 'FAILED';
export type TaskState = 'PENDING' | 'QUEUED' | 'RUNNING' | 'DONE' | 'FAILED';

export interface JobStatusResponse {
  job_id: string;
  state: JobState;
  final_result: string;
  error: string;
}

export interface TaskDetail {
  task_id: string;
  group_index: number;
  kind: string;
  state: TaskState;
  input: string;
  result: string;
  error: string;
  assigned_device_id: string;
  assigned_device_name: string;
  elapsed_ms: number;
}

export interface JobDetailResponse {
  job_id: string;
  state: JobState;
  tasks: TaskDetail[];
  final_result: string;
  error: string;
}

// Plan types
export interface TaskPlan {
  kind: string;
  input: string;
}

export interface TaskGroup {
  tasks: TaskPlan[];
}

export interface ExecutionPlan {
  groups: TaskGroup[];
  reduce_op: string;
}

export interface PlanPreviewRequest {
  text: string;
  max_workers?: number;
}

export interface PlanPreviewResponse {
  plan: ExecutionPlan;
  used_ai: boolean;
  rationale: string;
  notes: string[];
}

export interface PlanCostRequest {
  plan: ExecutionPlan;
}

export interface PlanCostResponse {
  estimated_latency_ms: number;
  estimated_memory_mb: number;
  device_recommendations: string[];
}

// Activity types
export interface RunningTask {
  task_id: string;
  job_id: string;
  device_id: string;
  device_name: string;
  kind: string;
  state: TaskState;
  input: string;
  elapsed_ms: number;
}

export interface DeviceActivity {
  device_id: string;
  device_name: string;
  running_count: number;
  last_metrics: DeviceMetrics | null;
}

export interface MetricsHistoryEntry {
  device_id: string;
  device_name: string;
  timestamp: string;
  cpu_percent: number;
  memory_percent: number;
  gpu_percent: number;
  npu_percent: number;
}

export interface ActivityResponse {
  running_tasks: RunningTask[];
  device_activities: DeviceActivity[];
  metrics_history: MetricsHistoryEntry[];
}

// Streaming types
export interface StreamStartRequest {
  policy: RoutingPolicy;
  force_device_id?: string;
  fps?: number;
  quality?: number;
  monitor_index?: number;
}

export interface StreamStartResponse {
  stream_id: string;
  offer_sdp: string;
  selected_device_id: string;
  selected_device_name: string;
  selected_device_addr: string;
}

export interface StreamAnswerRequest {
  stream_id: string;
  answer_sdp: string;
}

export interface StreamAnswerResponse {
  success: boolean;
}

export interface StreamStopRequest {
  stream_id: string;
}

export interface StreamStopResponse {
  success: boolean;
}

// Download types
export interface DownloadRequest {
  device_id: string;
  file_path: string;
}

export interface DownloadTicketResponse {
  ticket_id: string;
  download_url: string;
  file_size: number;
  expires_at: string;
}

// Assistant types
export interface AssistantRequest {
  message: string;
}

export interface AssistantResponse {
  reply: string;
  mode: string;
  job_id: string;
  plan: ExecutionPlan | null;
}

// Agent types
export interface AgentRequest {
  message: string;
}

export interface AgentResponse {
  reply: string;
  tool_name: string;
  iterations: number;
}

export interface AgentHealthResponse {
  provider: string;
  model: string;
  available: boolean;
  error: string;
}

export interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
}

export interface ChatMemoryResponse {
  messages: ChatMessage[];
}

// QAI Hub types
export interface QAIHubDoctorResponse {
  qai_hub_found: boolean;
  qai_hub_version: string;
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
  job_id: string;
  out_dir: string;
  raw_output_path: string;
  notes: string[];
  error: string;
}
