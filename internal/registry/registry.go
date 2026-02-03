// Package registry provides an in-memory device registry for multi-device orchestration
package registry

import (
	"sync"
	"time"

	pb "github.com/edgecli/edgecli/proto"
)

// DeviceEntry holds device info and status
type DeviceEntry struct {
	Info     *pb.DeviceInfo
	LastSeen time.Time
	Status   *pb.DeviceStatus
}

// Registry manages registered devices
type Registry struct {
	devices map[string]*DeviceEntry
	mu      sync.RWMutex
}

// NewRegistry creates a new device registry
func NewRegistry() *Registry {
	return &Registry{
		devices: make(map[string]*DeviceEntry),
	}
}

// Upsert adds or updates a device in the registry
// Returns the registration timestamp
func (r *Registry) Upsert(info *pb.DeviceInfo) time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	entry, exists := r.devices[info.DeviceId]
	if exists {
		// Update existing entry
		entry.Info = info
		entry.LastSeen = now
	} else {
		// Create new entry
		r.devices[info.DeviceId] = &DeviceEntry{
			Info:     info,
			LastSeen: now,
			Status: &pb.DeviceStatus{
				DeviceId: info.DeviceId,
				LastSeen: now.Unix(),
			},
		}
	}
	return now
}

// List returns all registered devices
func (r *Registry) List() []*pb.DeviceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	devices := make([]*pb.DeviceInfo, 0, len(r.devices))
	for _, entry := range r.devices {
		devices = append(devices, entry.Info)
	}
	return devices
}

// Get returns a device entry by ID
func (r *Registry) Get(deviceID string) (*DeviceEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.devices[deviceID]
	return entry, ok
}

// GetStatus returns the status of a device
func (r *Registry) GetStatus(deviceID string) *pb.DeviceStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.devices[deviceID]
	if !ok {
		// Return empty status for unknown device
		return &pb.DeviceStatus{
			DeviceId: deviceID,
			LastSeen: 0,
		}
	}

	return &pb.DeviceStatus{
		DeviceId:   deviceID,
		LastSeen:   entry.LastSeen.Unix(),
		CpuLoad:    entry.Status.CpuLoad,
		MemUsedMb:  entry.Status.MemUsedMb,
		MemTotalMb: entry.Status.MemTotalMb,
	}
}

// UpdateStatus updates the status of a device
func (r *Registry) UpdateStatus(deviceID string, status *pb.DeviceStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.devices[deviceID]
	if ok {
		entry.Status = status
		entry.LastSeen = time.Now()
	}
}

// Count returns the number of registered devices
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.devices)
}

// SelectBestDevice selects the best device for AI task routing
// Priority: has_npu > has_gpu > has_cpu
func (r *Registry) SelectBestDevice() (*pb.DeviceInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.devices) == 0 {
		return nil, false
	}

	var npuDevice, gpuDevice, cpuDevice *pb.DeviceInfo

	for _, entry := range r.devices {
		if entry.Info.HasNpu {
			npuDevice = entry.Info
			break // NPU is highest priority
		}
		if entry.Info.HasGpu && gpuDevice == nil {
			gpuDevice = entry.Info
		}
		if entry.Info.HasCpu && cpuDevice == nil {
			cpuDevice = entry.Info
		}
	}

	// Return by priority
	if npuDevice != nil {
		return npuDevice, true
	}
	if gpuDevice != nil {
		return gpuDevice, true
	}
	if cpuDevice != nil {
		return cpuDevice, true
	}

	// Fallback to first device
	for _, entry := range r.devices {
		return entry.Info, true
	}

	return nil, false
}
