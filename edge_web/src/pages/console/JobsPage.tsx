import { motion } from 'framer-motion';
import { GlassCard } from '@/components/GlassCard';
import { Progress } from '@/components/ui/progress';
import { mockJobs } from '@/lib/mock-data';
import { CheckCircle, XCircle, Loader2, Clock, Bot, Cpu } from 'lucide-react';

const statusIcons = {
  running: Loader2,
  completed: CheckCircle,
  failed: XCircle,
  queued: Clock,
};

const statusColors = {
  running: 'text-info-blue',
  completed: 'text-safe-green',
  failed: 'text-danger-pink',
  queued: 'text-muted-foreground',
};

export const JobsPage = () => {
  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Jobs</h1>
        <p className="text-muted-foreground mt-1">Monitor distributed job execution.</p>
      </div>

      <div className="space-y-4">
        {mockJobs.map((job, index) => {
          const StatusIcon = statusIcons[job.status];
          return (
            <motion.div
              key={job.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.1 }}
            >
              <GlassCard className="p-5">
                <div className="flex items-center gap-4">
                  <div className={`${statusColors[job.status]} ${job.status === 'running' ? 'animate-spin' : ''}`}>
                    <StatusIcon className="w-6 h-6" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="font-semibold">{job.name}</h3>
                      <span className="font-mono text-xs text-muted-foreground">{job.id}</span>
                    </div>
                    <div className="flex items-center gap-4 text-xs text-muted-foreground mb-2">
                      <span><Cpu className="w-3 h-3 inline mr-1" />{job.nodesCount} nodes</span>
                      <span>{job.startTime}</span>
                    </div>
                    <Progress value={job.progress} className="h-1.5" />
                  </div>
                  <div className="flex gap-2">
                    {job.tags.map(tag => (
                      <span key={tag} className="px-2 py-0.5 rounded bg-surface-variant text-xs">
                        {tag === 'AI' && <Bot className="w-3 h-3 inline mr-1" />}{tag}
                      </span>
                    ))}
                  </div>
                </div>
              </GlassCard>
            </motion.div>
          );
        })}
      </div>
    </div>
  );
};
