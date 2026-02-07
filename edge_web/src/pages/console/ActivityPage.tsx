import { motion } from 'framer-motion';
import { GlassCard } from '@/components/GlassCard';
import { CircularProgress } from '@/components/CircularProgress';
import { MetricsChart } from '@/components/MetricsChart';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useActivityPolling } from '@/hooks/useActivityPolling';
import {
  Play,
  Pause,
  RefreshCw,
  Activity,
  Cpu,
  HardDrive,
  Zap,
  Clock,
  AlertCircle,
} from 'lucide-react';

const taskStateBadgeVariant = (state: string) => {
  switch (state) {
    case 'RUNNING':
      return 'default';
    case 'DONE':
      return 'outline';
    case 'FAILED':
      return 'destructive';
    case 'QUEUED':
      return 'secondary';
    default:
      return 'outline';
  }
};

export const ActivityPage = () => {
  const {
    data,
    error,
    loading,
    isPolling,
    togglePolling,
    refresh,
  } = useActivityPolling({ enabled: false, includeMetricsHistory: true });

  const runningTasks = data?.activity?.running_tasks || [];
  const deviceActivities = (data?.activity?.device_activities || [])
    .slice()
    .sort((a, b) => a.device_id.localeCompare(b.device_id));
  const deviceMetrics = data?.device_metrics || {};

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Activity className="w-6 h-6 text-primary" />
            Activity Monitor
          </h1>
          <p className="text-muted-foreground mt-1">
            Real-time device metrics and running tasks
          </p>
        </div>

        <div className="flex items-center gap-3">
          <Button
            onClick={togglePolling}
            variant={isPolling ? 'destructive' : 'default'}
            className="gap-2"
          >
            {isPolling ? (
              <>
                <Pause className="w-4 h-4" />
                Stop Polling
              </>
            ) : (
              <>
                <Play className="w-4 h-4" />
                Start Polling
              </>
            )}
          </Button>
          <Button
            variant="outline"
            size="icon"
            onClick={refresh}
            disabled={loading}
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </Button>
        </div>
      </div>

      {/* Polling Status */}
      {isPolling && (
        <div className="flex items-center gap-2 text-sm text-safe-green">
          <span className="relative flex h-2 w-2">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-safe-green opacity-75" />
            <span className="relative inline-flex rounded-full h-2 w-2 bg-safe-green" />
          </span>
          Polling every 2 seconds...
        </div>
      )}

      {/* Error State */}
      {error && (
        <div className="flex items-center gap-3 p-4 rounded-lg bg-danger-pink/10 border border-danger-pink/20">
          <AlertCircle className="w-5 h-5 text-danger-pink" />
          <span className="text-danger-pink">{error}</span>
          <Button variant="outline" size="sm" onClick={refresh} className="ml-auto">
            Retry
          </Button>
        </div>
      )}

      {/* Device Resources */}
      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-4">
        {deviceActivities.map((activity, index) => (
          <motion.div
            key={activity.device_id}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.05 }}
          >
            <GlassCard className="p-5">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <h3 className="font-semibold">{activity.device_name}</h3>
                  <p className="text-xs text-muted-foreground font-mono">
                    {activity.device_id.slice(0, 12)}...
                  </p>
                </div>
                {(activity.running_task_count ?? 0) > 0 && (
                  <Badge variant="default" className="bg-safe-green">
                    {activity.running_task_count} running
                  </Badge>
                )}
              </div>

              <div className="flex items-center justify-around relative">
                <CircularProgress
                  value={Math.max(0, (activity.current_status?.cpu_load ?? 0) * 100)}
                  label="CPU"
                  variant="cpu"
                  size={70}
                  strokeWidth={6}
                />
                <CircularProgress
                  value={
                    activity.current_status?.mem_total_mb && activity.current_status?.mem_used_mb
                      ? (activity.current_status.mem_used_mb / activity.current_status.mem_total_mb) * 100
                      : 0
                  }
                  label="Memory"
                  variant="memory"
                  size={70}
                  strokeWidth={6}
                />
                <CircularProgress
                  value={
                    Math.max(
                      0,
                      Math.max(
                        activity.current_status?.gpu_load ?? -1,
                        activity.current_status?.npu_load ?? -1
                      ) * 100
                    )
                  }
                  label="GPU/NPU"
                  variant="gpu"
                  size={70}
                  strokeWidth={6}
                />
              </div>
            </GlassCard>
          </motion.div>
        ))}

        {deviceActivities.length === 0 && !loading && (
          <div className="col-span-full text-center py-8">
            <p className="text-muted-foreground">
              {isPolling
                ? 'Waiting for device data...'
                : 'Start polling to see device activity'}
            </p>
          </div>
        )}
      </div>

      {/* Running Tasks */}
      <GlassCard className="p-5">
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Clock className="w-5 h-5 text-primary" />
          Running Tasks
          {runningTasks.length > 0 && (
            <Badge variant="outline">{runningTasks.length}</Badge>
          )}
        </h2>

        {runningTasks.length > 0 ? (
          <div className="space-y-3">
            {runningTasks.map((task, index) => (
              <motion.div
                key={task.task_id}
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: index * 0.05 }}
                className={`p-4 rounded-lg border border-outline bg-surface-2 ${task.state === 'RUNNING' ? 'animate-pulse' : ''
                  }`}
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-medium">{task.kind}</span>
                      <Badge variant={taskStateBadgeVariant(task.state)}>
                        {task.state}
                      </Badge>
                    </div>
                    <p className="text-sm text-muted-foreground truncate">
                      {task.input}
                    </p>
                    <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
                      <span>Device: {task.device_name}</span>
                      <span>
                        Job: {task.job_id.slice(0, 8)}...
                      </span>
                    </div>
                  </div>
                  <div className="text-right text-sm">
                    <span className="font-mono text-warning-amber">
                      {(task.elapsed_ms / 1000).toFixed(1)}s
                    </span>
                  </div>
                </div>
              </motion.div>
            ))}
          </div>
        ) : (
          <p className="text-muted-foreground text-center py-6">
            No running tasks
          </p>
        )}
      </GlassCard>

      {/* Metrics Charts */}
      <GlassCard className="p-5">
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Zap className="w-5 h-5 text-primary" />
          Metrics History (120s)
        </h2>

        <Tabs defaultValue="cpu" className="w-full">
          <TabsList className="mb-4">
            <TabsTrigger value="cpu" className="gap-1">
              <Cpu className="w-3 h-3" />
              CPU
            </TabsTrigger>
            <TabsTrigger value="memory" className="gap-1">
              <HardDrive className="w-3 h-3" />
              Memory
            </TabsTrigger>
            <TabsTrigger value="gpu" className="gap-1">
              <Zap className="w-3 h-3" />
              GPU/NPU
            </TabsTrigger>
          </TabsList>

          <TabsContent value="cpu">
            <MetricsChart data={deviceMetrics} metric="cpu" height={250} />
          </TabsContent>

          <TabsContent value="memory">
            <MetricsChart data={deviceMetrics} metric="memory" height={250} />
          </TabsContent>

          <TabsContent value="gpu">
            <MetricsChart data={deviceMetrics} metric="gpu" height={250} />
          </TabsContent>
        </Tabs>
      </GlassCard>
    </div>
  );
};
