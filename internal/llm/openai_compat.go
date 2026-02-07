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

const systemPrompt = `You are a task planner for a distributed orchestration system.
You must output ONLY valid JSON — no markdown, no commentary, no explanation.

The JSON must have this exact structure:
{
  "groups": [
    {
      "index": 0,
      "tasks": [
        {
          "task_id": "<unique short id>",
          "kind": "SYSINFO" or "ECHO" or "LLM_GENERATE",
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
  * LLM_GENERATE: run LLM inference (summarize, generate text, answer questions, generate code)
- For LLM_GENERATE tasks:
  * input = the prompt text
  * prompt_tokens = estimated input tokens (rough: ~4 chars per token)
  * max_output_tokens = expected output length (summary=200, code=500, chat=300)
  * Prefer NPU devices (Windows Snapdragon), then GPU, then CPU
  * Prefer devices with more RAM for larger models
- target_device_id: use a specific device_id from the devices list, or leave empty for auto-assignment
- File paths must be relative to ./shared, no absolute paths, no ".." traversal
- reduce.kind must be "CONCAT"
- task_id must be unique across all tasks

Model/Task Mapping (use LLM_GENERATE for these):
- User wants to summarize text → LLM_GENERATE on NPU device
- User wants to generate code → LLM_GENERATE on NPU device  
- User wants to chat/answer questions → LLM_GENERATE on NPU device
- User wants system info → SYSINFO (not LLM)
- User wants to test echo → ECHO (not LLM)`

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
