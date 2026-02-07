import { useState } from 'react';
import { Link } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { GlassCard, GlassContainer } from '@/components/GlassCard';
import { MetricCard } from '@/components/MetricCard';
import { TerminalPanel } from '@/components/TerminalPanel';
import { MiniGauge } from '@/components/MiniGauge';
import {
  Monitor,
  Wrench,
  Briefcase,
  ChevronDown,
  ChevronRight,
  FileText,
  Eye,
  Download,
  Clock,
} from 'lucide-react';
import {
  mockExecutions,
  getDeviceStats,
  getComputeDistribution,
  getDeviceRunDistribution,
} from '@/lib/mock-data';
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from 'recharts';

export const DashboardPage = () => {
  const stats = getDeviceStats();
  const computeData = getComputeDistribution();
  const deviceData = getDeviceRunDistribution();
  const [expandedExecution, setExpandedExecution] = useState<string | null>(null);

  const quickActions = [
    { icon: FileText, label: 'Requests', href: '/console/dashboard/requests', count: 3 },
    { icon: Eye, label: 'Live View', href: '/console/dashboard/live' },
    { icon: Download, label: 'Downloads', href: '/console/dashboard/downloads' },
  ];

  return (
    <div className="p-6 space-y-6">
      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <MetricCard
          title="Connected Devices"
          value={stats.connectedDevices}
          icon={Monitor}
          variant="primary"
          trend="up"
          trendValue={`${stats.totalDevices} total`}
        />
        <MetricCard
          title="Tools Available"
          value={stats.toolsAvailable}
          icon={Wrench}
          variant="info"
        />
        <MetricCard
          title="Active Jobs"
          value={stats.activeJobs}
          icon={Briefcase}
          variant="safe"
        />
      </div>

      {/* Usage Overview */}
      <GlassContainer>
        <h2 className="text-lg font-semibold mb-6">Usage Overview</h2>
        
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          {/* Charts */}
          <div className="grid grid-cols-2 gap-6">
            {/* Device Distribution */}
            <div className="space-y-2">
              <h3 className="text-sm text-muted-foreground text-center">Runs by Device</h3>
              <div className="h-40">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={deviceData}
                      cx="50%"
                      cy="50%"
                      innerRadius={40}
                      outerRadius={60}
                      dataKey="value"
                      strokeWidth={0}
                    >
                      {deviceData.map((entry, index) => (
                        <Cell key={index} fill={entry.color} />
                      ))}
                    </Pie>
                    <Tooltip
                      contentStyle={{
                        backgroundColor: 'hsl(220, 18%, 8%)',
                        border: '1px solid hsl(215, 14%, 21%)',
                        borderRadius: '8px',
                      }}
                    />
                  </PieChart>
                </ResponsiveContainer>
              </div>
              <div className="flex flex-wrap justify-center gap-2">
                {deviceData.map((item) => (
                  <div key={item.name} className="flex items-center gap-1.5 text-xs">
                    <div
                      className="w-2 h-2 rounded-full"
                      style={{ backgroundColor: item.color }}
                    />
                    <span className="text-muted-foreground truncate max-w-[80px]">
                      {item.name.split(' ')[0]}
                    </span>
                  </div>
                ))}
              </div>
            </div>

            {/* Compute Distribution */}
            <div className="space-y-2">
              <h3 className="text-sm text-muted-foreground text-center">Compute Used</h3>
              <div className="h-40">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={computeData}
                      cx="50%"
                      cy="50%"
                      innerRadius={40}
                      outerRadius={60}
                      dataKey="value"
                      strokeWidth={0}
                    >
                      {computeData.map((entry, index) => (
                        <Cell key={index} fill={entry.color} />
                      ))}
                    </Pie>
                    <Tooltip
                      contentStyle={{
                        backgroundColor: 'hsl(220, 18%, 8%)',
                        border: '1px solid hsl(215, 14%, 21%)',
                        borderRadius: '8px',
                      }}
                    />
                  </PieChart>
                </ResponsiveContainer>
              </div>
              <div className="flex justify-center gap-4">
                {computeData.map((item) => (
                  <div key={item.name} className="flex items-center gap-1.5 text-xs">
                    <div
                      className="w-2 h-2 rounded-full"
                      style={{ backgroundColor: item.color }}
                    />
                    <span className="text-muted-foreground">{item.name}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Mini Gauges */}
          <div className="flex items-center justify-center gap-8">
            <MiniGauge value={stats.avgMemory} label="Avg Memory" variant="primary" />
            <MiniGauge value={stats.avgCpu} label="Avg CPU" variant="info" />
            <MiniGauge value={12.4} max={60} label="Avg Time" unit="s" variant="safe" />
          </div>
        </div>
      </GlassContainer>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {quickActions.map((action) => (
          <Link key={action.label} to={action.href}>
            <GlassCard hover className="p-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="p-2 rounded-lg bg-surface-variant">
                    <action.icon className="w-5 h-5 text-primary" />
                  </div>
                  <span className="font-medium">{action.label}</span>
                </div>
                {action.count && (
                  <span className="px-2 py-0.5 rounded-full bg-primary/20 text-primary text-sm font-medium">
                    {action.count}
                  </span>
                )}
                <ChevronRight className="w-5 h-5 text-muted-foreground" />
              </div>
            </GlassCard>
          </Link>
        ))}
      </div>

      {/* Recent Activity */}
      <GlassContainer>
        <h2 className="text-lg font-semibold mb-4">Recent Activity</h2>
        <div className="space-y-3">
          {mockExecutions.map((exec) => (
            <motion.div key={exec.id} layout>
              <GlassCard className="p-0 overflow-hidden">
                <button
                  onClick={() =>
                    setExpandedExecution(expandedExecution === exec.id ? null : exec.id)
                  }
                  className="w-full p-4 flex items-center justify-between hover:bg-surface-variant/50 transition-colors"
                >
                  <div className="flex items-center gap-4">
                    <div
                      className={`w-2 h-2 rounded-full ${
                        exec.exitCode === 0 ? 'bg-safe-green' : 'bg-danger-pink'
                      }`}
                    />
                    <div className="text-left">
                      <p className="font-mono text-sm truncate max-w-[300px]">
                        {exec.cmd}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {exec.deviceName} • {exec.time}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <div className="flex items-center gap-1 text-xs text-muted-foreground">
                      <Clock className="w-3 h-3" />
                      {(exec.totalTimeMs / 1000).toFixed(1)}s
                    </div>
                    <ChevronDown
                      className={`w-5 h-5 text-muted-foreground transition-transform ${
                        expandedExecution === exec.id ? 'rotate-180' : ''
                      }`}
                    />
                  </div>
                </button>

                <AnimatePresence>
                  {expandedExecution === exec.id && (
                    <motion.div
                      initial={{ height: 0, opacity: 0 }}
                      animate={{ height: 'auto', opacity: 1 }}
                      exit={{ height: 0, opacity: 0 }}
                      transition={{ duration: 0.2 }}
                      className="overflow-hidden"
                    >
                      <div className="px-4 pb-4">
                        <TerminalPanel
                          output={exec.output}
                          exitCode={exec.exitCode}
                        />
                      </div>
                    </motion.div>
                  )}
                </AnimatePresence>
              </GlassCard>
            </motion.div>
          ))}
        </div>
      </GlassContainer>
    </div>
  );
};
