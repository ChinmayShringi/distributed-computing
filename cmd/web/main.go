// Package main implements a minimal HTTP server for the EdgeCLI web UI demo
package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/edgecli/edgecli/proto"
)

//go:embed index.html
var staticFS embed.FS

const (
	defaultHTTPAddr = ":8080"
	defaultGRPCAddr = "localhost:50051"
	defaultDevKey   = "dev"
	dialTimeout     = 5 * time.Second
	requestTimeout  = 30 * time.Second
)

// WebServer handles HTTP requests and forwards them to gRPC
type WebServer struct {
	grpcClient pb.OrchestratorServiceClient
	grpcConn   *grpc.ClientConn
	devKey     string
}

// DeviceResponse is the JSON response for /api/devices
type DeviceResponse struct {
	DeviceID         string   `json:"device_id"`
	DeviceName       string   `json:"device_name"`
	Platform         string   `json:"platform"`
	Arch             string   `json:"arch"`
	Capabilities     []string `json:"capabilities"`
	GRPCAddr         string   `json:"grpc_addr"`
	CanScreenCapture bool     `json:"can_screen_capture"`
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
	Policy        string `json:"policy"`          // BEST_AVAILABLE, PREFER_REMOTE, FORCE_DEVICE_ID
	ForceDeviceID string `json:"force_device_id"` // used if FORCE_DEVICE_ID
	FPS           int32  `json:"fps"`             // default 8
	Quality       int32  `json:"quality"`         // default 60
	MonitorIndex  int32  `json:"monitor_index"`   // default 0
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

func main() {
	// Get configuration from environment
	httpAddr := os.Getenv("WEB_ADDR")
	if httpAddr == "" {
		httpAddr = defaultHTTPAddr
	}

	grpcAddr := os.Getenv("GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = defaultGRPCAddr
	}

	devKey := os.Getenv("DEV_KEY")
	if devKey == "" {
		devKey = defaultDevKey
	}

	// Connect to gRPC server
	log.Printf("[INFO] Connecting to gRPC server at %s", grpcAddr)

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("[ERROR] Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client := pb.NewOrchestratorServiceClient(conn)

	server := &WebServer{
		grpcClient: client,
		grpcConn:   conn,
		devKey:     devKey,
	}

	// Setup routes
	http.HandleFunc("/", server.handleIndex)
	http.HandleFunc("/api/devices", server.handleDevices)
	http.HandleFunc("/api/routed-cmd", server.handleRoutedCmd)
	http.HandleFunc("/api/assistant", server.handleAssistant)
	http.HandleFunc("/api/submit-job", server.handleSubmitJob)
	http.HandleFunc("/api/job", server.handleGetJob)
	http.HandleFunc("/api/plan", server.handlePreviewPlan)
	http.HandleFunc("/api/stream/start", server.handleStreamStart)
	http.HandleFunc("/api/stream/answer", server.handleStreamAnswer)
	http.HandleFunc("/api/stream/stop", server.handleStreamStop)

	log.Printf("[INFO] Web server listening on %s", httpAddr)
	log.Printf("[INFO] Connected to gRPC server at %s", grpcAddr)
	log.Printf("[INFO] Open http://localhost%s in your browser", httpAddr)

	if err := http.ListenAndServe(httpAddr, nil); err != nil {
		log.Fatalf("[ERROR] Server failed: %v", err)
	}
}

// handleIndex serves the embedded index.html
func (s *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	content, err := staticFS.ReadFile("index.html")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

// handleDevices returns all registered devices
func (s *WebServer) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	// ListDevices doesn't require a session
	resp, err := s.grpcClient.ListDevices(ctx, &pb.ListDevicesRequest{})
	if err != nil {
		log.Printf("[ERROR] ListDevices failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("gRPC error: %v", err))
		return
	}

	// Convert to JSON response
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
			DeviceID:         d.DeviceId,
			DeviceName:       d.DeviceName,
			Platform:         d.Platform,
			Arch:             d.Arch,
			Capabilities:     caps,
			GRPCAddr:         d.GrpcAddr,
			CanScreenCapture: d.CanScreenCapture,
		})
	}

	s.writeJSON(w, http.StatusOK, devices)
}

// handleRoutedCmd executes a command on the best available device
func (s *WebServer) handleRoutedCmd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse request body
	var req RoutedCmdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.Cmd == "" {
		s.writeError(w, http.StatusBadRequest, "cmd is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	// Create session
	sessionResp, err := s.grpcClient.CreateSession(ctx, &pb.AuthRequest{
		DeviceName:  "web-ui",
		SecurityKey: s.devKey,
	})
	if err != nil {
		log.Printf("[ERROR] CreateSession failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Session error: %v", err))
		return
	}

	// Build routing policy
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

	// Execute routed command
	cmdResp, err := s.grpcClient.ExecuteRoutedCommand(ctx, &pb.RoutedCommandRequest{
		SessionId: sessionResp.SessionId,
		Policy:    policy,
		Command:   req.Cmd,
		Args:      req.Args,
	})
	if err != nil {
		log.Printf("[ERROR] ExecuteRoutedCommand failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Command error: %v", err))
		return
	}

	// Build response
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

	s.writeJSON(w, http.StatusOK, resp)
}

// handleAssistant processes natural language commands
func (s *WebServer) handleAssistant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse request body
	var req AssistantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	text := strings.ToLower(strings.TrimSpace(req.Text))

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	var reply string
	var raw interface{}

	switch {
	case strings.Contains(text, "list devices") || strings.Contains(text, "show devices") || strings.Contains(text, "devices"):
		// List devices
		resp, err := s.grpcClient.ListDevices(ctx, &pb.ListDevicesRequest{})
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
		reply, raw = s.executeAssistantCommand(ctx, "pwd", nil)

	case strings.Contains(text, "ls") || strings.Contains(text, "list files"):
		reply, raw = s.executeAssistantCommand(ctx, "ls", nil)

	case strings.Contains(text, "cat"):
		// Try to extract file path
		parts := strings.Fields(text)
		var filePath string
		for i, p := range parts {
			if p == "cat" && i+1 < len(parts) {
				filePath = parts[i+1]
				break
			}
		}
		if filePath != "" {
			reply, raw = s.executeAssistantCommand(ctx, "cat", []string{filePath})
		} else {
			reply = "Please specify a file path. Example: 'cat ./shared/test.txt'"
		}

	default:
		reply = "I can help you run commands on your devices.\n\n" +
			"Try:\n" +
			"- 'list devices' - show all registered devices\n" +
			"- 'pwd' - print working directory\n" +
			"- 'ls' - list files\n" +
			"- 'cat ./shared/<file>' - show file contents\n\n" +
			"For more control, use the Routed Command section above."
	}

	s.writeJSON(w, http.StatusOK, AssistantResponse{
		Reply: reply,
		Raw:   raw,
	})
}

// executeAssistantCommand runs a command and returns formatted output
func (s *WebServer) executeAssistantCommand(ctx context.Context, cmd string, args []string) (string, interface{}) {
	// Create session
	sessionResp, err := s.grpcClient.CreateSession(ctx, &pb.AuthRequest{
		DeviceName:  "web-ui-assistant",
		SecurityKey: s.devKey,
	})
	if err != nil {
		return fmt.Sprintf("Session error: %v", err), nil
	}

	// Execute with BEST_AVAILABLE policy
	cmdResp, err := s.grpcClient.ExecuteRoutedCommand(ctx, &pb.RoutedCommandRequest{
		SessionId: sessionResp.SessionId,
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

// handleSubmitJob submits a distributed job to all devices
func (s *WebServer) handleSubmitJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse request body
	var req SubmitJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	// Create session
	sessionResp, err := s.grpcClient.CreateSession(ctx, &pb.AuthRequest{
		DeviceName:  "web-ui",
		SecurityKey: s.devKey,
	})
	if err != nil {
		log.Printf("[ERROR] CreateSession failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Session error: %v", err))
		return
	}

	// Submit job
	jobResp, err := s.grpcClient.SubmitJob(ctx, &pb.JobRequest{
		SessionId:  sessionResp.SessionId,
		Text:       req.Text,
		MaxWorkers: req.MaxWorkers,
	})
	if err != nil {
		log.Printf("[ERROR] SubmitJob failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Job error: %v", err))
		return
	}

	s.writeJSON(w, http.StatusOK, JobInfoResponse{
		JobID:     jobResp.JobId,
		CreatedAt: jobResp.CreatedAt,
		Summary:   jobResp.Summary,
	})
}

// handleGetJob returns the status of a job
func (s *WebServer) handleGetJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		s.writeError(w, http.StatusBadRequest, "id parameter is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	// Get job status
	jobResp, err := s.grpcClient.GetJob(ctx, &pb.JobId{JobId: jobID})
	if err != nil {
		log.Printf("[ERROR] GetJob failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Job error: %v", err))
		return
	}

	// Convert tasks
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

	s.writeJSON(w, http.StatusOK, JobStatusResponse{
		JobID:        jobResp.JobId,
		State:        jobResp.State,
		Tasks:        tasks,
		FinalResult:  jobResp.FinalResult,
		CurrentGroup: jobResp.CurrentGroup,
		TotalGroups:  jobResp.TotalGroups,
	})
}

// handlePreviewPlan generates a plan preview without creating a job
func (s *WebServer) handlePreviewPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req PreviewPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	// Create session
	sessionResp, err := s.grpcClient.CreateSession(ctx, &pb.AuthRequest{
		DeviceName:  "web-ui",
		SecurityKey: s.devKey,
	})
	if err != nil {
		log.Printf("[ERROR] handlePreviewPlan: CreateSession failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Session error: %v", err))
		return
	}

	// Call PreviewPlan
	planResp, err := s.grpcClient.PreviewPlan(ctx, &pb.PlanPreviewRequest{
		SessionId:  sessionResp.SessionId,
		Text:       req.Text,
		MaxWorkers: req.MaxWorkers,
	})
	if err != nil {
		log.Printf("[ERROR] handlePreviewPlan: PreviewPlan failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Plan error: %v", err))
		return
	}

	// Convert plan to a JSON-friendly structure
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

	s.writeJSON(w, http.StatusOK, PreviewPlanResponse{
		UsedAi:    planResp.UsedAi,
		Notes:     planResp.Notes,
		Rationale: planResp.Rationale,
		Plan:      plan,
		Reduce:    reduce,
	})
}

// handleStreamStart initiates a WebRTC stream from a selected device
func (s *WebServer) handleStreamStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req StreamStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	// Create session
	sessionResp, err := s.grpcClient.CreateSession(ctx, &pb.AuthRequest{
		DeviceName:  "web-stream",
		SecurityKey: s.devKey,
	})
	if err != nil {
		log.Printf("[ERROR] handleStreamStart: CreateSession failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Session error: %v", err))
		return
	}

	// List devices and select based on policy
	devicesResp, err := s.grpcClient.ListDevices(ctx, &pb.ListDevicesRequest{})
	if err != nil {
		log.Printf("[ERROR] handleStreamStart: ListDevices failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("ListDevices error: %v", err))
		return
	}

	if len(devicesResp.Devices) == 0 {
		s.writeError(w, http.StatusNotFound, "No devices available")
		return
	}

	// Select device based on policy
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
			s.writeError(w, http.StatusNotFound, fmt.Sprintf("Device not found: %s", req.ForceDeviceID))
			return
		}
	case "PREFER_REMOTE":
		// Prefer non-local (first device with different address)
		for _, d := range devicesResp.Devices {
			if d.GrpcAddr != "" && d.GrpcAddr != "localhost:50051" && d.GrpcAddr != "127.0.0.1:50051" {
				selectedDevice = d
				break
			}
		}
		if selectedDevice == nil {
			selectedDevice = devicesResp.Devices[0] // Fallback to first
		}
	default: // BEST_AVAILABLE
		// Prefer NPU > GPU > CPU
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

	// Dial the selected device directly
	dialCtx, dialCancel := context.WithTimeout(ctx, 5*time.Second)
	defer dialCancel()

	conn, err := grpc.DialContext(dialCtx, selectedDevice.GrpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("[ERROR] handleStreamStart: failed to dial device %s: %v", selectedDevice.GrpcAddr, err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to connect to device: %v", err))
		return
	}
	defer conn.Close()

	deviceClient := pb.NewOrchestratorServiceClient(conn)

	// Call StartWebRTC on the device
	webrtcResp, err := deviceClient.StartWebRTC(ctx, &pb.WebRTCConfig{
		SessionId:    sessionResp.SessionId,
		TargetFps:    req.FPS,
		JpegQuality:  req.Quality,
		MonitorIndex: req.MonitorIndex,
	})
	if err != nil {
		log.Printf("[ERROR] handleStreamStart: StartWebRTC failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("StartWebRTC error: %v", err))
		return
	}

	log.Printf("[INFO] handleStreamStart: stream %s started on %s", webrtcResp.StreamId, selectedDevice.DeviceName)

	s.writeJSON(w, http.StatusOK, StreamStartResponse{
		SelectedDeviceID:   selectedDevice.DeviceId,
		SelectedDeviceName: selectedDevice.DeviceName,
		SelectedDeviceAddr: selectedDevice.GrpcAddr,
		StreamID:           webrtcResp.StreamId,
		OfferSDP:           webrtcResp.Sdp,
	})
}

// handleStreamAnswer completes the WebRTC handshake with the answer SDP
func (s *WebServer) handleStreamAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req StreamAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.SelectedDeviceAddr == "" || req.StreamID == "" || req.AnswerSDP == "" {
		s.writeError(w, http.StatusBadRequest, "selected_device_addr, stream_id, and answer_sdp are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	// Dial the device
	dialCtx, dialCancel := context.WithTimeout(ctx, 5*time.Second)
	defer dialCancel()

	conn, err := grpc.DialContext(dialCtx, req.SelectedDeviceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("[ERROR] handleStreamAnswer: failed to dial %s: %v", req.SelectedDeviceAddr, err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to connect to device: %v", err))
		return
	}
	defer conn.Close()

	deviceClient := pb.NewOrchestratorServiceClient(conn)

	// Call CompleteWebRTC
	_, err = deviceClient.CompleteWebRTC(ctx, &pb.WebRTCAnswer{
		StreamId: req.StreamID,
		Sdp:      req.AnswerSDP,
	})
	if err != nil {
		log.Printf("[ERROR] handleStreamAnswer: CompleteWebRTC failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("CompleteWebRTC error: %v", err))
		return
	}

	log.Printf("[INFO] handleStreamAnswer: stream %s connected", req.StreamID)

	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleStreamStop stops an active WebRTC stream
func (s *WebServer) handleStreamStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req StreamStopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.SelectedDeviceAddr == "" || req.StreamID == "" {
		s.writeError(w, http.StatusBadRequest, "selected_device_addr and stream_id are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	// Dial the device
	dialCtx, dialCancel := context.WithTimeout(ctx, 5*time.Second)
	defer dialCancel()

	conn, err := grpc.DialContext(dialCtx, req.SelectedDeviceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("[ERROR] handleStreamStop: failed to dial %s: %v", req.SelectedDeviceAddr, err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to connect to device: %v", err))
		return
	}
	defer conn.Close()

	deviceClient := pb.NewOrchestratorServiceClient(conn)

	// Call StopWebRTC
	_, err = deviceClient.StopWebRTC(ctx, &pb.WebRTCStop{
		StreamId: req.StreamID,
	})
	if err != nil {
		log.Printf("[ERROR] handleStreamStop: StopWebRTC failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("StopWebRTC error: %v", err))
		return
	}

	log.Printf("[INFO] handleStreamStop: stream %s stopped", req.StreamID)

	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// writeJSON writes a JSON response
func (s *WebServer) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response
func (s *WebServer) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, ErrorResponse{Error: message})
}
