package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/edgecli/edgecli/internal/tools"
	pb "github.com/edgecli/edgecli/proto"
)

// ToolExecutor executes LLM tools via gRPC
type ToolExecutor struct {
	grpcAddr     string
	sessionID    string
	selfDeviceID string
	conn         *grpc.ClientConn
	client       pb.OrchestratorServiceClient
}

// NewToolExecutor creates a new tool executor
func NewToolExecutor(grpcAddr string) (*ToolExecutor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server at %s: %w", grpcAddr, err)
	}

	client := pb.NewOrchestratorServiceClient(conn)

	// Create session
	sessionResp, err := client.CreateSession(context.Background(), &pb.AuthRequest{
		DeviceName:  "agent-executor",
		SecurityKey: "agent-internal",
	})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &ToolExecutor{
		grpcAddr:  grpcAddr,
		sessionID: sessionResp.SessionId,
		conn:      conn,
		client:    client,
	}, nil
}

// Close closes the gRPC connection
func (e *ToolExecutor) Close() error {
	if e.conn != nil {
		return e.conn.Close()
	}
	return nil
}

// Execute runs a tool call and returns the result as JSON
func (e *ToolExecutor) Execute(ctx context.Context, toolCall ToolCall) (interface{}, error) {
	switch toolCall.Function.Name {
	case "get_capabilities":
		return e.executeGetCapabilities(ctx, toolCall.Function.Arguments)
	case "execute_shell_cmd":
		return e.executeShellCmd(ctx, toolCall.Function.Arguments)
	case "get_file":
		return e.executeGetFile(ctx, toolCall.Function.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolCall.Function.Name)
	}
}

// GetCapabilitiesParams defines parameters for get_capabilities
type GetCapabilitiesParams struct {
	IncludeBenchmarks bool `json:"include_benchmarks"`
}

// GetCapabilitiesResult is the result of get_capabilities
type GetCapabilitiesResult struct {
	Devices []DeviceCapability `json:"devices"`
}

// DeviceCapability describes a device's capabilities
type DeviceCapability struct {
	DeviceID           string   `json:"device_id"`
	DeviceName         string   `json:"device_name"`
	Platform           string   `json:"platform"`
	Arch               string   `json:"arch"`
	HasCPU             bool     `json:"has_cpu"`
	HasGPU             bool     `json:"has_gpu"`
	HasNPU             bool     `json:"has_npu"`
	GRPCAddr           string   `json:"grpc_addr"`
	HTTPAddr           string   `json:"http_addr,omitempty"`
	CanScreenCapture   bool     `json:"can_screen_capture"`
	RAMFreeMB          uint64   `json:"ram_free_mb,omitempty"`
	Models             []string `json:"models"` // Empty for now, placeholder for future
	LLMPrefillToksPerS float64  `json:"llm_prefill_toks_per_s,omitempty"`
	LLMDecodeToksPerS  float64  `json:"llm_decode_toks_per_s,omitempty"`
}

func (e *ToolExecutor) executeGetCapabilities(ctx context.Context, argsJSON string) (interface{}, error) {
	var params GetCapabilitiesParams
	if argsJSON != "" && argsJSON != "{}" {
		if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}

	// Call ListDevices RPC
	resp, err := e.client.ListDevices(ctx, &pb.ListDevicesRequest{})
	if err != nil {
		return nil, fmt.Errorf("ListDevices RPC failed: %w", err)
	}

	// Convert to result format
	result := GetCapabilitiesResult{
		Devices: make([]DeviceCapability, len(resp.Devices)),
	}

	for i, d := range resp.Devices {
		cap := DeviceCapability{
			DeviceID:         d.DeviceId,
			DeviceName:       d.DeviceName,
			Platform:         d.Platform,
			Arch:             d.Arch,
			HasCPU:           d.HasCpu,
			HasGPU:           d.HasGpu,
			HasNPU:           d.HasNpu,
			GRPCAddr:         d.GrpcAddr,
			HTTPAddr:         d.HttpAddr,
			CanScreenCapture: d.CanScreenCapture,
			RAMFreeMB:        d.RamFreeMb,
			Models:           []string{}, // Placeholder
		}

		if params.IncludeBenchmarks {
			cap.LLMPrefillToksPerS = d.LlmPrefillToksPerS
			cap.LLMDecodeToksPerS = d.LlmDecodeToksPerS
		}

		result.Devices[i] = cap
	}

	return result, nil
}

// ExecuteShellCmdParams defines parameters for execute_shell_cmd
type ExecuteShellCmdParams struct {
	DeviceID   string `json:"device_id"`
	Command    string `json:"command"`
	TimeoutMs  int    `json:"timeout_ms"`
	WorkingDir string `json:"working_dir"`
}

// ExecuteShellCmdResult is the result of execute_shell_cmd
type ExecuteShellCmdResult struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	Command    string `json:"command"`
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

func (e *ToolExecutor) executeShellCmd(ctx context.Context, argsJSON string) (interface{}, error) {
	var params ExecuteShellCmdParams
	if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Command == "" {
		return ExecuteShellCmdResult{
			Error: "command is required",
		}, nil
	}

	// Validate command safety
	if err := tools.ValidateShellCommand(params.Command); err != nil {
		return ExecuteShellCmdResult{
			Command: params.Command,
			Error:   err.Error(),
		}, nil
	}

	// Parse command into executable and args
	// Simple split by spaces (doesn't handle quoted args properly, but works for basic cases)
	parts := strings.Fields(params.Command)
	if len(parts) == 0 {
		return ExecuteShellCmdResult{
			Error: "empty command",
		}, nil
	}

	cmd := parts[0]
	args := parts[1:]

	// Set up timeout
	timeoutMs := params.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if timeoutMs < 1000 {
		timeoutMs = 1000
	}
	if timeoutMs > 300000 {
		timeoutMs = 300000
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	// Determine routing policy
	policy := &pb.RoutingPolicy{
		Mode: pb.RoutingPolicy_BEST_AVAILABLE,
	}
	if params.DeviceID != "" {
		policy.Mode = pb.RoutingPolicy_FORCE_DEVICE_ID
		policy.DeviceId = params.DeviceID
	}

	start := time.Now()

	// Call ExecuteRoutedCommand RPC
	resp, err := e.client.ExecuteRoutedCommand(execCtx, &pb.RoutedCommandRequest{
		SessionId: e.sessionID,
		Policy:    policy,
		Command:   cmd,
		Args:      args,
	})
	if err != nil {
		return ExecuteShellCmdResult{
			Command: params.Command,
			Error:   fmt.Sprintf("RPC failed: %v", err),
		}, nil
	}

	duration := time.Since(start)

	return ExecuteShellCmdResult{
		DeviceID:   resp.SelectedDeviceId,
		DeviceName: resp.SelectedDeviceName,
		Command:    params.Command,
		ExitCode:   int(resp.Output.ExitCode),
		Stdout:     resp.Output.Stdout,
		Stderr:     resp.Output.Stderr,
		DurationMs: duration.Milliseconds(),
	}, nil
}

// GetFileParams defines parameters for get_file
type GetFileParams struct {
	DeviceID string `json:"device_id"`
	Path     string `json:"path"`
	ReadMode string `json:"read_mode"`
	MaxBytes int    `json:"max_bytes"`
	Offset   int64  `json:"offset"`
	Length   int64  `json:"length"`
}

// GetFileResult is the result of get_file
type GetFileResult struct {
	DeviceID       string `json:"device_id,omitempty"`
	Path           string `json:"path"`
	ReadMode       string `json:"read_mode"`
	SizeBytes      int64  `json:"size_bytes"`
	BytesReturned  int64  `json:"bytes_returned"`
	Truncated      bool   `json:"truncated"`
	ContentBase64  string `json:"content_base64,omitempty"`
	ContentPreview string `json:"content_preview,omitempty"`
	Error          string `json:"error,omitempty"`
}

func (e *ToolExecutor) executeGetFile(ctx context.Context, argsJSON string) (interface{}, error) {
	var params GetFileParams
	if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Path == "" {
		return GetFileResult{
			Error: "path is required",
		}, nil
	}

	// Map read mode string to proto enum
	mode := pb.ReadMode_READ_MODE_FULL
	switch strings.ToLower(params.ReadMode) {
	case "head":
		mode = pb.ReadMode_READ_MODE_HEAD
	case "tail":
		mode = pb.ReadMode_READ_MODE_TAIL
	case "range":
		mode = pb.ReadMode_READ_MODE_RANGE
	case "full", "":
		mode = pb.ReadMode_READ_MODE_FULL
	}

	maxBytes := int32(params.MaxBytes)
	if maxBytes <= 0 {
		maxBytes = 65536
	}

	// Call ReadFile RPC
	resp, err := e.client.ReadFile(ctx, &pb.ReadFileRequest{
		SessionId: e.sessionID,
		DeviceId:  params.DeviceID,
		Path:      params.Path,
		Mode:      mode,
		MaxBytes:  maxBytes,
		Offset:    params.Offset,
		Length:    params.Length,
	})
	if err != nil {
		return GetFileResult{
			Path:  params.Path,
			Error: fmt.Sprintf("RPC failed: %v", err),
		}, nil
	}

	if resp.Error != "" {
		return GetFileResult{
			Path:  params.Path,
			Error: resp.Error,
		}, nil
	}

	// Encode content as base64
	contentBase64 := ""
	if len(resp.Content) > 0 {
		contentBase64 = base64.StdEncoding.EncodeToString(resp.Content)
	}

	return GetFileResult{
		DeviceID:       params.DeviceID,
		Path:           params.Path,
		ReadMode:       params.ReadMode,
		SizeBytes:      resp.SizeBytes,
		BytesReturned:  resp.BytesReturned,
		Truncated:      resp.Truncated,
		ContentBase64:  contentBase64,
		ContentPreview: resp.ContentPreview,
	}, nil
}
