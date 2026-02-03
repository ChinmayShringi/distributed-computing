// Package sysinfo provides lightweight host status sampling
package sysinfo

import (
	"runtime"
)

// HostStatus contains current host metrics
type HostStatus struct {
	CPULoad    float64 // -1 if unavailable (no external deps)
	MemUsedMB  uint64
	MemTotalMB uint64
}

// GetHostStatus samples current host status
// Uses runtime.ReadMemStats for memory (Go heap as approximation)
// CPU load is set to -1 as it requires external dependencies or platform-specific code
func GetHostStatus() *HostStatus {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Use Go heap memory as an approximation
	// Alloc is bytes allocated and still in use
	// Sys is total bytes obtained from the OS
	memUsedMB := memStats.Alloc / (1024 * 1024)
	memTotalMB := memStats.Sys / (1024 * 1024)

	return &HostStatus{
		CPULoad:    -1, // Not available without external deps
		MemUsedMB:  memUsedMB,
		MemTotalMB: memTotalMB,
	}
}
