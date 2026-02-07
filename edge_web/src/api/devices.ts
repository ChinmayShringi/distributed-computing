import { apiGet } from './client';
import type { Device, DeviceMetrics, DeviceMetricsHistory } from './types';

export async function listDevices(): Promise<Device[]> {
  return apiGet<Device[]>('/api/devices');
}

export async function getDeviceStatus(deviceId: string): Promise<DeviceMetrics> {
  return apiGet<DeviceMetrics>(`/api/device-status?device_id=${encodeURIComponent(deviceId)}`);
}

export async function getDeviceMetrics(deviceId: string): Promise<DeviceMetricsHistory> {
  return apiGet<DeviceMetricsHistory>(`/api/device-metrics?device_id=${encodeURIComponent(deviceId)}`);
}
