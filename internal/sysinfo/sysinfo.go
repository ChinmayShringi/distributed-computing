// Package sysinfo provides lightweight host status sampling with OS-level metrics
package sysinfo

import "time"

// HostStatus contains current host metrics
type HostStatus struct {
	CPULoad       float64 // 0.0-1.0, or -1 if unavailable
	MemUsedMB     uint64  // OS-level memory used
	MemTotalMB    uint64  // OS-level memory total
	GPULoad       float64 // 0.0-1.0, or -1 if unavailable
	GPUMemUsedMB  uint64  // GPU memory used (0 if unavailable)
	GPUMemTotalMB uint64  // GPU memory total (0 if unavailable)
	NPULoad       float64 // 0.0-1.0, or -1 if unavailable
	Timestamp     int64   // Unix milliseconds
}

// GetHostStatus samples current host status using platform-specific code
// Implemented in sysinfo_darwin.go, sysinfo_windows.go, sysinfo_linux.go
func GetHostStatus() *HostStatus {
	status := getPlatformHostStatus()
	status.Timestamp = time.Now().UnixMilli()
	return status
}
