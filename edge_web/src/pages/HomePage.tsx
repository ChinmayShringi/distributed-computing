import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { Button } from '@/components/ui/button';
import { GlassCard } from '@/components/GlassCard';
import { 
  Zap, 
  Shield, 
  Cpu, 
  Globe, 
  Lock, 
  Activity,
  ChevronRight,
  ArrowRight,
  Server,
  Layers,
  FileText,
  Check
} from 'lucide-react';

const features = [
  {
    icon: Zap,
    title: 'Instant Deployment',
    description: 'Deploy workloads to edge nodes in milliseconds with zero-config setup.',
  },
  {
    icon: Globe,
    title: 'Global Distribution',
    description: 'Run compute across your entire device mesh, anywhere in the world.',
  },
  {
    icon: Cpu,
    title: 'Heterogeneous Compute',
    description: 'Leverage CPU, GPU, and NPU resources seamlessly across devices.',
  },
  {
    icon: Shield,
    title: 'Enterprise Security',
    description: 'mTLS encryption, RBAC policies, and comprehensive audit logging.',
  },
  {
    icon: Activity,
    title: 'Real-time Monitoring',
    description: 'Live telemetry and health metrics from every connected node.',
  },
  {
    icon: Lock,
    title: 'Zero Trust Architecture',
    description: 'Every request is authenticated and authorized at the edge.',
  },
];

const securityBadges = [
  { icon: Lock, label: 'mTLS Encryption' },
  { icon: Shield, label: 'RBAC Policies' },
  { icon: Check, label: 'App Verification' },
  { icon: FileText, label: 'Audit Logs' },
];

export const HomePage = () => {
  return (
    <div className="relative">
      {/* Hero Section */}
      <section className="relative min-h-[90vh] flex items-center justify-center overflow-hidden">
        {/* Background effects */}
        <div className="absolute inset-0 bg-grid opacity-50" />
        <div className="absolute inset-0 bg-gradient-radial from-primary/10 via-transparent to-transparent" />
        
        <div className="relative z-10 max-w-5xl mx-auto px-4 text-center">
          <motion.div
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
            className="space-y-6"
          >
            <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-surface-variant border border-outline text-sm text-muted-foreground">
              <span className="w-2 h-2 rounded-full bg-safe-green animate-pulse" />
              Now in Public Beta
            </div>
            
            <h1 className="text-5xl md:text-7xl font-bold tracking-tight">
              <span className="text-foreground">Distributed Compute</span>
              <br />
              <span className="text-primary text-glow-primary">at the Edge</span>
            </h1>
            
            <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
              Connect your devices into a powerful mesh network. Run workloads anywhere, 
              monitor everything, and scale without limits.
            </p>
            
            <div className="flex flex-col sm:flex-row items-center justify-center gap-4 pt-4">
              <Button asChild size="lg" className="bg-primary hover:bg-primary/90 text-primary-foreground px-8">
                <Link to="/console/connect">
                  Open Console
                  <ArrowRight className="ml-2 w-5 h-5" />
                </Link>
              </Button>
              <Button asChild variant="outline" size="lg">
                <Link to="/features">
                  See Features
                  <ChevronRight className="ml-2 w-5 h-5" />
                </Link>
              </Button>
            </div>
          </motion.div>
        </div>

        {/* Floating orbs */}
        <motion.div
          className="absolute top-1/4 left-10 w-64 h-64 rounded-full bg-primary/20 blur-3xl"
          animate={{ y: [0, -20, 0], opacity: [0.3, 0.5, 0.3] }}
          transition={{ duration: 6, repeat: Infinity }}
        />
        <motion.div
          className="absolute bottom-1/4 right-10 w-96 h-96 rounded-full bg-info-blue/10 blur-3xl"
          animate={{ y: [0, 20, 0], opacity: [0.2, 0.4, 0.2] }}
          transition={{ duration: 8, repeat: Infinity }}
        />
      </section>

      {/* Features Grid */}
      <section className="py-24 px-4">
        <div className="max-w-6xl mx-auto">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-16"
          >
            <h2 className="text-3xl md:text-4xl font-bold mb-4">
              Everything you need for <span className="text-primary">edge computing</span>
            </h2>
            <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
              A complete platform for managing distributed workloads across your device fleet.
            </p>
          </motion.div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {features.map((feature, index) => (
              <motion.div
                key={feature.title}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <GlassCard hover className="h-full">
                  <div className="p-2">
                    <div className="w-12 h-12 rounded-xl bg-primary/20 flex items-center justify-center mb-4">
                      <feature.icon className="w-6 h-6 text-primary" />
                    </div>
                    <h3 className="text-lg font-semibold mb-2">{feature.title}</h3>
                    <p className="text-sm text-muted-foreground">{feature.description}</p>
                  </div>
                </GlassCard>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Security Badges */}
      <section className="py-16 px-4 bg-surface-1/50">
        <div className="max-w-4xl mx-auto">
          <motion.div
            initial={{ opacity: 0 }}
            whileInView={{ opacity: 1 }}
            viewport={{ once: true }}
            className="flex flex-wrap items-center justify-center gap-6"
          >
            {securityBadges.map((badge, index) => (
              <motion.div
                key={badge.label}
                initial={{ opacity: 0, scale: 0.9 }}
                whileInView={{ opacity: 1, scale: 1 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
                className="flex items-center gap-2 px-4 py-2 rounded-lg bg-surface-variant border border-outline"
              >
                <badge.icon className="w-4 h-4 text-safe-green" />
                <span className="text-sm font-medium">{badge.label}</span>
              </motion.div>
            ))}
          </motion.div>
        </div>
      </section>

      {/* Console Preview */}
      <section className="py-24 px-4">
        <div className="max-w-6xl mx-auto">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-12"
          >
            <h2 className="text-3xl md:text-4xl font-bold mb-4">
              A <span className="text-primary">powerful console</span> for complete control
            </h2>
            <p className="text-lg text-muted-foreground">
              Monitor, manage, and execute across your entire edge infrastructure.
            </p>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 40 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="relative"
          >
            {/* Mock console preview */}
            <div className="glass-card overflow-hidden">
              {/* Header */}
              <div className="flex items-center gap-4 px-4 py-3 bg-surface-2 border-b border-outline">
                <div className="flex gap-2">
                  <div className="w-3 h-3 rounded-full bg-danger-pink" />
                  <div className="w-3 h-3 rounded-full bg-warning-amber" />
                  <div className="w-3 h-3 rounded-full bg-safe-green" />
                </div>
                <div className="flex-1 text-center">
                  <span className="text-sm font-mono text-muted-foreground">Edge Mesh Console</span>
                </div>
              </div>

              {/* Content */}
              <div className="flex">
                {/* Sidebar */}
                <div className="w-48 bg-surface-1 border-r border-outline p-4 hidden md:block">
                  <div className="space-y-2">
                    {['Dashboard', 'Devices', 'Run', 'Jobs', 'Settings'].map((item, i) => (
                      <div
                        key={item}
                        className={`px-3 py-2 rounded-lg text-sm ${
                          i === 0 ? 'bg-primary/20 text-primary' : 'text-muted-foreground'
                        }`}
                      >
                        {item}
                      </div>
                    ))}
                  </div>
                </div>

                {/* Main */}
                <div className="flex-1 p-6 min-h-[300px]">
                  <div className="grid grid-cols-3 gap-4 mb-6">
                    {[
                      { label: 'Connected Devices', value: '5', icon: Server },
                      { label: 'Active Jobs', value: '3', icon: Layers },
                      { label: 'Tools Available', value: '12', icon: Cpu },
                    ].map((stat) => (
                      <div key={stat.label} className="glass-subtle rounded-lg p-4">
                        <div className="flex items-center justify-between mb-2">
                          <span className="text-xs text-muted-foreground">{stat.label}</span>
                          <stat.icon className="w-4 h-4 text-muted-icon" />
                        </div>
                        <span className="text-2xl font-bold text-primary">{stat.value}</span>
                      </div>
                    ))}
                  </div>
                  
                  <div className="grid grid-cols-2 gap-4">
                    <div className="glass-subtle rounded-lg p-4 h-32">
                      <span className="text-xs text-muted-foreground">Compute Distribution</span>
                    </div>
                    <div className="glass-subtle rounded-lg p-4 h-32">
                      <span className="text-xs text-muted-foreground">Device Activity</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            {/* Glow effect */}
            <div className="absolute inset-0 -z-10 bg-gradient-radial from-primary/20 via-transparent to-transparent blur-2xl" />
          </motion.div>
        </div>
      </section>

      {/* Final CTA */}
      <section className="py-24 px-4">
        <div className="max-w-3xl mx-auto text-center">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="glass-panel p-12"
          >
            <h2 className="text-3xl md:text-4xl font-bold mb-4">
              Ready to build your <span className="text-primary">edge mesh</span>?
            </h2>
            <p className="text-lg text-muted-foreground mb-8">
              Start connecting devices and running distributed workloads in minutes.
            </p>
            <Button asChild size="lg" className="bg-primary hover:bg-primary/90 px-8">
              <Link to="/console/connect">
                Get Started Free
                <ArrowRight className="ml-2 w-5 h-5" />
              </Link>
            </Button>
          </motion.div>
        </div>
      </section>
    </div>
  );
};
