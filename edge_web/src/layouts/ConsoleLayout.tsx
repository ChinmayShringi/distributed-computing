import { Outlet, NavLink, useNavigate, useLocation } from 'react-router-dom';
import { useEffect, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { EdgeMeshWordmark } from '@/components/EdgeMeshWordmark';
import { StatusStrip } from '@/components/StatusStrip';
import { getConnectionState } from '@/lib/connection';
import {
  LayoutDashboard,
  Monitor,
  Play,
  MessageSquare,
  Briefcase,
  Settings,
  Bell,
  User,
  Menu,
  X,
  Activity,
  Bot,
  Cpu,
  Eye,
  Download,
} from 'lucide-react';
import { cn } from '@/lib/utils';

const navItems = [
  { icon: LayoutDashboard, label: 'Dashboard', href: '/console/dashboard' },
  { icon: Monitor, label: 'Devices', href: '/console/devices' },
  { icon: Activity, label: 'Activity', href: '/console/activity' },
  { icon: Play, label: 'Run', href: '/console/run' },
  { icon: MessageSquare, label: 'Chat', href: '/console/chat' },
  { icon: Bot, label: 'Agent', href: '/console/agent' },
  { icon: Briefcase, label: 'Jobs', href: '/console/jobs' },
  { icon: Eye, label: 'Live View', href: '/console/live' },
  { icon: Download, label: 'Downloads', href: '/console/downloads' },
  { icon: Cpu, label: 'QAI Hub', href: '/console/qaihub' },
  { icon: Settings, label: 'Settings', href: '/console/settings' },
];

export const ConsoleLayout = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const state = getConnectionState();
    setConnected(state.connected);
    
    // Redirect to connect if not connected
    if (!state.connected && !location.pathname.includes('/console/connect')) {
      navigate('/console/connect');
    }
  }, [location.pathname, navigate]);

  // Listen for storage changes
  useEffect(() => {
    const handleStorageChange = () => {
      const state = getConnectionState();
      setConnected(state.connected);
      if (!state.connected && !location.pathname.includes('/console/connect')) {
        navigate('/console/connect');
      }
    };

    window.addEventListener('storage', handleStorageChange);
    return () => window.removeEventListener('storage', handleStorageChange);
  }, [location.pathname, navigate]);

  // If on connect page, render differently
  if (location.pathname === '/console/connect') {
    return (
      <div className="min-h-screen bg-background bg-noise bg-grid">
        <Outlet />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background flex">
      {/* Desktop Sidebar */}
      <aside className="hidden md:flex flex-col w-64 bg-surface-1 border-r border-outline">
        {/* Logo */}
        <div className="p-4 border-b border-outline">
          <EdgeMeshWordmark size="lg" />
        </div>

        {/* Navigation */}
        <nav className="flex-1 p-4">
          <ul className="space-y-2">
            {navItems.map((item) => (
              <li key={item.href}>
                <NavLink
                  to={item.href}
                  className={({ isActive }) =>
                    cn(
                      'flex items-center gap-3 px-4 py-3 rounded-lg transition-all',
                      isActive
                        ? 'bg-primary/20 text-primary'
                        : 'text-muted-foreground hover:bg-surface-variant hover:text-foreground'
                    )
                  }
                >
                  <item.icon className="w-5 h-5" />
                  <span className="font-medium">{item.label}</span>
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
      </aside>

      {/* Mobile Sidebar Overlay */}
      <AnimatePresence>
        {sidebarOpen && (
          <>
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="fixed inset-0 bg-black/50 z-40 md:hidden"
              onClick={() => setSidebarOpen(false)}
            />
            <motion.aside
              initial={{ x: -280 }}
              animate={{ x: 0 }}
              exit={{ x: -280 }}
              transition={{ type: 'tween' }}
              className="fixed left-0 top-0 bottom-0 w-64 bg-surface-1 border-r border-outline z-50 md:hidden"
            >
              <div className="p-4 border-b border-outline flex items-center justify-between">
                <EdgeMeshWordmark size="md" />
                <button onClick={() => setSidebarOpen(false)}>
                  <X className="w-5 h-5" />
                </button>
              </div>
              <nav className="p-4">
                <ul className="space-y-2">
                  {navItems.map((item) => (
                    <li key={item.href}>
                      <NavLink
                        to={item.href}
                        onClick={() => setSidebarOpen(false)}
                        className={({ isActive }) =>
                          cn(
                            'flex items-center gap-3 px-4 py-3 rounded-lg transition-all',
                            isActive
                              ? 'bg-primary/20 text-primary'
                              : 'text-muted-foreground hover:bg-surface-variant'
                          )
                        }
                      >
                        <item.icon className="w-5 h-5" />
                        <span className="font-medium">{item.label}</span>
                      </NavLink>
                    </li>
                  ))}
                </ul>
              </nav>
            </motion.aside>
          </>
        )}
      </AnimatePresence>

      {/* Main Content */}
      <div className="flex-1 flex flex-col min-h-screen">
        {/* Header */}
        <header className="h-14 bg-surface-1 border-b border-outline flex items-center justify-between px-4">
          <div className="flex items-center gap-4">
            <button
              className="md:hidden p-2"
              onClick={() => setSidebarOpen(true)}
            >
              <Menu className="w-5 h-5" />
            </button>
            <div className="md:hidden">
              <EdgeMeshWordmark size="sm" />
            </div>
          </div>

          <div className="flex items-center gap-4">
            <button className="p-2 rounded-lg hover:bg-surface-variant transition-colors relative">
              <Bell className="w-5 h-5 text-muted-foreground" />
              <span className="absolute top-1.5 right-1.5 w-2 h-2 rounded-full bg-primary" />
            </button>
            <button className="p-2 rounded-lg hover:bg-surface-variant transition-colors">
              <User className="w-5 h-5 text-muted-foreground" />
            </button>
          </div>
        </header>

        {/* Status Strip */}
        {connected && <StatusStrip />}

        {/* Page Content */}
        <main className="flex-1 overflow-auto">
          <Outlet />
        </main>
      </div>
    </div>
  );
};
