//go:build !darwin && !windows && !linux

package sysinfo

import "runtime"

// getPlatformHostStatus returns basic metrics for unsupported platforms
// Falls back to Go runtime memory stats
func getPlatformHostStatus() *HostStatus {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Use Go heap memory as an approximation
	memUsedMB := memStats.Alloc / (1024 * 1024)
	memTotalMB := memStats.Sys / (1024 * 1024)

	return &HostStatus{
		CPULoad:       -1, // Not available
		MemUsedMB:     memUsedMB,
		MemTotalMB:    memTotalMB,
		GPULoad:       -1,
		GPUMemUsedMB:  0,
		GPUMemTotalMB: 0,
		NPULoad:       -1,
	}
}
