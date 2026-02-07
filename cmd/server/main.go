// Package main implements the gRPC orchestrator server
package main

import (
	"bytes"
	"context"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/kbinani/screenshot"

	"github.com/edgecli/edgecli/internal/allowlist"
	"github.com/edgecli/edgecli/internal/brain"
	"github.com/edgecli/edgecli/internal/cost"
	"github.com/edgecli/edgecli/internal/deviceid"
	"github.com/edgecli/edgecli/internal/exec"
	"github.com/edgecli/edgecli/internal/jobs"
	"github.com/edgecli/edgecli/internal/registry"
	"github.com/edgecli/edgecli/internal/sysinfo"
	"github.com/edgecli/edgecli/internal/transfer"
	"github.com/edgecli/edgecli/internal/webrtcstream"
	pb "github.com/edgecli/edgecli/proto"
)

const (
	defaultAddr         = ":50051"
	defaultBulkHTTPAddr = ":8081"
	defaultSharedDir    = "./shared"
	defaultBulkTTL      = 60
	remoteDialTimeout   = 2 * time.Second
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
	brain         *brain.Brain
	selfDeviceID  string
	selfAddr      string
	ticketManager *transfer.Manager
	sharedRoot    string
	bulkHTTPAddr  string
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

	return &OrchestratorServer{
		sessions:      make(map[string]*Session),
		runner:        exec.NewRunner(),
		registry:      registry.NewRegistry(),
		jobManager:    jobs.NewManager(),
		webrtcManager: webrtcstream.NewManager(),
		brain:         brain.New(),
		selfDeviceID:  selfID,
		selfAddr:      selfAddr,
		ticketManager: transfer.NewManager(time.Duration(bulkTTL) * time.Second),
		sharedRoot:    sharedRootAbs,
		bulkHTTPAddr:  bulkHTTPAddr,
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
			DeviceId:   s.selfDeviceID,
			LastSeen:   time.Now().Unix(),
			CpuLoad:    hostStatus.CPULoad,
			MemUsedMb:  hostStatus.MemUsedMB,
			MemTotalMb: hostStatus.MemTotalMB,
		}, nil
	}

	// Return status from registry
	deviceStatus := s.registry.GetStatus(req.DeviceId)

	log.Printf("[DEBUG] GetDeviceStatus: device=%s last_seen=%d", req.DeviceId, deviceStatus.LastSeen)

	return deviceStatus, nil
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
	return &pb.DeviceInfo{
		DeviceId:         s.selfDeviceID,
		DeviceName:       hostname,
		Platform:         runtime.GOOS,
		Arch:             runtime.GOARCH,
		HasCpu:           true,
		HasGpu:           false,
		HasNpu:           false,
		GrpcAddr:         s.selfAddr,
		CanScreenCapture: detectScreenCapture(),
		HttpAddr:         s.deriveBulkHTTPAddr(),
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

	default:
		return &pb.TaskResult{
			TaskId: req.TaskId,
			Ok:     false,
			Error:  "unknown task kind: " + req.Kind,
			TimeMs: float64(time.Since(start).Milliseconds()),
		}, nil
	}
}

// runLLMGenerate sends a prompt to an OpenAI-compatible endpoint running on
// this device (Ollama, LM Studio, vLLM, or any /v1/chat/completions server).
// Configure via LLM_ENDPOINT and LLM_MODEL env vars on each device.
func (s *OrchestratorServer) runLLMGenerate(ctx context.Context, prompt string) (string, error) {
	endpoint := os.Getenv("LLM_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://127.0.0.1:11434" // Ollama default
	}
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "qwen3:8b" // Default model
	}

	// Build OpenAI-compatible chat completions request
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
		"max_tokens":  1024,
		"stream":      false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Try OpenAI-compatible endpoint first (/v1/chat/completions)
	url := strings.TrimRight(endpoint, "/") + "/v1/chat/completions"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		// Fallback: try Ollama native /api/chat endpoint
		return s.runLLMGenerateOllama(ctx, endpoint, model, prompt)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fallback to Ollama
		return s.runLLMGenerateOllama(ctx, endpoint, model, prompt)
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("parse response: %w (body: %s)", err, string(respBody[:min(200, len(respBody))]))
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned 0 choices")
	}

	result := chatResp.Choices[0].Message.Content
	log.Printf("[INFO] LLM_GENERATE completed: model=%s prompt_len=%d result_len=%d", model, len(prompt), len(result))
	return result, nil
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

	// Try to generate plan using brain if available and no plan provided
	plan := req.Plan
	reduce := req.Reduce
	if (plan == nil || len(plan.Groups) == 0) && s.brain != nil && s.brain.IsAvailable() {
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

	// Create job with tasks (plan and reduce will use defaults if nil)
	job, err := s.jobManager.CreateJob(devices, int(req.MaxWorkers), plan, reduce)
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

	// Fall back to default plan
	if plan == nil {
		plan = s.jobManager.GenerateDefaultPlan(selectedDevices)
		reduce = &pb.ReduceSpec{Kind: "CONCAT"}
		notes = "Brain not available (non-Windows or disabled)"
		rationale = fmt.Sprintf("Default: 1 SYSINFO per device, %d of %d devices selected", len(selectedDevices), len(devices))
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

func main() {
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

	// Create gRPC server
	grpcServer := grpc.NewServer()
	orchestrator := NewOrchestratorServer(addr)

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

	// Auto-register with coordinator if COORDINATOR_ADDR is set.
	// This lets any device join the mesh automatically on startup.
	if coordinatorAddr := os.Getenv("COORDINATOR_ADDR"); coordinatorAddr != "" {
		go orchestrator.autoRegisterWithCoordinator(coordinatorAddr)
	}

	// Serve gRPC
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
