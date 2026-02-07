import { Toaster } from "@/components/ui/toaster";
import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";

// Layouts
import { PublicLayout } from "@/layouts/PublicLayout";
import { ConsoleLayout } from "@/layouts/ConsoleLayout";

// Public Pages
import { HomePage } from "@/pages/HomePage";
import { FeaturesPage } from "@/pages/FeaturesPage";
import { PricingPage } from "@/pages/PricingPage";
import { AboutPage } from "@/pages/AboutPage";

// Console Pages
import { ConnectPage } from "@/pages/console/ConnectPage";
import { DashboardPage } from "@/pages/console/DashboardPage";
import { RequestsPage } from "@/pages/console/RequestsPage";
import { LiveViewPage } from "@/pages/console/LiveViewPage";
import { DownloadsPage } from "@/pages/console/DownloadsPage";
import { DevicesPage } from "@/pages/console/DevicesPage";
import { DeviceDetailPage } from "@/pages/console/DeviceDetailPage";
import { RunPage } from "@/pages/console/RunPage";
import { JobsPage } from "@/pages/console/JobsPage";
import { SettingsPage } from "@/pages/console/SettingsPage";
import { ActivityPage } from "@/pages/console/ActivityPage";
import { ChatPage } from "@/pages/console/ChatPage";
import { AgentPage } from "@/pages/console/AgentPage";
import { QAIHubPage } from "@/pages/console/QAIHubPage";

const queryClient = new QueryClient();

const App = () => (
  <QueryClientProvider client={queryClient}>
    <TooltipProvider>
      <Toaster />
      <Sonner />
      <BrowserRouter>
        <Routes>
          {/* Public Website */}
          <Route element={<PublicLayout />}>
            <Route path="/" element={<HomePage />} />
            <Route path="/features" element={<FeaturesPage />} />
            <Route path="/pricing" element={<PricingPage />} />
            <Route path="/about" element={<AboutPage />} />
          </Route>

          {/* Console */}
          <Route path="/console" element={<ConsoleLayout />}>
            <Route index element={<Navigate to="/console/connect" replace />} />
            <Route path="connect" element={<ConnectPage />} />
            <Route path="dashboard" element={<DashboardPage />} />
            <Route path="dashboard/requests" element={<RequestsPage />} />
            <Route path="dashboard/live" element={<LiveViewPage />} />
            <Route path="dashboard/downloads" element={<DownloadsPage />} />
            <Route path="devices" element={<DevicesPage />} />
            <Route path="devices/:id" element={<DeviceDetailPage />} />
            <Route path="run" element={<RunPage />} />
            <Route path="jobs" element={<JobsPage />} />
            <Route path="activity" element={<ActivityPage />} />
            <Route path="chat" element={<ChatPage />} />
            <Route path="agent" element={<AgentPage />} />
            <Route path="qaihub" element={<QAIHubPage />} />
            <Route path="settings" element={<SettingsPage />} />
          </Route>

          {/* Catch all */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </TooltipProvider>
  </QueryClientProvider>
);

export default App;
