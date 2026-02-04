// Package brain provides integration with the Windows AI CLI for AI-powered plan generation.
package brain

import (
	"os"

	pb "github.com/edgecli/edgecli/proto"
)

// Environment variables for brain configuration.
const (
	// EnvWindowsAiCliPath is the path to the WindowsAiCli.exe executable.
	EnvWindowsAiCliPath = "WINDOWS_AI_CLI_PATH"

	// EnvUseWindowsAiPlanner enables/disables the Windows AI planner.
	// Set to "true" to enable.
	EnvUseWindowsAiPlanner = "USE_WINDOWS_AI_PLANNER"
)

// Brain provides AI-powered plan generation and text processing
// by shelling out to the Windows AI CLI.
type Brain struct {
	cliPath string
	enabled bool
}

// New creates a new Brain instance configured from environment variables.
func New() *Brain {
	return &Brain{
		cliPath: os.Getenv(EnvWindowsAiCliPath),
		enabled: os.Getenv(EnvUseWindowsAiPlanner) == "true",
	}
}

// IsAvailable returns true if the brain is configured and available.
// This is platform-specific: returns true on Windows when enabled and CLI path is set.
func (b *Brain) IsAvailable() bool {
	return isAvailable(b)
}

// GeneratePlan generates an execution plan for the given devices.
// Returns the generated plan, reduce spec, and any error.
// Falls back to returning nil (allowing the caller to use default plan generation)
// if the CLI is unavailable or fails.
func (b *Brain) GeneratePlan(text string, devices []*pb.DeviceInfo, maxWorkers int) (*pb.Plan, *pb.ReduceSpec, error) {
	return generatePlan(b, text, devices, maxWorkers)
}

// Summarize summarizes the given text using Windows AI.
// Returns the summary, whether AI was used, and any error.
func (b *Brain) Summarize(text string) (string, bool, error) {
	return summarize(b, text)
}

// PlanRequest is the JSON request format for the plan command.
type PlanRequest struct {
	Text       string       `json:"text"`
	MaxWorkers int          `json:"max_workers"`
	Devices    []DeviceInfo `json:"devices"`
}

// DeviceInfo is the JSON format for device information.
type DeviceInfo struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	HasNpu     bool   `json:"has_npu"`
	HasGpu     bool   `json:"has_gpu"`
	HasCpu     bool   `json:"has_cpu"`
	GrpcAddr   string `json:"grpc_addr"`
}

// PlanResponse is the JSON response format for the plan command.
type PlanResponse struct {
	Ok     bool        `json:"ok"`
	Plan   *Plan       `json:"plan,omitempty"`
	Reduce *ReduceSpec `json:"reduce,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// Plan is the JSON format for an execution plan.
type Plan struct {
	Groups []TaskGroup `json:"groups"`
}

// TaskGroup is the JSON format for a task group.
type TaskGroup struct {
	Index int        `json:"index"`
	Tasks []TaskSpec `json:"tasks"`
}

// TaskSpec is the JSON format for a task specification.
type TaskSpec struct {
	TaskID         string `json:"task_id"`
	Kind           string `json:"kind"`
	Input          string `json:"input"`
	TargetDeviceID string `json:"target_device_id"`
}

// ReduceSpec is the JSON format for a reduce specification.
type ReduceSpec struct {
	Kind string `json:"kind"`
}

// SummarizeResponse is the JSON response format for the summarize command.
type SummarizeResponse struct {
	Ok      bool   `json:"ok"`
	Summary string `json:"summary,omitempty"`
	UsedAi  bool   `json:"used_ai"`
	Error   string `json:"error,omitempty"`
}

// CapabilitiesResponse is the JSON response format for the capabilities command.
type CapabilitiesResponse struct {
	Ok       bool     `json:"ok"`
	Features []string `json:"features,omitempty"`
	Notes    string   `json:"notes,omitempty"`
	Error    string   `json:"error,omitempty"`
}

// convertDevicesToJSON converts proto DeviceInfo to JSON format.
func convertDevicesToJSON(devices []*pb.DeviceInfo) []DeviceInfo {
	result := make([]DeviceInfo, len(devices))
	for i, d := range devices {
		result[i] = DeviceInfo{
			DeviceID:   d.DeviceId,
			DeviceName: d.DeviceName,
			HasNpu:     d.HasNpu,
			HasGpu:     d.HasGpu,
			HasCpu:     d.HasCpu,
			GrpcAddr:   d.GrpcAddr,
		}
	}
	return result
}

// convertPlanToProto converts JSON Plan to proto format.
func convertPlanToProto(plan *Plan) *pb.Plan {
	if plan == nil {
		return nil
	}

	groups := make([]*pb.TaskGroup, len(plan.Groups))
	for i, g := range plan.Groups {
		tasks := make([]*pb.TaskSpec, len(g.Tasks))
		for j, t := range g.Tasks {
			tasks[j] = &pb.TaskSpec{
				TaskId:         t.TaskID,
				Kind:           t.Kind,
				Input:          t.Input,
				TargetDeviceId: t.TargetDeviceID,
			}
		}
		groups[i] = &pb.TaskGroup{
			Index: int32(g.Index),
			Tasks: tasks,
		}
	}

	return &pb.Plan{Groups: groups}
}

// convertReduceToProto converts JSON ReduceSpec to proto format.
func convertReduceToProto(reduce *ReduceSpec) *pb.ReduceSpec {
	if reduce == nil {
		return nil
	}
	return &pb.ReduceSpec{Kind: reduce.Kind}
}
