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
import type { DeviceMetricsHistory } from '@/api';

type MetricType = 'cpu' | 'memory' | 'gpu';

interface MetricsChartProps {
  data: Record<string, DeviceMetricsHistory>;
  metric: MetricType;
  height?: number;
}

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
  // Transform device_metrics into chart data format
  const chartData = useMemo(() => {
    const deviceEntries = Object.values(data);
    if (deviceEntries.length === 0) {
      return { data: [], devices: [] };
    }

    const devices = deviceEntries.map((d) => d.device_name);
    const byTimestamp = new Map<number, Record<string, number | string>>();

    deviceEntries.forEach((device) => {
      device.samples.forEach((sample) => {
        if (!byTimestamp.has(sample.timestamp_ms)) {
          byTimestamp.set(sample.timestamp_ms, {
            time: new Date(sample.timestamp_ms).toLocaleTimeString([], {
              hour: '2-digit',
              minute: '2-digit',
              second: '2-digit',
            }),
            timestamp: sample.timestamp_ms,
          });
        }

        const record = byTimestamp.get(sample.timestamp_ms)!;

        // Get the value based on metric type
        let value = 0;
        if (metric === 'cpu') {
          value = sample.cpu_load >= 0 ? sample.cpu_load * 100 : 0;
        } else if (metric === 'memory') {
          if (sample.mem_total_mb && sample.mem_used_mb) {
            value = (sample.mem_used_mb / sample.mem_total_mb) * 100;
          }
        } else if (metric === 'gpu') {
          const gpuVal = sample.gpu_load >= 0 ? sample.gpu_load : 0;
          const npuVal = sample.npu_load >= 0 ? sample.npu_load : 0;
          value = Math.max(gpuVal, npuVal) * 100;
        }

        record[device.device_name] = value;
      });
    });

    // Sort by timestamp and take last 60 samples
    const sortedData = Array.from(byTimestamp.values())
      .sort((a, b) => (a.timestamp as number) - (b.timestamp as number))
      .slice(-60);

    return { data: sortedData, devices };
  }, [data, metric]);

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
