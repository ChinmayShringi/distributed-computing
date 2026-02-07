import { motion } from 'framer-motion';
import { GlassCard } from '@/components/GlassCard';
import {
  Zap,
  Shield,
  Cpu,
  Globe,
  Lock,
  Activity,
  Terminal,
  Layers,
  Network,
  Bot,
  HardDrive,
  Eye,
} from 'lucide-react';

const features = [
  {
    icon: Globe,
    title: 'Global Device Mesh',
    description: 'Connect devices anywhere in the world into a unified compute mesh. Works across networks, through NAT, and behind firewalls.',
    highlights: ['Auto-discovery', 'NAT traversal', 'Mesh networking'],
  },
  {
    icon: Cpu,
    title: 'Heterogeneous Compute',
    description: 'Seamlessly leverage CPU, GPU, and NPU resources across your entire device fleet. Intelligent workload routing based on capabilities.',
    highlights: ['GPU acceleration', 'NPU support', 'Smart routing'],
  },
  {
    icon: Terminal,
    title: 'Remote Execution',
    description: 'Run scripts, commands, and tools on any connected device. Full terminal access with real-time output streaming.',
    highlights: ['Real-time output', 'Script library', 'Batch execution'],
  },
  {
    icon: Activity,
    title: 'Live Monitoring',
    description: 'Real-time telemetry from every node. CPU, memory, network, and custom metrics with alerting and historical data.',
    highlights: ['Live dashboards', 'Custom metrics', 'Alerts'],
  },
  {
    icon: Bot,
    title: 'AI Integration',
    description: 'Built-in AI assistant for natural language device management. Ask questions, run diagnostics, and automate tasks.',
    highlights: ['Natural language', 'Smart suggestions', 'Automation'],
  },
  {
    icon: Layers,
    title: 'Job Orchestration',
    description: 'Create and manage distributed jobs across multiple nodes. Dependencies, retries, and progress tracking included.',
    highlights: ['Multi-node jobs', 'Dependencies', 'Progress tracking'],
  },
  {
    icon: HardDrive,
    title: 'File Transfers',
    description: 'Secure file transfers between devices. Generate time-limited download tickets for controlled access.',
    highlights: ['Secure transfers', 'TTL tickets', 'Resume support'],
  },
  {
    icon: Eye,
    title: 'Remote View',
    description: 'View device screens in real-time. Perfect for debugging, monitoring, and remote support scenarios.',
    highlights: ['Live streaming', 'Low latency', 'Quality control'],
  },
  {
    icon: Shield,
    title: 'Enterprise Security',
    description: 'Zero-trust architecture with mTLS encryption, role-based access control, and comprehensive audit logging.',
    highlights: ['mTLS', 'RBAC', 'Audit logs'],
  },
  {
    icon: Lock,
    title: 'Access Control',
    description: 'Fine-grained permissions with approval workflows. Request-based access for sensitive operations.',
    highlights: ['Approval workflows', 'Permissions', 'Request system'],
  },
  {
    icon: Network,
    title: 'Network Tools',
    description: 'Built-in network diagnostics and scanning tools. Discover devices, check connectivity, and troubleshoot issues.',
    highlights: ['Port scanning', 'Connectivity tests', 'Discovery'],
  },
  {
    icon: Zap,
    title: 'Instant Deployment',
    description: 'Zero-config agent deployment. Single binary that works on Linux, macOS, Windows, and ARM devices.',
    highlights: ['Single binary', 'Cross-platform', 'Auto-update'],
  },
];

export const FeaturesPage = () => {
  return (
    <div className="py-24 px-4">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-center mb-16"
        >
          <h1 className="text-4xl md:text-5xl font-bold mb-4">
            Powerful <span className="text-primary">Features</span>
          </h1>
          <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
            Everything you need to build, manage, and scale your distributed edge infrastructure.
          </p>
        </motion.div>

        {/* Features Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {features.map((feature, index) => (
            <motion.div
              key={feature.title}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.05 }}
            >
              <GlassCard hover className="h-full">
                <div className="p-2 space-y-4">
                  <div className="w-12 h-12 rounded-xl bg-primary/20 flex items-center justify-center">
                    <feature.icon className="w-6 h-6 text-primary" />
                  </div>
                  <div>
                    <h3 className="text-lg font-semibold mb-2">{feature.title}</h3>
                    <p className="text-sm text-muted-foreground mb-4">{feature.description}</p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {feature.highlights.map((highlight) => (
                      <span
                        key={highlight}
                        className="px-2 py-1 rounded-md bg-surface-variant text-xs text-muted-foreground"
                      >
                        {highlight}
                      </span>
                    ))}
                  </div>
                </div>
              </GlassCard>
            </motion.div>
          ))}
        </div>
      </div>
    </div>
  );
};
