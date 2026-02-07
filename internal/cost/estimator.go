// Package cost provides plan cost estimation for the orchestrator.
package cost

import (
	pb "github.com/edgecli/edgecli/proto"
)

// Default throughput values (tokens/sec) for different platforms
const (
	DefaultLaptopPrefillTPS = 300.0 // macos, windows, linux
	DefaultLaptopDecodeTPS  = 30.0
	DefaultPhonePrefillTPS  = 120.0 // android, ios
	DefaultPhoneDecodeTPS   = 12.0
	UnknownStepPenaltyMS    = 250.0 // penalty for unknown step types
	DefaultMemoryMB         = 2048  // conservative LLM memory estimate
)

// Estimator calculates cost estimates for execution plans.
type Estimator struct{}

// NewEstimator creates a new cost estimator.
func NewEstimator() *Estimator {
	return &Estimator{}
}

// EstimatePlanCost evaluates a plan against multiple devices and returns
// cost estimates with a recommendation for the best device.
func (e *Estimator) EstimatePlanCost(plan *pb.Plan, devices []*pb.DeviceInfo) *pb.PlanCostResponse {
	if plan == nil || len(devices) == 0 {
		return &pb.PlanCostResponse{
			Warning: "no plan or devices provided",
		}
	}

	var bestDevice *pb.DeviceInfo
	var bestCost float64 = -1
	hasUnknownCosts := false
	deviceCosts := make([]*pb.DeviceCostEstimate, 0, len(devices))

	for _, device := range devices {
		estimate := e.estimateForDevice(plan, device)
		deviceCosts = append(deviceCosts, estimate)

		// Check for unknown costs
		for _, step := range estimate.StepCosts {
			if step.UnknownCost {
				hasUnknownCosts = true
			}
		}

		// Track best device (lowest total cost)
		if bestCost < 0 || estimate.TotalMs < bestCost {
			bestCost = estimate.TotalMs
			bestDevice = device
		}
	}

	resp := &pb.PlanCostResponse{
		DeviceCosts:     deviceCosts,
		HasUnknownCosts: hasUnknownCosts,
	}

	if bestDevice != nil {
		resp.TotalPredictedMs = bestCost
		resp.RecommendedDeviceId = bestDevice.DeviceId
		resp.RecommendedDeviceName = bestDevice.DeviceName
	}

	if hasUnknownCosts {
		resp.Warning = "some steps have unknown cost (using penalty estimate)"
	}

	return resp
}

// estimateForDevice calculates cost for running the plan on a specific device.
func (e *Estimator) estimateForDevice(plan *pb.Plan, device *pb.DeviceInfo) *pb.DeviceCostEstimate {
	var totalMs float64
	var peakMemoryMB uint64
	stepCosts := make([]*pb.StepCostEstimate, 0)

	for _, group := range plan.Groups {
		groupCost, groupSteps, groupMemory := e.estimateGroupCost(group, device)
		totalMs += groupCost // Groups run sequentially
		stepCosts = append(stepCosts, groupSteps...)
		if groupMemory > peakMemoryMB {
			peakMemoryMB = groupMemory
		}
	}

	// Check RAM sufficiency
	ramSufficient := true
	if device.RamFreeMb > 0 && peakMemoryMB > device.RamFreeMb {
		ramSufficient = false
	}

	return &pb.DeviceCostEstimate{
		DeviceId:           device.DeviceId,
		DeviceName:         device.DeviceName,
		TotalMs:            totalMs,
		StepCosts:          stepCosts,
		EstimatedPeakRamMb: peakMemoryMB,
		RamSufficient:      ramSufficient,
	}
}

// estimateGroupCost calculates cost for a task group.
// Tasks within a group run in parallel, so we take the max cost.
func (e *Estimator) estimateGroupCost(group *pb.TaskGroup, device *pb.DeviceInfo) (float64, []*pb.StepCostEstimate, uint64) {
	var maxCost float64
	var maxMemory uint64
	stepCosts := make([]*pb.StepCostEstimate, 0, len(group.Tasks))

	for _, task := range group.Tasks {
		stepCost := e.estimateStep(task, device)
		stepCosts = append(stepCosts, stepCost)

		// Parallel tasks: take the max latency
		if stepCost.PredictedMs > maxCost {
			maxCost = stepCost.PredictedMs
		}

		// Memory: take the max (all tasks run concurrently)
		memMB := uint64(stepCost.PredictedMemoryMb)
		if memMB > maxMemory {
			maxMemory = memMB
		}
	}

	return maxCost, stepCosts, maxMemory
}

// estimateStep calculates cost for a single task based on its kind.
func (e *Estimator) estimateStep(task *pb.TaskSpec, device *pb.DeviceInfo) *pb.StepCostEstimate {
	switch task.Kind {
	case "LLM_GENERATE":
		return e.estimateLLMStep(task, device)
	case "SYSINFO", "ECHO":
		// Fast local operations - minimal cost
		return &pb.StepCostEstimate{
			TaskId:            task.TaskId,
			Kind:              task.Kind,
			PredictedMs:       10, // ~10ms for local ops
			PredictedMemoryMb: 0,
			UnknownCost:       false,
		}
	default:
		return e.estimateUnknownStep(task)
	}
}

// estimateLLMStep calculates latency for LLM_GENERATE step.
// Formula: predicted_ms = (prompt_tokens / prefill_tps + max_output_tokens / decode_tps) * 1000
func (e *Estimator) estimateLLMStep(task *pb.TaskSpec, device *pb.DeviceInfo) *pb.StepCostEstimate {
	prefillTPS, decodeTPS := e.getDeviceThroughput(device)

	var latencyMS float64
	var notes string

	// Calculate latency based on tokens
	if task.PromptTokens > 0 || task.MaxOutputTokens > 0 {
		prefillTime := float64(task.PromptTokens) / prefillTPS
		decodeTime := float64(task.MaxOutputTokens) / decodeTPS
		latencyMS = (prefillTime + decodeTime) * 1000
	} else {
		// No token info provided, use a default estimate
		latencyMS = 1000 // 1 second default
		notes = "no token counts provided, using default estimate"
	}

	// Check if we used defaults
	if device.LlmPrefillToksPerS == 0 || device.LlmDecodeToksPerS == 0 {
		if notes != "" {
			notes += "; "
		}
		notes += "using default throughput for platform"
	}

	return &pb.StepCostEstimate{
		TaskId:            task.TaskId,
		Kind:              task.Kind,
		PredictedMs:       latencyMS,
		PredictedMemoryMb: DefaultMemoryMB,
		UnknownCost:       false,
		Notes:             notes,
	}
}

// getDeviceThroughput returns throughput values for a device, using defaults if not reported.
func (e *Estimator) getDeviceThroughput(device *pb.DeviceInfo) (prefill, decode float64) {
	// Use device-reported values if available
	if device.LlmPrefillToksPerS > 0 && device.LlmDecodeToksPerS > 0 {
		return device.LlmPrefillToksPerS, device.LlmDecodeToksPerS
	}

	// Fallback to defaults based on platform
	switch device.Platform {
	case "android", "ios":
		return DefaultPhonePrefillTPS, DefaultPhoneDecodeTPS
	default:
		// macos, windows, linux, or unknown
		return DefaultLaptopPrefillTPS, DefaultLaptopDecodeTPS
	}
}

// estimateUnknownStep returns a penalty estimate for unrecognized step types.
func (e *Estimator) estimateUnknownStep(task *pb.TaskSpec) *pb.StepCostEstimate {
	return &pb.StepCostEstimate{
		TaskId:            task.TaskId,
		Kind:              task.Kind,
		PredictedMs:       UnknownStepPenaltyMS,
		PredictedMemoryMb: 0,
		UnknownCost:       true,
		Notes:             "unknown step type, using penalty estimate",
	}
}
