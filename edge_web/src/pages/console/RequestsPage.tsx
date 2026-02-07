import { useState } from 'react';
import { motion } from 'framer-motion';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { RiskBadge } from '@/components/RiskBadge';
import { Button } from '@/components/ui/button';
import { mockPendingRequests, PendingRequest } from '@/lib/mock-data';
import { Check, X, Monitor, Clock } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

export const RequestsPage = () => {
  const { toast } = useToast();
  const [requests, setRequests] = useState<PendingRequest[]>(mockPendingRequests);

  const handleAllow = (id: string) => {
    setRequests(requests.filter(r => r.id !== id));
    toast({
      title: 'Request Approved',
      description: 'The request has been allowed.',
    });
  };

  const handleDeny = (id: string) => {
    setRequests(requests.filter(r => r.id !== id));
    toast({
      title: 'Request Denied',
      description: 'The request has been denied.',
      variant: 'destructive',
    });
  };

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Pending Requests</h1>
        <p className="text-muted-foreground mt-1">
          Review and manage access requests from your devices.
        </p>
      </div>

      {requests.length === 0 ? (
        <GlassContainer className="text-center py-12">
          <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-surface-variant flex items-center justify-center">
            <Check className="w-8 h-8 text-safe-green" />
          </div>
          <h3 className="text-lg font-semibold mb-2">All Clear</h3>
          <p className="text-muted-foreground">No pending requests to review.</p>
        </GlassContainer>
      ) : (
        <div className="space-y-4">
          {requests.map((request, index) => (
            <motion.div
              key={request.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.1 }}
            >
              <GlassCard className="p-5">
                <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                  <div className="flex-1 space-y-2">
                    <div className="flex items-center gap-3">
                      <RiskBadge level={request.riskLevel} />
                      <span className="font-semibold">{request.action}</span>
                    </div>
                    <p className="text-sm text-muted-foreground">
                      {request.description}
                    </p>
                    <div className="flex items-center gap-4 text-xs text-muted-foreground">
                      <div className="flex items-center gap-1">
                        <Monitor className="w-3 h-3" />
                        {request.deviceName}
                      </div>
                      <div className="flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        {request.requestedAt}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-3">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleDeny(request.id)}
                      className="border-danger-pink/50 text-danger-pink hover:bg-danger-pink/20"
                    >
                      <X className="w-4 h-4 mr-1" />
                      Deny
                    </Button>
                    <Button
                      size="sm"
                      onClick={() => handleAllow(request.id)}
                      className="bg-safe-green hover:bg-safe-green/90 text-background"
                    >
                      <Check className="w-4 h-4 mr-1" />
                      Allow
                    </Button>
                  </div>
                </div>
              </GlassCard>
            </motion.div>
          ))}
        </div>
      )}
    </div>
  );
};
