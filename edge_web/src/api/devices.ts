// Device API endpoints
import { get } from './client';
import type { DevicesResponse, Device, DeviceMetricsResponse } from './types';

// List all registered devices
export async function listDevices(): Promise<Device[]> {
  const response = await get<DevicesResponse>('/api/devices');
  return response.devices || [];
}

// Get metrics history for a specific device
export async function getDeviceMetrics(deviceId: string): Promise<DeviceMetricsResponse> {
  return get<DeviceMetricsResponse>('/api/device-metrics', { device_id: deviceId });
}
