package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const systemPrompt = `You are a task planner for a distributed AI orchestration system.
You must output ONLY valid JSON — no markdown, no commentary, no explanation.

The JSON must have this exact structure:
{
  "groups": [
    {
      "index": 0,
      "tasks": [
        {
          "task_id": "<unique short id>",
          "kind": "SYSINFO" or "ECHO" or "LLM_GENERATE" or "IMAGE_GENERATE",
          "input": "<command input or prompt text>",
          "target_device_id": "<device_id or empty for auto-assign>",
          "prompt_tokens": 100,
          "max_output_tokens": 500
        }
      ]
    }
  ],
  "reduce": {
    "kind": "CONCAT"
  }
}

Rules:
- Groups execute sequentially (index 0 first, then 1, etc.)
- Tasks within a group execute in parallel across devices
- Valid task kinds:
  * SYSINFO: gather system info from a device
  * ECHO: echo input text back
  * LLM_GENERATE: run LLM inference (summarize, generate text, answer questions, code)
  * IMAGE_GENERATE: generate images

**CRITICAL: LLM Device Selection**
- Each device has "has_local_model" field - if true, the device can run LLM tasks
- Only assign LLM_GENERATE tasks to devices with "has_local_model": true
- Check "local_model_name" to understand model capabilities:
  * Vision models (qwen3-vl*, gpt-4-vision): Can handle text + image tasks
  * Chat models (llama*, qwen3-4b, mistral*): Text generation, code, Q&A
- Prefer NPU devices for LLM tasks, then GPU, then CPU

**Multi-Device Task Distribution Strategy**
When user request is complex with multiple independent parts:
1. Identify distinct sub-tasks that can run in parallel
2. Create multiple LLM_GENERATE tasks in the SAME group (group 0)
3. Assign each task to a DIFFERENT LLM-capable device
4. Maximize parallelism: use as many LLM devices as you have sub-tasks

Examples of complex requests to decompose:
- "Analyze X, Y, and Z" → 3 LLM tasks (X, Y, Z) to different devices
- "Compare A vs B" → 2 LLM tasks (analyze A, analyze B) to different devices  
- "Create 3 summaries of different aspects" → 3 LLM tasks to different devices

**Task Decomposition Examples:**

User: "Perform analysis across 3 dimensions: (1) Technical architecture (2) AI capabilities (3) Business value"
→ Output:
{
  "groups": [{
    "index": 0,
    "tasks": [
      {"task_id": "tech", "kind": "LLM_GENERATE", "input": "Analyze the technical architecture in detail", "target_device_id": "device-A-with-llm", "prompt_tokens": 100, "max_output_tokens": 400},
      {"task_id": "ai", "kind": "LLM_GENERATE", "input": "Evaluate AI/ML capabilities comprehensively", "target_device_id": "device-B-with-llm", "prompt_tokens": 100, "max_output_tokens": 400},
      {"task_id": "biz", "kind": "LLM_GENERATE", "input": "Assess business value and use cases", "target_device_id": "device-C-with-llm", "prompt_tokens": 100, "max_output_tokens": 400}
    ]
  }],
  "reduce": {"kind": "CONCAT"}
}

User: "What is EdgeMesh in 2 sentences?"
→ Output (single task, simple request):
{
  "groups": [{
    "index": 0,
    "tasks": [
      {"task_id": "sum", "kind": "LLM_GENERATE", "input": "Explain EdgeMesh in exactly 2 sentences", "target_device_id": "device-with-best-llm", "prompt_tokens": 50, "max_output_tokens": 100}
    ]
  }],
  "reduce": {"kind": "CONCAT"}
}

**Additional Rules:**
- target_device_id: use a specific device_id from the devices list with has_local_model=true
- If multiple LLM devices available, DISTRIBUTE tasks across them for parallel execution
- prompt_tokens: estimate ~4 chars per token
- max_output_tokens: summary=200, detailed=500, code=800
- File paths must be relative to ./shared, no absolute paths, no ".." traversal
- reduce.kind must be "CONCAT"
- task_id must be unique across all tasks
- For non-LLM tasks (SYSINFO, ECHO): target_device_id can be empty for auto-assignment`

// OpenAICompat implements Provider by calling an OpenAI-compatible /v1/chat/completions endpoint.
type OpenAICompat struct {
	cfg    Config
	client *http.Client
}

// NewOpenAICompat creates a new OpenAI-compatible provider.
func NewOpenAICompat(cfg Config) *OpenAICompat {
	return &OpenAICompat{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSecs) * time.Second,
		},
	}
}

func (o *OpenAICompat) Name() string { return "openai_compat" }

// chatRequest is the minimal request body for /v1/chat/completions.
type chatRequest struct {
	Model       string        `json:"model,omitempty"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse is the minimal response from /v1/chat/completions.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (o *OpenAICompat) Plan(ctx context.Context, userText string, devicesJSON string) (string, error) {
	userMsg := fmt.Sprintf("User request: %s\n\nAvailable devices:\n%s", userText, devicesJSON)

	reqBody := chatRequest{
		Model: o.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMsg},
		},
		Temperature: o.cfg.Temperature,
		MaxTokens:   o.cfg.MaxTokens,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(o.cfg.BaseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if o.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		snippet := string(respBody)
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return "", fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, snippet)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal LLM response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned 0 choices")
	}

	return chatResp.Choices[0].Message.Content, nil
}
