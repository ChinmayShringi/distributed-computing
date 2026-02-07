//go:build linux

package sysinfo

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// CPU sampling state
	prevCPUStats cpuStats
	cpuMu        sync.Mutex
)

type cpuStats struct {
	user    uint64
	nice    uint64
	system  uint64
	idle    uint64
	iowait  uint64
	irq     uint64
	softirq uint64
	total   uint64
}

// getPlatformHostStatus returns OS-level metrics for Linux
func getPlatformHostStatus() *HostStatus {
	status := &HostStatus{
		CPULoad:       -1,
		GPULoad:       -1,
		NPULoad:       -1,
		GPUMemUsedMB:  0,
		GPUMemTotalMB: 0,
	}

	// Get CPU load from /proc/stat
	status.CPULoad = getCPULoad()

	// Get memory from /proc/meminfo
	used, total := getMemoryUsage()
	status.MemUsedMB = used
	status.MemTotalMB = total

	// Get GPU metrics (NVIDIA or AMD)
	gpuLoad, gpuMemUsed, gpuMemTotal := getGPUMetrics()
	status.GPULoad = gpuLoad
	status.GPUMemUsedMB = gpuMemUsed
	status.GPUMemTotalMB = gpuMemTotal

	return status
}

// getCPULoad returns CPU usage as 0.0-1.0 from /proc/stat
func getCPULoad() float64 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return getLoadAverage()
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) >= 8 {
				var stats cpuStats
				stats.user, _ = strconv.ParseUint(fields[1], 10, 64)
				stats.nice, _ = strconv.ParseUint(fields[2], 10, 64)
				stats.system, _ = strconv.ParseUint(fields[3], 10, 64)
				stats.idle, _ = strconv.ParseUint(fields[4], 10, 64)
				stats.iowait, _ = strconv.ParseUint(fields[5], 10, 64)
				stats.irq, _ = strconv.ParseUint(fields[6], 10, 64)
				stats.softirq, _ = strconv.ParseUint(fields[7], 10, 64)
				stats.total = stats.user + stats.nice + stats.system + stats.idle +
					stats.iowait + stats.irq + stats.softirq

				cpuMu.Lock()
				prev := prevCPUStats
				prevCPUStats = stats
				cpuMu.Unlock()

				if prev.total == 0 {
					return getLoadAverage() // Need two samples
				}

				totalDelta := stats.total - prev.total
				idleDelta := stats.idle - prev.idle

				if totalDelta == 0 {
					return 0
				}

				usage := 1.0 - (float64(idleDelta) / float64(totalDelta))
				if usage < 0 {
					usage = 0
				}
				if usage > 1 {
					usage = 1
				}
				return usage
			}
		}
	}

	return getLoadAverage()
}

// getLoadAverage returns 1-minute load average normalized by CPU count
func getLoadAverage() float64 {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return -1
	}

	fields := strings.Fields(string(data))
	if len(fields) >= 1 {
		load, err := strconv.ParseFloat(fields[0], 64)
		if err == nil {
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
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 1
	}

	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "processor") {
			count++
		}
	}
	if count == 0 {
		return 1
	}
	return count
}

// getMemoryUsage returns used and total memory in MB from /proc/meminfo
func getMemoryUsage() (uint64, uint64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	var memTotal, memFree, memAvailable, buffers, cached uint64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.TrimSuffix(fields[0], ":")
		value, _ := strconv.ParseUint(fields[1], 10, 64)

		switch key {
		case "MemTotal":
			memTotal = value
		case "MemFree":
			memFree = value
		case "MemAvailable":
			memAvailable = value
		case "Buffers":
			buffers = value
		case "Cached":
			cached = value
		}
	}

	// Convert from KB to MB
	totalMB := memTotal / 1024

	// Calculate used memory
	var usedMB uint64
	if memAvailable > 0 {
		// Use MemAvailable if present (more accurate)
		usedMB = (memTotal - memAvailable) / 1024
	} else {
		// Fall back to Total - Free - Buffers - Cached
		usedMB = (memTotal - memFree - buffers - cached) / 1024
	}

	return usedMB, totalMB
}

// getGPUMetrics returns GPU load and memory
func getGPUMetrics() (float64, uint64, uint64) {
	// Try NVIDIA first
	load, used, total := getNvidiaGPUMetrics()
	if load >= 0 {
		return load, used, total
	}

	// Try AMD ROCm
	load, used, total = getAMDGPUMetrics()
	if load >= 0 {
		return load, used, total
	}

	return -1, 0, 0
}

// getNvidiaGPUMetrics returns NVIDIA GPU metrics via nvidia-smi
func getNvidiaGPUMetrics() (float64, uint64, uint64) {
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=utilization.gpu,memory.used,memory.total",
		"--format=csv,noheader,nounits")

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

// getAMDGPUMetrics returns AMD GPU metrics via rocm-smi
func getAMDGPUMetrics() (float64, uint64, uint64) {
	cmd := exec.Command("rocm-smi", "--showuse", "--showmemuse", "--json")

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

	// Parse rocm-smi JSON output
	// Format varies, simplified parsing here
	str := string(output)
	if strings.Contains(str, "GPU use") {
		// Basic parsing - would need JSON unmarshaling for production
		return -1, 0, 0 // Placeholder
	}

	return -1, 0, 0
}
