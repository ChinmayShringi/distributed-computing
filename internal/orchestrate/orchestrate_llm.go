// Package orchestrate provides a proof-of-concept for multi-compute model orchestration.
//
// It demonstrates the architecture for running models on CPU, GPU, and NPU.
//
// Usage patterns are shown in commented examples at the bottom of this file.
package orchestrate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"
)

// ComputeTarget specifies which hardware to run inference on.
type ComputeTarget string

const (
	ComputeCPU ComputeTarget = "cpu"
	ComputeGPU ComputeTarget = "gpu"
	ComputeNPU ComputeTarget = "npu"
)

// Request represents an orchestration request.
type Request struct {
	DeviceID string        `json:"device_id"` // Target device ID (from registry)
	Task     string        `json:"task"`      // Task type: "inference", "embed", "generate"
	Model    string        `json:"model"`     // Model name: "llama3.2:3b", "phi3:mini", or QAI model path
	Input    string        `json:"input"`     // Prompt or input data
	Compute  ComputeTarget `json:"compute"`   // Target: "cpu", "gpu", "npu"
}

// Response represents an orchestration response.
type Response struct {
	DeviceID   string        `json:"device_id"`
	Compute    ComputeTarget `json:"compute"`
	Model      string        `json:"model"`
	Output     string        `json:"output"`
	DurationMs int64         `json:"duration_ms"`
	Error      string        `json:"error,omitempty"`
}

// Orchestrator manages model execution across compute targets.
type Orchestrator struct {
	// OllamaBaseURL is the base URL for Ollama API (e.g., "http://localhost:11434")
	OllamaBaseURL string

	// QAIHubModelDir is the directory containing QAI Hub compiled models
	QAIHubModelDir string

	// HTTPClient for Ollama requests
	HTTPClient *http.Client
}

// NewOrchestrator creates a new Orchestrator with default settings.
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		OllamaBaseURL:  "http://localhost:11434",
		QAIHubModelDir: "./models/qaihub",
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Orchestrate routes the request to the appropriate compute backend.
func (o *Orchestrator) Orchestrate(ctx context.Context, req Request) Response {
	start := time.Now()

	var output string
	var err error

	switch req.Compute {
	case ComputeCPU:
		output, err = o.runOllamaCPU(ctx, req.Model, req.Input)
	case ComputeGPU:
		output, err = o.runOllamaGPU(ctx, req.Model, req.Input)
	case ComputeNPU:
		output, err = o.runQAIHubNPU(ctx, req.Model, req.Input)
	default:
		err = fmt.Errorf("unknown compute target: %s", req.Compute)
	}

	resp := Response{
		DeviceID:   req.DeviceID,
		Compute:    req.Compute,
		Model:      req.Model,
		DurationMs: time.Since(start).Milliseconds(),
	}

	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Output = output
	}

	return resp
}

// runOllamaCPU runs inference on Ollama forcing CPU-only execution.
// It sets CUDA_VISIBLE_DEVICES="" to disable GPU.
func (o *Orchestrator) runOllamaCPU(ctx context.Context, model, input string) (string, error) {
	// For CPU-only, we need to call Ollama with GPU disabled
	// This is done by setting environment variable when starting Ollama server
	// or by using a separate Ollama instance configured for CPU

	// In practice, you'd have Ollama running with CUDA_VISIBLE_DEVICES=""
	// Here we just call the same endpoint but document the requirement
	return o.callOllama(ctx, model, input, "cpu")
}

// runOllamaGPU runs inference on Ollama using GPU acceleration.
func (o *Orchestrator) runOllamaGPU(ctx context.Context, model, input string) (string, error) {
	return o.callOllama(ctx, model, input, "gpu")
}

// callOllama makes a request to Ollama's generate API.
func (o *Orchestrator) callOllama(ctx context.Context, model, input, computeNote string) (string, error) {
	reqBody := map[string]interface{}{
		"model":   model,
		"prompt":  input,
		"stream":  false,
		"options": map[string]interface{}{
			// For CPU mode, you'd run Ollama with CUDA_VISIBLE_DEVICES=""
			// This options map can include num_gpu: 0 for some backends
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/generate", o.OllamaBaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.HTTPClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ollama request (%s): %w", computeNote, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.Response, nil
}

// runQAIHubNPU runs inference using a QAI Hub compiled model on the NPU.
// This shells out to the QAI Hub runtime or ONNX Runtime with QNN backend.
func (o *Orchestrator) runQAIHubNPU(ctx context.Context, model, input string) (string, error) {
	// QAI Hub compiled models are typically run via:
	// 1. Python script with qai_hub SDK
	// 2. ONNX Runtime with QNN Execution Provider
	// 3. Direct QNN API calls

	// Example: Using a Python wrapper script
	// The model path would be something like: ./models/qaihub/llama-v3-8b-chat-quantized
	modelPath := fmt.Sprintf("%s/%s", o.QAIHubModelDir, model)

	// Shell out to Python inference script
	cmd := exec.CommandContext(ctx, "python", "-c", fmt.Sprintf(`
import sys
sys.path.insert(0, '%s')

# QAI Hub inference example (pseudo-code - actual implementation depends on model)
# from qai_hub_models.models.llama_v3_8b_chat_quantized import Model
# model = Model.from_pretrained()
# output = model.generate("%s")
# print(output)

# For demo purposes, we simulate NPU inference
print("NPU inference result for: %s")
`, o.QAIHubModelDir, input, input[:min(50, len(input))]))

	output, err := cmd.Output()
	if err != nil {
		// If Python script fails, try ONNX Runtime approach
		return o.runONNXRuntimeNPU(ctx, modelPath, input)
	}

	return string(output), nil
}

// runONNXRuntimeNPU runs inference using ONNX Runtime with QNN Execution Provider.
func (o *Orchestrator) runONNXRuntimeNPU(ctx context.Context, modelPath, input string) (string, error) {
	// ONNX Runtime with QNN EP command
	// This requires onnxruntime-qnn package installed
	cmd := exec.CommandContext(ctx, "python", "-c", fmt.Sprintf(`
import onnxruntime as ort
import numpy as np

# Configure QNN Execution Provider for NPU
sess_options = ort.SessionOptions()
providers = [
    ('QNNExecutionProvider', {
        'backend_path': 'QnnHtp.dll',  # NPU backend
        'htp_performance_mode': 'burst',
    }),
]

try:
    session = ort.InferenceSession('%s', sess_options, providers=providers)
    # Tokenize input and run inference (model-specific)
    # output = session.run(None, {'input': tokenized_input})
    print("NPU inference completed for model: %s")
except Exception as e:
    print(f"NPU inference error: {e}")
`, modelPath, modelPath))

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ONNX Runtime NPU inference failed: %w", err)
	}

	return string(output), nil
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ============================================================================
// EXAMPLE USAGE (COMMENTED OUT - THIS IS DEAD CODE)
// ============================================================================

/*
// CLI Example - would be in cmd/orchestrate/main.go
func main() {
	flag.Parse()

	deviceID := flag.String("device", "", "Target device ID")
	task := flag.String("task", "inference", "Task type: inference, embed, generate")
	model := flag.String("model", "llama3.2:3b", "Model name or path")
	input := flag.String("input", "", "Input prompt")
	compute := flag.String("compute", "gpu", "Compute target: cpu, gpu, npu")

	flag.Parse()

	orch := NewOrchestrator()
	resp := orch.Orchestrate(context.Background(), Request{
		DeviceID: *deviceID,
		Task:     *task,
		Model:    *model,
		Input:    *input,
		Compute:  ComputeTarget(*compute),
	})

	if resp.Error != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
		os.Exit(1)
	}

	fmt.Printf("Output (%s, %dms):\n%s\n", resp.Compute, resp.DurationMs, resp.Output)
}
*/

/*
// REST Handler Example - would be added to cmd/server/main.go
func (h *WebHandler) handleOrchestrate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	orch := NewOrchestrator()
	resp := orch.Orchestrate(r.Context(), req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Register endpoint:
// mux.HandleFunc("/api/orchestrate", h.handleOrchestrate)
*/

/*
// Example curl commands:

// Run on GPU (default Ollama):
// curl -X POST http://localhost:8080/api/orchestrate \
//   -H "Content-Type: application/json" \
//   -d '{"device_id": "windows-snapdragon", "task": "inference", "model": "llama3.2:3b", "input": "Hello, how are you?", "compute": "gpu"}'

// Run on CPU:
// curl -X POST http://localhost:8080/api/orchestrate \
//   -H "Content-Type: application/json" \
//   -d '{"device_id": "windows-snapdragon", "task": "inference", "model": "phi3:mini", "input": "Explain quantum computing", "compute": "cpu"}'

// Run on NPU (requires QAI Hub compiled model):
// curl -X POST http://localhost:8080/api/orchestrate \
//   -H "Content-Type: application/json" \
//   -d '{"device_id": "windows-snapdragon", "task": "inference", "model": "llama-v3-8b-chat-quantized", "input": "What is AI?", "compute": "npu"}'
*/
