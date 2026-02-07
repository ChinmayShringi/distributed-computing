import { useState, useEffect, useCallback } from 'react';
import { motion } from 'framer-motion';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { Progress } from '@/components/ui/progress';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  submitJob,
  getJobStatus,
  getJobDetail,
  previewPlan,
  type JobStatusResponse,
  type JobDetailResponse,
  type PlanPreviewResponse,
} from '@/api';
import {
  CheckCircle,
  XCircle,
  Loader2,
  Clock,
  Bot,
  Cpu,
  Play,
  Eye,
  RefreshCw,
  ChevronDown,
  FileText,
  AlertCircle,
} from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

const stateColors: Record<string, string> = {
  SUBMITTED: 'text-info-blue',
  QUEUED: 'text-muted-foreground',
  RUNNING: 'text-warning-amber',
  DONE: 'text-safe-green',
  FAILED: 'text-danger-pink',
  PENDING: 'text-muted-foreground',
};

const stateIcons: Record<string, typeof Loader2> = {
  SUBMITTED: Clock,
  QUEUED: Clock,
  RUNNING: Loader2,
  DONE: CheckCircle,
  FAILED: XCircle,
  PENDING: Clock,
};

export const JobsPage = () => {
  const { toast } = useToast();

  // Form state
  const [jobText, setJobText] = useState('collect status');
  const [maxWorkers, setMaxWorkers] = useState(0);

  // Job state
  const [submitting, setSubmitting] = useState(false);
  const [currentJobId, setCurrentJobId] = useState<string | null>(null);
  const [jobStatus, setJobStatus] = useState<JobStatusResponse | null>(null);
  const [jobDetail, setJobDetail] = useState<JobDetailResponse | null>(null);
  const [polling, setPolling] = useState(false);

  // Plan preview state
  const [previewing, setPreviewing] = useState(false);
  const [planPreview, setPlanPreview] = useState<PlanPreviewResponse | null>(null);
  const [planOpen, setPlanOpen] = useState(false);

  // Error state
  const [error, setError] = useState<string | null>(null);

  // Poll job status
  const pollJob = useCallback(async (jobId: string) => {
    try {
      const status = await getJobStatus(jobId);
      setJobStatus(status);

      // Stop polling if job is done
      if (status.state === 'DONE' || status.state === 'FAILED') {
        setPolling(false);
        // Fetch detailed info
        try {
          const detail = await getJobDetail(jobId);
          setJobDetail(detail);
        } catch (err) {
          console.error('Failed to fetch job detail:', err);
        }
      }
    } catch (err) {
      console.error('Failed to poll job:', err);
    }
  }, []);

  // Polling effect
  useEffect(() => {
    if (!polling || !currentJobId) return;

    const interval = setInterval(() => {
      pollJob(currentJobId);
    }, 500);

    return () => clearInterval(interval);
  }, [polling, currentJobId, pollJob]);

  // Handle job submission
  const handleSubmit = async () => {
    if (!jobText.trim()) {
      toast({
        title: 'Error',
        description: 'Please enter a job description',
        variant: 'destructive',
      });
      return;
    }

    setSubmitting(true);
    setError(null);
    setJobStatus(null);
    setJobDetail(null);
    setPlanPreview(null);

    try {
      const response = await submitJob(jobText, maxWorkers > 0 ? maxWorkers : undefined);
      setCurrentJobId(response.job_id);
      setPolling(true);

      toast({
        title: 'Job Submitted',
        description: `Job ${response.job_id.slice(0, 8)}... created`,
      });

      // Initial poll
      await pollJob(response.job_id);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to submit job';
      setError(message);
      toast({
        title: 'Submission Failed',
        description: message,
        variant: 'destructive',
      });
    } finally {
      setSubmitting(false);
    }
  };

  // Handle plan preview
  const handlePreview = async () => {
    if (!jobText.trim()) {
      toast({
        title: 'Error',
        description: 'Please enter a job description',
        variant: 'destructive',
      });
      return;
    }

    setPreviewing(true);
    setError(null);
    setPlanPreview(null);

    try {
      const response = await previewPlan(jobText, maxWorkers > 0 ? maxWorkers : undefined);
      setPlanPreview(response);
      setPlanOpen(true);

      toast({
        title: 'Plan Generated',
        description: response.used_ai ? 'AI-generated plan' : 'Default plan',
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to preview plan';
      setError(message);
      toast({
        title: 'Preview Failed',
        description: message,
        variant: 'destructive',
      });
    } finally {
      setPreviewing(false);
    }
  };

  // Refresh job status
  const handleRefresh = async () => {
    if (!currentJobId) return;
    await pollJob(currentJobId);
    try {
      const detail = await getJobDetail(currentJobId);
      setJobDetail(detail);
    } catch (err) {
      console.error('Failed to refresh job detail:', err);
    }
  };

  // Calculate progress
  const getProgress = () => {
    if (!jobDetail?.tasks?.length) return 0;
    const done = jobDetail.tasks.filter(t => t.state === 'DONE' || t.state === 'FAILED').length;
    return Math.round((done / jobDetail.tasks.length) * 100);
  };

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Jobs</h1>
        <p className="text-muted-foreground mt-1">
          Submit and monitor distributed job execution.
        </p>
      </div>

      <Tabs defaultValue="submit" className="space-y-6">
        <TabsList className="bg-surface-2">
          <TabsTrigger value="submit">
            <Play className="w-4 h-4 mr-2" />
            Submit Job
          </TabsTrigger>
          <TabsTrigger value="monitor" disabled={!currentJobId}>
            <Eye className="w-4 h-4 mr-2" />
            Monitor
            {polling && (
              <span className="ml-2 relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-warning-amber opacity-75" />
                <span className="relative inline-flex rounded-full h-2 w-2 bg-warning-amber" />
              </span>
            )}
          </TabsTrigger>
        </TabsList>

        {/* Submit Tab */}
        <TabsContent value="submit" className="space-y-4">
          <GlassContainer>
            <div className="space-y-6">
              {/* Job Form */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="md:col-span-2 space-y-2">
                  <Label htmlFor="jobText">Job Description</Label>
                  <Input
                    id="jobText"
                    value={jobText}
                    onChange={(e) => setJobText(e.target.value)}
                    placeholder="e.g., collect status, run tests"
                    className="bg-surface-2 border-outline"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="maxWorkers">Max Workers (0 = all)</Label>
                  <Input
                    id="maxWorkers"
                    type="number"
                    min={0}
                    value={maxWorkers}
                    onChange={(e) => setMaxWorkers(parseInt(e.target.value) || 0)}
                    className="bg-surface-2 border-outline"
                  />
                </div>
              </div>

              {/* Action Buttons */}
              <div className="flex gap-3">
                <Button
                  onClick={handleSubmit}
                  disabled={!jobText.trim() || submitting}
                  className="bg-safe-green hover:bg-safe-green/90 text-background"
                >
                  {submitting ? (
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  ) : (
                    <Play className="w-4 h-4 mr-2" />
                  )}
                  Submit Job
                </Button>
                <Button
                  onClick={handlePreview}
                  disabled={!jobText.trim() || previewing}
                  variant="outline"
                >
                  {previewing ? (
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  ) : (
                    <Eye className="w-4 h-4 mr-2" />
                  )}
                  Preview Plan
                </Button>
              </div>

              {/* Error */}
              {error && (
                <div className="flex items-center gap-2 p-4 rounded-lg bg-danger-pink/10 border border-danger-pink/20 text-danger-pink">
                  <AlertCircle className="w-5 h-5" />
                  <span>{error}</span>
                </div>
              )}

              {/* Plan Preview */}
              {planPreview && (
                <Collapsible open={planOpen} onOpenChange={setPlanOpen}>
                  <CollapsibleTrigger asChild>
                    <Button variant="ghost" className="w-full justify-between">
                      <div className="flex items-center gap-2">
                        <FileText className="w-4 h-4" />
                        <span>Execution Plan</span>
                        {planPreview.used_ai && (
                          <Badge variant="secondary" className="gap-1">
                            <Bot className="w-3 h-3" />
                            AI
                          </Badge>
                        )}
                      </div>
                      <ChevronDown className={`w-4 h-4 transition-transform ${planOpen ? 'rotate-180' : ''}`} />
                    </Button>
                  </CollapsibleTrigger>
                  <CollapsibleContent className="mt-2">
                    <div className="p-4 rounded-lg bg-surface-2 border border-outline space-y-3">
                      {planPreview.rationale && (
                        <div>
                          <span className="text-sm text-muted-foreground">Rationale:</span>
                          <p className="text-sm mt-1">{planPreview.rationale}</p>
                        </div>
                      )}
                      {planPreview.notes?.length > 0 && (
                        <div>
                          <span className="text-sm text-muted-foreground">Notes:</span>
                          <ul className="list-disc list-inside text-sm mt-1">
                            {planPreview.notes.map((note, i) => (
                              <li key={i}>{note}</li>
                            ))}
                          </ul>
                        </div>
                      )}
                      <div>
                        <span className="text-sm text-muted-foreground">Plan JSON:</span>
                        <pre className="mt-1 p-3 rounded bg-background text-xs overflow-auto max-h-[300px] font-mono">
                          {JSON.stringify(planPreview.plan, null, 2)}
                        </pre>
                      </div>
                    </div>
                  </CollapsibleContent>
                </Collapsible>
              )}
            </div>
          </GlassContainer>
        </TabsContent>

        {/* Monitor Tab */}
        <TabsContent value="monitor" className="space-y-4">
          {currentJobId && (
            <>
              {/* Job Status Card */}
              <GlassCard className="p-5">
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center gap-3">
                    {jobStatus && (
                      <>
                        {(() => {
                          const Icon = stateIcons[jobStatus.state] || Clock;
                          return (
                            <Icon
                              className={`w-6 h-6 ${stateColors[jobStatus.state]} ${
                                jobStatus.state === 'RUNNING' ? 'animate-spin' : ''
                              }`}
                            />
                          );
                        })()}
                        <div>
                          <h3 className="font-semibold font-mono">
                            {currentJobId.slice(0, 12)}...
                          </h3>
                          <Badge
                            variant={jobStatus.state === 'DONE' ? 'default' : 'outline'}
                            className={stateColors[jobStatus.state]}
                          >
                            {jobStatus.state}
                          </Badge>
                        </div>
                      </>
                    )}
                  </div>
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={handleRefresh}
                  >
                    <RefreshCw className="w-4 h-4" />
                  </Button>
                </div>

                {/* Progress Bar */}
                <Progress value={getProgress()} className="h-2 mb-2" />
                <p className="text-xs text-muted-foreground text-right">
                  {getProgress()}% complete
                </p>

                {/* Final Result */}
                {jobStatus?.final_result && (
                  <div className="mt-4 p-3 rounded-lg bg-surface-2 border border-outline">
                    <span className="text-sm text-muted-foreground">Result:</span>
                    <pre className="mt-1 text-sm font-mono whitespace-pre-wrap">
                      {jobStatus.final_result}
                    </pre>
                  </div>
                )}

                {/* Error */}
                {jobStatus?.error && (
                  <div className="mt-4 p-3 rounded-lg bg-danger-pink/10 border border-danger-pink/20">
                    <span className="text-sm text-danger-pink">Error:</span>
                    <p className="mt-1 text-sm text-danger-pink">{jobStatus.error}</p>
                  </div>
                )}
              </GlassCard>

              {/* Task Visualization Table */}
              {jobDetail?.tasks && jobDetail.tasks.length > 0 && (
                <GlassCard className="p-5">
                  <h3 className="font-semibold mb-4 flex items-center gap-2">
                    <Cpu className="w-4 h-4 text-primary" />
                    Task Details
                    <Badge variant="outline">{jobDetail.tasks.length} tasks</Badge>
                  </h3>

                  <div className="overflow-x-auto">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead className="w-[60px]">Group</TableHead>
                          <TableHead>Device</TableHead>
                          <TableHead>Kind</TableHead>
                          <TableHead>State</TableHead>
                          <TableHead>Input</TableHead>
                          <TableHead>Output / Error</TableHead>
                          <TableHead className="text-right">Time</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {jobDetail.tasks.map((task) => {
                          const Icon = stateIcons[task.state] || Clock;
                          return (
                            <TableRow key={task.task_id}>
                              <TableCell className="font-mono text-xs">
                                {task.group_index}
                              </TableCell>
                              <TableCell className="text-sm">
                                {task.assigned_device_name || '-'}
                              </TableCell>
                              <TableCell>
                                <Badge variant="outline">{task.kind}</Badge>
                              </TableCell>
                              <TableCell>
                                <div className="flex items-center gap-1">
                                  <Icon
                                    className={`w-4 h-4 ${stateColors[task.state]} ${
                                      task.state === 'RUNNING' ? 'animate-spin' : ''
                                    }`}
                                  />
                                  <span className={`text-xs ${stateColors[task.state]}`}>
                                    {task.state}
                                  </span>
                                </div>
                              </TableCell>
                              <TableCell className="max-w-[150px] truncate font-mono text-xs">
                                {task.input}
                              </TableCell>
                              <TableCell className="max-w-[200px]">
                                {task.error ? (
                                  <span className="text-xs text-danger-pink truncate block">
                                    {task.error}
                                  </span>
                                ) : task.result ? (
                                  <span className="text-xs font-mono truncate block">
                                    {task.result.slice(0, 50)}...
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell className="text-right font-mono text-xs">
                                {task.elapsed_ms ? `${task.elapsed_ms}ms` : '-'}
                              </TableCell>
                            </TableRow>
                          );
                        })}
                      </TableBody>
                    </Table>
                  </div>
                </GlassCard>
              )}
            </>
          )}

          {!currentJobId && (
            <div className="text-center py-12 text-muted-foreground">
              Submit a job to start monitoring
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
};
