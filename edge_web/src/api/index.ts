// API Module - Re-export all API functions and types

// Client utilities
export { ApiError, getBaseUrl } from './client';

// Device APIs
export { listDevices, getDeviceMetrics } from './devices';

// Activity APIs
export { getActivity } from './activity';

// Command APIs
export { executeRoutedCommand } from './commands';

// Job APIs
export {
  submitJob,
  getJobStatus,
  getJobDetail,
  previewPlan,
  previewPlanCost,
} from './jobs';

// Assistant APIs
export { sendAssistantMessage } from './assistant';

// Streaming APIs
export { startStream, sendStreamAnswer, stopStream } from './streaming';

// Download APIs
export { requestDownload } from './downloads';

// QAI Hub APIs
export { runQAIHubDoctor, compileModel } from './qaihub';

// Agent APIs
export { getAgentHealth, sendAgentMessage, getChatMemory } from './agent';

// Types
export * from './types';
