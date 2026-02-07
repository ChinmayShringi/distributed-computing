import { useMemo } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import type { DeviceMetricsHistory, MetricsSample } from '@/api';

interface MetricsChartProps {
  data: DeviceMetricsHistory[];
  metric: 'cpu' | 'memory' | 'gpu';
  height?: number;
}

// Generate unique colors for each device
const deviceColors = [
  '#3b82f6', // blue
  '#22c55e', // green
  '#f59e0b', // amber
  '#ec4899', // pink
  '#8b5cf6', // purple
  '#06b6d4', // cyan
  '#f97316', // orange
  '#14b8a6', // teal
];

interface ChartDataPoint {
  time: number;
  [deviceName: string]: number;
}

export const MetricsChart = ({
  data,
  metric,
  height = 200,
}: MetricsChartProps) => {
  // Transform data for recharts
  const { chartData, devices } = useMemo(() => {
    const devices: { name: string; color: string }[] = [];
    const timeMap = new Map<number, ChartDataPoint>();

    data.forEach((deviceHistory, idx) => {
      devices.push({
        name: deviceHistory.device_name,
        color: deviceColors[idx % deviceColors.length],
      });

      deviceHistory.samples.forEach((sample) => {
        const timestamp = new Date(sample.timestamp).getTime();
        const existing = timeMap.get(timestamp) || { time: timestamp };

        // Get the metric value
        let value = 0;
        switch (metric) {
          case 'cpu':
            value = sample.cpu_percent;
            break;
          case 'memory':
            value = sample.memory_percent;
            break;
          case 'gpu':
            value = Math.max(sample.gpu_percent, sample.npu_percent);
            break;
        }

        existing[deviceHistory.device_name] = value;
        timeMap.set(timestamp, existing);
      });
    });

    // Sort by time and take last 120 seconds worth
    const sortedData = Array.from(timeMap.values())
      .sort((a, b) => a.time - b.time)
      .slice(-60); // ~60 data points at 2s intervals = 120s

    return { chartData: sortedData, devices };
  }, [data, metric]);

  // Format time for X axis
  const formatTime = (timestamp: number) => {
    const now = Date.now();
    const secondsAgo = Math.round((now - timestamp) / 1000);
    return `-${secondsAgo}s`;
  };

  const metricLabels = {
    cpu: 'CPU Usage',
    memory: 'Memory Usage',
    gpu: 'GPU/NPU Usage',
  };

  if (chartData.length === 0) {
    return (
      <div
        className="flex items-center justify-center text-muted-foreground text-sm"
        style={{ height }}
      >
        No data available. Start polling to collect metrics.
      </div>
    );
  }

  return (
    <div style={{ height }}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart
          data={chartData}
          margin={{ top: 5, right: 20, left: 0, bottom: 5 }}
        >
          <CartesianGrid
            strokeDasharray="3 3"
            stroke="hsl(var(--outline))"
            opacity={0.3}
          />
          <XAxis
            dataKey="time"
            tickFormatter={formatTime}
            stroke="hsl(var(--muted-foreground))"
            fontSize={10}
            tickLine={false}
            axisLine={false}
          />
          <YAxis
            domain={[0, 100]}
            stroke="hsl(var(--muted-foreground))"
            fontSize={10}
            tickLine={false}
            axisLine={false}
            tickFormatter={(value) => `${value}%`}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: 'hsl(var(--surface-2))',
              border: '1px solid hsl(var(--outline))',
              borderRadius: '8px',
              fontSize: '12px',
            }}
            labelFormatter={(timestamp) => {
              const date = new Date(timestamp as number);
              return date.toLocaleTimeString();
            }}
            formatter={(value: number, name: string) => [
              `${value.toFixed(1)}%`,
              name,
            ]}
          />
          {devices.length > 1 && (
            <Legend
              wrapperStyle={{ fontSize: '10px' }}
              iconType="line"
            />
          )}
          {devices.map((device) => (
            <Line
              key={device.name}
              type="monotone"
              dataKey={device.name}
              stroke={device.color}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4 }}
              connectNulls
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
};
