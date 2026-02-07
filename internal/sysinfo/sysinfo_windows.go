//go:build windows

package sysinfo

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemTimes       = kernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")

	// CPU sampling state
	prevIdleTime   uint64
	prevKernelTime uint64
	prevUserTime   uint64
	prevSampleTime time.Time
)

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

// getPlatformHostStatus returns OS-level metrics for Windows
func getPlatformHostStatus() *HostStatus {
	status := &HostStatus{
		CPULoad:       -1,
		GPULoad:       -1,
		NPULoad:       -1,
		GPUMemUsedMB:  0,
		GPUMemTotalMB: 0,
	}

	// Get CPU load
	status.CPULoad = getCPULoad()

	// Get memory
	used, total := getMemoryUsage()
	status.MemUsedMB = used
	status.MemTotalMB = total

	// Get GPU metrics (NVIDIA via nvidia-smi)
	gpuLoad, gpuMemUsed, gpuMemTotal := getGPUMetrics()
	status.GPULoad = gpuLoad
	status.GPUMemUsedMB = gpuMemUsed
	status.GPUMemTotalMB = gpuMemTotal

	// NPU metrics for Qualcomm (placeholder - requires SDK)
	status.NPULoad = getNPULoad()

	return status
}

// getCPULoad returns CPU usage as 0.0-1.0 using GetSystemTimes
func getCPULoad() float64 {
	var idleTime, kernelTime, userTime syscall.Filetime

	ret, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idleTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)

	if ret == 0 {
		return -1
	}

	idle := filetimeToUint64(idleTime)
	kernel := filetimeToUint64(kernelTime)
	user := filetimeToUint64(userTime)

	now := time.Now()

	// Calculate delta from previous sample
	if prevSampleTime.IsZero() {
		prevIdleTime = idle
		prevKernelTime = kernel
		prevUserTime = user
		prevSampleTime = now
		return -1 // Need two samples to calculate
	}

	idleDelta := idle - prevIdleTime
	kernelDelta := kernel - prevKernelTime
	userDelta := user - prevUserTime
	totalDelta := kernelDelta + userDelta

	// Update previous values
	prevIdleTime = idle
	prevKernelTime = kernel
	prevUserTime = user
	prevSampleTime = now

	if totalDelta == 0 {
		return 0
	}

	// CPU usage = 1 - (idle / total)
	// Note: kernel time includes idle time
	cpuUsage := 1.0 - (float64(idleDelta) / float64(totalDelta))
	if cpuUsage < 0 {
		cpuUsage = 0
	}
	if cpuUsage > 1 {
		cpuUsage = 1
	}

	return cpuUsage
}

func filetimeToUint64(ft syscall.Filetime) uint64 {
	return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
}

// getMemoryUsage returns used and total memory in MB
func getMemoryUsage() (uint64, uint64) {
	var memStatus memoryStatusEx
	memStatus.Length = uint32(unsafe.Sizeof(memStatus))

	ret, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if ret == 0 {
		return 0, 0
	}

	totalMB := memStatus.TotalPhys / (1024 * 1024)
	availMB := memStatus.AvailPhys / (1024 * 1024)
	usedMB := totalMB - availMB

	return usedMB, totalMB
}

// getGPUMetrics returns GPU load and memory (NVIDIA via nvidia-smi)
func getGPUMetrics() (float64, uint64, uint64) {
	// Try nvidia-smi for NVIDIA GPUs
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=utilization.gpu,memory.used,memory.total",
		"--format=csv,noheader,nounits")

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

// getNPULoad returns NPU utilization for Qualcomm devices
// Currently returns -1 as it requires Qualcomm SDK integration
func getNPULoad() float64 {
	// Qualcomm NPU metrics would require:
	// 1. Check if running on Snapdragon (Windows on ARM)
	// 2. Use Qualcomm Neural Processing SDK or similar
	// For now, return -1 (unavailable)

	// Check if we're on ARM64 (potential Snapdragon)
	// In the future, this could integrate with QNN SDK
	return -1
}
