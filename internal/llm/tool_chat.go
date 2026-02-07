package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ToolChatProvider extends ChatProvider with tool calling support
type ToolChatProvider interface {
	// Name returns the provider name
	Name() string

	// Health checks if the chat service is reachable
	Health(ctx context.Context) (*HealthResult, error)

	// ChatWithTools sends messages with tool definitions and returns structured response
	ChatWithTools(ctx context.Context, messages []ToolChatMessage, tools []ToolDefinition) (*ToolChatResponse, error)
}

// ToolChatMessage represents a message in a tool-enabled conversation
type ToolChatMessage struct {
	Role       string     `json:"role"`                   // "system", "user", "assistant", "tool"
	Content    string     `json:"content,omitempty"`      // Text content
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // For assistant messages with tool calls
	ToolCallID string     `json:"tool_call_id,omitempty"` // For tool result messages
	Name       string     `json:"name,omitempty"`         // Tool name for tool result messages
}

// ToolCall represents a function call requested by the LLM
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall contains the function name and arguments
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolChatResponse contains the LLM's response with potential tool calls
type ToolChatResponse struct {
	Content      string     `json:"content,omitempty"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string     `json:"finish_reason"` // "stop", "tool_calls"
}

// OpenAIToolChat implements ToolChatProvider using OpenAI-compatible API
type OpenAIToolChat struct {
	cfg    ChatConfig
	client *http.Client
}

// NewOpenAIToolChat creates a new OpenAI-compatible tool chat provider
func NewOpenAIToolChat(cfg ChatConfig) *OpenAIToolChat {
	return &OpenAIToolChat{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSecs) * time.Second,
		},
	}
}

func (o *OpenAIToolChat) Name() string { return "openai-tools" }

func (o *OpenAIToolChat) Health(ctx context.Context) (*HealthResult, error) {
	// Reuse the simple OpenAI health check
	chat := NewOpenAIChat(o.cfg)
	return chat.Health(ctx)
}

// openaiToolChatRequest is the request body for /v1/chat/completions with tools
type openaiToolChatRequest struct {
	Model       string            `json:"model,omitempty"`
	Messages    []json.RawMessage `json:"messages"`
	Tools       []ToolDefinition  `json:"tools,omitempty"`
	ToolChoice  string            `json:"tool_choice,omitempty"` // "auto", "none", or specific tool
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
}

// openaiToolChatResponse is the response from /v1/chat/completions with tools
type openaiToolChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string     `json:"role"`
			Content   *string    `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// ChatWithTools sends a chat request with tool definitions
func (o *OpenAIToolChat) ChatWithTools(ctx context.Context, messages []ToolChatMessage, tools []ToolDefinition) (*ToolChatResponse, error) {
	// Convert messages to JSON format compatible with OpenAI
	jsonMsgs := make([]json.RawMessage, len(messages))
	for i, m := range messages {
		msgBytes, err := json.Marshal(m)
		if err != nil {
			return nil, fmt.Errorf("marshal message %d: %w", i, err)
		}
		jsonMsgs[i] = msgBytes
	}

	reqBody := openaiToolChatRequest{
		Model:       o.cfg.Model,
		Messages:    jsonMsgs,
		Tools:       tools,
		ToolChoice:  "auto",
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := o.cfg.BaseURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if o.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		snippet := string(respBody)
		if len(snippet) > 500 {
			snippet = snippet[:500] + "..."
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, snippet)
	}

	var chatResp openaiToolChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("API returned 0 choices")
	}

	choice := chatResp.Choices[0]
	content := ""
	if choice.Message.Content != nil {
		content = *choice.Message.Content
	}

	return &ToolChatResponse{
		Content:      content,
		ToolCalls:    choice.Message.ToolCalls,
		FinishReason: choice.FinishReason,
	}, nil
}

// NewToolChatFromEnv creates a ToolChatProvider from environment variables
// Uses the same env vars as the regular chat provider (CHAT_PROVIDER, CHAT_BASE_URL, etc.)
func NewToolChatFromEnv() (ToolChatProvider, error) {
	provider := envOrDefault("CHAT_PROVIDER", "ollama")
	timeoutSecs := envIntOrDefault("CHAT_TIMEOUT_SECONDS", 120) // Longer default for tool calling

	cfg := ChatConfig{
		Provider:    provider,
		APIKey:      envOrDefault("CHAT_API_KEY", ""),
		TimeoutSecs: timeoutSecs,
	}

	switch provider {
	case "openai":
		cfg.BaseURL = envOrDefault("CHAT_BASE_URL", "http://localhost:1234") // LM Studio default
		cfg.Model = envOrDefault("CHAT_MODEL", "")
		return NewOpenAIToolChat(cfg), nil

	case "ollama":
		// Ollama doesn't support native tool calling in the same way
		// Fall back to OpenAI-compatible endpoint if available
		cfg.BaseURL = envOrDefault("CHAT_BASE_URL", "http://localhost:11434")
		cfg.Model = envOrDefault("CHAT_MODEL", "llama2")
		// Try OpenAI-compatible mode (Ollama supports this at /v1/chat/completions)
		return NewOpenAIToolChat(ChatConfig{
			BaseURL:     cfg.BaseURL,
			Model:       cfg.Model,
			APIKey:      cfg.APIKey,
			TimeoutSecs: cfg.TimeoutSecs,
		}), nil

	case "echo", "mock":
		return NewEchoToolChat(), nil

	default:
		return nil, fmt.Errorf("unknown chat provider for tool calling: %s (valid: openai, ollama, echo)", provider)
	}
}

// EchoToolChat is a mock provider for testing tool calling
type EchoToolChat struct{}

// NewEchoToolChat creates a new echo tool chat provider
func NewEchoToolChat() *EchoToolChat {
	return &EchoToolChat{}
}

func (e *EchoToolChat) Name() string { return "echo-tools" }

func (e *EchoToolChat) Health(ctx context.Context) (*HealthResult, error) {
	return &HealthResult{
		Ok:       true,
		Provider: "echo-tools",
		BaseURL:  "local",
		Model:    "echo-mock",
	}, nil
}

func (e *EchoToolChat) ChatWithTools(ctx context.Context, messages []ToolChatMessage, tools []ToolDefinition) (*ToolChatResponse, error) {
	// Find last user message
	var lastUserMsg string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUserMsg = messages[i].Content
			break
		}
	}

	// Check if this looks like a tool result (last message is a tool message)
	if len(messages) > 0 && messages[len(messages)-1].Role == "tool" {
		// This is a follow-up after tool execution, return final response
		return &ToolChatResponse{
			Content:      fmt.Sprintf("Echo: Processed tool results. Original request was: %q\n\nThis is a mock response. Set CHAT_PROVIDER=openai for real LLM tool calling.", lastUserMsg),
			FinishReason: "stop",
		}, nil
	}

	// First turn: always call get_capabilities as an example
	if len(tools) > 0 {
		return &ToolChatResponse{
			Content: "",
			ToolCalls: []ToolCall{
				{
					ID:   "call_echo_1",
					Type: "function",
					Function: FunctionCall{
						Name:      "get_capabilities",
						Arguments: `{"include_benchmarks": true}`,
					},
				},
			},
			FinishReason: "tool_calls",
		}, nil
	}

	return &ToolChatResponse{
		Content:      fmt.Sprintf("Echo: You said: %q\n\nNo tools available.", lastUserMsg),
		FinishReason: "stop",
	}, nil
}
