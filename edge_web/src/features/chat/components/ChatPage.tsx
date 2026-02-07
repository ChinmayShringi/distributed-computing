import { useState, useCallback } from 'react';
import { useToast } from '@/hooks/use-toast';
import { useChatMessages } from '../hooks/useChatMessages';
import { useJobPolling } from '../hooks/useJobPolling';
import { sendAssistantMessage, submitJob } from '../api';
import type { PlanPayload, DevicePayload, ResultPayload } from '../types';
import { WelcomeScreen } from './WelcomeScreen';
import { MessageList } from './MessageList';
import { ComposerBar } from './ComposerBar';
import { ActionSheet } from './ActionSheet';

export function ChatPage() {
  const { toast } = useToast();
  const {
    messages,
    isThinking,
    addUserMessage,
    addAssistantMessage,
    setThinking,
  } = useChatMessages();

  const [actionSheetOpen, setActionSheetOpen] = useState(false);

  const { startPolling } = useJobPolling({
    onResult: (result) => {
      if (result.state === 'DONE' && result.results?.[0]) {
        const r = result.results[0];
        addAssistantMessage('result', undefined, {
          cmd: 'Executed command',
          device: r.device_id,
          host_compute: 'CPU',
          time: `${r.elapsed_ms}ms`,
          exit_code: r.exit_code,
          output: r.stdout || r.stderr || 'No output',
        } as ResultPayload);
      } else if (result.state === 'FAILED') {
        addAssistantMessage('text', `Job failed: ${result.error || 'Unknown error'}`);
      }
    },
    onError: (error) => {
      toast({
        title: 'Error',
        description: error.message,
        variant: 'destructive',
      });
    },
  });

  const handleSendMessage = useCallback(
    async (text: string) => {
      // Add user message
      addUserMessage(text);
      setThinking(true);

      try {
        const response = await sendAssistantMessage(text);

        // Determine message type based on response
        if (response.plan) {
          // It's a plan response
          addAssistantMessage('plan', undefined, {
            steps: (response.plan as { steps?: string[] }).steps || [],
            job_id: response.job_id,
            device: (response.plan as { device?: string }).device || 'Unknown',
            policy: (response.plan as { policy?: string }).policy || 'BEST_AVAILABLE',
            risk: (response.plan as { risk?: string }).risk || 'SAFE',
            cmd: (response.plan as { cmd?: string }).cmd || '',
          } as PlanPayload);
        } else if (response.raw && Array.isArray(response.raw)) {
          // It's a device list
          addAssistantMessage('devices', undefined, {
            devices: response.raw,
          } as DevicePayload);
        } else if (response.job_id) {
          // Job was created, poll for results
          addAssistantMessage('text', response.reply);
          startPolling(response.job_id);
        } else {
          // Regular text response
          addAssistantMessage('text', response.reply);
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to send message';
        addAssistantMessage('text', `Error: ${message}`);
        toast({
          title: 'Error',
          description: message,
          variant: 'destructive',
        });
      } finally {
        setThinking(false);
      }
    },
    [addUserMessage, addAssistantMessage, setThinking, startPolling, toast]
  );

  const handleRunPlan = useCallback(
    async (payload: PlanPayload) => {
      setThinking(true);
      addAssistantMessage('text', 'Executing plan...');

      try {
        const result = await submitJob({
          plan: {
            steps: payload.steps,
            device: payload.device,
            policy: payload.policy,
            cmd: payload.cmd,
          },
        });

        if (result.job_id) {
          startPolling(result.job_id);
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to execute plan';
        addAssistantMessage('text', `Error: ${message}`);
        toast({
          title: 'Error',
          description: message,
          variant: 'destructive',
        });
        setThinking(false);
      }
    },
    [addAssistantMessage, setThinking, startPolling, toast]
  );

  const handleActionSheetAction = useCallback(
    (message: string) => {
      handleSendMessage(message);
    },
    [handleSendMessage]
  );

  const handleSuggestionClick = useCallback(
    (text: string) => {
      handleSendMessage(text);
    },
    [handleSendMessage]
  );

  return (
    <div className="flex flex-col h-full bg-background">
      {messages.length === 0 ? (
        <WelcomeScreen onSuggestionClick={handleSuggestionClick} />
      ) : (
        <MessageList
          messages={messages}
          isThinking={isThinking}
          onRunPlan={handleRunPlan}
        />
      )}

      <ComposerBar
        onSend={handleSendMessage}
        onOpenActions={() => setActionSheetOpen(true)}
        disabled={isThinking}
      />

      <ActionSheet
        open={actionSheetOpen}
        onOpenChange={setActionSheetOpen}
        onAction={handleActionSheetAction}
      />
    </div>
  );
}
