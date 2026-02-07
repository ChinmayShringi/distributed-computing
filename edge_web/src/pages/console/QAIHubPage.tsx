import { useState } from 'react';
import { motion } from 'framer-motion';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import {
  runQAIHubDoctor,
  compileModel,
  type QAIHubDoctorResponse,
  type QAIHubCompileResponse,
} from '@/api';
import {
  Cpu,
  Stethoscope,
  Play,
  Loader2,
  CheckCircle,
  XCircle,
  AlertCircle,
  FileCode,
  FolderOutput,
  Key,
} from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

export const QAIHubPage = () => {
  const { toast } = useToast();

  // Doctor state
  const [doctorLoading, setDoctorLoading] = useState(false);
  const [doctorResult, setDoctorResult] = useState<QAIHubDoctorResponse | null>(null);
  const [doctorError, setDoctorError] = useState<string | null>(null);

  // Compile form state
  const [onnxPath, setOnnxPath] = useState('');
  const [target, setTarget] = useState('Samsung Galaxy S24');
  const [runtime, setRuntime] = useState('');

  // Compile state
  const [compiling, setCompiling] = useState(false);
  const [compileResult, setCompileResult] = useState<QAIHubCompileResponse | null>(null);
  const [compileError, setCompileError] = useState<string | null>(null);

  // Run doctor check
  const handleDoctor = async () => {
    setDoctorLoading(true);
    setDoctorError(null);
    setDoctorResult(null);

    try {
      const result = await runQAIHubDoctor();
      setDoctorResult(result);
      toast({
        title: result.qai_hub_found ? 'QAI Hub Available' : 'QAI Hub Not Found',
        description: result.qai_hub_found
          ? `Version: ${result.qai_hub_version}`
          : 'Install qai-hub CLI to use this feature',
        variant: result.qai_hub_found ? 'default' : 'destructive',
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Doctor check failed';
      setDoctorError(message);
      toast({
        title: 'Error',
        description: message,
        variant: 'destructive',
      });
    } finally {
      setDoctorLoading(false);
    }
  };

  // Run compile
  const handleCompile = async () => {
    if (!onnxPath.trim() || !target.trim()) {
      toast({
        title: 'Error',
        description: 'Please enter ONNX path and target device',
        variant: 'destructive',
      });
      return;
    }

    setCompiling(true);
    setCompileError(null);
    setCompileResult(null);

    try {
      const result = await compileModel(
        onnxPath,
        target,
        runtime.trim() || undefined
      );
      setCompileResult(result);

      if (result.submitted) {
        toast({
          title: 'Compilation Started',
          description: `Job ID: ${result.job_id}`,
        });
      } else {
        toast({
          title: 'Compilation Failed',
          description: result.error || 'Unknown error',
          variant: 'destructive',
        });
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Compilation failed';
      setCompileError(message);
      toast({
        title: 'Error',
        description: message,
        variant: 'destructive',
      });
    } finally {
      setCompiling(false);
    }
  };

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <Cpu className="w-6 h-6 text-primary" />
          Qualcomm AI Hub
        </h1>
        <p className="text-muted-foreground mt-1">
          Compile and optimize models for Qualcomm devices.
        </p>
      </div>

      {/* Doctor Check */}
      <GlassContainer>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <Stethoscope className="w-5 h-5 text-primary" />
            Health Check
          </h2>
          <Button
            onClick={handleDoctor}
            disabled={doctorLoading}
            variant="outline"
          >
            {doctorLoading ? (
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            ) : (
              <Stethoscope className="w-4 h-4 mr-2" />
            )}
            Run Doctor
          </Button>
        </div>

        {doctorError && (
          <div className="flex items-center gap-2 p-3 rounded-lg bg-danger-pink/10 border border-danger-pink/20 text-danger-pink text-sm">
            <AlertCircle className="w-4 h-4" />
            {doctorError}
          </div>
        )}

        {doctorResult && (
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            className="space-y-4"
          >
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {/* CLI Status */}
              <GlassCard className="p-4">
                <div className="flex items-center gap-3">
                  {doctorResult.qai_hub_found ? (
                    <CheckCircle className="w-8 h-8 text-safe-green" />
                  ) : (
                    <XCircle className="w-8 h-8 text-danger-pink" />
                  )}
                  <div>
                    <p className="font-medium">QAI Hub CLI</p>
                    <p className="text-sm text-muted-foreground">
                      {doctorResult.qai_hub_found
                        ? `v${doctorResult.qai_hub_version}`
                        : 'Not installed'}
                    </p>
                  </div>
                </div>
              </GlassCard>

              {/* Token Status */}
              <GlassCard className="p-4">
                <div className="flex items-center gap-3">
                  {doctorResult.token_env_present ? (
                    <Key className="w-8 h-8 text-safe-green" />
                  ) : (
                    <Key className="w-8 h-8 text-warning-amber" />
                  )}
                  <div>
                    <p className="font-medium">API Token</p>
                    <p className="text-sm text-muted-foreground">
                      {doctorResult.token_env_present
                        ? 'Configured'
                        : 'Not configured'}
                    </p>
                  </div>
                </div>
              </GlassCard>

              {/* Notes */}
              {doctorResult.notes && doctorResult.notes.length > 0 && (
                <GlassCard className="p-4 md:col-span-2 lg:col-span-1">
                  <div className="flex items-start gap-3">
                    <AlertCircle className="w-5 h-5 text-info-blue flex-shrink-0 mt-0.5" />
                    <div>
                      <p className="font-medium mb-1">Notes</p>
                      <ul className="text-sm text-muted-foreground space-y-1">
                        {doctorResult.notes.map((note, i) => (
                          <li key={i}>{note}</li>
                        ))}
                      </ul>
                    </div>
                  </div>
                </GlassCard>
              )}
            </div>
          </motion.div>
        )}
      </GlassContainer>

      {/* Compile Model */}
      <GlassContainer>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <FileCode className="w-5 h-5 text-primary" />
          Compile Model
        </h2>

        <div className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="onnxPath">ONNX Model Path</Label>
              <Input
                id="onnxPath"
                value={onnxPath}
                onChange={(e) => setOnnxPath(e.target.value)}
                placeholder="/path/to/model.onnx"
                className="font-mono bg-surface-2 border-outline"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="target">Target Device</Label>
              <Input
                id="target"
                value={target}
                onChange={(e) => setTarget(e.target.value)}
                placeholder="Samsung Galaxy S24"
                className="bg-surface-2 border-outline"
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="runtime">Runtime (optional)</Label>
            <Input
              id="runtime"
              value={runtime}
              onChange={(e) => setRuntime(e.target.value)}
              placeholder="e.g., precompiled_qnn_onnx"
              className="font-mono bg-surface-2 border-outline"
            />
          </div>

          <Button
            onClick={handleCompile}
            disabled={!onnxPath.trim() || !target.trim() || compiling}
            className="bg-safe-green hover:bg-safe-green/90 text-background"
          >
            {compiling ? (
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            ) : (
              <Play className="w-4 h-4 mr-2" />
            )}
            Compile Model
          </Button>

          {compileError && (
            <div className="flex items-center gap-2 p-3 rounded-lg bg-danger-pink/10 border border-danger-pink/20 text-danger-pink text-sm">
              <AlertCircle className="w-4 h-4" />
              {compileError}
            </div>
          )}

          {compileResult && (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
            >
              <GlassCard className="p-4">
                <div className="flex items-center gap-3 mb-4">
                  {compileResult.submitted ? (
                    <CheckCircle className="w-6 h-6 text-safe-green" />
                  ) : (
                    <XCircle className="w-6 h-6 text-danger-pink" />
                  )}
                  <div>
                    <p className="font-medium">
                      {compileResult.submitted
                        ? 'Compilation Submitted'
                        : 'Compilation Failed'}
                    </p>
                    {compileResult.job_id && (
                      <Badge variant="outline" className="font-mono text-xs mt-1">
                        Job: {compileResult.job_id}
                      </Badge>
                    )}
                  </div>
                </div>

                {compileResult.out_dir && (
                  <div className="flex items-center gap-2 text-sm mb-2">
                    <FolderOutput className="w-4 h-4 text-muted-foreground" />
                    <span className="text-muted-foreground">Output:</span>
                    <span className="font-mono">{compileResult.out_dir}</span>
                  </div>
                )}

                {compileResult.raw_output_path && (
                  <div className="flex items-center gap-2 text-sm mb-2">
                    <FileCode className="w-4 h-4 text-muted-foreground" />
                    <span className="text-muted-foreground">Log:</span>
                    <span className="font-mono text-xs">{compileResult.raw_output_path}</span>
                  </div>
                )}

                {compileResult.notes && compileResult.notes.length > 0 && (
                  <div className="mt-3 pt-3 border-t border-outline">
                    <p className="text-sm text-muted-foreground mb-1">Notes:</p>
                    <ul className="text-sm space-y-1">
                      {compileResult.notes.map((note, i) => (
                        <li key={i}>{note}</li>
                      ))}
                    </ul>
                  </div>
                )}

                {compileResult.error && (
                  <div className="mt-3 pt-3 border-t border-outline">
                    <p className="text-sm text-danger-pink">{compileResult.error}</p>
                  </div>
                )}
              </GlassCard>
            </motion.div>
          )}
        </div>
      </GlassContainer>

      {/* Info */}
      <GlassCard className="p-4">
        <div className="flex items-start gap-3">
          <AlertCircle className="w-5 h-5 text-info-blue flex-shrink-0 mt-0.5" />
          <div className="text-sm text-muted-foreground">
            <p className="font-medium text-foreground mb-1">About Qualcomm AI Hub</p>
            <ul className="list-disc list-inside space-y-1">
              <li>Requires qai-hub CLI to be installed on the server</li>
              <li>API token must be configured via QAI_HUB_TOKEN environment variable</li>
              <li>Supports ONNX models for compilation to Qualcomm NPU</li>
              <li>Target devices include Samsung Galaxy S24, Google Pixel 8, and more</li>
            </ul>
          </div>
        </div>
      </GlassCard>
    </div>
  );
};
