// Mock data for EDGE MESH console

export interface Device {
  id: string;
  name: string;
  os: string;
  type: 'desktop' | 'laptop' | 'server' | 'mobile' | 'iot';
  status: 'online' | 'offline';
  cpuPct: number;
  memPct: number;
  capabilities: string[];
  ip?: string;
  lastSeen?: string;
  hardware?: {
    cpu: string;
    memory: string;
    gpu?: string;
    storage: string;
  };
}

export interface Execution {
  id: string;
  cmd: string;
  deviceId: string;
  deviceName: string;
  computeType: 'CPU' | 'GPU' | 'NPU';
  memoryUsedMb: number;
  totalTimeMs: number;
  exitCode: number;
  time: string;
  output: string;
}

export interface Job {
  id: string;
  name: string;
  status: 'running' | 'completed' | 'failed' | 'queued';
  progress: number;
  nodesCount: number;
  usedAi: boolean;
  startTime: string;
  tags: string[];
}

export interface PendingRequest {
  id: string;
  deviceId: string;
  deviceName: string;
  action: string;
  description: string;
  riskLevel: 'low' | 'medium' | 'high';
  requestedAt: string;
}

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: string;
}

export const mockDevices: Device[] = [
  {
    id: 'dev-001',
    name: 'Workstation Alpha',
    os: 'Ubuntu 22.04',
    type: 'desktop',
    status: 'online',
    cpuPct: 45,
    memPct: 62,
    capabilities: ['GPU Compute', 'Docker', 'SSH'],
    ip: '192.168.1.101',
    lastSeen: 'Just now',
    hardware: {
      cpu: 'AMD Ryzen 9 5900X',
      memory: '64 GB DDR4',
      gpu: 'NVIDIA RTX 4090',
      storage: '2 TB NVMe SSD',
    },
  },
  {
    id: 'dev-002',
    name: 'MacBook Pro M3',
    os: 'macOS Sonoma',
    type: 'laptop',
    status: 'online',
    cpuPct: 28,
    memPct: 45,
    capabilities: ['NPU', 'Docker', 'SSH'],
    ip: '192.168.1.102',
    lastSeen: 'Just now',
    hardware: {
      cpu: 'Apple M3 Max',
      memory: '48 GB Unified',
      gpu: 'Integrated 40-core',
      storage: '1 TB SSD',
    },
  },
  {
    id: 'dev-003',
    name: 'Edge Node Beta',
    os: 'Debian 12',
    type: 'server',
    status: 'online',
    cpuPct: 72,
    memPct: 81,
    capabilities: ['GPU Compute', 'Kubernetes', 'Docker'],
    ip: '192.168.1.103',
    lastSeen: 'Just now',
    hardware: {
      cpu: 'Intel Xeon W-2295',
      memory: '128 GB ECC',
      gpu: 'NVIDIA A100',
      storage: '8 TB NVMe RAID',
    },
  },
  {
    id: 'dev-004',
    name: 'Pi Cluster Node',
    os: 'Raspberry Pi OS',
    type: 'iot',
    status: 'offline',
    cpuPct: 0,
    memPct: 0,
    capabilities: ['SSH', 'GPIO'],
    ip: '192.168.1.104',
    lastSeen: '3 hours ago',
    hardware: {
      cpu: 'ARM Cortex-A76',
      memory: '8 GB',
      storage: '256 GB SD',
    },
  },
  {
    id: 'dev-005',
    name: 'Dev Phone',
    os: 'Android 14',
    type: 'mobile',
    status: 'offline',
    cpuPct: 0,
    memPct: 0,
    capabilities: ['ADB', 'Screen Mirror'],
    ip: '192.168.1.105',
    lastSeen: '1 day ago',
    hardware: {
      cpu: 'Snapdragon 8 Gen 3',
      memory: '12 GB',
      storage: '256 GB',
    },
  },
];

export const mockExecutions: Execution[] = [
  {
    id: 'exec-001',
    cmd: 'python train_model.py --epochs 100',
    deviceId: 'dev-001',
    deviceName: 'Workstation Alpha',
    computeType: 'GPU',
    memoryUsedMb: 8420,
    totalTimeMs: 45230,
    exitCode: 0,
    time: '2 minutes ago',
    output: `[INFO] Loading dataset...
[INFO] Model initialized with 45M parameters
[INFO] Starting training on GPU:0 (RTX 4090)
Epoch 1/100: loss=2.341, acc=0.42
Epoch 2/100: loss=1.892, acc=0.56
...
Epoch 100/100: loss=0.124, acc=0.98
[SUCCESS] Training complete. Model saved to ./checkpoints/model_final.pt`,
  },
  {
    id: 'exec-002',
    cmd: 'docker compose up -d',
    deviceId: 'dev-003',
    deviceName: 'Edge Node Beta',
    computeType: 'CPU',
    memoryUsedMb: 512,
    totalTimeMs: 3200,
    exitCode: 0,
    time: '15 minutes ago',
    output: `[+] Running 4/4
 ✔ Network app_default      Created
 ✔ Container redis          Started
 ✔ Container postgres       Started
 ✔ Container api            Started`,
  },
  {
    id: 'exec-003',
    cmd: 'npm run build',
    deviceId: 'dev-002',
    deviceName: 'MacBook Pro M3',
    computeType: 'NPU',
    memoryUsedMb: 2048,
    totalTimeMs: 12400,
    exitCode: 0,
    time: '1 hour ago',
    output: `> edge-mesh-client@1.0.0 build
> vite build

vite v5.0.0 building for production...
✓ 1247 modules transformed.
dist/index.html                 0.45 kB
dist/assets/index-abc123.js   245.32 kB │ gzip: 78.21 kB
dist/assets/index-def456.css   12.45 kB │ gzip: 3.12 kB
✓ built in 12.4s`,
  },
  {
    id: 'exec-004',
    cmd: 'kubectl apply -f deployment.yaml',
    deviceId: 'dev-003',
    deviceName: 'Edge Node Beta',
    computeType: 'CPU',
    memoryUsedMb: 128,
    totalTimeMs: 1850,
    exitCode: 1,
    time: '2 hours ago',
    output: `error: error validating "deployment.yaml": error validating data: ValidationError(Deployment.spec.template.spec.containers[0]): unknown field "imagePullSecrets" in io.k8s.api.core.v1.Container`,
  },
];

export const mockJobs: Job[] = [
  {
    id: 'job-7x9k2m',
    name: 'ML Pipeline Batch',
    status: 'running',
    progress: 67,
    nodesCount: 3,
    usedAi: true,
    startTime: '10 minutes ago',
    tags: ['AI', 'GPU'],
  },
  {
    id: 'job-3f8n1p',
    name: 'Data Sync Task',
    status: 'completed',
    progress: 100,
    nodesCount: 2,
    usedAi: false,
    startTime: '1 hour ago',
    tags: ['Sync'],
  },
  {
    id: 'job-9w2q5r',
    name: 'Security Scan',
    status: 'queued',
    progress: 0,
    nodesCount: 1,
    usedAi: false,
    startTime: 'Pending',
    tags: ['Security'],
  },
  {
    id: 'job-1m4k8v',
    name: 'Build & Deploy',
    status: 'failed',
    progress: 45,
    nodesCount: 2,
    usedAi: false,
    startTime: '3 hours ago',
    tags: ['CI/CD'],
  },
];

export const mockPendingRequests: PendingRequest[] = [
  {
    id: 'req-001',
    deviceId: 'dev-001',
    deviceName: 'Workstation Alpha',
    action: 'File System Access',
    description: 'Request to read files from /home/user/projects directory',
    riskLevel: 'low',
    requestedAt: '2 minutes ago',
  },
  {
    id: 'req-002',
    deviceId: 'dev-003',
    deviceName: 'Edge Node Beta',
    action: 'Network Configuration',
    description: 'Modify firewall rules to allow inbound connections on port 8080',
    riskLevel: 'medium',
    requestedAt: '5 minutes ago',
  },
  {
    id: 'req-003',
    deviceId: 'dev-002',
    deviceName: 'MacBook Pro M3',
    action: 'System Restart',
    description: 'Initiate system restart for pending updates',
    riskLevel: 'high',
    requestedAt: '10 minutes ago',
  },
];

export const mockChatMessages: ChatMessage[] = [
  {
    id: 'msg-001',
    role: 'assistant',
    content: 'Hello! I\'m your AI assistant for Edge Mesh. I can help you manage devices, run diagnostics, analyze logs, and more. What would you like to do today?',
    timestamp: '10:30 AM',
  },
  {
    id: 'msg-002',
    role: 'user',
    content: 'Can you check the status of all my connected devices?',
    timestamp: '10:31 AM',
  },
  {
    id: 'msg-003',
    role: 'assistant',
    content: 'I\'ve checked your device fleet. Here\'s a summary:\n\n**Online (3 devices):**\n- Workstation Alpha (CPU: 45%, Memory: 62%)\n- MacBook Pro M3 (CPU: 28%, Memory: 45%)\n- Edge Node Beta (CPU: 72%, Memory: 81%)\n\n**Offline (2 devices):**\n- Pi Cluster Node (last seen 3 hours ago)\n- Dev Phone (last seen 1 day ago)\n\nEdge Node Beta is running at high utilization. Would you like me to investigate?',
    timestamp: '10:31 AM',
  },
];

// Helper to get aggregate stats
export const getDeviceStats = () => {
  const onlineDevices = mockDevices.filter(d => d.status === 'online');
  return {
    connectedDevices: onlineDevices.length,
    totalDevices: mockDevices.length,
    toolsAvailable: 12,
    activeJobs: mockJobs.filter(j => j.status === 'running').length,
    avgCpu: Math.round(onlineDevices.reduce((acc, d) => acc + d.cpuPct, 0) / onlineDevices.length),
    avgMemory: Math.round(onlineDevices.reduce((acc, d) => acc + d.memPct, 0) / onlineDevices.length),
  };
};

export const getComputeDistribution = () => [
  { name: 'CPU', value: 45, color: 'hsl(217, 91%, 60%)' },
  { name: 'GPU', value: 40, color: 'hsl(346, 100%, 62%)' },
  { name: 'NPU', value: 15, color: 'hsl(157, 93%, 55%)' },
];

export const getDeviceRunDistribution = () => [
  { name: 'Workstation Alpha', value: 42, color: 'hsl(346, 100%, 62%)' },
  { name: 'Edge Node Beta', value: 35, color: 'hsl(157, 93%, 55%)' },
  { name: 'MacBook Pro M3', value: 23, color: 'hsl(217, 91%, 60%)' },
];
