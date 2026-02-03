// Package main implements the gRPC orchestrator server
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/edgecli/edgecli/internal/allowlist"
	"github.com/edgecli/edgecli/internal/deviceid"
	"github.com/edgecli/edgecli/internal/exec"
	"github.com/edgecli/edgecli/internal/registry"
	"github.com/edgecli/edgecli/internal/sysinfo"
	pb "github.com/edgecli/edgecli/proto"
)

const defaultAddr = ":50051"

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
	sessions     map[string]*Session
	mu           sync.RWMutex
	runner       *exec.Runner
	registry     *registry.Registry
	selfDeviceID string
	selfAddr     string
}

// NewOrchestratorServer creates a new server instance
func NewOrchestratorServer(addr string) *OrchestratorServer {
	// Get or create device ID for this server
	selfID, err := deviceid.GetOrCreate()
	if err != nil {
		log.Printf("[WARN] Could not get device ID: %v", err)
		selfID = uuid.New().String()
	}

	return &OrchestratorServer{
		sessions:     make(map[string]*Session),
		runner:       exec.NewRunner(),
		registry:     registry.NewRegistry(),
		selfDeviceID: selfID,
		selfAddr:     addr,
	}
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
		DeviceId:   s.selfDeviceID,
		DeviceName: hostname,
		Platform:   runtime.GOOS,
		Arch:       runtime.GOARCH,
		HasCpu:     true,
		HasGpu:     false,
		HasNpu:     false,
		GrpcAddr:   s.selfAddr,
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
	pb.RegisterOrchestratorServiceServer(grpcServer, orchestrator)

	log.Printf("[INFO] Orchestrator server listening on %s", addr)
	log.Printf("[INFO] Server device ID: %s", orchestrator.selfDeviceID)
	log.Printf("[INFO] Allowed commands: %v", allowlist.ListAllowed())

	// Serve
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
