//go:build darwin

package sysinfo

import (
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// CPU sampling state for calculating usage
	prevCPUTotal uint64
	prevCPUIdle  uint64
	cpuMu        sync.Mutex
)

// getPlatformHostStatus returns OS-level metrics for macOS
func getPlatformHostStatus() *HostStatus {
	status := &HostStatus{
		CPULoad:       -1,
		GPULoad:       -1,
		NPULoad:       -1,
		GPUMemUsedMB:  0,
		GPUMemTotalMB: 0,
	}

	// Get CPU load using vm_stat approach or load average
	status.CPULoad = getCPULoad()

	// Get memory using vm_stat
	used, total := getMemoryUsage()
	status.MemUsedMB = used
	status.MemTotalMB = total

	// GPU metrics (Apple Silicon doesn't expose GPU utilization easily)
	// For NVIDIA eGPU or similar, we could check nvidia-smi
	gpuLoad, gpuMemUsed, gpuMemTotal := getGPUMetrics()
	status.GPULoad = gpuLoad
	status.GPUMemUsedMB = gpuMemUsed
	status.GPUMemTotalMB = gpuMemTotal

	return status
}

// getCPULoad returns CPU usage as 0.0-1.0
func getCPULoad() float64 {
	// Use top command to get CPU usage
	cmd := exec.Command("top", "-l", "1", "-n", "0", "-stats", "cpu")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to load average
		return getLoadAverage()
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "CPU usage:") {
			// Parse: "CPU usage: 5.26% user, 10.52% sys, 84.21% idle"
			parts := strings.Split(line, ",")
			for _, part := range parts {
				if strings.Contains(part, "idle") {
					idleStr := strings.TrimSpace(part)
					idleStr = strings.TrimSuffix(idleStr, "% idle")
					idleStr = strings.TrimSpace(idleStr)
					idle, err := strconv.ParseFloat(idleStr, 64)
					if err == nil {
						return (100.0 - idle) / 100.0
					}
				}
			}
		}
	}

	return getLoadAverage()
}

// getLoadAverage returns 1-minute load average normalized by CPU count
func getLoadAverage() float64 {
	cmd := exec.Command("sysctl", "-n", "vm.loadavg")
	output, err := cmd.Output()
	if err != nil {
		return -1
	}

	// Parse: "{ 1.23 1.45 1.67 }"
	str := strings.Trim(string(output), "{ }\n")
	parts := strings.Fields(str)
	if len(parts) >= 1 {
		load, err := strconv.ParseFloat(parts[0], 64)
		if err == nil {
			// Normalize by CPU count
			cpuCount := getCPUCount()
			if cpuCount > 0 {
				normalized := load / float64(cpuCount)
				if normalized > 1.0 {
					normalized = 1.0
				}
				return normalized
			}
		}
	}
	return -1
}

// getCPUCount returns the number of CPUs
func getCPUCount() int {
	cmd := exec.Command("sysctl", "-n", "hw.ncpu")
	output, err := cmd.Output()
	if err != nil {
		return 1
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 1
	}
	return count
}

// getMemoryUsage returns used and total memory in MB
func getMemoryUsage() (uint64, uint64) {
	// Get total physical memory
	totalCmd := exec.Command("sysctl", "-n", "hw.memsize")
	totalOutput, err := totalCmd.Output()
	if err != nil {
		return 0, 0
	}
	totalBytes, err := strconv.ParseUint(strings.TrimSpace(string(totalOutput)), 10, 64)
	if err != nil {
		return 0, 0
	}
	totalMB := totalBytes / (1024 * 1024)

	// Get memory usage from vm_stat
	vmCmd := exec.Command("vm_stat")
	vmOutput, err := vmCmd.Output()
	if err != nil {
		return 0, totalMB
	}

	// Parse vm_stat output
	var pagesFree, pagesActive, pagesWired, pagesSpeculative uint64
	_ = pagesFree            // May be used for free memory calculation
	pageSize := uint64(4096) // Default page size

	lines := strings.Split(string(vmOutput), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Mach Virtual Memory Statistics") {
			// Extract page size if mentioned
			if strings.Contains(line, "page size of") {
				parts := strings.Split(line, "page size of")
				if len(parts) > 1 {
					sizeStr := strings.TrimSpace(strings.Split(parts[1], " ")[1])
					if size, err := strconv.ParseUint(sizeStr, 10, 64); err == nil {
						pageSize = size
					}
				}
			}
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(strings.TrimSuffix(parts[1], "."))
		value, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			continue
		}

		switch key {
		case "Pages free":
			pagesFree = value
		case "Pages active":
			pagesActive = value
		case "Pages wired down":
			pagesWired = value
		case "Pages speculative":
			pagesSpeculative = value
		}
	}

	// Calculate used memory (active + wired + speculative)
	usedPages := pagesActive + pagesWired + pagesSpeculative
	usedMB := (usedPages * pageSize) / (1024 * 1024)

	// If parsing failed, estimate from total - free
	if usedMB == 0 && pagesFree > 0 {
		freeBytes := pagesFree * pageSize
		usedMB = totalMB - (freeBytes / (1024 * 1024))
	}

	return usedMB, totalMB
}

// getGPUMetrics returns GPU load and memory (for NVIDIA eGPUs)
func getGPUMetrics() (float64, uint64, uint64) {
	// Check for NVIDIA GPU via nvidia-smi
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=utilization.gpu,memory.used,memory.total",
		"--format=csv,noheader,nounits")
	cmd.Env = append(cmd.Env, "PATH=/usr/local/bin:/usr/bin:/bin")

	// Set timeout
	done := make(chan error, 1)
	var output []byte
	var err error

	go func() {
		output, err = cmd.Output()
		done <- err
	}()

	select {
	case <-done:
		if err != nil {
			return -1, 0, 0
		}
	case <-time.After(2 * time.Second):
		cmd.Process.Kill()
		return -1, 0, 0
	}

	// Parse: "35, 1234, 8192"
	line := strings.TrimSpace(string(output))
	parts := strings.Split(line, ",")
	if len(parts) >= 3 {
		load, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		memUsed, _ := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
		memTotal, _ := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)
		return load / 100.0, memUsed, memTotal
	}

	return -1, 0, 0
}
