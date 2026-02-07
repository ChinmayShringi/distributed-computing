// API Types
export * from './types';

// API Client
export { ApiError } from './client';

// Device APIs
export { listDevices, getDeviceStatus, getDeviceMetrics } from './devices';

// Command APIs
export { executeRoutedCommand } from './commands';

// Job APIs
export { submitJob, getJobStatus, getJobDetail, previewPlan, previewPlanCost } from './jobs';

// Activity APIs
export { getActivity } from './activity';

// Assistant APIs
export { sendAssistantMessage } from './assistant';

// Agent APIs
export { sendAgentMessage, getAgentHealth, getChatMemory } from './agent';

// Streaming APIs
export { startStream, sendStreamAnswer, stopStream } from './streaming';
export type { StreamOptions } from './streaming';

// Download APIs
export { requestDownload, triggerDownload } from './downloads';

// QAI Hub APIs
export { runQAIHubDoctor, compileModel } from './qaihub';
