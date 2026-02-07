// Package jobs provides in-memory job and task management for distributed execution
package jobs

import (
	pb "github.com/edgecli/edgecli/proto"
)

// LLM device selection helpers

// HasLLMCapability checks if a device has LLM inference capability
// A device is LLM-capable if it has a local model endpoint (Ollama/LM Studio)
func HasLLMCapability(device *pb.DeviceInfo) bool {
	return device.HasLocalModel
}

// FilterLLMDevices returns only devices that have LLM capability
func FilterLLMDevices(devices []*pb.DeviceInfo) []*pb.DeviceInfo {
	llmDevices := make([]*pb.DeviceInfo, 0)
	for _, d := range devices {
		if HasLLMCapability(d) {
			llmDevices = append(llmDevices, d)
		}
	}
	return llmDevices
}

// SelectBestLLMDevice selects the best device for LLM tasks
// Priority: NPU > GPU > CPU, then prefer devices with advertised TPS
func SelectBestLLMDevice(devices []*pb.DeviceInfo) *pb.DeviceInfo {
	if len(devices) == 0 {
		return nil
	}

	var best *pb.DeviceInfo

	// Phase 1: Prefer NPU devices
	for _, d := range devices {
		if d.HasNpu {
			if best == nil {
				best = d
			} else if d.LlmPrefillToksPerS > best.LlmPrefillToksPerS {
				// If multiple NPU devices, pick faster one
				best = d
			}
		}
	}
	if best != nil {
		return best
	}

	// Phase 2: Prefer GPU devices
	for _, d := range devices {
		if d.HasGpu {
			if best == nil {
				best = d
			} else if d.LlmPrefillToksPerS > best.LlmPrefillToksPerS {
				best = d
			}
		}
	}
	if best != nil {
		return best
	}

	// Phase 3: Fall back to CPU, prefer devices with TPS info, else first available
	for _, d := range devices {
		if best == nil {
			best = d
		} else if d.LlmPrefillToksPerS > 0 && d.LlmPrefillToksPerS > best.LlmPrefillToksPerS {
			best = d
		}
	}

	return best
}
