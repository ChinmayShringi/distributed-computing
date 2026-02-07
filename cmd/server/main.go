// Package main implements the gRPC orchestrator server with embedded HTTP web UI
package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/kbinani/screenshot"

	"github.com/edgecli/edgecli/internal/allowlist"
	"github.com/edgecli/edgecli/internal/brain"
	"github.com/edgecli/edgecli/internal/chatmem"
	"github.com/edgecli/edgecli/internal/cost"
	"github.com/edgecli/edgecli/internal/deviceid"
	"github.com/edgecli/edgecli/internal/discovery"
	"github.com/edgecli/edgecli/internal/exec"
	"github.com/edgecli/edgecli/internal/jobs"
	"github.com/edgecli/edgecli/internal/llm"
	"github.com/edgecli/edgecli/internal/metrics"
	"github.com/edgecli/edgecli/internal/qaihub"
	"github.com/edgecli/edgecli/internal/registry"
	"github.com/edgecli/edgecli/internal/sysinfo"
	"github.com/edgecli/edgecli/internal/transfer"
	"github.com/edgecli/edgecli/internal/webrtcstream"
	pb "github.com/edgecli/edgecli/proto"
)

//go:embed webui/*
var staticFS embed.FS

const (
	defaultAddr         = ":50051"
	defaultWebAddr      = ":8080"
	defaultBulkHTTPAddr = ":8081"
	defaultSharedDir    = "./shared"
	defaultBulkTTL      = 60
	defaultDevKey       = "dev"
	remoteDialTimeout   = 15 * time.Second
	webRequestTimeout   = 30 * time.Second
)

// Session represents an authenticated client session
type Session struct {
	ID          string
	DeviceName  string
	HostName    string
	ConnectedAt time.Time
}

// OrchestratorServer implements the OrchestratorService gRPC interface
type OrchestratorServer struct {
	pb.UnimplementedOrchestratorServiceServer
	sessions      map[string]*Session
	mu            sync.RWMutex
	runner        *exec.Runner
	registry      *registry.Registry
	jobManager    *jobs.Manager
	webrtcManager *webrtcstream.Manager
	brain         *brain.Brain // Windows AI CLI planner (platform-specific)
	llmProvider   llm.Provider // Cross-platform LLM planner (openai_compat, etc.)
	chatMem       *chatmem.ChatMemory
	selfDeviceID  string
	selfAddr      string
	ticketManager *transfer.Manager
	sharedRoot    string
	bulkHTTPAddr  string
	metricsStore  *metrics.MetricsStore
}

// WebHandler handles HTTP requests using in-process calls to OrchestratorServer
type WebHandler struct {
	orchestrator *OrchestratorServer
	devKey       string
	llm          llm.Provider     // nil if disabled
	chat         llm.ChatProvider // nil if disabled
	agent        *llm.AgentLoop   // LLM tool-calling agent (nil if disabled)
	qaihubClient *qaihub.Client   // qai-hub CLI wrapper
}

// ---- HTTP Request/Response Types ----

// DeviceResponse is the JSON response for /api/devices
type DeviceResponse struct {
	DeviceID          string   `json:"device_id"`
	DeviceName        string   `json:"device_name"`
	Platform          string   `json:"platform"`
	Arch              string   `json:"arch"`
	Capabilities      []string `json:"capabilities"`
	GRPCAddr          string   `json:"grpc_addr"`
	CanScreenCapture  bool     `json:"can_screen_capture"`
	HttpAddr          string   `json:"http_addr"`
	HasLocalModel     bool     `json:"has_local_model"`
	LocalModelName    string   `json:"local_model_name,omitempty"`
	LocalChatEndpoint string   `json:"local_chat_endpoint,omitempty"`
}

// RoutedCmdRequest is the JSON request for /api/routed-cmd
type RoutedCmdRequest struct {
	Cmd           string   `json:"cmd"`
	Args          []string `json:"args"`
	Policy        string   `json:"policy"`
	ForceDeviceID string   `json:"force_device_id"`
}

// RoutedCmdResponse is the JSON response for /api/routed-cmd
type RoutedCmdResponse struct {
	SelectedDeviceName string  `json:"selected_device_name"`
	SelectedDeviceID   string  `json:"selected_device_id"`
	SelectedDeviceAddr string  `json:"selected_device_addr"`
	ExecutedLocally    bool    `json:"executed_locally"`
	TotalTimeMs        float64 `json:"total_time_ms"`
	ExitCode           int32   `json:"exit_code"`
	Stdout             string  `json:"stdout"`
	Stderr             string  `json:"stderr"`
}

// AssistantRequest is the JSON request for /api/assistant
type AssistantRequest struct {
	Text string `json:"text"`
}

// AssistantResponse is the JSON response for /api/assistant
type AssistantResponse struct {
	Reply string      `json:"reply"`
	Raw   interface{} `json:"raw,omitempty"`
	Mode  string      `json:"mode,omitempty"`
	JobID string      `json:"job_id,omitempty"`
	Plan  interface{} `json:"plan,omitempty"`
}

// ErrorResponse is the JSON error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// PreviewPlanRequest is the JSON request for /api/plan
type PreviewPlanRequest struct {
	Text       string `json:"text"`
	MaxWorkers int32  `json:"max_workers"`
}

// PreviewPlanResponse is the JSON response for /api/plan
type PreviewPlanResponse struct {
	UsedAi    bool        `json:"used_ai"`
	Notes     string      `json:"notes"`
	Rationale string      `json:"rationale"`
	Plan      interface{} `json:"plan"`
	Reduce    interface{} `json:"reduce"`
}

// PlanCostRequest is the JSON request for /api/plan-cost
type PlanCostRequest struct {
	Plan      interface{} `json:"plan"`
	DeviceIDs []string    `json:"device_ids"`
}

// StepCostResponse is the cost for a single step
type StepCostResponse struct {
	TaskID            string  `json:"task_id"`
	Kind              string  `json:"kind"`
	PredictedMs       float64 `json:"predicted_ms"`
	PredictedMemoryMB float64 `json:"predicted_memory_mb"`
	UnknownCost       bool    `json:"unknown_cost"`
	Notes             string  `json:"notes,omitempty"`
}

// DeviceCostResponse is the cost breakdown for a single device
type DeviceCostResponse struct {
	DeviceID           string             `json:"device_id"`
	DeviceName         string             `json:"device_name"`
	TotalMs            float64            `json:"total_ms"`
	StepCosts          []StepCostResponse `json:"step_costs"`
	EstimatedPeakRAMMB uint64             `json:"estimated_peak_ram_mb"`
	RAMSufficient      bool               `json:"ram_sufficient"`
}

// PlanCostResponse is the JSON response for /api/plan-cost
type PlanCostResponse struct {
	TotalPredictedMs      float64              `json:"total_predicted_ms"`
	DeviceCosts           []DeviceCostResponse `json:"device_costs"`
	RecommendedDeviceID   string               `json:"recommended_device_id"`
	RecommendedDeviceName string               `json:"recommended_device_name"`
	HasUnknownCosts       bool                 `json:"has_unknown_costs"`
	Warning               string               `json:"warning,omitempty"`
}

// DownloadRequest is the JSON request for /api/request-download
type DownloadRequest struct {
	DeviceID string `json:"device_id"`
	Path     string `json:"path"`
}

// DownloadResponse is the JSON response for /api/request-download
type DownloadResponse struct {
	DownloadURL   string `json:"download_url"`
	Filename      string `json:"filename"`
	SizeBytes     int64  `json:"size_bytes"`
	ExpiresUnixMs int64  `json:"expires_unix_ms"`
}

// SubmitJobRequest is the JSON request for /api/submit-job
type SubmitJobRequest struct {
	Text       string `json:"text"`
	MaxWorkers int32  `json:"max_workers"`
}

// JobInfoResponse is the JSON response for /api/submit-job
type JobInfoResponse struct {
	JobID     string `json:"job_id"`
	CreatedAt int64  `json:"created_at"`
	Summary   string `json:"summary"`
}

// TaskStatusResponse represents a task in the job status
type TaskStatusResponse struct {
	TaskID             string `json:"task_id"`
	AssignedDeviceID   string `json:"assigned_device_id"`
	AssignedDeviceName string `json:"assigned_device_name"`
	State              string `json:"state"`
	Result             string `json:"result"`
	Error              string `json:"error"`
}

// JobStatusResponse is the JSON response for /api/job
type JobStatusResponse struct {
	JobID        string               `json:"job_id"`
	State        string               `json:"state"`
	Tasks        []TaskStatusResponse `json:"tasks"`
	FinalResult  string               `json:"final_result"`
	CurrentGroup int32                `json:"current_group"`
	TotalGroups  int32                `json:"total_groups"`
}

// StreamStartRequest is the JSON request for /api/stream/start
type StreamStartRequest struct {
	Policy        string `json:"policy"`
	ForceDeviceID string `json:"force_device_id"`
	FPS           int32  `json:"fps"`
	Quality       int32  `json:"quality"`
	MonitorIndex  int32  `json:"monitor_index"`
}

// StreamStartResponse is the JSON response for /api/stream/start
type StreamStartResponse struct {
	SelectedDeviceID   string `json:"selected_device_id"`
	SelectedDeviceName string `json:"selected_device_name"`
	SelectedDeviceAddr string `json:"selected_device_addr"`
	StreamID           string `json:"stream_id"`
	OfferSDP           string `json:"offer_sdp"`
}

// StreamAnswerRequest is the JSON request for /api/stream/answer
type StreamAnswerRequest struct {
	SelectedDeviceAddr string `json:"selected_device_addr"`
	StreamID           string `json:"stream_id"`
	AnswerSDP          string `json:"answer_sdp"`
}

// StreamStopRequest is the JSON request for /api/stream/stop
type StreamStopRequest struct {
	SelectedDeviceAddr string `json:"selected_device_addr"`
	StreamID           string `json:"stream_id"`
}

// CompileRequest is the JSON request for /api/qaihub/compile
type CompileRequest struct {
	ONNXPath string `json:"onnx_path"`
	Target   string `json:"target"`
	Runtime  string `json:"runtime"`
}

// ChatRequest is the JSON request for /api/chat
type ChatRequest struct {
	Messages []llm.ChatMessage `json:"messages"`
}

// ChatResponse is the JSON response for /api/chat
type ChatResponse struct {
	Reply string `json:"reply"`
}

// AgentRequest is the JSON request for /api/agent
type AgentRequest struct {
	Message string `json:"message"`
}

// AgentToolCallInfo records a tool call made during agent execution
type AgentToolCallInfo struct {
	Iteration int    `json:"iteration"`
	ToolName  string `json:"tool_name"`
	Arguments string `json:"arguments"`
	ResultLen int    `json:"result_len"`
}

// AgentResponseJSON is the JSON response for /api/agent
type AgentResponseJSON struct {
	Reply      string              `json:"reply"`
	Iterations int                 `json:"iterations"`
	ToolCalls  []AgentToolCallInfo `json:"tool_calls,omitempty"`
	Error      string              `json:"error,omitempty"`
}

// QaihubJobStatusRequest is the JSON request for /api/qaihub/job-status
type QaihubJobStatusRequest struct {
	JobID string `json:"job_id"`
}

// QaihubSubmitCompileRequest is the JSON request for /api/qaihub/submit-compile
type QaihubSubmitCompileRequest struct {
	Model      string `json:"model"`
	DeviceName string `json:"device_name"`
	Options    string `json:"options"`
}

// NewOrchestratorServer creates a new server instance
func NewOrchestratorServer(addr string) *OrchestratorServer {
	// Get device ID from env or generate/persist one
	selfID := os.Getenv("DEVICE_ID")
	if selfID == "" {
		var err error
		selfID, err = deviceid.GetOrCreate()
		if err != nil {
			log.Printf("[WARN] Could not get device ID: %v", err)
			selfID = uuid.New().String()
		}
	}

	// Normalize selfAddr for local demo
	// If addr is like ":50051", convert to "127.0.0.1:50051" for local dialing
	selfAddr := addr
	if strings.HasPrefix(addr, ":") {
		selfAddr = "127.0.0.1" + addr
	}

	// Shared root directory
	sharedDir := os.Getenv("SHARED_DIR")
	if sharedDir == "" {
		sharedDir = defaultSharedDir
	}
	sharedRootAbs, err := filepath.Abs(sharedDir)
	if err != nil {
		log.Fatalf("[FATAL] Cannot resolve shared dir: %v", err)
	}

	// Bulk TTL
	bulkTTL := defaultBulkTTL
	if ttlStr := os.Getenv("BULK_TTL_SECONDS"); ttlStr != "" {
		if parsed, parseErr := strconv.Atoi(ttlStr); parseErr == nil && parsed > 0 {
			bulkTTL = parsed
		}
	}

	// Bulk HTTP address
	bulkHTTPAddr := os.Getenv("BULK_HTTP_ADDR")
	if bulkHTTPAddr == "" {
		bulkHTTPAddr = defaultBulkHTTPAddr
	}

	// Load chat memory from file
	chatMemPath, _ := chatmem.DefaultFilePath()
	chatMem, err := chatmem.LoadFromFile(chatMemPath)
	if err != nil {
		log.Printf("[WARN] Could not load chat memory from %s: %v", chatMemPath, err)
		chatMem = chatmem.New()
	} else {
		log.Printf("[INFO] Loaded chat memory from %s (%d messages)", chatMemPath, len(chatMem.GetMessages()))
	}

	return &OrchestratorServer{
		sessions:      make(map[string]*Session),
		runner:        exec.NewRunner(),
		registry:      registry.NewRegistry(),
		jobManager:    jobs.NewManager(),
		webrtcManager: webrtcstream.NewManager(),
		brain:         brain.New(),
		chatMem:       chatMem,
		selfDeviceID:  selfID,
		selfAddr:      selfAddr,
		ticketManager: transfer.NewManager(time.Duration(bulkTTL) * time.Second),
		sharedRoot:    sharedRootAbs,
		bulkHTTPAddr:  bulkHTTPAddr,
		metricsStore:  metrics.NewMetricsStore(),
	}
}

// registerSelf registers this server as a device in its own registry
func (s *OrchestratorServer) registerSelf() {
	selfInfo := s.getSelfDeviceInfo()
	s.registry.Upsert(selfInfo)
	log.Printf("[INFO] Self-registered as device: id=%s name=%s grpc=%s http=%s",
		selfInfo.DeviceId, selfInfo.DeviceName, selfInfo.GrpcAddr, selfInfo.HttpAddr)
}

// CreateSession authenticates a client and creates a new session
func (s *OrchestratorServer) CreateSession(ctx context.Context, req *pb.AuthRequest) (*pb.SessionInfo, error) {
	// Validate security key (simple non-empty check for now)
	if req.SecurityKey == "" {
		log.Printf("[ERROR] CreateSession: empty security key from device %q", req.DeviceName)
		return nil, status.Error(codes.Unauthenticated, "security_key is required")
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Get hostname
	hostName, err := os.Hostname()
	if err != nil {
		hostName = "unknown"
	}

	// Create session
	session := &Session{
		ID:          sessionID,
		DeviceName:  req.DeviceName,
		HostName:    hostName,
		ConnectedAt: time.Now(),
	}

	// Store session
	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	log.Printf("[INFO] Session created: id=%s device=%s host=%s", sessionID, req.DeviceName, hostName)

	return &pb.SessionInfo{
		SessionId:   sessionID,
		HostName:    hostName,
		ConnectedAt: session.ConnectedAt.Unix(),
	}, nil
}

// Heartbeat verifies a session is still valid
func (s *OrchestratorServer) Heartbeat(ctx context.Context, req *pb.SessionInfo) (*pb.Empty, error) {
	s.mu.RLock()
	session, exists := s.sessions[req.SessionId]
	s.mu.RUnlock()

	if !exists {
		log.Printf("[ERROR] Heartbeat: session not found: %s", req.SessionId)
		return nil, status.Error(codes.NotFound, "session not found")
	}

	log.Printf("[DEBUG] Heartbeat: session=%s device=%s", req.SessionId, session.DeviceName)
	return &pb.Empty{}, nil
}

// ExecuteCommand executes an allowed command and returns the result
func (s *OrchestratorServer) ExecuteCommand(ctx context.Context, req *pb.CommandRequest) (*pb.CommandResponse, error) {
	// Verify session
	s.mu.RLock()
	session, exists := s.sessions[req.SessionId]
	s.mu.RUnlock()

	if !exists {
		log.Printf("[ERROR] ExecuteCommand: session not found: %s", req.SessionId)
		return nil, status.Error(codes.Unauthenticated, "session not found")
	}

	// Validate command against allowlist
	cmdSpec, err := allowlist.ValidateCommand(req.Command, req.Args)
	if err != nil {
		log.Printf("[ERROR] ExecuteCommand: command rejected: session=%s cmd=%s error=%v",
			req.SessionId, req.Command, err)
		return &pb.CommandResponse{
			ExitCode: 1,
			Stdout:   "",
			Stderr:   fmt.Sprintf("command not allowed: %v", err),
		}, nil
	}

	// Execute command using the internal runner
	log.Printf("[INFO] ExecuteCommand: session=%s device=%s cmd=%s args=%v",
		req.SessionId, session.DeviceName, cmdSpec.Executable, cmdSpec.Args)

	result := s.runner.Run(ctx, cmdSpec.Executable, cmdSpec.Args...)

	log.Printf("[INFO] ExecuteCommand completed: session=%s cmd=%s exit_code=%d duration=%s",
		req.SessionId, req.Command, result.ExitCode, result.Duration)

	return &pb.CommandResponse{
		ExitCode: int32(result.ExitCode),
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
	}, nil
}

// RegisterDevice registers a device in the registry
func (s *OrchestratorServer) RegisterDevice(ctx context.Context, req *pb.DeviceInfo) (*pb.DeviceAck, error) {
	// Validate required fields
	if req.DeviceId == "" {
		log.Printf("[ERROR] RegisterDevice: empty device_id")
		return nil, status.Error(codes.InvalidArgument, "device_id is required")
	}
	if req.GrpcAddr == "" {
		log.Printf("[ERROR] RegisterDevice: empty grpc_addr for device %s", req.DeviceId)
		return nil, status.Error(codes.InvalidArgument, "grpc_addr is required")
	}

	// Register device
	registeredAt := s.registry.Upsert(req)

	log.Printf("[INFO] Device registered: id=%s name=%s platform=%s arch=%s addr=%s",
		req.DeviceId, req.DeviceName, req.Platform, req.Arch, req.GrpcAddr)

	return &pb.DeviceAck{
		Ok:           true,
		RegisteredAt: registeredAt.Unix(),
	}, nil
}

// ListDevices returns all registered devices
func (s *OrchestratorServer) ListDevices(ctx context.Context, req *pb.ListDevicesRequest) (*pb.ListDevicesResponse, error) {
	devices := s.registry.List()

	log.Printf("[DEBUG] ListDevices: returning %d devices", len(devices))

	return &pb.ListDevicesResponse{
		Devices: devices,
	}, nil
}

// GetDeviceStatus returns the status of a device
func (s *OrchestratorServer) GetDeviceStatus(ctx context.Context, req *pb.DeviceId) (*pb.DeviceStatus, error) {
	// If requesting self status, sample current host metrics
	if req.DeviceId == s.selfDeviceID {
		hostStatus := sysinfo.GetHostStatus()
		return &pb.DeviceStatus{
			DeviceId:      s.selfDeviceID,
			LastSeen:      time.Now().Unix(),
			CpuLoad:       hostStatus.CPULoad,
			MemUsedMb:     hostStatus.MemUsedMB,
			MemTotalMb:    hostStatus.MemTotalMB,
			GpuLoad:       hostStatus.GPULoad,
			GpuMemUsedMb:  hostStatus.GPUMemUsedMB,
			GpuMemTotalMb: hostStatus.GPUMemTotalMB,
			NpuLoad:       hostStatus.NPULoad,
			TimestampMs:   hostStatus.Timestamp,
		}, nil
	}

	// Return status from registry
	deviceStatus := s.registry.GetStatus(req.DeviceId)

	log.Printf("[DEBUG] GetDeviceStatus: device=%s last_seen=%d", req.DeviceId, deviceStatus.LastSeen)

	return deviceStatus, nil
}

// GetActivity returns current activity data (running tasks, device activities, metrics)
func (s *OrchestratorServer) GetActivity(ctx context.Context, req *pb.GetActivityRequest) (*pb.GetActivityResponse, error) {
	now := time.Now().UnixMilli()

	// Get running tasks from job manager
	runningTasks := s.jobManager.GetRunningTasks()
	pbRunningTasks := make([]*pb.RunningTask, 0, len(runningTasks))
	for _, task := range runningTasks {
		elapsed := now - task.StartedAt
		if task.StartedAt == 0 {
			elapsed = 0
		}
		pbRunningTasks = append(pbRunningTasks, &pb.RunningTask{
			TaskId:      task.ID,
			JobId:       task.JobID,
			Kind:        task.Kind,
			Input:       task.Input,
			DeviceId:    task.DeviceID,
			DeviceName:  task.DeviceName,
			StartedAtMs: task.StartedAt,
			ElapsedMs:   elapsed,
		})
	}

	// Get running task counts per device
	taskCounts := s.jobManager.GetRunningTaskCountByDevice()

	// Build device activities
	devices := s.registry.List()
	deviceActivities := make([]*pb.DeviceActivity, 0, len(devices))
	for _, d := range devices {
		count := taskCounts[d.DeviceId]
		status, _ := s.GetDeviceStatus(ctx, &pb.DeviceId{DeviceId: d.DeviceId})
		deviceActivities = append(deviceActivities, &pb.DeviceActivity{
			DeviceId:         d.DeviceId,
			DeviceName:       d.DeviceName,
			RunningTaskCount: int32(count),
			CurrentStatus:    status,
		})
	}

	resp := &pb.GetActivityResponse{
		Activity: &pb.ActivityData{
			RunningTasks:     pbRunningTasks,
			DeviceActivities: deviceActivities,
		},
	}

	// Include metrics history if requested
	if req.IncludeMetricsHistory {
		sinceMs := req.MetricsSinceMs
		allHistory := s.metricsStore.GetAllHistory(sinceMs)
		resp.DeviceMetrics = make(map[string]*pb.MetricsHistoryResponse)
		for deviceID, samples := range allHistory {
			deviceName, _ := s.metricsStore.GetDeviceInfo(deviceID)
			pbSamples := make([]*pb.MetricsSample, len(samples))
			for i, sample := range samples {
				pbSamples[i] = &pb.MetricsSample{
					TimestampMs:   sample.Timestamp,
					CpuLoad:       sample.CPULoad,
					MemUsedMb:     sample.MemUsedMB,
					MemTotalMb:    sample.MemTotalMB,
					GpuLoad:       sample.GPULoad,
					GpuMemUsedMb:  sample.GPUMemUsedMB,
					GpuMemTotalMb: sample.GPUMemTotalMB,
					NpuLoad:       sample.NPULoad,
				}
			}
			resp.DeviceMetrics[deviceID] = &pb.MetricsHistoryResponse{
				DeviceId:   deviceID,
				DeviceName: deviceName,
				Samples:    pbSamples,
			}
		}
	}

	return resp, nil
}

// GetDeviceMetrics returns metrics history for a specific device
func (s *OrchestratorServer) GetDeviceMetrics(ctx context.Context, req *pb.DeviceId) (*pb.MetricsHistoryResponse, error) {
	samples := s.metricsStore.GetHistory(req.DeviceId, 0)
	deviceName, _ := s.metricsStore.GetDeviceInfo(req.DeviceId)

	pbSamples := make([]*pb.MetricsSample, len(samples))
	for i, sample := range samples {
		pbSamples[i] = &pb.MetricsSample{
			TimestampMs:   sample.Timestamp,
			CpuLoad:       sample.CPULoad,
			MemUsedMb:     sample.MemUsedMB,
			MemTotalMb:    sample.MemTotalMB,
			GpuLoad:       sample.GPULoad,
			GpuMemUsedMb:  sample.GPUMemUsedMB,
			GpuMemTotalMb: sample.GPUMemTotalMB,
			NpuLoad:       sample.NPULoad,
		}
	}

	return &pb.MetricsHistoryResponse{
		DeviceId:   req.DeviceId,
		DeviceName: deviceName,
		Samples:    pbSamples,
	}, nil
}

// GetJobDetail returns enhanced job details for visualization
func (s *OrchestratorServer) GetJobDetail(ctx context.Context, req *pb.JobId) (*pb.JobDetailResponse, error) {
	job, ok := s.jobManager.Get(req.JobId)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "job not found: %s", req.JobId)
	}

	tasks := make([]*pb.TaskStatusEnhanced, len(job.Tasks))
	for i, task := range job.Tasks {
		tasks[i] = &pb.TaskStatusEnhanced{
			TaskId:             task.ID,
			JobId:              task.JobID,
			AssignedDeviceId:   task.DeviceID,
			AssignedDeviceName: task.DeviceName,
			Kind:               task.Kind,
			Input:              task.Input,
			State:              string(task.State),
			Result:             task.Result,
			Error:              task.Error,
			GroupIndex:         int32(task.GroupIndex),
			StartedAtMs:        task.StartedAt,
			EndedAtMs:          task.EndedAt,
		}
	}

	return &pb.JobDetailResponse{
		JobId:        job.ID,
		State:        string(job.State),
		Tasks:        tasks,
		FinalResult:  job.FinalResult,
		CurrentGroup: int32(job.CurrentGroup),
		TotalGroups:  int32(job.TotalGroups),
		CreatedAtMs:  job.CreatedAt.UnixMilli(),
		StartedAtMs:  job.StartedAt.UnixMilli(),
		EndedAtMs:    job.EndedAt.UnixMilli(),
	}, nil
}

// RunAITask routes an AI task to the best available device (stub implementation)
func (s *OrchestratorServer) RunAITask(ctx context.Context, req *pb.AITaskRequest) (*pb.AITaskResponse, error) {
	// Verify session
	s.mu.RLock()
	_, exists := s.sessions[req.SessionId]
	s.mu.RUnlock()

	if !exists {
		log.Printf("[ERROR] RunAITask: session not found: %s", req.SessionId)
		return nil, status.Error(codes.Unauthenticated, "session not found")
	}

	// Select best device for the task
	selectedDevice, found := s.registry.SelectBestDevice()

	var deviceID, deviceAddr, deviceName string
	var wouldUseNPU bool

	if found {
		deviceID = selectedDevice.DeviceId
		deviceAddr = selectedDevice.GrpcAddr
		deviceName = selectedDevice.DeviceName
		wouldUseNPU = selectedDevice.HasNpu
	} else {
		// Route to self if no devices registered
		deviceID = s.selfDeviceID
		deviceAddr = s.selfAddr
		hostname, _ := os.Hostname()
		deviceName = hostname
		wouldUseNPU = false
	}

	result := fmt.Sprintf("ROUTED: %s to %s", req.Task, deviceName)

	log.Printf("[INFO] RunAITask: task=%s routed to device=%s addr=%s npu=%v",
		req.Task, deviceName, deviceAddr, wouldUseNPU)

	return &pb.AITaskResponse{
		SelectedDeviceId:   deviceID,
		SelectedDeviceAddr: deviceAddr,
		WouldUseNpu:        wouldUseNPU,
		Result:             result,
	}, nil
}

// getSelfDeviceInfo returns device info for this server
func (s *OrchestratorServer) getSelfDeviceInfo() *pb.DeviceInfo {
	hostname, _ := os.Hostname()

	// Detect local Ollama/LLM availability
	hasLocalModel, localModelName, localChatEndpoint := detectLocalModel()

	return &pb.DeviceInfo{
		DeviceId:          s.selfDeviceID,
		DeviceName:        hostname,
		Platform:          runtime.GOOS,
		Arch:              runtime.GOARCH,
		HasCpu:            true,
		HasGpu:            false,
		HasNpu:            false,
		GrpcAddr:          s.selfAddr,
		CanScreenCapture:  detectScreenCapture(),
		HttpAddr:          s.deriveBulkHTTPAddr(),
		HasLocalModel:     hasLocalModel,
		LocalModelName:    localModelName,
		LocalChatEndpoint: localChatEndpoint,
	}
}

// deriveBulkHTTPAddr combines the host from selfAddr with the port from bulkHTTPAddr.
func (s *OrchestratorServer) deriveBulkHTTPAddr() string {
	host := s.selfAddr
	if idx := strings.LastIndex(s.selfAddr, ":"); idx != -1 {
		host = s.selfAddr[:idx]
	}
	port := s.bulkHTTPAddr
	if idx := strings.LastIndex(s.bulkHTTPAddr, ":"); idx != -1 {
		port = s.bulkHTTPAddr[idx+1:]
	}
	return host + ":" + port
}

// detectScreenCapture tests whether this machine can capture the screen
func detectScreenCapture() bool {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return false
	}
	bounds := screenshot.GetDisplayBounds(0)
	_, err := screenshot.CaptureRect(bounds)
	return err == nil
}

// detectLocalModel checks if Ollama or other local LLM is available
// Returns (hasLocalModel, modelName, chatEndpoint)
func detectLocalModel() (bool, string, string) {
	// Check CHAT_BASE_URL first (configured Ollama endpoint)
	baseURL := os.Getenv("CHAT_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// Try to hit Ollama API
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(baseURL)
	if err != nil {
		return false, "", ""
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, "", ""
	}

	// Ollama is running, get loaded model
	modelName := getOllamaModel(client, baseURL)
	if modelName == "" {
		// Use configured model as fallback
		modelName = os.Getenv("CHAT_MODEL")
		if modelName == "" {
			modelName = "llama3.2:3b"
		}
	}

	return true, modelName, baseURL
}

// getOllamaModel queries Ollama /api/tags to get available models
func getOllamaModel(client *http.Client, baseURL string) string {
	resp, err := client.Get(baseURL + "/api/tags")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	if len(result.Models) > 0 {
		return result.Models[0].Name
	}

	return ""
}

// autoRegisterWithCoordinator registers this device with a remote coordinator.
// It retries periodically so that if the coordinator restarts, we re-register.
func (s *OrchestratorServer) autoRegisterWithCoordinator(coordinatorAddr string) {
	selfInfo := s.getSelfDeviceInfo()

	// Fix self-address: if we're listening on 0.0.0.0, resolve to our LAN IP
	// so the coordinator can actually reach us back.
	if strings.HasPrefix(selfInfo.GrpcAddr, "0.0.0.0:") {
		port := selfInfo.GrpcAddr[len("0.0.0.0"):]
		if lanIP := detectLANIP(); lanIP != "" {
			selfInfo.GrpcAddr = lanIP + port
			selfInfo.HttpAddr = lanIP + ":" + strings.TrimLeft(s.bulkHTTPAddr, ":")
			log.Printf("[INFO] Auto-register: resolved self address to %s", selfInfo.GrpcAddr)
		}
	}

	// Detect NPU on Windows ARM64 (Snapdragon)
	if runtime.GOOS == "windows" && runtime.GOARCH == "arm64" {
		selfInfo.HasNpu = true
	}

	for {
		log.Printf("[INFO] Auto-registering with coordinator at %s ...", coordinatorAddr)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		conn, err := grpc.DialContext(ctx, coordinatorAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			cancel()
			log.Printf("[WARN] Could not reach coordinator at %s: %v — retrying in 10s", coordinatorAddr, err)
			time.Sleep(10 * time.Second)
			continue
		}

		client := pb.NewOrchestratorServiceClient(conn)

		// Register ourselves
		ack, err := client.RegisterDevice(ctx, selfInfo)
		cancel()
		conn.Close()

		if err != nil {
			log.Printf("[WARN] Registration with coordinator failed: %v — retrying in 10s", err)
			time.Sleep(10 * time.Second)
			continue
		}

		log.Printf("[INFO] Successfully registered with coordinator at %s (ack=%v)", coordinatorAddr, ack.Ok)

		// Also sync: pull the coordinator's device list and add to our local registry
		s.syncDevicesFromCoordinator(coordinatorAddr)

		// Re-register periodically (heartbeat) every 30 seconds
		time.Sleep(30 * time.Second)
	}
}

// syncDevicesFromCoordinator pulls the device list from a coordinator and adds
// them to our local registry. This means any device can see all other devices.
func (s *OrchestratorServer) syncDevicesFromCoordinator(coordinatorAddr string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, coordinatorAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("[WARN] Could not connect to coordinator for sync: %v", err)
		return
	}
	defer conn.Close()

	client := pb.NewOrchestratorServiceClient(conn)
	resp, err := client.ListDevices(ctx, &pb.ListDevicesRequest{})
	if err != nil {
		log.Printf("[WARN] ListDevices from coordinator failed: %v", err)
		return
	}

	added := 0
	for _, d := range resp.Devices {
		if d.DeviceId == s.selfDeviceID {
			continue // Skip ourselves
		}
		s.registry.Upsert(d)
		added++
	}
	if added > 0 {
		log.Printf("[INFO] Synced %d device(s) from coordinator", added)
	}
}

// runImageGenerate calls a Stable Diffusion API endpoint to generate an image.
// Configure via IMAGE_API_ENDPOINT env var (defaults to Automatic1111 WebUI format).
// Returns the path to the generated image in the shared directory.
func (s *OrchestratorServer) runImageGenerate(ctx context.Context, prompt string) (string, error) {
	endpoint := os.Getenv("IMAGE_API_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://127.0.0.1:7860" // Automatic1111 WebUI default
	}

	// Build Stable Diffusion txt2img request
	reqBody := map[string]interface{}{
		"prompt":       prompt,
		"steps":        20,
		"width":        512,
		"height":       512,
		"sampler_name": "Euler a",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(endpoint, "/") + "/sdapi/v1/txt2img"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 180 * time.Second} // 3 min for image gen
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("Image API unreachable at %s: %w (set IMAGE_API_ENDPOINT env var)", endpoint, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Image API returned %d: %s", resp.StatusCode, string(respBody[:min(200, len(respBody))]))
	}

	var sdResp struct {
		Images []string `json:"images"` // base64 encoded images
	}
	if err := json.Unmarshal(respBody, &sdResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(sdResp.Images) == 0 {
		return "", fmt.Errorf("no images generated")
	}

	// Save first image to shared directory
	imageData := sdResp.Images[0]
	decoded, err := base64.StdEncoding.DecodeString(imageData)
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("image_%d.png", time.Now().Unix())
	imagePath := filepath.Join(s.sharedRoot, filename)

	if err := os.WriteFile(imagePath, decoded, 0644); err != nil {
		return "", fmt.Errorf("save image: %w", err)
	}

	log.Printf("[INFO] IMAGE_GENERATE completed: prompt_len=%d image_saved=%s", len(prompt), imagePath)
	return imagePath, nil
}

// detectLANIP finds the first non-loopback IPv4 address on this machine.
func detectLANIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			return ipNet.IP.String()
		}
	}
	return ""
}

// HealthCheck returns the server's health status
func (s *OrchestratorServer) HealthCheck(ctx context.Context, req *pb.Empty) (*pb.HealthStatus, error) {
	return &pb.HealthStatus{
		DeviceId:   s.selfDeviceID,
		ServerTime: time.Now().Unix(),
		Message:    "ok",
	}, nil
}

// ExecuteRoutedCommand executes a command on the best available device
func (s *OrchestratorServer) ExecuteRoutedCommand(ctx context.Context, req *pb.RoutedCommandRequest) (*pb.RoutedCommandResponse, error) {
	startTime := time.Now()

	// Verify session
	s.mu.RLock()
	_, exists := s.sessions[req.SessionId]
	s.mu.RUnlock()

	if !exists {
		log.Printf("[ERROR] ExecuteRoutedCommand: session not found: %s", req.SessionId)
		return nil, status.Error(codes.Unauthenticated, "session not found")
	}

	// Select target device based on policy
	result := s.registry.SelectDevice(req.Policy, s.selfDeviceID)
	if result.Error != nil {
		log.Printf("[ERROR] ExecuteRoutedCommand: device selection failed: %v", result.Error)
		return nil, status.Error(codes.FailedPrecondition, result.Error.Error())
	}

	device := result.Device
	log.Printf("[INFO] ExecuteRoutedCommand: selected device=%s name=%s addr=%s local=%v",
		device.DeviceId, device.DeviceName, device.GrpcAddr, result.ExecutedLocally)

	var cmdResp *pb.CommandResponse
	var err error

	if result.ExecutedLocally {
		// Execute locally
		cmdResp, err = s.ExecuteCommand(ctx, &pb.CommandRequest{
			SessionId: req.SessionId,
			Command:   req.Command,
			Args:      req.Args,
		})
	} else {
		// Forward to remote device
		cmdResp, err = s.forwardCommand(ctx, device.GrpcAddr, req)
	}

	if err != nil {
		return nil, err
	}

	totalTime := time.Since(startTime).Seconds() * 1000 // Convert to ms

	return &pb.RoutedCommandResponse{
		Output:             cmdResp,
		SelectedDeviceId:   device.DeviceId,
		SelectedDeviceName: device.DeviceName,
		SelectedDeviceAddr: device.GrpcAddr,
		TotalTimeMs:        totalTime,
		ExecutedLocally:    result.ExecutedLocally,
	}, nil
}

// forwardCommand forwards a command to a remote device
func (s *OrchestratorServer) forwardCommand(ctx context.Context, targetAddr string, req *pb.RoutedCommandRequest) (*pb.CommandResponse, error) {
	// Create context with timeout for dialing
	dialCtx, cancel := context.WithTimeout(ctx, remoteDialTimeout)
	defer cancel()

	// Dial remote server
	conn, err := grpc.DialContext(dialCtx, targetAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("[ERROR] forwardCommand: failed to dial %s: %v", targetAddr, err)
		return nil, status.Errorf(codes.Unavailable, "failed to connect to remote device at %s: %v", targetAddr, err)
	}
	defer conn.Close()

	client := pb.NewOrchestratorServiceClient(conn)

	// Optional: health check first
	healthCtx, healthCancel := context.WithTimeout(ctx, time.Second)
	defer healthCancel()
	healthResp, err := client.HealthCheck(healthCtx, &pb.Empty{})
	if err != nil {
		log.Printf("[WARN] forwardCommand: health check failed for %s: %v", targetAddr, err)
		// Continue anyway, the actual command will fail if server is truly down
	} else {
		log.Printf("[DEBUG] forwardCommand: health check ok for %s (device=%s)", targetAddr, healthResp.DeviceId)
	}

	// Create a session on the remote server
	sessionResp, err := client.CreateSession(ctx, &pb.AuthRequest{
		DeviceName:  "coordinator-forward",
		SecurityKey: "internal-routing", // Internal key for forwarded requests
	})
	if err != nil {
		log.Printf("[ERROR] forwardCommand: failed to create session on %s: %v", targetAddr, err)
		return nil, status.Errorf(codes.Internal, "failed to create session on remote device: %v", err)
	}

	// Execute command on remote
	cmdResp, err := client.ExecuteCommand(ctx, &pb.CommandRequest{
		SessionId: sessionResp.SessionId,
		Command:   req.Command,
		Args:      req.Args,
	})
	if err != nil {
		log.Printf("[ERROR] forwardCommand: command execution failed on %s: %v", targetAddr, err)
		return nil, status.Errorf(codes.Internal, "command execution failed on remote device: %v", err)
	}

	log.Printf("[INFO] forwardCommand: command completed on %s exit_code=%d", targetAddr, cmdResp.ExitCode)
	return cmdResp, nil
}

// RunTask executes a task locally on this device (worker RPC)
func (s *OrchestratorServer) RunTask(ctx context.Context, req *pb.TaskRequest) (*pb.TaskResult, error) {
	start := time.Now()
	log.Printf("[INFO] RunTask: task_id=%s job_id=%s kind=%s", req.TaskId, req.JobId, req.Kind)

	switch req.Kind {
	case "SYSINFO":
		info := s.collectSysInfo()
		return &pb.TaskResult{
			TaskId: req.TaskId,
			Ok:     true,
			Output: info,
			TimeMs: float64(time.Since(start).Milliseconds()),
		}, nil

	case "ECHO":
		return &pb.TaskResult{
			TaskId: req.TaskId,
			Ok:     true,
			Output: "echo: " + req.Input,
			TimeMs: float64(time.Since(start).Milliseconds()),
		}, nil

	case "LLM_GENERATE":
		output, err := s.runLLMGenerate(ctx, req.Input)
		if err != nil {
			return &pb.TaskResult{
				TaskId: req.TaskId,
				Ok:     false,
				Error:  fmt.Sprintf("LLM_GENERATE failed: %v", err),
				TimeMs: float64(time.Since(start).Milliseconds()),
			}, nil
		}
		return &pb.TaskResult{
			TaskId: req.TaskId,
			Ok:     true,
			Output: output,
			TimeMs: float64(time.Since(start).Milliseconds()),
		}, nil

	case "IMAGE_GENERATE":
		imagePath, err := s.runImageGenerate(ctx, req.Input)
		if err != nil {
			return &pb.TaskResult{
				TaskId: req.TaskId,
				Ok:     false,
				Error:  fmt.Sprintf("IMAGE_GENERATE failed: %v", err),
				TimeMs: float64(time.Since(start).Milliseconds()),
			}, nil
		}
		return &pb.TaskResult{
			TaskId: req.TaskId,
			Ok:     true,
			Output: imagePath, // Return path to generated image
			TimeMs: float64(time.Since(start).Milliseconds()),
		}, nil

	default:
		return &pb.TaskResult{
			TaskId: req.TaskId,
			Ok:     false,
			Error:  "unknown task kind: " + req.Kind,
			TimeMs: float64(time.Since(start).Milliseconds()),
		}, nil
	}
}

// runLLMGenerate sends a prompt to this device's /api/assistant HTTP endpoint,
// which wraps the local LLM (Ollama, LM Studio, etc.) configured via CHAT_PROVIDER.
// This ensures each device uses its own local model for task execution.
func (s *OrchestratorServer) runLLMGenerate(ctx context.Context, prompt string) (string, error) {
	// Use device's own /api/assistant endpoint
	// This device's HTTP server is at http://localhost:<WEB_PORT>/api/assistant
	webPort := os.Getenv("WEB_ADDR")
	if webPort == "" {
		webPort = ":8080"
	}
	// Strip leading : if present
	if strings.HasPrefix(webPort, ":") {
		webPort = "localhost" + webPort
	}

	url := fmt.Sprintf("http://%s/api/assistant", webPort)

	reqBody := map[string]interface{}{
		"text": prompt,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("HTTP request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, url, string(respBody[:min(200, len(respBody))]))
	}

	var assistantResp struct {
		Text string `json:"text"`
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if err := json.Unmarshal(respBody, &assistantResp); err != nil {
		return "", fmt.Errorf("parse response: %w (body: %s)", err, string(respBody[:min(200, len(respBody))]))
	}

	if assistantResp.Text == "" {
		return "", fmt.Errorf("assistant returned empty response")
	}

	log.Printf("[INFO] LLM_GENERATE completed via /api/assistant: prompt_len=%d result_len=%d", len(prompt), len(assistantResp.Text))
	return assistantResp.Text, nil
}

// runLLMGenerateOllama is a fallback that uses Ollama's native /api/chat endpoint.
func (s *OrchestratorServer) runLLMGenerateOllama(ctx context.Context, endpoint, model, prompt string) (string, error) {
	url := strings.TrimRight(endpoint, "/") + "/api/chat"

	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"stream": false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("LLM endpoint unreachable at %s: %w (set LLM_ENDPOINT env var)", endpoint, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(respBody[:min(200, len(respBody))]))
	}

	var ollamaResp struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return "", fmt.Errorf("parse ollama response: %w", err)
	}

	log.Printf("[INFO] LLM_GENERATE (ollama) completed: model=%s result_len=%d", model, len(ollamaResp.Message.Content))
	return ollamaResp.Message.Content, nil
}

// collectSysInfo gathers system information for this device
func (s *OrchestratorServer) collectSysInfo() string {
	hostname, _ := os.Hostname()
	hostStatus := sysinfo.GetHostStatus()
	now := time.Now().Format(time.RFC3339)

	return fmt.Sprintf("Device: %s\nDevice ID: %s\nPlatform: %s/%s\nMemory: %d MB total, %d MB used\nTime: %s",
		hostname,
		s.selfDeviceID,
		runtime.GOOS, runtime.GOARCH,
		hostStatus.MemTotalMB, hostStatus.MemUsedMB,
		now)
}

// SubmitJob accepts a job request and distributes tasks to devices
func (s *OrchestratorServer) SubmitJob(ctx context.Context, req *pb.JobRequest) (*pb.JobInfo, error) {
	// Verify session
	s.mu.RLock()
	_, exists := s.sessions[req.SessionId]
	s.mu.RUnlock()

	if !exists {
		log.Printf("[ERROR] SubmitJob: session not found: %s", req.SessionId)
		return nil, status.Error(codes.Unauthenticated, "session not found")
	}

	// Get devices from registry
	devices := s.registry.List()
	if len(devices) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "no devices available")
	}

	// Try to generate plan using LLM provider or brain if available and no plan provided
	plan := req.Plan
	reduce := req.Reduce
	if plan == nil || len(plan.Groups) == 0 {
		// Prefer cross-platform LLM provider over Windows AI brain
		if s.llmProvider != nil {
			// Use LLM provider for planning
			var err error
			plan, reduce, err = s.generatePlanWithLLM(req.Text, devices, int(req.MaxWorkers))
			if err == nil && plan != nil {
				log.Printf("[INFO] SubmitJob: LLM plan generated, groups=%d", len(plan.Groups))
			} else if err != nil {
				log.Printf("[WARN] SubmitJob: LLM plan generation failed, using default: %v", err)
				plan, reduce = nil, nil
			}
		} else if s.brain != nil && s.brain.IsAvailable() {
			// Fallback to Windows AI brain
			result, err := s.brain.GeneratePlan(req.Text, devices, int(req.MaxWorkers))
			if err == nil && result != nil && result.Plan != nil {
				plan = result.Plan
				if reduce == nil {
					reduce = result.Reduce
				}
				log.Printf("[INFO] SubmitJob: brain plan used_ai=%v rationale=%q groups=%d",
					result.UsedAi, result.Rationale, len(plan.Groups))
			} else if err != nil {
				log.Printf("[WARN] SubmitJob: brain plan generation failed, using default: %v", err)
			}
		}
	}

	// Create job with tasks (plan and reduce will use smart defaults if nil)
	job, err := s.jobManager.CreateJob(req.Text, devices, int(req.MaxWorkers), plan, reduce)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create job: %v", err)
	}

	log.Printf("[INFO] SubmitJob: job_id=%s tasks=%d groups=%d text=%q",
		job.ID, len(job.Tasks), job.TotalGroups, req.Text)

	// Execute groups sequentially (tasks within groups run in parallel)
	go s.executeJobGroups(job)

	return &pb.JobInfo{
		JobId:     job.ID,
		CreatedAt: job.CreatedAt.Unix(),
		Summary:   fmt.Sprintf("distributed to %d devices in %d group(s)", len(job.Tasks), job.TotalGroups),
	}, nil
}

// executeJobGroups executes task groups sequentially (tasks within groups run in parallel)
func (s *OrchestratorServer) executeJobGroups(job *jobs.Job) {
	s.jobManager.SetJobRunning(job.ID)

	// Start metrics polling for active devices
	metricsCtx, cancelMetrics := context.WithCancel(context.Background())
	go s.pollDeviceMetrics(metricsCtx, job)
	defer cancelMetrics()

	var allResults []string
	var totalFailed int

	// Execute groups sequentially
	for groupIdx := 0; groupIdx < job.TotalGroups; groupIdx++ {
		log.Printf("[INFO] executeJobGroups: executing group %d/%d for job %s",
			groupIdx+1, job.TotalGroups, job.ID)

		s.jobManager.SetCurrentGroup(job.ID, groupIdx)

		// Get tasks for this group
		groupTasks := s.jobManager.GetTasksForGroup(job.ID, groupIdx)

		// Execute all tasks in this group in parallel
		groupResults, failedCount := s.executeTaskGroup(job, groupTasks)
		allResults = append(allResults, groupResults...)
		totalFailed += failedCount

		// Check if group failed - stop execution if it did
		if s.jobManager.IsGroupFailed(job.ID, groupIdx) {
			log.Printf("[WARN] executeJobGroups: group %d failed, stopping job %s", groupIdx, job.ID)
			break
		}
	}

	// Apply reduce to combine results
	finalResult := s.applyReduce(job.ReduceSpec, allResults)
	if totalFailed > 0 {
		finalResult = fmt.Sprintf("Warning: %d task(s) failed\n\n%s", totalFailed, finalResult)
	}
	s.jobManager.SetJobDone(job.ID, finalResult)

	log.Printf("[INFO] executeJobGroups: job=%s completed, %d succeeded, %d failed",
		job.ID, len(allResults), totalFailed)
}

// pollDeviceMetrics polls metrics from active devices during job execution
func (s *OrchestratorServer) pollDeviceMetrics(ctx context.Context, job *jobs.Job) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get devices that have running tasks
			activeDeviceIDs := s.jobManager.GetActiveDeviceIDs()
			if len(activeDeviceIDs) == 0 {
				continue
			}

			// Poll each active device
			for _, deviceID := range activeDeviceIDs {
				go s.fetchAndStoreDeviceMetrics(ctx, deviceID)
			}
		}
	}
}

// fetchAndStoreDeviceMetrics fetches metrics from a device and stores them
func (s *OrchestratorServer) fetchAndStoreDeviceMetrics(ctx context.Context, deviceID string) {
	// Get device info
	device, ok := s.registry.Get(deviceID)
	if !ok || device == nil {
		return
	}

	var status *pb.DeviceStatus
	var err error

	// For self, use local GetDeviceStatus
	if deviceID == s.selfDeviceID {
		status, err = s.GetDeviceStatus(ctx, &pb.DeviceId{DeviceId: deviceID})
	} else {
		// For remote devices, make gRPC call
		status, err = s.fetchRemoteDeviceStatus(ctx, device.Info.GrpcAddr, deviceID)
	}

	if err != nil {
		// Don't log for remote connection failures (too noisy)
		return
	}

	if status == nil || status.CpuLoad < 0 {
		// No valid metrics
		return
	}

	// Convert to metrics sample and store
	sample := metrics.MetricsSample{
		Timestamp:     time.Now().UnixMilli(),
		CPULoad:       status.CpuLoad,
		MemUsedMB:     status.MemUsedMb,
		MemTotalMB:    status.MemTotalMb,
		GPULoad:       status.GpuLoad,
		GPUMemUsedMB:  status.GpuMemUsedMb,
		GPUMemTotalMB: status.GpuMemTotalMb,
		NPULoad:       status.NpuLoad,
	}

	s.metricsStore.AddSample(deviceID, device.Info.DeviceName, sample)

	// Also update registry status so Activity panel shows current data
	s.registry.UpdateStatus(deviceID, status)
}

// fetchRemoteDeviceStatus makes a gRPC call to get status from a remote device
func (s *OrchestratorServer) fetchRemoteDeviceStatus(ctx context.Context, targetAddr string, deviceID string) (*pb.DeviceStatus, error) {
	if targetAddr == "" {
		return nil, fmt.Errorf("no gRPC address for device %s", deviceID)
	}

	// Short timeout for metrics polling
	dialCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, targetAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pb.NewOrchestratorServiceClient(conn)

	// Call GetDeviceStatus on remote device
	statusCtx, statusCancel := context.WithTimeout(ctx, 2*time.Second)
	defer statusCancel()

	status, err := client.GetDeviceStatus(statusCtx, &pb.DeviceId{DeviceId: deviceID})
	if err != nil {
		return nil, err
	}

	return status, nil
}

// startContinuousMetricsPolling polls all registered devices for metrics every 2 seconds
func (s *OrchestratorServer) startContinuousMetricsPolling(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Printf("[INFO] Starting continuous metrics polling")

	for {
		select {
		case <-ctx.Done():
			log.Printf("[INFO] Stopping continuous metrics polling")
			return
		case <-ticker.C:
			// Poll all registered devices
			devices := s.registry.List()
			for _, device := range devices {
				go s.fetchAndStoreDeviceMetrics(ctx, device.DeviceId)
			}
		}
	}
}

// executeTaskGroup runs all tasks in a group in parallel
func (s *OrchestratorServer) executeTaskGroup(job *jobs.Job, tasks []*jobs.Task) ([]string, int) {
	var wg sync.WaitGroup
	var results []string
	var resultsMu sync.Mutex
	var failedCount int

	for _, task := range tasks {
		wg.Add(1)
		go func(t *jobs.Task) {
			defer wg.Done()

			log.Printf("[INFO] executeTaskGroup: executing task=%s on device=%s addr=%s",
				t.ID, t.DeviceName, t.DeviceAddr)

			// Mark task as running
			s.jobManager.SetTaskRunning(job.ID, t.ID)

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Dial the device
			conn, err := grpc.DialContext(ctx, t.DeviceAddr,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithBlock(),
			)
			if err != nil {
				log.Printf("[ERROR] executeTaskGroup: failed to dial %s: %v", t.DeviceAddr, err)
				s.jobManager.UpdateTask(job.ID, t.ID, jobs.TaskFailed, "", err.Error())
				resultsMu.Lock()
				failedCount++
				resultsMu.Unlock()
				return
			}
			defer conn.Close()

			client := pb.NewOrchestratorServiceClient(conn)

			// Call RunTask on the device
			result, err := client.RunTask(ctx, &pb.TaskRequest{
				TaskId: t.ID,
				JobId:  job.ID,
				Kind:   t.Kind,
				Input:  t.Input,
			})

			if err != nil {
				log.Printf("[ERROR] executeTaskGroup: RunTask failed on %s: %v", t.DeviceAddr, err)
				s.jobManager.UpdateTask(job.ID, t.ID, jobs.TaskFailed, "", err.Error())
				resultsMu.Lock()
				failedCount++
				resultsMu.Unlock()
				return
			}

			if !result.Ok {
				log.Printf("[ERROR] executeTaskGroup: task failed on %s: %s", t.DeviceAddr, result.Error)
				s.jobManager.UpdateTask(job.ID, t.ID, jobs.TaskFailed, "", result.Error)
				resultsMu.Lock()
				failedCount++
				resultsMu.Unlock()
				return
			}

			// Task succeeded
			s.jobManager.UpdateTask(job.ID, t.ID, jobs.TaskDone, result.Output, "")
			resultsMu.Lock()
			results = append(results, fmt.Sprintf("=== %s (%s) ===\n%s",
				t.DeviceName, t.DeviceID[:8], result.Output))
			resultsMu.Unlock()

			log.Printf("[INFO] executeTaskGroup: task=%s completed on %s in %.2fms",
				t.ID, t.DeviceName, result.TimeMs)
		}(task)
	}

	wg.Wait()
	return results, failedCount
}

// applyReduce combines results based on the reduce specification
func (s *OrchestratorServer) applyReduce(spec *jobs.ReduceSpec, results []string) string {
	if spec == nil {
		spec = &jobs.ReduceSpec{Kind: "CONCAT"}
	}

	switch spec.Kind {
	case "CONCAT":
		return strings.Join(results, "\n\n")
	default:
		// Default to CONCAT for unknown reduce kinds
		return strings.Join(results, "\n\n")
	}
}

// GetJob returns the status of a job
func (s *OrchestratorServer) GetJob(ctx context.Context, req *pb.JobId) (*pb.JobStatus, error) {
	job, ok := s.jobManager.Get(req.JobId)
	if !ok {
		return nil, status.Error(codes.NotFound, "job not found")
	}

	// Convert tasks to proto
	tasks := make([]*pb.TaskStatus, len(job.Tasks))
	for i, t := range job.Tasks {
		tasks[i] = &pb.TaskStatus{
			TaskId:             t.ID,
			AssignedDeviceId:   t.DeviceID,
			AssignedDeviceName: t.DeviceName,
			State:              string(t.State),
			Result:             t.Result,
			Error:              t.Error,
		}
	}

	return &pb.JobStatus{
		JobId:        job.ID,
		State:        string(job.State),
		Tasks:        tasks,
		FinalResult:  job.FinalResult,
		CurrentGroup: int32(job.CurrentGroup),
		TotalGroups:  int32(job.TotalGroups),
	}, nil
}

// PreviewPlan generates an execution plan without creating a job
func (s *OrchestratorServer) PreviewPlan(ctx context.Context, req *pb.PlanPreviewRequest) (*pb.PlanPreviewResponse, error) {
	// Verify session
	s.mu.RLock()
	_, exists := s.sessions[req.SessionId]
	s.mu.RUnlock()

	if !exists {
		log.Printf("[ERROR] PreviewPlan: session not found: %s", req.SessionId)
		return nil, status.Error(codes.Unauthenticated, "session not found")
	}

	// Get devices from registry
	devices := s.registry.List()
	if len(devices) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "no devices available")
	}

	// Select devices (limit by maxWorkers if specified)
	selectedDevices := devices
	if req.MaxWorkers > 0 && int(req.MaxWorkers) < len(devices) {
		selectedDevices = devices[:req.MaxWorkers]
	}

	var plan *pb.Plan
	var reduce *pb.ReduceSpec
	usedAi := false
	notes := ""
	rationale := ""

	// Try brain if available
	if s.brain != nil && s.brain.IsAvailable() {
		result, err := s.brain.GeneratePlan(req.Text, devices, int(req.MaxWorkers))
		if err == nil && result != nil && result.Plan != nil {
			plan = result.Plan
			reduce = result.Reduce
			usedAi = result.UsedAi
			notes = result.Notes
			rationale = result.Rationale
			log.Printf("[INFO] PreviewPlan: brain plan used_ai=%v rationale=%q", usedAi, rationale)
		} else if err != nil {
			log.Printf("[WARN] PreviewPlan: brain failed, using default: %v", err)
		}
	}

	// Fall back to smart plan based on user text
	if plan == nil {
		plan = s.jobManager.GenerateSmartPlan(req.Text, selectedDevices)
		reduce = &pb.ReduceSpec{Kind: "CONCAT"}
		notes = "Brain not available (non-Windows or disabled)"

		// Determine rationale based on what kind of plan was generated
		isLLMTask := false
		if len(plan.Groups) > 0 && len(plan.Groups[0].Tasks) > 0 {
			isLLMTask = plan.Groups[0].Tasks[0].Kind == "LLM_GENERATE"
		}
		if isLLMTask {
			rationale = fmt.Sprintf("Smart: detected LLM request, routing to best NPU device")
		} else {
			rationale = fmt.Sprintf("Default: 1 SYSINFO per device, %d of %d devices selected", len(selectedDevices), len(devices))
		}
	}

	return &pb.PlanPreviewResponse{
		UsedAi:    usedAi,
		Notes:     notes,
		Rationale: rationale,
		Plan:      plan,
		Reduce:    reduce,
	}, nil
}

// PreviewPlanCost estimates execution cost for a plan without running it
func (s *OrchestratorServer) PreviewPlanCost(ctx context.Context, req *pb.PlanCostRequest) (*pb.PlanCostResponse, error) {
	// Verify session
	s.mu.RLock()
	_, exists := s.sessions[req.SessionId]
	s.mu.RUnlock()

	if !exists {
		log.Printf("[ERROR] PreviewPlanCost: session not found: %s", req.SessionId)
		return nil, status.Error(codes.Unauthenticated, "session not found")
	}

	// Get devices from registry
	devices := s.registry.List()
	if len(devices) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "no devices available")
	}

	// Filter devices if device_ids specified
	if len(req.DeviceIds) > 0 {
		deviceSet := make(map[string]bool)
		for _, id := range req.DeviceIds {
			deviceSet[id] = true
		}
		filtered := make([]*pb.DeviceInfo, 0)
		for _, d := range devices {
			if deviceSet[d.DeviceId] {
				filtered = append(filtered, d)
			}
		}
		if len(filtered) == 0 {
			return nil, status.Error(codes.InvalidArgument, "none of the specified devices found")
		}
		devices = filtered
	}

	// Validate plan
	if req.Plan == nil || len(req.Plan.Groups) == 0 {
		return nil, status.Error(codes.InvalidArgument, "plan is required and must have at least one group")
	}

	// Estimate costs
	estimator := cost.NewEstimator()
	resp := estimator.EstimatePlanCost(req.Plan, devices)

	log.Printf("[INFO] PreviewPlanCost: devices=%d total_ms=%.2f recommended=%s has_unknown=%v",
		len(devices), resp.TotalPredictedMs, resp.RecommendedDeviceId, resp.HasUnknownCosts)

	return resp, nil
}

// StartWebRTC creates a new WebRTC peer connection and returns an offer SDP
func (s *OrchestratorServer) StartWebRTC(ctx context.Context, req *pb.WebRTCConfig) (*pb.WebRTCOffer, error) {
	log.Printf("[INFO] StartWebRTC: session=%s fps=%d quality=%d monitor=%d",
		req.SessionId, req.TargetFps, req.JpegQuality, req.MonitorIndex)

	streamID, offerSDP, err := s.webrtcManager.Start(
		req.SessionId,
		int(req.TargetFps),
		int(req.JpegQuality),
		int(req.MonitorIndex),
	)
	if err != nil {
		log.Printf("[ERROR] StartWebRTC failed: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to start WebRTC: %v", err)
	}

	log.Printf("[INFO] StartWebRTC: created stream %s", streamID)
	return &pb.WebRTCOffer{
		StreamId: streamID,
		Sdp:      offerSDP,
	}, nil
}

// CompleteWebRTC sets the remote description (answer) for a stream
func (s *OrchestratorServer) CompleteWebRTC(ctx context.Context, req *pb.WebRTCAnswer) (*pb.Empty, error) {
	log.Printf("[INFO] CompleteWebRTC: stream=%s", req.StreamId)

	if err := s.webrtcManager.Complete(req.StreamId, req.Sdp); err != nil {
		log.Printf("[ERROR] CompleteWebRTC failed: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to complete WebRTC: %v", err)
	}

	log.Printf("[INFO] CompleteWebRTC: stream %s connected", req.StreamId)
	return &pb.Empty{}, nil
}

// StopWebRTC closes a stream and cleans up resources
func (s *OrchestratorServer) StopWebRTC(ctx context.Context, req *pb.WebRTCStop) (*pb.Empty, error) {
	log.Printf("[INFO] StopWebRTC: stream=%s", req.StreamId)

	if err := s.webrtcManager.Stop(req.StreamId); err != nil {
		log.Printf("[ERROR] StopWebRTC failed: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to stop WebRTC: %v", err)
	}

	log.Printf("[INFO] StopWebRTC: stream %s stopped", req.StreamId)
	return &pb.Empty{}, nil
}

// CreateDownloadTicket validates a file path and issues a one-time download token
func (s *OrchestratorServer) CreateDownloadTicket(ctx context.Context, req *pb.DownloadTicketRequest) (*pb.DownloadTicketResponse, error) {
	path := req.Path
	if path == "" {
		return nil, status.Error(codes.InvalidArgument, "path is required")
	}

	// Resolve path: absolute paths used directly, relative paths resolved under sharedRoot
	var fullPath string
	if filepath.IsAbs(path) {
		fullPath = filepath.Clean(path)
	} else {
		if strings.Contains(path, "..") {
			return nil, status.Error(codes.InvalidArgument, "path must not contain '..'")
		}
		cleaned := filepath.Clean(path)
		fullPath = filepath.Join(s.sharedRoot, cleaned)
	}

	// Verify file exists and is regular
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, status.Errorf(codes.NotFound, "file not found: %s", fullPath)
		}
		return nil, status.Errorf(codes.Internal, "stat error: %v", err)
	}
	if !info.Mode().IsRegular() {
		return nil, status.Error(codes.InvalidArgument, "path is not a regular file")
	}

	// Mint ticket
	ticket, err := s.ticketManager.Create(fullPath, info.Name(), info.Size())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create ticket: %v", err)
	}

	log.Printf("[INFO] CreateDownloadTicket: path=%s size=%d token=%s...%s expires=%v",
		fullPath, info.Size(), ticket.Token[:4], ticket.Token[len(ticket.Token)-4:],
		ticket.ExpiresAt.Format(time.RFC3339))

	return &pb.DownloadTicketResponse{
		Token:         ticket.Token,
		Filename:      ticket.Filename,
		SizeBytes:     ticket.SizeBytes,
		ExpiresUnixMs: ticket.ExpiresAt.UnixMilli(),
	}, nil
}

// ReadFile reads a file from local or remote device (for LLM tool calling)
func (s *OrchestratorServer) ReadFile(ctx context.Context, req *pb.ReadFileRequest) (*pb.ReadFileResponse, error) {
	// Verify session
	s.mu.RLock()
	_, exists := s.sessions[req.SessionId]
	s.mu.RUnlock()

	if !exists {
		log.Printf("[ERROR] ReadFile: session not found: %s", req.SessionId)
		return nil, status.Error(codes.Unauthenticated, "session not found")
	}

	// If device_id specified and not self, forward to remote device
	if req.DeviceId != "" && req.DeviceId != s.selfDeviceID {
		return s.forwardReadFile(ctx, req)
	}

	// Read local file
	return s.readLocalFile(ctx, req)
}

// readLocalFile reads a file from the local filesystem
func (s *OrchestratorServer) readLocalFile(ctx context.Context, req *pb.ReadFileRequest) (*pb.ReadFileResponse, error) {
	path := req.Path
	if path == "" {
		return &pb.ReadFileResponse{Error: "path is required"}, nil
	}

	// Validate no path traversal
	if strings.Contains(path, "..") {
		return &pb.ReadFileResponse{Error: "path must not contain '..'"}, nil
	}

	// Resolve path: absolute paths used directly, relative under sharedRoot
	var fullPath string
	if filepath.IsAbs(path) {
		fullPath = filepath.Clean(path)
	} else {
		cleaned := filepath.Clean(path)
		fullPath = filepath.Join(s.sharedRoot, cleaned)
	}

	// Open file
	f, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &pb.ReadFileResponse{Error: fmt.Sprintf("file not found: %s", fullPath)}, nil
		}
		return &pb.ReadFileResponse{Error: fmt.Sprintf("open error: %v", err)}, nil
	}
	defer f.Close()

	// Get file info
	info, err := f.Stat()
	if err != nil {
		return &pb.ReadFileResponse{Error: fmt.Sprintf("stat error: %v", err)}, nil
	}
	if !info.Mode().IsRegular() {
		return &pb.ReadFileResponse{Error: "path is not a regular file"}, nil
	}

	fileSize := info.Size()

	// Determine max bytes (default 64KB, max 10MB)
	maxBytes := int64(65536)
	if req.MaxBytes > 0 {
		maxBytes = int64(req.MaxBytes)
	}
	if maxBytes > 10*1024*1024 {
		maxBytes = 10 * 1024 * 1024
	}

	var content []byte
	var truncated bool

	switch req.Mode {
	case pb.ReadMode_READ_MODE_HEAD, pb.ReadMode_READ_MODE_FULL:
		// Read from beginning
		toRead := fileSize
		if toRead > maxBytes {
			toRead = maxBytes
			truncated = true
		}
		content = make([]byte, toRead)
		n, err := f.Read(content)
		if err != nil && err != io.EOF {
			return &pb.ReadFileResponse{Error: fmt.Sprintf("read error: %v", err)}, nil
		}
		content = content[:n]

	case pb.ReadMode_READ_MODE_TAIL:
		// Read from end
		toRead := fileSize
		if toRead > maxBytes {
			toRead = maxBytes
			truncated = true
		}
		offset := fileSize - toRead
		if offset < 0 {
			offset = 0
		}
		if _, err := f.Seek(offset, 0); err != nil {
			return &pb.ReadFileResponse{Error: fmt.Sprintf("seek error: %v", err)}, nil
		}
		content = make([]byte, toRead)
		n, err := f.Read(content)
		if err != nil && err != io.EOF {
			return &pb.ReadFileResponse{Error: fmt.Sprintf("read error: %v", err)}, nil
		}
		content = content[:n]

	case pb.ReadMode_READ_MODE_RANGE:
		// Read specific range
		offset := req.Offset
		length := req.Length
		if length <= 0 {
			length = maxBytes
		}
		if length > maxBytes {
			length = maxBytes
			truncated = true
		}
		if offset >= fileSize {
			return &pb.ReadFileResponse{
				SizeBytes:     fileSize,
				BytesReturned: 0,
				Error:         "offset beyond file end",
			}, nil
		}
		if _, err := f.Seek(offset, 0); err != nil {
			return &pb.ReadFileResponse{Error: fmt.Sprintf("seek error: %v", err)}, nil
		}
		content = make([]byte, length)
		n, err := f.Read(content)
		if err != nil && err != io.EOF {
			return &pb.ReadFileResponse{Error: fmt.Sprintf("read error: %v", err)}, nil
		}
		content = content[:n]

	default:
		// Default to full read
		toRead := fileSize
		if toRead > maxBytes {
			toRead = maxBytes
			truncated = true
		}
		content = make([]byte, toRead)
		n, err := f.Read(content)
		if err != nil && err != io.EOF {
			return &pb.ReadFileResponse{Error: fmt.Sprintf("read error: %v", err)}, nil
		}
		content = content[:n]
	}

	// Generate preview (first ~2KB if text-like)
	preview := ""
	if len(content) > 0 && isTextLike(content) {
		previewLen := 2048
		if len(content) < previewLen {
			previewLen = len(content)
		}
		preview = string(content[:previewLen])
	}

	log.Printf("[INFO] ReadFile: path=%s mode=%v size=%d returned=%d truncated=%v",
		fullPath, req.Mode, fileSize, len(content), truncated)

	return &pb.ReadFileResponse{
		Content:        content,
		SizeBytes:      fileSize,
		BytesReturned:  int64(len(content)),
		Truncated:      truncated,
		ContentPreview: preview,
	}, nil
}

// forwardReadFile forwards a ReadFile request to a remote device
func (s *OrchestratorServer) forwardReadFile(ctx context.Context, req *pb.ReadFileRequest) (*pb.ReadFileResponse, error) {
	// Find device in registry
	devices := s.registry.List()
	var targetDevice *pb.DeviceInfo
	for _, d := range devices {
		if d.DeviceId == req.DeviceId {
			targetDevice = d
			break
		}
	}
	if targetDevice == nil {
		return &pb.ReadFileResponse{Error: fmt.Sprintf("device not found: %s", req.DeviceId)}, nil
	}

	// Dial remote device
	dialCtx, cancel := context.WithTimeout(ctx, remoteDialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, targetDevice.GrpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return &pb.ReadFileResponse{Error: fmt.Sprintf("failed to connect to device %s: %v", req.DeviceId, err)}, nil
	}
	defer conn.Close()

	client := pb.NewOrchestratorServiceClient(conn)

	// Create session on remote
	sessionResp, err := client.CreateSession(ctx, &pb.AuthRequest{
		DeviceName:  "coordinator-readfile",
		SecurityKey: "internal-routing",
	})
	if err != nil {
		return &pb.ReadFileResponse{Error: fmt.Sprintf("failed to create session on remote: %v", err)}, nil
	}

	// Forward request with remote session
	remoteReq := *req
	remoteReq.SessionId = sessionResp.SessionId
	remoteReq.DeviceId = "" // Clear device_id so remote reads locally

	return client.ReadFile(ctx, &remoteReq)
}

// isTextLike checks if content appears to be text (no null bytes, mostly printable)
func isTextLike(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	// Check first 512 bytes
	checkLen := 512
	if len(data) < checkLen {
		checkLen = len(data)
	}
	for i := 0; i < checkLen; i++ {
		b := data[i]
		// Null byte = binary
		if b == 0 {
			return false
		}
	}
	return true
}

// handleBulkDownload serves file bytes for a valid, unexpired token
func (s *OrchestratorServer) handleBulkDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract token from URL: /bulk/download/<token>
	const prefix = "/bulk/download/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	token := strings.TrimPrefix(r.URL.Path, prefix)
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	// Consume ticket (atomic: check + delete)
	ticket := s.ticketManager.Consume(token)
	if ticket == nil {
		http.Error(w, "invalid or expired token", http.StatusForbidden)
		return
	}

	// Open file
	f, err := os.Open(ticket.FilePath)
	if err != nil {
		log.Printf("[ERROR] handleBulkDownload: failed to open %s: %v", ticket.FilePath, err)
		http.Error(w, "file not accessible", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Set headers for download
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, ticket.Filename))
	w.Header().Set("Content-Length", strconv.FormatInt(ticket.SizeBytes, 10))

	// Stream file bytes
	written, err := io.Copy(w, f)
	if err != nil {
		log.Printf("[ERROR] handleBulkDownload: stream error after %d bytes: %v", written, err)
		return
	}

	log.Printf("[INFO] handleBulkDownload: served %s (%d bytes)", ticket.Filename, written)
}

// startBulkHTTP starts the HTTP server for bulk file downloads
func (s *OrchestratorServer) startBulkHTTP() {
	mux := http.NewServeMux()
	mux.HandleFunc("/bulk/download/", s.handleBulkDownload)

	log.Printf("[INFO] Bulk HTTP server listening on %s", s.bulkHTTPAddr)
	if err := http.ListenAndServe(s.bulkHTTPAddr, mux); err != nil {
		log.Fatalf("[FATAL] Bulk HTTP server failed: %v", err)
	}
}

// SyncChatMemory synchronizes chat memory between devices.
// If the sender's memory is newer, we update ours and return updated=true.
// If our memory is newer, we return our memory for the sender to update.
func (s *OrchestratorServer) SyncChatMemory(ctx context.Context, req *pb.ChatMemorySync) (*pb.ChatMemorySyncResponse, error) {
	// If sender is pushing their memory
	if req.MemoryJson != "" {
		incoming, err := chatmem.ParseFromJSON(req.MemoryJson)
		if err != nil {
			log.Printf("[ERROR] SyncChatMemory: failed to parse incoming JSON: %v", err)
			return nil, status.Error(codes.InvalidArgument, "invalid memory JSON")
		}

		// Merge if incoming is newer
		if s.chatMem.Merge(incoming) {
			if err := s.chatMem.SaveToFile(); err != nil {
				log.Printf("[WARN] SyncChatMemory: failed to save memory: %v", err)
			}
			log.Printf("[INFO] SyncChatMemory: updated local memory from remote (ts=%d)", incoming.GetLastUpdated())

			// Broadcast to all other registered devices
			go s.broadcastChatMemory()

			return &pb.ChatMemorySyncResponse{
				Updated:    true,
				MemoryJson: "",
			}, nil
		}
	}

	// Our memory is newer or same - return it
	memJSON, err := s.chatMem.ToJSON()
	if err != nil {
		log.Printf("[ERROR] SyncChatMemory: failed to serialize memory: %v", err)
		return nil, status.Error(codes.Internal, "failed to serialize memory")
	}

	return &pb.ChatMemorySyncResponse{
		Updated:    false,
		MemoryJson: memJSON,
	}, nil
}

// GetChatMemory returns the current chat memory.
func (s *OrchestratorServer) GetChatMemory(ctx context.Context, req *pb.Empty) (*pb.ChatMemoryData, error) {
	memJSON, err := s.chatMem.ToJSON()
	if err != nil {
		log.Printf("[ERROR] GetChatMemory: failed to serialize memory: %v", err)
		return nil, status.Error(codes.Internal, "failed to serialize memory")
	}

	log.Printf("[DEBUG] GetChatMemory: returning %d messages", len(s.chatMem.GetMessages()))

	return &pb.ChatMemoryData{
		MemoryJson: memJSON,
	}, nil
}

// RunLLMTask executes an LLM inference task on this device's local model
func (s *OrchestratorServer) RunLLMTask(ctx context.Context, req *pb.LLMTaskRequest) (*pb.LLMTaskResponse, error) {
	// Check if this device has a local model
	selfInfo := s.getSelfDeviceInfo()
	if !selfInfo.HasLocalModel {
		return &pb.LLMTaskResponse{
			Error: "no local LLM model available on this device",
		}, nil
	}

	log.Printf("[INFO] RunLLMTask: processing prompt (%d chars) with model %s", len(req.Prompt), selfInfo.LocalModelName)

	// Call Ollama directly
	model := req.Model
	if model == "" {
		model = selfInfo.LocalModelName
	}

	result, err := callOllamaChat(ctx, selfInfo.LocalChatEndpoint, model, req.Prompt)
	if err != nil {
		log.Printf("[ERROR] RunLLMTask: chat failed: %v", err)
		return &pb.LLMTaskResponse{
			Error: err.Error(),
		}, nil
	}

	log.Printf("[INFO] RunLLMTask: completed, output %d chars", len(result))

	return &pb.LLMTaskResponse{
		Output:          result,
		ModelUsed:       model,
		TokensGenerated: int64(len(result) / 4), // rough estimate: ~4 chars per token
	}, nil
}

// callOllamaChat calls the Ollama API directly for LLM inference
func callOllamaChat(ctx context.Context, baseURL, model, prompt string) (string, error) {
	reqBody := struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Stream bool `json:"stream"`
	}{
		Model: model,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/chat", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Message.Content, nil
}

// broadcastChatMemory pushes our chat memory to all registered devices.
func (s *OrchestratorServer) broadcastChatMemory() {
	devices := s.registry.List()
	if len(devices) <= 1 {
		return // Only self registered
	}

	memJSON, err := s.chatMem.ToJSON()
	if err != nil {
		log.Printf("[WARN] broadcastChatMemory: failed to serialize: %v", err)
		return
	}

	for _, device := range devices {
		if device.DeviceId == s.selfDeviceID {
			continue
		}

		go func(d *pb.DeviceInfo) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn, err := grpc.DialContext(ctx, d.GrpcAddr,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithBlock(),
			)
			if err != nil {
				log.Printf("[WARN] broadcastChatMemory: failed to dial %s: %v", d.GrpcAddr, err)
				return
			}
			defer conn.Close()

			client := pb.NewOrchestratorServiceClient(conn)
			_, err = client.SyncChatMemory(ctx, &pb.ChatMemorySync{
				LastUpdatedMs: s.chatMem.GetLastUpdated(),
				MemoryJson:    memJSON,
			})
			if err != nil {
				log.Printf("[WARN] broadcastChatMemory: sync to %s failed: %v", d.DeviceName, err)
			} else {
				log.Printf("[INFO] broadcastChatMemory: synced to %s", d.DeviceName)
			}
		}(device)
	}
}

// AddChatMessage adds a message to chat memory and broadcasts to all devices.
// This is called by the web server after LLM responses.
func (s *OrchestratorServer) AddChatMessage(role, content string) {
	s.chatMem.AddMessage(role, content)
	if err := s.chatMem.SaveToFile(); err != nil {
		log.Printf("[WARN] AddChatMessage: failed to save: %v", err)
	}
	go s.broadcastChatMemory()
}

// CreateInternalSession creates a session for internal web handler use
func (s *OrchestratorServer) CreateInternalSession(name string) string {
	sessionID := uuid.New().String()
	session := &Session{
		ID:          sessionID,
		DeviceName:  name,
		HostName:    "internal",
		ConnectedAt: time.Now(),
	}
	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()
	return sessionID
}

// ---- WebHandler HTTP Methods ----

// handleIndex serves the embedded index.html from webui folder
func (h *WebHandler) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Serve index.html for root path and any non-API, non-asset paths (SPA fallback)
	if r.URL.Path == "/" || (!strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasPrefix(r.URL.Path, "/assets/")) {
		content, err := staticFS.ReadFile("webui/index.html")
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(content)
		return
	}
	http.NotFound(w, r)
}

// handleAssets serves static assets (JS, CSS, images) from webui/assets folder
func (h *WebHandler) handleAssets(w http.ResponseWriter, r *http.Request) {
	// Strip /assets/ prefix and read from webui/assets/
	assetPath := "webui" + r.URL.Path

	content, err := staticFS.ReadFile(assetPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Set content type based on file extension
	ext := filepath.Ext(r.URL.Path)
	switch ext {
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	case ".json":
		w.Header().Set("Content-Type", "application/json")
	case ".woff", ".woff2":
		w.Header().Set("Content-Type", "font/woff2")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// Cache assets for 1 year (they have content hashes in filenames)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Write(content)
}

// handleDevices returns all registered devices
func (h *WebHandler) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), webRequestTimeout)
	defer cancel()

	resp, err := h.orchestrator.ListDevices(ctx, &pb.ListDevicesRequest{})
	if err != nil {
		log.Printf("[ERROR] ListDevices failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("gRPC error: %v", err))
		return
	}

	devices := make([]DeviceResponse, 0, len(resp.Devices))
	for _, d := range resp.Devices {
		caps := []string{"cpu"}
		if d.HasGpu {
			caps = append(caps, "gpu")
		}
		if d.HasNpu {
			caps = append(caps, "npu")
		}

		devices = append(devices, DeviceResponse{
			DeviceID:          d.DeviceId,
			DeviceName:        d.DeviceName,
			Platform:          d.Platform,
			Arch:              d.Arch,
			Capabilities:      caps,
			GRPCAddr:          d.GrpcAddr,
			CanScreenCapture:  d.CanScreenCapture,
			HttpAddr:          d.HttpAddr,
			HasLocalModel:     d.HasLocalModel,
			LocalModelName:    d.LocalModelName,
			LocalChatEndpoint: d.LocalChatEndpoint,
		})
	}

	h.writeJSON(w, http.StatusOK, devices)
}

// handleRoutedCmd executes a command on the best available device
func (h *WebHandler) handleRoutedCmd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req RoutedCmdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.Cmd == "" {
		h.writeError(w, http.StatusBadRequest, "cmd is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), webRequestTimeout)
	defer cancel()

	sessionID := h.orchestrator.CreateInternalSession("web-ui")

	policy := &pb.RoutingPolicy{
		Mode: pb.RoutingPolicy_BEST_AVAILABLE,
	}

	switch strings.ToUpper(req.Policy) {
	case "PREFER_REMOTE":
		policy.Mode = pb.RoutingPolicy_PREFER_REMOTE
	case "REQUIRE_NPU":
		policy.Mode = pb.RoutingPolicy_REQUIRE_NPU
	case "FORCE_DEVICE_ID":
		policy.Mode = pb.RoutingPolicy_FORCE_DEVICE_ID
		policy.DeviceId = req.ForceDeviceID
	}

	cmdResp, err := h.orchestrator.ExecuteRoutedCommand(ctx, &pb.RoutedCommandRequest{
		SessionId: sessionID,
		Policy:    policy,
		Command:   req.Cmd,
		Args:      req.Args,
	})
	if err != nil {
		log.Printf("[ERROR] ExecuteRoutedCommand failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Command error: %v", err))
		return
	}

	resp := RoutedCmdResponse{
		SelectedDeviceName: cmdResp.SelectedDeviceName,
		SelectedDeviceID:   cmdResp.SelectedDeviceId,
		SelectedDeviceAddr: cmdResp.SelectedDeviceAddr,
		ExecutedLocally:    cmdResp.ExecutedLocally,
		TotalTimeMs:        cmdResp.TotalTimeMs,
		ExitCode:           cmdResp.Output.ExitCode,
		Stdout:             cmdResp.Output.Stdout,
		Stderr:             cmdResp.Output.Stderr,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// handleAssistant processes natural language commands
func (h *WebHandler) handleAssistant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req AssistantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	text := strings.ToLower(strings.TrimSpace(req.Text))

	ctx, cancel := context.WithTimeout(r.Context(), webRequestTimeout)
	defer cancel()

	var reply string
	var raw interface{}
	var mode string
	var jobID string
	var planDebug interface{}

	switch {
	case strings.Contains(text, "list devices") || strings.Contains(text, "show devices") || strings.Contains(text, "devices"):
		resp, err := h.orchestrator.ListDevices(ctx, &pb.ListDevicesRequest{})
		if err != nil {
			reply = fmt.Sprintf("Error listing devices: %v", err)
		} else if len(resp.Devices) == 0 {
			reply = "No devices registered."
		} else {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Found %d device(s):\n", len(resp.Devices)))
			for i, d := range resp.Devices {
				caps := "cpu"
				if d.HasGpu {
					caps += ",gpu"
				}
				if d.HasNpu {
					caps += ",npu"
				}
				sb.WriteString(fmt.Sprintf("%d. %s (%s/%s) [%s] - %s\n",
					i+1, d.DeviceName, d.Platform, d.Arch, caps, d.GrpcAddr))
			}
			reply = sb.String()
			raw = resp.Devices
		}

	case strings.Contains(text, "pwd"):
		reply, raw = h.executeAssistantCommand(ctx, "pwd", nil)

	case strings.Contains(text, "ls") || strings.Contains(text, "list files"):
		reply, raw = h.executeAssistantCommand(ctx, "ls", nil)

	case strings.Contains(text, "cat"):
		parts := strings.Fields(text)
		var filePath string
		for i, p := range parts {
			if p == "cat" && i+1 < len(parts) {
				filePath = parts[i+1]
				break
			}
		}
		if filePath != "" {
			reply, raw = h.executeAssistantCommand(ctx, "cat", []string{filePath})
		} else {
			reply = "Please specify a file path. Example: 'cat ./shared/test.txt'"
		}

	default:
		reply, mode, jobID, planDebug = h.handleAssistantDefault(ctx, req.Text)
	}

	h.writeJSON(w, http.StatusOK, AssistantResponse{
		Reply: reply,
		Raw:   raw,
		Mode:  mode,
		JobID: jobID,
		Plan:  planDebug,
	})
}

// executeAssistantCommand runs a command and returns formatted output
func (h *WebHandler) executeAssistantCommand(ctx context.Context, cmd string, args []string) (string, interface{}) {
	sessionID := h.orchestrator.CreateInternalSession("web-ui-assistant")

	cmdResp, err := h.orchestrator.ExecuteRoutedCommand(ctx, &pb.RoutedCommandRequest{
		SessionId: sessionID,
		Policy:    &pb.RoutingPolicy{Mode: pb.RoutingPolicy_BEST_AVAILABLE},
		Command:   cmd,
		Args:      args,
	})
	if err != nil {
		return fmt.Sprintf("Command error: %v", err), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Executed on: %s\n", cmdResp.SelectedDeviceName))
	sb.WriteString(fmt.Sprintf("Time: %.2f ms\n", cmdResp.TotalTimeMs))
	sb.WriteString("---\n")

	if cmdResp.Output.Stdout != "" {
		sb.WriteString(cmdResp.Output.Stdout)
	}
	if cmdResp.Output.Stderr != "" {
		sb.WriteString("\n[stderr]\n")
		sb.WriteString(cmdResp.Output.Stderr)
	}
	if cmdResp.Output.ExitCode != 0 {
		sb.WriteString(fmt.Sprintf("\n[exit code: %d]", cmdResp.Output.ExitCode))
	}

	return sb.String(), cmdResp
}

// handleAssistantDefault handles the default case in the assistant
func (h *WebHandler) handleAssistantDefault(ctx context.Context, userText string) (reply, mode, jobID string, planDebug interface{}) {
	devicesResp, err := h.orchestrator.ListDevices(ctx, &pb.ListDevicesRequest{})
	if err != nil {
		return fmt.Sprintf("Error listing devices: %v", err), "", "", nil
	}
	if len(devicesResp.Devices) == 0 {
		return "No devices registered. Register a device first.", "", "", nil
	}

	if h.llm != nil {
		type deviceCompact struct {
			DeviceID string `json:"device_id"`
			Name     string `json:"name"`
			HasNPU   bool   `json:"has_npu"`
			HasGPU   bool   `json:"has_gpu"`
			HasCPU   bool   `json:"has_cpu"`
		}
		devList := make([]deviceCompact, len(devicesResp.Devices))
		for i, d := range devicesResp.Devices {
			devList[i] = deviceCompact{
				DeviceID: d.DeviceId,
				Name:     d.DeviceName,
				HasNPU:   d.HasNpu,
				HasGPU:   d.HasGpu,
				HasCPU:   d.HasCpu,
			}
		}
		devJSON, _ := json.MarshalIndent(devList, "", "  ")

		planRaw, err := h.llm.Plan(ctx, userText, string(devJSON))
		if err != nil {
			log.Printf("[WARN] LLM plan generation failed: %v — falling back", err)
		} else {
			plan, reduce, err := llm.ParsePlanJSON(planRaw)
			if err != nil {
				log.Printf("[WARN] LLM plan validation failed: %v — falling back", err)
			} else {
				sessionID := h.orchestrator.CreateInternalSession("web-ui-assistant")

				jobResp, err := h.orchestrator.SubmitJob(ctx, &pb.JobRequest{
					SessionId: sessionID,
					Text:      userText,
					Plan:      plan,
					Reduce:    reduce,
				})
				if err != nil {
					log.Printf("[WARN] LLM plan submit failed: %v — falling back", err)
				} else {
					var planJSON interface{}
					json.Unmarshal([]byte(planRaw), &planJSON)

					return fmt.Sprintf("Job submitted via LLM planner.\nJob ID: %s", jobResp.JobId),
						"llm", jobResp.JobId, planJSON
				}
			}
		}
	}

	sessionID := h.orchestrator.CreateInternalSession("web-ui-assistant")

	jobResp, err := h.orchestrator.SubmitJob(ctx, &pb.JobRequest{
		SessionId: sessionID,
		Text:      userText,
	})
	if err != nil {
		return fmt.Sprintf("Job submission error: %v", err), "", "", nil
	}

	return fmt.Sprintf("Job submitted via fallback planner.\nJob ID: %s", jobResp.JobId),
		"fallback", jobResp.JobId, nil
}

// handleSubmitJob submits a distributed job to all devices
func (h *WebHandler) handleSubmitJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req SubmitJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), webRequestTimeout)
	defer cancel()

	sessionID := h.orchestrator.CreateInternalSession("web-ui")

	jobResp, err := h.orchestrator.SubmitJob(ctx, &pb.JobRequest{
		SessionId:  sessionID,
		Text:       req.Text,
		MaxWorkers: req.MaxWorkers,
	})
	if err != nil {
		log.Printf("[ERROR] SubmitJob failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Job error: %v", err))
		return
	}

	h.writeJSON(w, http.StatusOK, JobInfoResponse{
		JobID:     jobResp.JobId,
		CreatedAt: jobResp.CreatedAt,
		Summary:   jobResp.Summary,
	})
}

// handleGetJob returns the status of a job
func (h *WebHandler) handleGetJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		h.writeError(w, http.StatusBadRequest, "id parameter is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), webRequestTimeout)
	defer cancel()

	jobResp, err := h.orchestrator.GetJob(ctx, &pb.JobId{JobId: jobID})
	if err != nil {
		log.Printf("[ERROR] GetJob failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Job error: %v", err))
		return
	}

	tasks := make([]TaskStatusResponse, len(jobResp.Tasks))
	for i, t := range jobResp.Tasks {
		tasks[i] = TaskStatusResponse{
			TaskID:             t.TaskId,
			AssignedDeviceID:   t.AssignedDeviceId,
			AssignedDeviceName: t.AssignedDeviceName,
			State:              t.State,
			Result:             t.Result,
			Error:              t.Error,
		}
	}

	h.writeJSON(w, http.StatusOK, JobStatusResponse{
		JobID:        jobResp.JobId,
		State:        jobResp.State,
		Tasks:        tasks,
		FinalResult:  jobResp.FinalResult,
		CurrentGroup: jobResp.CurrentGroup,
		TotalGroups:  jobResp.TotalGroups,
	})
}

// handleActivity returns current activity data for the Activity Panel
func (h *WebHandler) handleActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	includeHistory := r.URL.Query().Get("include_metrics_history") == "true"
	sinceMs := int64(0)
	if sinceStr := r.URL.Query().Get("since_ms"); sinceStr != "" {
		sinceMs, _ = strconv.ParseInt(sinceStr, 10, 64)
	}

	ctx, cancel := context.WithTimeout(r.Context(), webRequestTimeout)
	defer cancel()

	resp, err := h.orchestrator.GetActivity(ctx, &pb.GetActivityRequest{
		IncludeMetricsHistory: includeHistory,
		MetricsSinceMs:        sinceMs,
	})
	if err != nil {
		log.Printf("[ERROR] GetActivity failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Activity error: %v", err))
		return
	}

	// Convert to JSON-friendly format
	result := map[string]interface{}{
		"activity": map[string]interface{}{
			"running_tasks":     resp.Activity.RunningTasks,
			"device_activities": resp.Activity.DeviceActivities,
		},
	}

	if includeHistory && resp.DeviceMetrics != nil {
		result["device_metrics"] = resp.DeviceMetrics
	}

	h.writeJSON(w, http.StatusOK, result)
}

// handleDeviceMetrics returns metrics history for a specific device
func (h *WebHandler) handleDeviceMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		h.writeError(w, http.StatusBadRequest, "device_id parameter is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), webRequestTimeout)
	defer cancel()

	resp, err := h.orchestrator.GetDeviceMetrics(ctx, &pb.DeviceId{DeviceId: deviceID})
	if err != nil {
		log.Printf("[ERROR] GetDeviceMetrics failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Metrics error: %v", err))
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// handleJobDetail returns enhanced job details for visualization
func (h *WebHandler) handleJobDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		h.writeError(w, http.StatusBadRequest, "id parameter is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), webRequestTimeout)
	defer cancel()

	resp, err := h.orchestrator.GetJobDetail(ctx, &pb.JobId{JobId: jobID})
	if err != nil {
		log.Printf("[ERROR] GetJobDetail failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Job detail error: %v", err))
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// handlePreviewPlan generates a plan preview without creating a job
func (h *WebHandler) handlePreviewPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req PreviewPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), webRequestTimeout)
	defer cancel()

	sessionID := h.orchestrator.CreateInternalSession("web-ui")

	planResp, err := h.orchestrator.PreviewPlan(ctx, &pb.PlanPreviewRequest{
		SessionId:  sessionID,
		Text:       req.Text,
		MaxWorkers: req.MaxWorkers,
	})
	if err != nil {
		log.Printf("[ERROR] handlePreviewPlan: PreviewPlan failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Plan error: %v", err))
		return
	}

	type taskJSON struct {
		TaskID         string `json:"task_id"`
		Kind           string `json:"kind"`
		Input          string `json:"input"`
		TargetDeviceID string `json:"target_device_id"`
	}
	type groupJSON struct {
		Index int32      `json:"index"`
		Tasks []taskJSON `json:"tasks"`
	}
	type planJSON struct {
		Groups []groupJSON `json:"groups"`
	}

	var plan planJSON
	if planResp.Plan != nil {
		for _, g := range planResp.Plan.Groups {
			group := groupJSON{Index: g.Index}
			for _, t := range g.Tasks {
				group.Tasks = append(group.Tasks, taskJSON{
					TaskID:         t.TaskId,
					Kind:           t.Kind,
					Input:          t.Input,
					TargetDeviceID: t.TargetDeviceId,
				})
			}
			plan.Groups = append(plan.Groups, group)
		}
	}

	var reduce interface{}
	if planResp.Reduce != nil {
		reduce = map[string]string{"kind": planResp.Reduce.Kind}
	}

	h.writeJSON(w, http.StatusOK, PreviewPlanResponse{
		UsedAi:    planResp.UsedAi,
		Notes:     planResp.Notes,
		Rationale: planResp.Rationale,
		Plan:      plan,
		Reduce:    reduce,
	})
}

// handlePlanCost estimates execution cost for a plan
func (h *WebHandler) handlePlanCost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req PlanCostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	planProto, err := h.convertJSONToPlan(req.Plan)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid plan: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), webRequestTimeout)
	defer cancel()

	sessionID := h.orchestrator.CreateInternalSession("web-ui")

	costResp, err := h.orchestrator.PreviewPlanCost(ctx, &pb.PlanCostRequest{
		SessionId: sessionID,
		Plan:      planProto,
		DeviceIds: req.DeviceIDs,
	})
	if err != nil {
		log.Printf("[ERROR] handlePlanCost: PreviewPlanCost failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Cost estimation error: %v", err))
		return
	}

	deviceCosts := make([]DeviceCostResponse, len(costResp.DeviceCosts))
	for i, dc := range costResp.DeviceCosts {
		stepCosts := make([]StepCostResponse, len(dc.StepCosts))
		for j, sc := range dc.StepCosts {
			stepCosts[j] = StepCostResponse{
				TaskID:            sc.TaskId,
				Kind:              sc.Kind,
				PredictedMs:       sc.PredictedMs,
				PredictedMemoryMB: sc.PredictedMemoryMb,
				UnknownCost:       sc.UnknownCost,
				Notes:             sc.Notes,
			}
		}
		deviceCosts[i] = DeviceCostResponse{
			DeviceID:           dc.DeviceId,
			DeviceName:         dc.DeviceName,
			TotalMs:            dc.TotalMs,
			StepCosts:          stepCosts,
			EstimatedPeakRAMMB: dc.EstimatedPeakRamMb,
			RAMSufficient:      dc.RamSufficient,
		}
	}

	h.writeJSON(w, http.StatusOK, PlanCostResponse{
		TotalPredictedMs:      costResp.TotalPredictedMs,
		DeviceCosts:           deviceCosts,
		RecommendedDeviceID:   costResp.RecommendedDeviceId,
		RecommendedDeviceName: costResp.RecommendedDeviceName,
		HasUnknownCosts:       costResp.HasUnknownCosts,
		Warning:               costResp.Warning,
	})
}

// convertJSONToPlan converts a JSON plan to a proto Plan
func (h *WebHandler) convertJSONToPlan(planJSON interface{}) (*pb.Plan, error) {
	jsonBytes, err := json.Marshal(planJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plan: %v", err)
	}

	type taskInput struct {
		TaskID          string `json:"task_id"`
		Kind            string `json:"kind"`
		Input           string `json:"input"`
		TargetDeviceID  string `json:"target_device_id"`
		PromptTokens    int32  `json:"prompt_tokens"`
		MaxOutputTokens int32  `json:"max_output_tokens"`
	}
	type groupInput struct {
		Index int32       `json:"index"`
		Tasks []taskInput `json:"tasks"`
	}
	type planInput struct {
		Groups []groupInput `json:"groups"`
	}

	var plan planInput
	if err := json.Unmarshal(jsonBytes, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan: %v", err)
	}

	if len(plan.Groups) == 0 {
		return nil, fmt.Errorf("plan must have at least one group")
	}

	protoPlan := &pb.Plan{
		Groups: make([]*pb.TaskGroup, len(plan.Groups)),
	}
	for i, g := range plan.Groups {
		protoGroup := &pb.TaskGroup{
			Index: g.Index,
			Tasks: make([]*pb.TaskSpec, len(g.Tasks)),
		}
		for j, t := range g.Tasks {
			protoGroup.Tasks[j] = &pb.TaskSpec{
				TaskId:          t.TaskID,
				Kind:            t.Kind,
				Input:           t.Input,
				TargetDeviceId:  t.TargetDeviceID,
				PromptTokens:    t.PromptTokens,
				MaxOutputTokens: t.MaxOutputTokens,
			}
		}
		protoPlan.Groups[i] = protoGroup
	}

	return protoPlan, nil
}

// handleStreamStart initiates a WebRTC stream from a selected device
func (h *WebHandler) handleStreamStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req StreamStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), webRequestTimeout)
	defer cancel()

	sessionID := h.orchestrator.CreateInternalSession("web-stream")

	devicesResp, err := h.orchestrator.ListDevices(ctx, &pb.ListDevicesRequest{})
	if err != nil {
		log.Printf("[ERROR] handleStreamStart: ListDevices failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("ListDevices error: %v", err))
		return
	}

	if len(devicesResp.Devices) == 0 {
		h.writeError(w, http.StatusNotFound, "No devices available")
		return
	}

	var selectedDevice *pb.DeviceInfo
	switch strings.ToUpper(req.Policy) {
	case "FORCE_DEVICE_ID":
		for _, d := range devicesResp.Devices {
			if d.DeviceId == req.ForceDeviceID {
				selectedDevice = d
				break
			}
		}
		if selectedDevice == nil {
			h.writeError(w, http.StatusNotFound, fmt.Sprintf("Device not found: %s", req.ForceDeviceID))
			return
		}
	case "PREFER_REMOTE":
		// Select non-local device by comparing device IDs
		for _, d := range devicesResp.Devices {
			if d.DeviceId != h.orchestrator.selfDeviceID {
				selectedDevice = d
				break
			}
		}
		if selectedDevice == nil {
			h.writeError(w, http.StatusNotFound, "No remote device available")
			return
		}
	default:
		for _, d := range devicesResp.Devices {
			if d.HasNpu {
				selectedDevice = d
				break
			}
		}
		if selectedDevice == nil {
			for _, d := range devicesResp.Devices {
				if d.HasGpu {
					selectedDevice = d
					break
				}
			}
		}
		if selectedDevice == nil {
			selectedDevice = devicesResp.Devices[0]
		}
	}

	log.Printf("[INFO] handleStreamStart: selected device %s (%s)", selectedDevice.DeviceName, selectedDevice.GrpcAddr)

	// For local device, call directly
	if selectedDevice.DeviceId == h.orchestrator.selfDeviceID {
		webrtcResp, err := h.orchestrator.StartWebRTC(ctx, &pb.WebRTCConfig{
			SessionId:    sessionID,
			TargetFps:    req.FPS,
			JpegQuality:  req.Quality,
			MonitorIndex: req.MonitorIndex,
		})
		if err != nil {
			log.Printf("[ERROR] handleStreamStart: StartWebRTC failed: %v", err)
			h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("StartWebRTC error: %v", err))
			return
		}

		h.writeJSON(w, http.StatusOK, StreamStartResponse{
			SelectedDeviceID:   selectedDevice.DeviceId,
			SelectedDeviceName: selectedDevice.DeviceName,
			SelectedDeviceAddr: selectedDevice.GrpcAddr,
			StreamID:           webrtcResp.StreamId,
			OfferSDP:           webrtcResp.Sdp,
		})
		return
	}

	// For remote device, dial via gRPC
	dialCtx, dialCancel := context.WithTimeout(ctx, 5*time.Second)
	defer dialCancel()

	conn, err := grpc.DialContext(dialCtx, selectedDevice.GrpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("[ERROR] handleStreamStart: failed to dial device %s: %v", selectedDevice.GrpcAddr, err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to connect to device: %v", err))
		return
	}
	defer conn.Close()

	deviceClient := pb.NewOrchestratorServiceClient(conn)

	webrtcResp, err := deviceClient.StartWebRTC(ctx, &pb.WebRTCConfig{
		SessionId:    sessionID,
		TargetFps:    req.FPS,
		JpegQuality:  req.Quality,
		MonitorIndex: req.MonitorIndex,
	})
	if err != nil {
		log.Printf("[ERROR] handleStreamStart: StartWebRTC failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("StartWebRTC error: %v", err))
		return
	}

	log.Printf("[INFO] handleStreamStart: stream %s started on %s", webrtcResp.StreamId, selectedDevice.DeviceName)

	h.writeJSON(w, http.StatusOK, StreamStartResponse{
		SelectedDeviceID:   selectedDevice.DeviceId,
		SelectedDeviceName: selectedDevice.DeviceName,
		SelectedDeviceAddr: selectedDevice.GrpcAddr,
		StreamID:           webrtcResp.StreamId,
		OfferSDP:           webrtcResp.Sdp,
	})
}

// handleStreamAnswer completes the WebRTC handshake with the answer SDP
func (h *WebHandler) handleStreamAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req StreamAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.SelectedDeviceAddr == "" || req.StreamID == "" || req.AnswerSDP == "" {
		h.writeError(w, http.StatusBadRequest, "selected_device_addr, stream_id, and answer_sdp are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), webRequestTimeout)
	defer cancel()

	// Check if local
	if req.SelectedDeviceAddr == h.orchestrator.selfAddr {
		_, err := h.orchestrator.CompleteWebRTC(ctx, &pb.WebRTCAnswer{
			StreamId: req.StreamID,
			Sdp:      req.AnswerSDP,
		})
		if err != nil {
			log.Printf("[ERROR] handleStreamAnswer: CompleteWebRTC failed: %v", err)
			h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("CompleteWebRTC error: %v", err))
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	dialCtx, dialCancel := context.WithTimeout(ctx, 5*time.Second)
	defer dialCancel()

	conn, err := grpc.DialContext(dialCtx, req.SelectedDeviceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("[ERROR] handleStreamAnswer: failed to dial %s: %v", req.SelectedDeviceAddr, err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to connect to device: %v", err))
		return
	}
	defer conn.Close()

	deviceClient := pb.NewOrchestratorServiceClient(conn)

	_, err = deviceClient.CompleteWebRTC(ctx, &pb.WebRTCAnswer{
		StreamId: req.StreamID,
		Sdp:      req.AnswerSDP,
	})
	if err != nil {
		log.Printf("[ERROR] handleStreamAnswer: CompleteWebRTC failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("CompleteWebRTC error: %v", err))
		return
	}

	log.Printf("[INFO] handleStreamAnswer: stream %s connected", req.StreamID)

	h.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleStreamStop stops an active WebRTC stream
func (h *WebHandler) handleStreamStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req StreamStopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.SelectedDeviceAddr == "" || req.StreamID == "" {
		h.writeError(w, http.StatusBadRequest, "selected_device_addr and stream_id are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), webRequestTimeout)
	defer cancel()

	// Check if local
	if req.SelectedDeviceAddr == h.orchestrator.selfAddr {
		_, err := h.orchestrator.StopWebRTC(ctx, &pb.WebRTCStop{
			StreamId: req.StreamID,
		})
		if err != nil {
			log.Printf("[ERROR] handleStreamStop: StopWebRTC failed: %v", err)
			h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("StopWebRTC error: %v", err))
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	dialCtx, dialCancel := context.WithTimeout(ctx, 5*time.Second)
	defer dialCancel()

	conn, err := grpc.DialContext(dialCtx, req.SelectedDeviceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("[ERROR] handleStreamStop: failed to dial %s: %v", req.SelectedDeviceAddr, err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to connect to device: %v", err))
		return
	}
	defer conn.Close()

	deviceClient := pb.NewOrchestratorServiceClient(conn)

	_, err = deviceClient.StopWebRTC(ctx, &pb.WebRTCStop{
		StreamId: req.StreamID,
	})
	if err != nil {
		log.Printf("[ERROR] handleStreamStop: StopWebRTC failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("StopWebRTC error: %v", err))
		return
	}

	log.Printf("[INFO] handleStreamStop: stream %s stopped", req.StreamID)

	h.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleRequestDownload creates a download ticket on a target device and returns a direct URL
func (h *WebHandler) handleRequestDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.DeviceID == "" {
		h.writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	if req.Path == "" {
		h.writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), webRequestTimeout)
	defer cancel()

	devicesResp, err := h.orchestrator.ListDevices(ctx, &pb.ListDevicesRequest{})
	if err != nil {
		log.Printf("[ERROR] handleRequestDownload: ListDevices failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("ListDevices error: %v", err))
		return
	}

	var targetDevice *pb.DeviceInfo
	for _, d := range devicesResp.Devices {
		if d.DeviceId == req.DeviceID {
			targetDevice = d
			break
		}
	}
	if targetDevice == nil {
		h.writeError(w, http.StatusNotFound, fmt.Sprintf("Device not found: %s", req.DeviceID))
		return
	}
	if targetDevice.HttpAddr == "" {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Device %s has no HTTP address configured", targetDevice.DeviceName))
		return
	}

	// For local device, call directly
	if targetDevice.DeviceId == h.orchestrator.selfDeviceID {
		ticketResp, err := h.orchestrator.CreateDownloadTicket(ctx, &pb.DownloadTicketRequest{
			Path: req.Path,
		})
		if err != nil {
			log.Printf("[ERROR] handleRequestDownload: CreateDownloadTicket failed: %v", err)
			h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Ticket error: %v", err))
			return
		}

		downloadURL := fmt.Sprintf("http://%s/bulk/download/%s", targetDevice.HttpAddr, ticketResp.Token)

		h.writeJSON(w, http.StatusOK, DownloadResponse{
			DownloadURL:   downloadURL,
			Filename:      ticketResp.Filename,
			SizeBytes:     ticketResp.SizeBytes,
			ExpiresUnixMs: ticketResp.ExpiresUnixMs,
		})
		return
	}

	dialCtx, dialCancel := context.WithTimeout(ctx, 5*time.Second)
	defer dialCancel()

	conn, err := grpc.DialContext(dialCtx, targetDevice.GrpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("[ERROR] handleRequestDownload: failed to dial device %s: %v", targetDevice.GrpcAddr, err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to connect to device: %v", err))
		return
	}
	defer conn.Close()

	deviceClient := pb.NewOrchestratorServiceClient(conn)

	ticketResp, err := deviceClient.CreateDownloadTicket(ctx, &pb.DownloadTicketRequest{
		Path: req.Path,
	})
	if err != nil {
		log.Printf("[ERROR] handleRequestDownload: CreateDownloadTicket failed: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Ticket error: %v", err))
		return
	}

	downloadURL := fmt.Sprintf("http://%s/bulk/download/%s", targetDevice.HttpAddr, ticketResp.Token)

	log.Printf("[INFO] handleRequestDownload: ticket for %s on %s, url=%s",
		req.Path, targetDevice.DeviceName, downloadURL)

	h.writeJSON(w, http.StatusOK, DownloadResponse{
		DownloadURL:   downloadURL,
		Filename:      ticketResp.Filename,
		SizeBytes:     ticketResp.SizeBytes,
		ExpiresUnixMs: ticketResp.ExpiresUnixMs,
	})
}

// handleQaihubDoctor runs qai-hub diagnostics
func (h *WebHandler) handleQaihubDoctor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	result, err := h.qaihubClient.Doctor(ctx)
	if err != nil {
		log.Printf("[ERROR] handleQaihubDoctor: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Doctor failed: %v", err))
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

// handleQaihubCompile runs model compilation using qai-hub
func (h *WebHandler) handleQaihubCompile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req CompileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.ONNXPath == "" {
		h.writeError(w, http.StatusBadRequest, "onnx_path is required")
		return
	}
	if req.Target == "" {
		h.writeError(w, http.StatusBadRequest, "target is required")
		return
	}

	if !filepath.IsAbs(req.ONNXPath) {
		h.writeError(w, http.StatusBadRequest, "onnx_path must be an absolute path")
		return
	}

	if strings.Contains(req.ONNXPath, "..") {
		h.writeError(w, http.StatusBadRequest, "onnx_path cannot contain path traversal (..)")
		return
	}

	info, err := os.Stat(req.ONNXPath)
	if os.IsNotExist(err) {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("ONNX file not found: %s", req.ONNXPath))
		return
	}
	if info.IsDir() {
		h.writeError(w, http.StatusBadRequest, "onnx_path must be a file, not a directory")
		return
	}

	if !h.qaihubClient.IsAvailable() {
		h.writeError(w, http.StatusServiceUnavailable, "qai-hub CLI is not available. Run qaihub doctor for setup instructions.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	outDir := filepath.Join("artifacts", "qaihub", time.Now().Format("20060102-150405"))

	result, err := h.qaihubClient.Compile(ctx, req.ONNXPath, req.Target, req.Runtime, outDir)
	if err != nil {
		log.Printf("[ERROR] handleQaihubCompile: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Compile failed: %v", err))
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

// handleQaihubDevices returns the QAI Hub cloud device catalog
func (h *WebHandler) handleQaihubDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if !h.qaihubClient.IsAvailable() {
		h.writeError(w, http.StatusServiceUnavailable, "qai-hub CLI is not available")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	result, err := h.qaihubClient.ListDevices(ctx)
	if err != nil {
		log.Printf("[ERROR] handleQaihubDevices: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("ListDevices failed: %v", err))
		return
	}

	nameFilter := r.URL.Query().Get("name")
	chipsetFilter := r.URL.Query().Get("chipset")
	vendorFilter := r.URL.Query().Get("vendor")

	devices := result.Devices
	if nameFilter != "" {
		var filtered []qaihub.TargetDevice
		for _, d := range devices {
			if strings.Contains(strings.ToLower(d.Name), strings.ToLower(nameFilter)) {
				filtered = append(filtered, d)
			}
		}
		devices = filtered
	}
	if chipsetFilter != "" {
		var filtered []qaihub.TargetDevice
		for _, d := range devices {
			if strings.Contains(strings.ToLower(d.Chipset), strings.ToLower(chipsetFilter)) {
				filtered = append(filtered, d)
			}
		}
		devices = filtered
	}
	if vendorFilter != "" {
		var filtered []qaihub.TargetDevice
		for _, d := range devices {
			if strings.Contains(strings.ToLower(d.Vendor), strings.ToLower(vendorFilter)) {
				filtered = append(filtered, d)
			}
		}
		devices = filtered
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":   len(devices),
		"devices": devices,
	})
}

// handleQaihubJobStatus checks the status of a QAI Hub compile job
func (h *WebHandler) handleQaihubJobStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var jobID string
	if r.Method == http.MethodGet {
		jobID = r.URL.Query().Get("job_id")
	} else {
		var req QaihubJobStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
			return
		}
		jobID = req.JobID
	}

	if jobID == "" {
		h.writeError(w, http.StatusBadRequest, "job_id is required")
		return
	}

	if !h.qaihubClient.IsAvailable() {
		h.writeError(w, http.StatusServiceUnavailable, "qai-hub CLI is not available")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := h.qaihubClient.GetJobStatus(ctx, jobID)
	if err != nil {
		log.Printf("[ERROR] handleQaihubJobStatus: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Job status check failed: %v", err))
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

// handleQaihubSubmitCompile submits a compile job via the Python SDK
func (h *WebHandler) handleQaihubSubmitCompile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req QaihubSubmitCompileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.Model == "" {
		h.writeError(w, http.StatusBadRequest, "model is required (ONNX path or model ID)")
		return
	}
	if req.DeviceName == "" {
		req.DeviceName = "Samsung Galaxy S24 (Family)"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	log.Printf("[INFO] handleQaihubSubmitCompile: model=%s device=%s", req.Model, req.DeviceName)

	result, err := h.qaihubClient.SubmitCompile(ctx, req.Model, req.DeviceName, req.Options)
	if err != nil {
		log.Printf("[ERROR] handleQaihubSubmitCompile: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Compile submission failed: %v", err))
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

// handleChat handles chat requests using the local LLM runtime
func (h *WebHandler) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if h.chat == nil {
		h.writeError(w, http.StatusServiceUnavailable, "Chat provider is not configured. Set CHAT_PROVIDER environment variable.")
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if len(req.Messages) == 0 {
		h.writeError(w, http.StatusBadRequest, "messages array is required and must not be empty")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	reply, err := h.chat.Chat(ctx, req.Messages)
	if err != nil {
		log.Printf("[ERROR] handleChat: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Chat failed: %v", err))
		return
	}

	// Update chat memory
	if len(req.Messages) > 0 {
		lastMsg := req.Messages[len(req.Messages)-1]
		if lastMsg.Role == "user" {
			h.orchestrator.chatMem.AddMessage("user", lastMsg.Content)
		}
	}
	h.orchestrator.chatMem.AddMessage("assistant", reply)

	// Trigger broadcast
	go h.orchestrator.broadcastChatMemory()

	// Trigger background summarization if needed
	h.orchestrator.chatMem.SummarizeAsync(h.performSummarization, func() {
		h.orchestrator.broadcastChatMemory()
	})

	h.writeJSON(w, http.StatusOK, ChatResponse{Reply: reply})
}

// handleChatMemory returns the current chat history
func (h *WebHandler) handleChatMemory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jsonStr, err := h.orchestrator.chatMem.ToJSON()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to serialize memory")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(jsonStr))
}

// handleChatHealth checks the chat runtime status
func (h *WebHandler) handleChatHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if h.chat == nil {
		h.writeJSON(w, http.StatusOK, &llm.HealthResult{
			Ok:       false,
			Provider: "none",
			Error:    "Chat provider is not configured",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	result, err := h.chat.Health(ctx)
	if err != nil {
		log.Printf("[ERROR] handleChatHealth: %v", err)
		h.writeJSON(w, http.StatusOK, &llm.HealthResult{
			Ok:       false,
			Provider: h.chat.Name(),
			Error:    err.Error(),
		})
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

// handleAgent handles LLM tool-calling agent requests
func (h *WebHandler) handleAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if h.agent == nil {
		h.writeError(w, http.StatusServiceUnavailable, "Agent is not available. Check CHAT_PROVIDER configuration and gRPC server connectivity.")
		return
	}

	var req AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.Message == "" {
		h.writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	log.Printf("[INFO] handleAgent: processing message: %q", truncate(req.Message, 100))

	var history []llm.ToolChatMessage

	summary := h.orchestrator.chatMem.GetSummary()
	if summary != "" {
		history = append(history, llm.ToolChatMessage{
			Role:    "system",
			Content: fmt.Sprintf("Summary of previous conversation:\n%s", summary),
		})
	}

	msgs := h.orchestrator.chatMem.GetMessages()
	for _, m := range msgs {
		history = append(history, llm.ToolChatMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	resp, err := h.agent.Run(ctx, req.Message, history)
	if err != nil {
		log.Printf("[ERROR] handleAgent: %v", err)
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Agent error: %v", err))
		return
	}

	h.orchestrator.chatMem.AddMessage("user", req.Message)
	h.orchestrator.chatMem.AddMessage("assistant", resp.Reply)
	go h.orchestrator.broadcastChatMemory()

	h.orchestrator.chatMem.SummarizeAsync(h.performSummarization, func() {
		h.orchestrator.broadcastChatMemory()
	})

	toolCalls := make([]AgentToolCallInfo, len(resp.ToolCalls))
	for i, tc := range resp.ToolCalls {
		toolCalls[i] = AgentToolCallInfo{
			Iteration: tc.Iteration,
			ToolName:  tc.ToolName,
			Arguments: tc.Arguments,
			ResultLen: tc.ResultLen,
		}
	}

	log.Printf("[INFO] handleAgent: completed in %d iterations, %d tool calls", resp.Iterations, len(toolCalls))

	h.writeJSON(w, http.StatusOK, AgentResponseJSON{
		Reply:      resp.Reply,
		Iterations: resp.Iterations,
		ToolCalls:  toolCalls,
		Error:      resp.Error,
	})
}

// handleAgentHealth checks the agent health status
func (h *WebHandler) handleAgentHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if h.agent == nil {
		h.writeJSON(w, http.StatusOK, map[string]interface{}{
			"ok":       false,
			"provider": "none",
			"error":    "Agent is not configured",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	result, err := h.agent.HealthCheck(ctx)
	if err != nil {
		log.Printf("[ERROR] handleAgentHealth: %v", err)
		h.writeJSON(w, http.StatusOK, map[string]interface{}{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

// handleLLMTask routes an LLM task to a device with a local model
func (h *WebHandler) handleLLMTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Prompt   string `json:"prompt"`
		Model    string `json:"model,omitempty"`
		DeviceID string `json:"device_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Prompt == "" {
		h.writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	// Select device with local model
	var policy *pb.RoutingPolicy
	if req.DeviceID != "" {
		policy = &pb.RoutingPolicy{
			Mode:     pb.RoutingPolicy_FORCE_DEVICE_ID,
			DeviceId: req.DeviceID,
		}
	} else {
		policy = &pb.RoutingPolicy{Mode: pb.RoutingPolicy_PREFER_LOCAL_MODEL}
	}

	result := h.orchestrator.registry.SelectDevice(policy, h.orchestrator.selfDeviceID)
	if result.Error != nil {
		h.writeError(w, http.StatusNotFound, result.Error.Error())
		return
	}

	if result.Device == nil || !result.Device.HasLocalModel {
		h.writeError(w, http.StatusNotFound, "No device with local model available")
		return
	}

	log.Printf("[INFO] handleLLMTask: routing to device %s (%s)", result.Device.DeviceName, result.Device.DeviceId[:8])

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	var resp *pb.LLMTaskResponse

	if result.ExecutedLocally {
		// Execute locally
		var err error
		resp, err = h.orchestrator.RunLLMTask(ctx, &pb.LLMTaskRequest{
			Prompt:    req.Prompt,
			Model:     req.Model,
			MaxTokens: 2048,
		})
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		// Dial remote device and call RunLLMTask
		conn, err := grpc.DialContext(ctx, result.Device.GrpcAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			h.writeError(w, http.StatusBadGateway, fmt.Sprintf("Failed to connect to device: %v", err))
			return
		}
		defer conn.Close()

		client := pb.NewOrchestratorServiceClient(conn)
		resp, err = client.RunLLMTask(ctx, &pb.LLMTaskRequest{
			Prompt:    req.Prompt,
			Model:     req.Model,
			MaxTokens: 2048,
		})
		if err != nil {
			h.writeError(w, http.StatusBadGateway, fmt.Sprintf("RPC failed: %v", err))
			return
		}
	}

	if resp.Error != "" {
		h.writeError(w, http.StatusInternalServerError, resp.Error)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"output":           resp.Output,
		"model_used":       resp.ModelUsed,
		"tokens_generated": resp.TokensGenerated,
		"device_id":        result.Device.DeviceId,
		"device_name":      result.Device.DeviceName,
	})
}

// performSummarization is the callback for rolling summaries
func (h *WebHandler) performSummarization(currentSummary string, msgs []chatmem.ChatMessage) (string, error) {
	if h.chat == nil {
		log.Printf("[WARN] performSummarization: no chat provider available, skipping summary")
		return "", fmt.Errorf("no chat provider available")
	}

	var sb strings.Builder
	sb.WriteString("Current summary of the conversation:\n\"\"\"\n")
	if currentSummary != "" {
		sb.WriteString(currentSummary)
	} else {
		sb.WriteString("(No previous summary)")
	}
	sb.WriteString("\n\"\"\"\n\nNew conversation lines to add:\n\"\"\"\n")
	for _, msg := range msgs {
		sb.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}
	sb.WriteString("\"\"\"\n\nPlease update the summary to include the new information, keeping it concise and preserving key details. Return ONLY the updated summary text.")

	prompt := sb.String()
	log.Printf("[INFO] performSummarization: summarizing %d new messages...", len(msgs))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	messages := []llm.ChatMessage{
		{Role: "system", Content: "You are an expert conversation summarizer. You update summaries concisely."},
		{Role: "user", Content: prompt},
	}
	summary, err := h.chat.Chat(ctx, messages)
	if err != nil {
		log.Printf("[ERROR] Summarization failed: %v", err)
		return "", err
	}
	log.Printf("[INFO] performSummarization: completed successfully. New summary len: %d", len(summary))
	return strings.TrimSpace(summary), nil
}

// writeJSON writes a JSON response
func (h *WebHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response
func (h *WebHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, ErrorResponse{Error: message})
}

// truncate shortens a string for logging
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("[INFO] No .env file found: %v", err)
	}

	// Get address from environment or use default
	addr := os.Getenv("GRPC_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	// Create listener
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	// Initialize LLM provider early (before orchestrator)
	llmProvider, err := llm.NewFromEnv()
	if err != nil {
		log.Fatalf("[FATAL] LLM provider init failed: %v", err)
	}
	if llmProvider != nil {
		log.Printf("[INFO] LLM provider: %s", llmProvider.Name())
	} else {
		log.Printf("[INFO] LLM provider: disabled")
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	orchestrator := NewOrchestratorServer(addr)
	orchestrator.llmProvider = llmProvider // Inject LLM provider

	// Auto-register self so list-devices always shows this server
	orchestrator.registerSelf()

	pb.RegisterOrchestratorServiceServer(grpcServer, orchestrator)

	log.Printf("[INFO] Orchestrator server listening on %s", addr)
	log.Printf("[INFO] Server device ID: %s", orchestrator.selfDeviceID)
	log.Printf("[INFO] Server gRPC address: %s", orchestrator.selfAddr)
	log.Printf("[INFO] Bulk HTTP address: %s", orchestrator.deriveBulkHTTPAddr())
	log.Printf("[INFO] Shared directory: %s", orchestrator.sharedRoot)
	log.Printf("[INFO] Allowed commands: %v", allowlist.ListAllowed())
	log.Printf("[INFO] Windows AI Brain available: %v", orchestrator.brain.IsAvailable())

	// Start bulk HTTP server in a goroutine
	go orchestrator.startBulkHTTP()

	// Start continuous metrics polling in a goroutine
	metricsCtx, metricsCancel := context.WithCancel(context.Background())
	defer metricsCancel()
	go orchestrator.startContinuousMetricsPolling(metricsCtx)

	// Get dev key from environment
	devKey := os.Getenv("DEV_KEY")
	if devKey == "" {
		devKey = defaultDevKey
	}

	// Start gRPC server in a goroutine
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Give gRPC server a moment to start before agent initialization
	time.Sleep(1 * time.Second)

	// Initialize Chat provider (optional, defaults to Ollama)
	chatProvider, err := llm.NewChatFromEnv()
	if err != nil {
		log.Printf("[WARN] Chat provider init failed: %v", err)
	}
	if chatProvider != nil {
		log.Printf("[INFO] Chat provider: %s", chatProvider.Name())
	} else {
		log.Printf("[INFO] Chat provider: disabled")
	}

	// Wrap with device-priority when CHAT_USE_DEVICE_LLM is set (use mobile/registered device's model)
	if os.Getenv("CHAT_USE_DEVICE_LLM") == "true" || os.Getenv("CHAT_USE_DEVICE_LLM") == "1" {
		chatTimeout := 120
		if v := os.Getenv("CHAT_TIMEOUT_SECONDS"); v != "" {
			if n, e := strconv.Atoi(v); e == nil && n > 0 {
				chatTimeout = n
			}
		}
		deviceResolver := func() (baseURL, modelName string, ok bool) {
			devices := orchestrator.registry.List()
			for _, d := range devices {
				if d.HasLocalModel && d.LocalChatEndpoint != "" {
					return d.LocalChatEndpoint, d.LocalModelName, true
				}
			}
			return "", "", false
		}
		chatProvider = llm.NewDeviceChatProvider(chatProvider, deviceResolver, chatTimeout)
		if llmProvider != nil {
			llmTimeout := 20
			if v := os.Getenv("LLM_TIMEOUT_SECONDS"); v != "" {
				if n, e := strconv.Atoi(v); e == nil && n > 0 {
					llmTimeout = n
				}
			}
			llmProvider = llm.NewDeviceLLMProvider(llmProvider, deviceResolver, llmTimeout)
		}
		log.Printf("[INFO] Chat + Assistant: device-priority enabled (will use registered device's LLM when available)")
	}

	// Initialize Agent (LLM tool-calling)
	agentGRPCAddr := "localhost:50051"
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		agentGRPCAddr = "localhost" + addr[idx:]
	}
	var agentLoop *llm.AgentLoop
	agentLoop, err = llm.NewAgentLoop(llm.AgentLoopConfig{
		GRPCAddr: agentGRPCAddr, // dial self for tools
	})
	if err != nil {
		log.Printf("[WARN] Agent init failed: %v — agent endpoint will be disabled", err)
	} else {
		log.Printf("[INFO] Agent: enabled (max iterations: 8)")
	}

	// Initialize QAI Hub client
	qaihubCli := qaihub.New()
	if qaihubCli.IsAvailable() {
		log.Printf("[INFO] QAI Hub CLI: available at %s", qaihubCli.Bin)
	} else {
		log.Printf("[INFO] QAI Hub CLI: not available")
	}

	// Create WebHandler for HTTP routes
	webHandler := &WebHandler{
		orchestrator: orchestrator,
		devKey:       devKey,
		llm:          llmProvider,
		chat:         chatProvider,
		agent:        agentLoop,
		qaihubClient: qaihubCli,
	}

	// Setup HTTP routes
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/", webHandler.handleIndex)
	httpMux.HandleFunc("/assets/", webHandler.handleAssets)
	httpMux.HandleFunc("/api/devices", webHandler.handleDevices)
	httpMux.HandleFunc("/api/routed-cmd", webHandler.handleRoutedCmd)
	httpMux.HandleFunc("/api/assistant", webHandler.handleAssistant)
	httpMux.HandleFunc("/api/submit-job", webHandler.handleSubmitJob)
	httpMux.HandleFunc("/api/job", webHandler.handleGetJob)
	httpMux.HandleFunc("/api/job-detail", webHandler.handleJobDetail)
	httpMux.HandleFunc("/api/activity", webHandler.handleActivity)
	httpMux.HandleFunc("/api/device-metrics", webHandler.handleDeviceMetrics)
	httpMux.HandleFunc("/api/plan", webHandler.handlePreviewPlan)
	httpMux.HandleFunc("/api/plan-cost", webHandler.handlePlanCost)
	httpMux.HandleFunc("/api/stream/start", webHandler.handleStreamStart)
	httpMux.HandleFunc("/api/stream/answer", webHandler.handleStreamAnswer)
	httpMux.HandleFunc("/api/stream/stop", webHandler.handleStreamStop)
	httpMux.HandleFunc("/api/request-download", webHandler.handleRequestDownload)

	// QAI Hub endpoints
	httpMux.HandleFunc("/api/qaihub/doctor", webHandler.handleQaihubDoctor)
	httpMux.HandleFunc("/api/qaihub/compile", webHandler.handleQaihubCompile)
	httpMux.HandleFunc("/api/qaihub/devices", webHandler.handleQaihubDevices)
	httpMux.HandleFunc("/api/qaihub/job-status", webHandler.handleQaihubJobStatus)
	httpMux.HandleFunc("/api/qaihub/submit-compile", webHandler.handleQaihubSubmitCompile)

	// Chat endpoints
	httpMux.HandleFunc("/api/chat", webHandler.handleChat)
	httpMux.HandleFunc("/api/chat/health", webHandler.handleChatHealth)
	httpMux.HandleFunc("/api/chat/memory", webHandler.handleChatMemory)

	// Agent endpoint (LLM tool-calling)
	httpMux.HandleFunc("/api/agent", webHandler.handleAgent)
	httpMux.HandleFunc("/api/agent/health", webHandler.handleAgentHealth)

	// LLM task routing endpoint
	httpMux.HandleFunc("/api/llm-task", webHandler.handleLLMTask)

	// Start HTTP Web UI server in goroutine
	webAddr := os.Getenv("WEB_ADDR")
	if webAddr == "" {
		webAddr = defaultWebAddr
	}

	// P2P Discovery: enabled by default, use UDP broadcast to find peers on LAN
	// Set P2P_DISCOVERY=false to disable
	if os.Getenv("P2P_DISCOVERY") != "false" {
		discoveryPort := discovery.DefaultPort
		if portStr := os.Getenv("DISCOVERY_PORT"); portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p < 65536 {
				discoveryPort = p
			}
		}

		// Get self device info and convert to discovery format
		selfInfo := orchestrator.getSelfDeviceInfo()
		selfDevice := &discovery.DeviceAnnounce{
			DeviceID:          selfInfo.DeviceId,
			DeviceName:        selfInfo.DeviceName,
			GrpcAddr:          selfInfo.GrpcAddr,
			HttpAddr:          selfInfo.HttpAddr,
			Platform:          selfInfo.Platform,
			Arch:              selfInfo.Arch,
			HasCPU:            selfInfo.HasCpu,
			HasGPU:            selfInfo.HasGpu,
			HasNPU:            selfInfo.HasNpu || (runtime.GOOS == "windows" && runtime.GOARCH == "arm64"),
			CanScreenCapture:  selfInfo.CanScreenCapture,
			HasLocalModel:     selfInfo.HasLocalModel,
			LocalModelName:    selfInfo.LocalModelName,
			LocalChatEndpoint: selfInfo.LocalChatEndpoint,
		}

		discoverySvc := discovery.NewService(discoveryPort, selfDevice, &discoveryCallback{registry: orchestrator.registry})

		// Add seed peers for cross-subnet discovery
		if seedPeers := os.Getenv("SEED_PEERS"); seedPeers != "" {
			for _, peer := range strings.Split(seedPeers, ",") {
				peer = strings.TrimSpace(peer)
				if peer == "" {
					continue
				}
				// Add discovery port if not specified
				if !strings.Contains(peer, ":") {
					peer = peer + ":" + strconv.Itoa(discoveryPort)
				}
				if err := discoverySvc.AddSeedPeer(peer); err != nil {
					log.Printf("[WARN] Invalid seed peer %s: %v", peer, err)
				} else {
					log.Printf("[INFO] Added seed peer: %s", peer)
				}
			}
		}

		if err := discoverySvc.Start(); err != nil {
			log.Printf("[WARN] Failed to start P2P discovery: %v", err)
		} else {
			log.Printf("[INFO] P2P discovery enabled on UDP port %d", discoveryPort)
			defer discoverySvc.Stop()
		}
	} else if coordinatorAddr := os.Getenv("COORDINATOR_ADDR"); coordinatorAddr != "" {
		// Legacy: Auto-register with coordinator if COORDINATOR_ADDR is set.
		go orchestrator.autoRegisterWithCoordinator(coordinatorAddr)
	}

	// Start HTTP Web UI server (blocking)
	log.Printf("[INFO] HTTP Web UI listening on %s", webAddr)
	// Extract port for localhost URL (e.g., "0.0.0.0:8080" -> ":8080")
	webPort := webAddr
	if idx := strings.LastIndex(webAddr, ":"); idx >= 0 {
		webPort = webAddr[idx:]
	}
	log.Printf("[INFO] Open http://localhost%s in your browser", webPort)
	if err := http.ListenAndServe(webAddr, httpMux); err != nil {
		log.Fatalf("[FATAL] HTTP server failed: %v", err)
	}
}

// discoveryCallback bridges discovery events to the device registry
type discoveryCallback struct {
	registry *registry.Registry
}

func (dc *discoveryCallback) OnDeviceDiscovered(device *discovery.DeviceAnnounce) {
	dc.registry.UpsertFromDiscovery(
		device.DeviceID,
		device.DeviceName,
		device.GrpcAddr,
		device.HttpAddr,
		device.Platform,
		device.Arch,
		device.HasCPU,
		device.HasGPU,
		device.HasNPU,
		device.CanScreenCapture,
		device.HasLocalModel,
		device.LocalModelName,
		device.LocalChatEndpoint,
	)
}

func (dc *discoveryCallback) OnDeviceLeft(deviceID string) {
	dc.registry.Remove(deviceID)
}
