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
import { useMemo } from 'react';
import type { MetricsHistoryEntry } from '@/api';

type MetricType = 'cpu' | 'memory' | 'gpu';

interface MetricsChartProps {
  data: MetricsHistoryEntry[];
  metric: MetricType;
  height?: number;
}

const metricConfig: Record<MetricType, { key: keyof MetricsHistoryEntry; label: string; color: string }> = {
  cpu: { key: 'cpu_percent', label: 'CPU %', color: '#3B82F6' }, // info-blue
  memory: { key: 'memory_percent', label: 'Memory %', color: '#8B5CF6' }, // primary purple
  gpu: { key: 'gpu_percent', label: 'GPU %', color: '#10B981' }, // safe-green
};

// Color palette for different devices
const deviceColors = [
  '#3B82F6', // blue
  '#10B981', // green
  '#F59E0B', // amber
  '#EF4444', // red
  '#8B5CF6', // purple
  '#EC4899', // pink
  '#06B6D4', // cyan
  '#84CC16', // lime
];

export function MetricsChart({ data, metric, height = 200 }: MetricsChartProps) {
  const config = metricConfig[metric];

  // Group data by timestamp and device for multi-line chart
  const chartData = useMemo(() => {
    const byTimestamp = new Map<string, Record<string, number>>();
    const devices = new Set<string>();

    data.forEach((entry) => {
      devices.add(entry.device_name);
      const time = new Date(entry.timestamp).toLocaleTimeString([], {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      });

      if (!byTimestamp.has(time)) {
        byTimestamp.set(time, { time });
      }

      const record = byTimestamp.get(time)!;
      record[entry.device_name] = entry[config.key] as number;
    });

    return {
      data: Array.from(byTimestamp.values()),
      devices: Array.from(devices),
    };
  }, [data, config.key]);

  if (chartData.data.length === 0) {
    return (
      <div
        className="flex items-center justify-center text-muted-foreground"
        style={{ height }}
      >
        No metrics data available
      </div>
    );
  }

  return (
    <ResponsiveContainer width="100%" height={height}>
      <LineChart data={chartData.data} margin={{ top: 5, right: 5, left: 0, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" className="stroke-outline/30" />
        <XAxis
          dataKey="time"
          tick={{ fontSize: 10 }}
          className="fill-muted-foreground"
          tickLine={false}
          axisLine={false}
        />
        <YAxis
          domain={[0, 100]}
          tick={{ fontSize: 10 }}
          className="fill-muted-foreground"
          tickLine={false}
          axisLine={false}
          width={35}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: 'hsl(var(--surface-2))',
            border: '1px solid hsl(var(--outline))',
            borderRadius: '8px',
            fontSize: '12px',
          }}
          labelStyle={{ color: 'hsl(var(--foreground))' }}
        />
        <Legend
          wrapperStyle={{ fontSize: '11px' }}
          iconSize={8}
        />
        {chartData.devices.map((device, index) => (
          <Line
            key={device}
            type="monotone"
            dataKey={device}
            name={device}
            stroke={deviceColors[index % deviceColors.length]}
            strokeWidth={2}
            dot={false}
            activeDot={{ r: 4 }}
          />
        ))}
      </LineChart>
    </ResponsiveContainer>
  );
}
