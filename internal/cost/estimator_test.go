package cost

import (
	"math"
	"testing"

	pb "github.com/edgecli/edgecli/proto"
)

func TestEstimatePlanCost_LLMSteps(t *testing.T) {
	// Device A: laptop defaults (prefill=300, decode=30)
	// Device B: fast device (prefill=600, decode=60)
	deviceA := &pb.DeviceInfo{
		DeviceId:   "device-a",
		DeviceName: "slow-laptop",
		Platform:   "linux",
		// No throughput set - will use defaults
	}
	deviceB := &pb.DeviceInfo{
		DeviceId:           "device-b",
		DeviceName:         "fast-device",
		Platform:           "linux",
		LlmPrefillToksPerS: 600,
		LlmDecodeToksPerS:  60,
	}

	// Plan with 2 LLM steps in same group (parallel)
	// Step 1: 500 prompt + 200 output
	// Step 2: 100 prompt + 50 output
	plan := &pb.Plan{
		Groups: []*pb.TaskGroup{
			{
				Index: 0,
				Tasks: []*pb.TaskSpec{
					{TaskId: "step-1", Kind: "LLM_GENERATE", PromptTokens: 500, MaxOutputTokens: 200},
					{TaskId: "step-2", Kind: "LLM_GENERATE", PromptTokens: 100, MaxOutputTokens: 50},
				},
			},
		},
	}

	estimator := NewEstimator()
	resp := estimator.EstimatePlanCost(plan, []*pb.DeviceInfo{deviceA, deviceB})

	// Verify device B is recommended (faster)
	if resp.RecommendedDeviceId != "device-b" {
		t.Errorf("expected device-b to be recommended, got %s", resp.RecommendedDeviceId)
	}

	// Calculate expected costs:
	// Device A (defaults: 300/30):
	//   Step 1: (500/300 + 200/30) * 1000 = (1.667 + 6.667) * 1000 = 8333.33ms
	//   Step 2: (100/300 + 50/30) * 1000 = (0.333 + 1.667) * 1000 = 2000ms
	//   Group cost (parallel): max(8333.33, 2000) = 8333.33ms
	// Device B (600/60):
	//   Step 1: (500/600 + 200/60) * 1000 = (0.833 + 3.333) * 1000 = 4166.67ms
	//   Step 2: (100/600 + 50/60) * 1000 = (0.167 + 0.833) * 1000 = 1000ms
	//   Group cost (parallel): max(4166.67, 1000) = 4166.67ms

	// Verify costs are approximately correct
	for _, dc := range resp.DeviceCosts {
		if dc.DeviceId == "device-a" {
			expectedA := 8333.33
			if math.Abs(dc.TotalMs-expectedA) > 1 {
				t.Errorf("device-a total: expected ~%.2f, got %.2f", expectedA, dc.TotalMs)
			}
		}
		if dc.DeviceId == "device-b" {
			expectedB := 4166.67
			if math.Abs(dc.TotalMs-expectedB) > 1 {
				t.Errorf("device-b total: expected ~%.2f, got %.2f", expectedB, dc.TotalMs)
			}
		}
	}

	// Verify best total matches device B
	expectedBest := 4166.67
	if math.Abs(resp.TotalPredictedMs-expectedBest) > 1 {
		t.Errorf("total predicted: expected ~%.2f, got %.2f", expectedBest, resp.TotalPredictedMs)
	}
}

func TestEstimatePlanCost_DefaultFallback(t *testing.T) {
	// Device with no throughput fields set (platform=linux)
	device := &pb.DeviceInfo{
		DeviceId:   "laptop-1",
		DeviceName: "my-laptop",
		Platform:   "linux",
	}

	plan := &pb.Plan{
		Groups: []*pb.TaskGroup{
			{
				Index: 0,
				Tasks: []*pb.TaskSpec{
					{TaskId: "task-1", Kind: "LLM_GENERATE", PromptTokens: 300, MaxOutputTokens: 30},
				},
			},
		},
	}

	estimator := NewEstimator()
	resp := estimator.EstimatePlanCost(plan, []*pb.DeviceInfo{device})

	// With laptop defaults (300/30):
	// (300/300 + 30/30) * 1000 = (1 + 1) * 1000 = 2000ms
	expected := 2000.0
	if math.Abs(resp.TotalPredictedMs-expected) > 0.01 {
		t.Errorf("expected %.2f ms, got %.2f ms", expected, resp.TotalPredictedMs)
	}

	// Verify notes indicate default throughput was used
	if len(resp.DeviceCosts) > 0 && len(resp.DeviceCosts[0].StepCosts) > 0 {
		notes := resp.DeviceCosts[0].StepCosts[0].Notes
		if notes == "" {
			t.Error("expected notes about using default throughput")
		}
	}
}

func TestEstimatePlanCost_PhoneDefaults(t *testing.T) {
	// Device with platform=android, no throughput set
	device := &pb.DeviceInfo{
		DeviceId:   "phone-1",
		DeviceName: "my-phone",
		Platform:   "android",
	}

	plan := &pb.Plan{
		Groups: []*pb.TaskGroup{
			{
				Index: 0,
				Tasks: []*pb.TaskSpec{
					{TaskId: "task-1", Kind: "LLM_GENERATE", PromptTokens: 120, MaxOutputTokens: 12},
				},
			},
		},
	}

	estimator := NewEstimator()
	resp := estimator.EstimatePlanCost(plan, []*pb.DeviceInfo{device})

	// With phone defaults (120/12):
	// (120/120 + 12/12) * 1000 = (1 + 1) * 1000 = 2000ms
	expected := 2000.0
	if math.Abs(resp.TotalPredictedMs-expected) > 0.01 {
		t.Errorf("expected %.2f ms, got %.2f ms", expected, resp.TotalPredictedMs)
	}
}

func TestEstimatePlanCost_UnknownStepType(t *testing.T) {
	device := &pb.DeviceInfo{
		DeviceId:   "device-1",
		DeviceName: "test-device",
		Platform:   "linux",
	}

	plan := &pb.Plan{
		Groups: []*pb.TaskGroup{
			{
				Index: 0,
				Tasks: []*pb.TaskSpec{
					{TaskId: "task-1", Kind: "CUSTOM_TASK", Input: "some input"},
				},
			},
		},
	}

	estimator := NewEstimator()
	resp := estimator.EstimatePlanCost(plan, []*pb.DeviceInfo{device})

	// Verify unknown_cost flag is set
	if !resp.HasUnknownCosts {
		t.Error("expected HasUnknownCosts to be true")
	}

	// Verify penalty is applied (250ms)
	expected := UnknownStepPenaltyMS
	if math.Abs(resp.TotalPredictedMs-expected) > 0.01 {
		t.Errorf("expected %.2f ms penalty, got %.2f ms", expected, resp.TotalPredictedMs)
	}

	// Verify step has unknown_cost flag
	if len(resp.DeviceCosts) > 0 && len(resp.DeviceCosts[0].StepCosts) > 0 {
		step := resp.DeviceCosts[0].StepCosts[0]
		if !step.UnknownCost {
			t.Error("expected step.UnknownCost to be true")
		}
	}
}

func TestEstimatePlanCost_ParallelTasks(t *testing.T) {
	device := &pb.DeviceInfo{
		DeviceId:           "device-1",
		DeviceName:         "test-device",
		Platform:           "linux",
		LlmPrefillToksPerS: 100,
		LlmDecodeToksPerS:  10,
	}

	// Group with 2 tasks that will have different costs
	// Task 1: 100 prompt + 100 output = (1 + 10) * 1000 = 11000ms
	// Task 2: 50 prompt + 50 output = (0.5 + 5) * 1000 = 5500ms
	plan := &pb.Plan{
		Groups: []*pb.TaskGroup{
			{
				Index: 0,
				Tasks: []*pb.TaskSpec{
					{TaskId: "slow-task", Kind: "LLM_GENERATE", PromptTokens: 100, MaxOutputTokens: 100},
					{TaskId: "fast-task", Kind: "LLM_GENERATE", PromptTokens: 50, MaxOutputTokens: 50},
				},
			},
		},
	}

	estimator := NewEstimator()
	resp := estimator.EstimatePlanCost(plan, []*pb.DeviceInfo{device})

	// Group cost should be max(11000, 5500) = 11000ms (parallel execution)
	expected := 11000.0
	if math.Abs(resp.TotalPredictedMs-expected) > 0.01 {
		t.Errorf("expected %.2f ms (max of parallel tasks), got %.2f ms", expected, resp.TotalPredictedMs)
	}
}

func TestEstimatePlanCost_SequentialGroups(t *testing.T) {
	device := &pb.DeviceInfo{
		DeviceId:           "device-1",
		DeviceName:         "test-device",
		Platform:           "linux",
		LlmPrefillToksPerS: 100,
		LlmDecodeToksPerS:  10,
	}

	// 2 groups that execute sequentially
	// Group 0: 50 prompt + 50 output = (0.5 + 5) * 1000 = 5500ms
	// Group 1: 30 prompt + 30 output = (0.3 + 3) * 1000 = 3300ms
	plan := &pb.Plan{
		Groups: []*pb.TaskGroup{
			{
				Index: 0,
				Tasks: []*pb.TaskSpec{
					{TaskId: "task-0", Kind: "LLM_GENERATE", PromptTokens: 50, MaxOutputTokens: 50},
				},
			},
			{
				Index: 1,
				Tasks: []*pb.TaskSpec{
					{TaskId: "task-1", Kind: "LLM_GENERATE", PromptTokens: 30, MaxOutputTokens: 30},
				},
			},
		},
	}

	estimator := NewEstimator()
	resp := estimator.EstimatePlanCost(plan, []*pb.DeviceInfo{device})

	// Total should be sum of groups = 5500 + 3300 = 8800ms
	expected := 8800.0
	if math.Abs(resp.TotalPredictedMs-expected) > 0.01 {
		t.Errorf("expected %.2f ms (sum of sequential groups), got %.2f ms", expected, resp.TotalPredictedMs)
	}
}

func TestEstimatePlanCost_RamCheck(t *testing.T) {
	// Device with limited RAM
	device := &pb.DeviceInfo{
		DeviceId:   "low-ram-device",
		DeviceName: "low-ram",
		Platform:   "linux",
		RamFreeMb:  1024, // Only 1GB free
	}

	plan := &pb.Plan{
		Groups: []*pb.TaskGroup{
			{
				Index: 0,
				Tasks: []*pb.TaskSpec{
					{TaskId: "task-1", Kind: "LLM_GENERATE", PromptTokens: 100, MaxOutputTokens: 50},
				},
			},
		},
	}

	estimator := NewEstimator()
	resp := estimator.EstimatePlanCost(plan, []*pb.DeviceInfo{device})

	// Default memory estimate is 2048MB, device has 1024MB
	if len(resp.DeviceCosts) > 0 {
		if resp.DeviceCosts[0].RamSufficient {
			t.Error("expected RamSufficient to be false for low-RAM device")
		}
	}
}

func TestEstimatePlanCost_EmptyPlan(t *testing.T) {
	device := &pb.DeviceInfo{
		DeviceId:   "device-1",
		DeviceName: "test-device",
		Platform:   "linux",
	}

	estimator := NewEstimator()

	// Empty plan
	resp := estimator.EstimatePlanCost(&pb.Plan{}, []*pb.DeviceInfo{device})
	if resp.TotalPredictedMs != 0 {
		t.Errorf("expected 0 for empty plan, got %f", resp.TotalPredictedMs)
	}

	// Nil plan
	resp = estimator.EstimatePlanCost(nil, []*pb.DeviceInfo{device})
	if resp.Warning == "" {
		t.Error("expected warning for nil plan")
	}

	// No devices
	resp = estimator.EstimatePlanCost(&pb.Plan{}, []*pb.DeviceInfo{})
	if resp.Warning == "" {
		t.Error("expected warning for no devices")
	}
}

func TestEstimatePlanCost_SysinfoAndEcho(t *testing.T) {
	device := &pb.DeviceInfo{
		DeviceId:   "device-1",
		DeviceName: "test-device",
		Platform:   "linux",
	}

	plan := &pb.Plan{
		Groups: []*pb.TaskGroup{
			{
				Index: 0,
				Tasks: []*pb.TaskSpec{
					{TaskId: "task-1", Kind: "SYSINFO"},
					{TaskId: "task-2", Kind: "ECHO", Input: "hello"},
				},
			},
		},
	}

	estimator := NewEstimator()
	resp := estimator.EstimatePlanCost(plan, []*pb.DeviceInfo{device})

	// SYSINFO and ECHO should have minimal cost (~10ms each)
	// Parallel execution means max(10, 10) = 10ms
	if resp.TotalPredictedMs > 20 {
		t.Errorf("expected low cost for SYSINFO/ECHO, got %f ms", resp.TotalPredictedMs)
	}

	// Should not be flagged as unknown cost
	if resp.HasUnknownCosts {
		t.Error("SYSINFO and ECHO should not have unknown costs")
	}
}
