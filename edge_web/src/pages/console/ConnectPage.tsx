import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { GlassCard } from '@/components/GlassCard';
import { EdgeMeshWordmark } from '@/components/EdgeMeshWordmark';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { connect } from '@/lib/connection';
import { Server, Key, Eye, EyeOff, Shield, Lock, Check } from 'lucide-react';

export const ConnectPage = () => {
  const navigate = useNavigate();
  const [address, setAddress] = useState('192.168.1.10:50051');
  const [signature, setSignature] = useState('dev');
  const [showSignature, setShowSignature] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleConnect = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    
    // Simulate connection delay
    await new Promise(resolve => setTimeout(resolve, 1500));
    
    connect(address);
    // Force a storage event for same-tab detection
    window.dispatchEvent(new Event('storage'));
    navigate('/console/dashboard');
  };

  const badges = [
    { icon: Check, label: 'App Check' },
    { icon: Shield, label: 'RBAC' },
    { icon: Lock, label: 'mTLS' },
  ];

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      {/* Background effects */}
      <div className="fixed inset-0 bg-gradient-radial from-primary/5 via-transparent to-transparent" />
      
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.4 }}
        className="w-full max-w-md relative z-10"
      >
        <GlassCard glow="primary" className="p-8">
          <div className="text-center mb-8">
            <EdgeMeshWordmark size="xl" animated />
            <p className="mt-4 text-muted-foreground">
              Connect to your edge gateway
            </p>
          </div>

          <form onSubmit={handleConnect} className="space-y-6">
            {/* Gateway Address */}
            <div className="space-y-2">
              <Label htmlFor="address" className="flex items-center gap-2">
                <Server className="w-4 h-4" />
                Gateway Address
              </Label>
              <Input
                id="address"
                type="text"
                value={address}
                onChange={(e) => setAddress(e.target.value)}
                placeholder="192.168.1.10:50051"
                className="font-mono bg-surface-2 border-outline"
              />
            </div>

            {/* Access Signature */}
            <div className="space-y-2">
              <Label htmlFor="signature" className="flex items-center gap-2">
                <Key className="w-4 h-4" />
                Access Signature
              </Label>
              <div className="relative">
                <Input
                  id="signature"
                  type={showSignature ? 'text' : 'password'}
                  value={signature}
                  onChange={(e) => setSignature(e.target.value)}
                  placeholder="Enter signature"
                  className="font-mono bg-surface-2 border-outline pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowSignature(!showSignature)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  {showSignature ? (
                    <EyeOff className="w-4 h-4" />
                  ) : (
                    <Eye className="w-4 h-4" />
                  )}
                </button>
              </div>
            </div>

            {/* Connect Button */}
            <Button
              type="submit"
              disabled={loading || !address || !signature}
              className="w-full bg-safe-green hover:bg-safe-green/90 text-background font-semibold"
            >
              {loading ? (
                <motion.div
                  className="w-5 h-5 border-2 border-background border-t-transparent rounded-full"
                  animate={{ rotate: 360 }}
                  transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                />
              ) : (
                'Connect'
              )}
            </Button>
          </form>

          {/* Security Badges */}
          <div className="mt-8 pt-6 border-t border-outline">
            <div className="flex items-center justify-center gap-4">
              {badges.map((badge) => (
                <div
                  key={badge.label}
                  className="flex items-center gap-1.5 text-xs text-muted-foreground"
                >
                  <badge.icon className="w-3.5 h-3.5 text-safe-green" />
                  <span>{badge.label}</span>
                </div>
              ))}
            </div>
          </div>
        </GlassCard>
      </motion.div>
    </div>
  );
};
