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
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/edgecli/edgecli/internal/chatmem"
	"github.com/edgecli/edgecli/internal/llm"
	"github.com/edgecli/edgecli/internal/qaihub"
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
	grpcClient   pb.OrchestratorServiceClient
	grpcConn     *grpc.ClientConn
	devKey       string
	llm          llm.Provider     // nil if disabled
	chat         llm.ChatProvider // nil if disabled
	agent        *llm.AgentLoop   // LLM tool-calling agent (nil if disabled)
	qaihubClient *qaihub.Client   // qai-hub CLI wrapper
	chatMem      *chatmem.ChatMemory
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
	HttpAddr         string   `json:"http_addr"`
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
	Mode  string      `json:"mode,omitempty"` // "llm", "fallback", or ""
	JobID string      `json:"job_id,omitempty"`
	Plan  interface{} `json:"plan,omitempty"` // plan JSON for debug
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
	Plan      interface{} `json:"plan"`       // Plan object with groups/tasks
	DeviceIDs []string    `json:"device_ids"` // Optional: limit to these devices
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

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("[INFO] No .env file found or error loading it: %v", err)
	}

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

	// Initialize LLM provider (optional)
	llmProvider, err := llm.NewFromEnv()
	if err != nil {
		log.Fatalf("[FATAL] LLM provider init failed: %v", err)
	}
	if llmProvider != nil {
		log.Printf("[INFO] LLM provider: %s", llmProvider.Name())
	} else {
		log.Printf("[INFO] LLM provider: disabled")
	}

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

	// Initialize Agent (LLM tool-calling)
	var agentLoop *llm.AgentLoop
	agentLoop, err = llm.NewAgentLoop(llm.AgentLoopConfig{
		GRPCAddr: grpcAddr,
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

	// Initialize Chat Memory
	chatMemPath, _ := chatmem.DefaultFilePath()
	chatMem, err := chatmem.LoadFromFile(chatMemPath)
	if err != nil {
		log.Printf("[WARN] Could not load chat memory from %s: %v", chatMemPath, err)
		chatMem = chatmem.New()
	}

	server := &WebServer{
		grpcClient:   client,
		grpcConn:     conn,
		devKey:       devKey,
		llm:          llmProvider,
		chat:         chatProvider,
		agent:        agentLoop,
		qaihubClient: qaihubCli,
		chatMem:      chatMem,
	}

	// Initial sync with orchestrator
	go server.syncChatToOrchestrator(true)

	// Setup routes
	http.HandleFunc("/", server.handleIndex)
	http.HandleFunc("/api/devices", server.handleDevices)
	http.HandleFunc("/api/routed-cmd", server.handleRoutedCmd)
	http.HandleFunc("/api/assistant", server.handleAssistant)
	http.HandleFunc("/api/submit-job", server.handleSubmitJob)
	http.HandleFunc("/api/job", server.handleGetJob)
	http.HandleFunc("/api/plan", server.handlePreviewPlan)
	http.HandleFunc("/api/plan-cost", server.handlePlanCost)
	http.HandleFunc("/api/stream/start", server.handleStreamStart)
	http.HandleFunc("/api/stream/answer", server.handleStreamAnswer)
	http.HandleFunc("/api/stream/stop", server.handleStreamStop)
	http.HandleFunc("/api/request-download", server.handleRequestDownload)

	// QAI Hub endpoints
	http.HandleFunc("/api/qaihub/doctor", server.handleQaihubDoctor)
	http.HandleFunc("/api/qaihub/compile", server.handleQaihubCompile)
	http.HandleFunc("/api/qaihub/devices", server.handleQaihubDevices)
	http.HandleFunc("/api/qaihub/job-status", server.handleQaihubJobStatus)
	http.HandleFunc("/api/qaihub/submit-compile", server.handleQaihubSubmitCompile)

	// Chat endpoints
	http.HandleFunc("/api/chat", server.handleChat)
	http.HandleFunc("/api/chat/health", server.handleChatHealth)
	http.HandleFunc("/api/chat/memory", server.handleChatMemory)

	// Agent endpoint (LLM tool-calling)
	http.HandleFunc("/api/agent", server.handleAgent)
	http.HandleFunc("/api/agent/health", server.handleAgentHealth)

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
			HttpAddr:         d.HttpAddr,
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
	var mode string
	var jobID string
	var planDebug interface{}

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
		reply, mode, jobID, planDebug = s.handleAssistantDefault(ctx, req.Text)
	}

	s.writeJSON(w, http.StatusOK, AssistantResponse{
		Reply: reply,
		Raw:   raw,
		Mode:  mode,
		JobID: jobID,
		Plan:  planDebug,
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

// handleAssistantDefault handles the default case in the assistant: tries LLM plan, falls back to deterministic.
func (s *WebServer) handleAssistantDefault(ctx context.Context, userText string) (reply, mode, jobID string, planDebug interface{}) {
	// Fetch device list for LLM context
	devicesResp, err := s.grpcClient.ListDevices(ctx, &pb.ListDevicesRequest{})
	if err != nil {
		return fmt.Sprintf("Error listing devices: %v", err), "", "", nil
	}
	if len(devicesResp.Devices) == 0 {
		return "No devices registered. Register a device first.", "", "", nil
	}

	// Try LLM plan generation
	if s.llm != nil {
		// Build compact devices JSON
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

		planRaw, err := s.llm.Plan(ctx, userText, string(devJSON))
		if err != nil {
			log.Printf("[WARN] LLM plan generation failed: %v — falling back", err)
		} else {
			plan, reduce, err := llm.ParsePlanJSON(planRaw)
			if err != nil {
				log.Printf("[WARN] LLM plan validation failed: %v — falling back", err)
			} else {
				// Create session and submit job with explicit plan
				sessionResp, err := s.grpcClient.CreateSession(ctx, &pb.AuthRequest{
					DeviceName:  "web-ui-assistant",
					SecurityKey: s.devKey,
				})
				if err != nil {
					return fmt.Sprintf("Session error: %v", err), "", "", nil
				}

				jobResp, err := s.grpcClient.SubmitJob(ctx, &pb.JobRequest{
					SessionId: sessionResp.SessionId,
					Text:      userText,
					Plan:      plan,
					Reduce:    reduce,
				})
				if err != nil {
					log.Printf("[WARN] LLM plan submit failed: %v — falling back", err)
				} else {
					// Parse planRaw for debug display
					var planJSON interface{}
					json.Unmarshal([]byte(planRaw), &planJSON)

					return fmt.Sprintf("Job submitted via LLM planner.\nJob ID: %s", jobResp.JobId),
						"llm", jobResp.JobId, planJSON
				}
			}
		}
	}

	// Fallback: submit job without explicit plan (server uses deterministic)
	sessionResp, err := s.grpcClient.CreateSession(ctx, &pb.AuthRequest{
		DeviceName:  "web-ui-assistant",
		SecurityKey: s.devKey,
	})
	if err != nil {
		return fmt.Sprintf("Session error: %v", err), "", "", nil
	}

	jobResp, err := s.grpcClient.SubmitJob(ctx, &pb.JobRequest{
		SessionId: sessionResp.SessionId,
		Text:      userText,
	})
	if err != nil {
		return fmt.Sprintf("Job submission error: %v", err), "", "", nil
	}

	return fmt.Sprintf("Job submitted via fallback planner.\nJob ID: %s", jobResp.JobId),
		"fallback", jobResp.JobId, nil
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

// handlePlanCost estimates execution cost for a plan
func (s *WebServer) handlePlanCost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req PlanCostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	// Convert request plan to proto Plan
	planProto, err := s.convertJSONToPlan(req.Plan)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid plan: %v", err))
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
		log.Printf("[ERROR] handlePlanCost: CreateSession failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Session error: %v", err))
		return
	}

	// Call PreviewPlanCost
	costResp, err := s.grpcClient.PreviewPlanCost(ctx, &pb.PlanCostRequest{
		SessionId: sessionResp.SessionId,
		Plan:      planProto,
		DeviceIds: req.DeviceIDs,
	})
	if err != nil {
		log.Printf("[ERROR] handlePlanCost: PreviewPlanCost failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Cost estimation error: %v", err))
		return
	}

	// Convert to JSON response
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

	s.writeJSON(w, http.StatusOK, PlanCostResponse{
		TotalPredictedMs:      costResp.TotalPredictedMs,
		DeviceCosts:           deviceCosts,
		RecommendedDeviceID:   costResp.RecommendedDeviceId,
		RecommendedDeviceName: costResp.RecommendedDeviceName,
		HasUnknownCosts:       costResp.HasUnknownCosts,
		Warning:               costResp.Warning,
	})
}

// convertJSONToPlan converts a JSON plan to a proto Plan
func (s *WebServer) convertJSONToPlan(planJSON interface{}) (*pb.Plan, error) {
	// Re-encode to JSON and then decode into a structured type
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

	// Convert to proto
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

// handleRequestDownload creates a download ticket on a target device and returns a direct URL
func (s *WebServer) handleRequestDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.DeviceID == "" {
		s.writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	if req.Path == "" {
		s.writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	// Find the device in the coordinator's registry
	devicesResp, err := s.grpcClient.ListDevices(ctx, &pb.ListDevicesRequest{})
	if err != nil {
		log.Printf("[ERROR] handleRequestDownload: ListDevices failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("ListDevices error: %v", err))
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
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Device not found: %s", req.DeviceID))
		return
	}
	if targetDevice.HttpAddr == "" {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Device %s has no HTTP address configured", targetDevice.DeviceName))
		return
	}

	// Dial the device's gRPC directly
	dialCtx, dialCancel := context.WithTimeout(ctx, 5*time.Second)
	defer dialCancel()

	conn, err := grpc.DialContext(dialCtx, targetDevice.GrpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("[ERROR] handleRequestDownload: failed to dial device %s: %v", targetDevice.GrpcAddr, err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to connect to device: %v", err))
		return
	}
	defer conn.Close()

	deviceClient := pb.NewOrchestratorServiceClient(conn)

	// Call CreateDownloadTicket on the device
	ticketResp, err := deviceClient.CreateDownloadTicket(ctx, &pb.DownloadTicketRequest{
		Path: req.Path,
	})
	if err != nil {
		log.Printf("[ERROR] handleRequestDownload: CreateDownloadTicket failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Ticket error: %v", err))
		return
	}

	// Build the direct download URL
	downloadURL := fmt.Sprintf("http://%s/bulk/download/%s", targetDevice.HttpAddr, ticketResp.Token)

	log.Printf("[INFO] handleRequestDownload: ticket for %s on %s, url=%s",
		req.Path, targetDevice.DeviceName, downloadURL)

	s.writeJSON(w, http.StatusOK, DownloadResponse{
		DownloadURL:   downloadURL,
		Filename:      ticketResp.Filename,
		SizeBytes:     ticketResp.SizeBytes,
		ExpiresUnixMs: ticketResp.ExpiresUnixMs,
	})
}

// handleQaihubDoctor runs qai-hub diagnostics
func (s *WebServer) handleQaihubDoctor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	result, err := s.qaihubClient.Doctor(ctx)
	if err != nil {
		log.Printf("[ERROR] handleQaihubDoctor: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Doctor failed: %v", err))
		return
	}

	s.writeJSON(w, http.StatusOK, result)
}

// handleQaihubCompile runs model compilation using qai-hub
func (s *WebServer) handleQaihubCompile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req CompileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	// Validate required fields
	if req.ONNXPath == "" {
		s.writeError(w, http.StatusBadRequest, "onnx_path is required")
		return
	}
	if req.Target == "" {
		s.writeError(w, http.StatusBadRequest, "target is required")
		return
	}

	// Validate path: must be absolute and exist
	if !filepath.IsAbs(req.ONNXPath) {
		s.writeError(w, http.StatusBadRequest, "onnx_path must be an absolute path")
		return
	}

	// Block path traversal
	if strings.Contains(req.ONNXPath, "..") {
		s.writeError(w, http.StatusBadRequest, "onnx_path cannot contain path traversal (..)")
		return
	}

	// Check file exists
	info, err := os.Stat(req.ONNXPath)
	if os.IsNotExist(err) {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("ONNX file not found: %s", req.ONNXPath))
		return
	}
	if info.IsDir() {
		s.writeError(w, http.StatusBadRequest, "onnx_path must be a file, not a directory")
		return
	}

	// Check if qai-hub is available
	if !s.qaihubClient.IsAvailable() {
		s.writeError(w, http.StatusServiceUnavailable, "qai-hub CLI is not available. Run qaihub doctor for setup instructions.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	// Default output directory
	outDir := filepath.Join("artifacts", "qaihub", time.Now().Format("20060102-150405"))

	result, err := s.qaihubClient.Compile(ctx, req.ONNXPath, req.Target, req.Runtime, outDir)
	if err != nil {
		log.Printf("[ERROR] handleQaihubCompile: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Compile failed: %v", err))
		return
	}

	s.writeJSON(w, http.StatusOK, result)
}

// handleQaihubDevices returns the QAI Hub cloud device catalog
func (s *WebServer) handleQaihubDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if !s.qaihubClient.IsAvailable() {
		s.writeError(w, http.StatusServiceUnavailable, "qai-hub CLI is not available")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	result, err := s.qaihubClient.ListDevices(ctx)
	if err != nil {
		log.Printf("[ERROR] handleQaihubDevices: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("ListDevices failed: %v", err))
		return
	}

	// Optional filtering via query params
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

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":   len(devices),
		"devices": devices,
	})
}

// QaihubJobStatusRequest is the JSON request for /api/qaihub/job-status
type QaihubJobStatusRequest struct {
	JobID string `json:"job_id"`
}

// handleQaihubJobStatus checks the status of a QAI Hub compile job
func (s *WebServer) handleQaihubJobStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var jobID string
	if r.Method == http.MethodGet {
		jobID = r.URL.Query().Get("job_id")
	} else {
		var req QaihubJobStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
			return
		}
		jobID = req.JobID
	}

	if jobID == "" {
		s.writeError(w, http.StatusBadRequest, "job_id is required")
		return
	}

	if !s.qaihubClient.IsAvailable() {
		s.writeError(w, http.StatusServiceUnavailable, "qai-hub CLI is not available")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := s.qaihubClient.GetJobStatus(ctx, jobID)
	if err != nil {
		log.Printf("[ERROR] handleQaihubJobStatus: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Job status check failed: %v", err))
		return
	}

	s.writeJSON(w, http.StatusOK, result)
}

// QaihubSubmitCompileRequest is the JSON request for /api/qaihub/submit-compile
type QaihubSubmitCompileRequest struct {
	Model      string `json:"model"`       // ONNX path or QAI Hub model ID
	DeviceName string `json:"device_name"` // e.g. "Samsung Galaxy S24 (Family)"
	Options    string `json:"options"`     // extra compile options
}

// handleQaihubSubmitCompile submits a compile job via the Python SDK
func (s *WebServer) handleQaihubSubmitCompile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req QaihubSubmitCompileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.Model == "" {
		s.writeError(w, http.StatusBadRequest, "model is required (ONNX path or model ID)")
		return
	}
	if req.DeviceName == "" {
		req.DeviceName = "Samsung Galaxy S24 (Family)"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	log.Printf("[INFO] handleQaihubSubmitCompile: model=%s device=%s", req.Model, req.DeviceName)

	result, err := s.qaihubClient.SubmitCompile(ctx, req.Model, req.DeviceName, req.Options)
	if err != nil {
		log.Printf("[ERROR] handleQaihubSubmitCompile: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Compile submission failed: %v", err))
		return
	}

	s.writeJSON(w, http.StatusOK, result)
}

// handleChat handles chat requests using the local LLM runtime
func (s *WebServer) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if s.chat == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Chat provider is not configured. Set CHAT_PROVIDER environment variable.")
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if len(req.Messages) == 0 {
		s.writeError(w, http.StatusBadRequest, "messages array is required and must not be empty")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	reply, err := s.chat.Chat(ctx, req.Messages)
	if err != nil {
		log.Printf("[ERROR] handleChat: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Chat failed: %v", err))
		return
	}

	// Update chat memory
	// Add user message(s) - assuming last one is new
	if len(req.Messages) > 0 {
		lastMsg := req.Messages[len(req.Messages)-1]
		if lastMsg.Role == "user" {
			s.chatMem.AddMessage("user", lastMsg.Content)
		}
	}
	// Add assistant reply
	s.chatMem.AddMessage("assistant", reply)

	// Sync to orchestrator
	go s.syncChatToOrchestrator(false)
	
	// Trigger background summarization if needed
	s.chatMem.SummarizeAsync(s.performSummarization, func() {
		s.syncChatToOrchestrator(false)
	})

	s.writeJSON(w, http.StatusOK, ChatResponse{Reply: reply})
}

// handleChatMemory returns the current chat history
func (s *WebServer) handleChatMemory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jsonStr, err := s.chatMem.ToJSON()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to serialize memory")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(jsonStr))
}

// syncChatToOrchestrator pushes local chat memory to the orchestrator,
// or pulls from it if pullFirst is true.
func (s *WebServer) syncChatToOrchestrator(pullOnly bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var req *pb.ChatMemorySync
	if pullOnly {
		// Empty JSON means we want to pull
		req = &pb.ChatMemorySync{
			LastUpdatedMs: 0,
			MemoryJson:    "",
		}
	} else {
		// Push our memory
		jsonStr, _ := s.chatMem.ToJSON()
		req = &pb.ChatMemorySync{
			LastUpdatedMs: s.chatMem.GetLastUpdated(),
			MemoryJson:    jsonStr,
		}
	}

	resp, err := s.grpcClient.SyncChatMemory(ctx, req)
	if err != nil {
		log.Printf("[WARN] SyncChatMemory failed: %v", err)
		return
	}

	// If orchestrator sent back updates (either because we pulled, or because ours was older)
	if resp.MemoryJson != "" {
		incoming, err := chatmem.ParseFromJSON(resp.MemoryJson)
		if err == nil {
			if s.chatMem.Merge(incoming) {
				log.Printf("[INFO] Chat memory updated from orchestrator (ts=%d)", incoming.GetLastUpdated())
				// We don't save to file here to avoid conflict with orchestrator if on same machine
				// syncChatToOrchestrator is enough
			}
		}
	}
}

// handleChatHealth checks the chat runtime status
func (s *WebServer) handleChatHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if s.chat == nil {
		s.writeJSON(w, http.StatusOK, &llm.HealthResult{
			Ok:       false,
			Provider: "none",
			Error:    "Chat provider is not configured",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	result, err := s.chat.Health(ctx)
	if err != nil {
		log.Printf("[ERROR] handleChatHealth: %v", err)
		s.writeJSON(w, http.StatusOK, &llm.HealthResult{
			Ok:       false,
			Provider: s.chat.Name(),
			Error:    err.Error(),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, result)
}

// handleAgent handles LLM tool-calling agent requests
func (s *WebServer) handleAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if s.agent == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Agent is not available. Check CHAT_PROVIDER configuration and gRPC server connectivity.")
		return
	}

	var req AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.Message == "" {
		s.writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	// Use a longer timeout for agent requests (2 minutes default)
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	log.Printf("[INFO] handleAgent: processing message: %q", truncate(req.Message, 100))

	// Construct history from chat memory
	var history []llm.ToolChatMessage
	
	// Add summary if available
	summary := s.chatMem.GetSummary()
	if summary != "" {
		history = append(history, llm.ToolChatMessage{
			Role:    "system",
			Content: fmt.Sprintf("Summary of previous conversation:\n%s", summary),
		})
	}
	
	// Add recent messages
	msgs := s.chatMem.GetMessages()
	for _, m := range msgs {
		history = append(history, llm.ToolChatMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	resp, err := s.agent.Run(ctx, req.Message, history)
	if err != nil {
		log.Printf("[ERROR] handleAgent: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Agent error: %v", err))
		return
	}

	// Update chat memory and sync
	s.chatMem.AddMessage("user", req.Message)
	s.chatMem.AddMessage("assistant", resp.Reply)
	go s.syncChatToOrchestrator(false)
	
	// Trigger background summarization if needed
	s.chatMem.SummarizeAsync(s.performSummarization, func() {
		s.syncChatToOrchestrator(false)
	})

	// Convert tool calls to response format
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

	s.writeJSON(w, http.StatusOK, AgentResponseJSON{
		Reply:      resp.Reply,
		Iterations: resp.Iterations,
		ToolCalls:  toolCalls,
		Error:      resp.Error,
	})
}

// handleAgentHealth checks the agent health status
func (s *WebServer) handleAgentHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if s.agent == nil {
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"ok":       false,
			"provider": "none",
			"error":    "Agent is not configured",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	result, err := s.agent.HealthCheck(ctx)
	if err != nil {
		log.Printf("[ERROR] handleAgentHealth: %v", err)
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, result)
}

// truncate shortens a string for logging
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// performSummarization is the callback for rolling summaries
func (s *WebServer) performSummarization(currentSummary string, msgs []chatmem.ChatMessage) (string, error) {
	if s.chat == nil {
		log.Printf("[WARN] performSummarization: no chat provider available, skipping summary")
		return "", fmt.Errorf("no chat provider available")
	}

	// Construct the prompt
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

	// Use Chat provider
	messages := []llm.ChatMessage{
		{Role: "system", Content: "You are an expert conversation summarizer. You update summaries concisely."},
		{Role: "user", Content: prompt},
	}
	summary, err := s.chat.Chat(ctx, messages)
	if err != nil {
		log.Printf("[ERROR] Summarization failed: %v", err)
		return "", err
	}
	log.Printf("[INFO] performSummarization: completed successfully. New summary len: %d", len(summary))
	return strings.TrimSpace(summary), nil
}
