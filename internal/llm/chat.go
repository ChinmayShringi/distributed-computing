package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ChatProvider is the interface for chat completion.
type ChatProvider interface {
	// Name returns the provider name (e.g., "ollama", "openai").
	Name() string

	// Health checks if the chat service is reachable and returns status.
	Health(ctx context.Context) (*HealthResult, error)

	// Chat sends messages and returns the assistant's reply.
	Chat(ctx context.Context, messages []ChatMessage) (string, error)
}

// ChatMessage represents a single message in a conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// HealthResult contains the status of a chat provider.
type HealthResult struct {
	Ok       bool   `json:"ok"`
	Provider string `json:"provider"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
	Error    string `json:"error,omitempty"`
}

// ChatConfig holds chat provider configuration.
type ChatConfig struct {
	Provider    string
	BaseURL     string
	Model       string
	APIKey      string
	TimeoutSecs int
}

// NewChatFromEnv creates a ChatProvider from environment variables.
// Environment variables:
//   - CHAT_PROVIDER: "ollama" (default) or "openai"
//   - CHAT_BASE_URL: base URL (default: http://localhost:11434 for Ollama)
//   - CHAT_MODEL: model name (default: "llama2" for Ollama)
//   - CHAT_API_KEY: API key (optional, for OpenAI-compatible providers)
//   - CHAT_TIMEOUT_SECONDS: request timeout (default: 60)
func NewChatFromEnv() (ChatProvider, error) {
	provider := envOrDefault("CHAT_PROVIDER", "ollama")
	timeoutSecs := envIntOrDefault("CHAT_TIMEOUT_SECONDS", 60)

	cfg := ChatConfig{
		Provider:    provider,
		APIKey:      os.Getenv("CHAT_API_KEY"),
		TimeoutSecs: timeoutSecs,
	}

	switch provider {
	case "ollama":
		cfg.BaseURL = envOrDefault("CHAT_BASE_URL", "http://localhost:11434")
		cfg.Model = envOrDefault("CHAT_MODEL", "llama2")
		return NewOllamaChat(cfg), nil

	case "openai":
		cfg.BaseURL = envOrDefault("CHAT_BASE_URL", "http://localhost:1234") // LM Studio default
		cfg.Model = os.Getenv("CHAT_MODEL")                                  // Let the provider use its default if not set
		return NewOpenAIChat(cfg), nil

	case "echo", "mock":
		// Echo provider for testing when no LLM runtime is available
		return NewEchoChat(), nil

	default:
		return nil, fmt.Errorf("unknown chat provider: %s (valid: ollama, openai)", provider)
	}
}

// OllamaChat implements ChatProvider using Ollama's native API.
type OllamaChat struct {
	cfg    ChatConfig
	client *http.Client
}

// NewOllamaChat creates a new Ollama chat provider.
func NewOllamaChat(cfg ChatConfig) *OllamaChat {
	return &OllamaChat{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSecs) * time.Second,
		},
	}
}

func (o *OllamaChat) Name() string { return "ollama" }

func (o *OllamaChat) Health(ctx context.Context) (*HealthResult, error) {
	result := &HealthResult{
		Provider: "ollama",
		BaseURL:  o.cfg.BaseURL,
		Model:    o.cfg.Model,
	}

	// Check if Ollama is running by hitting the root endpoint
	url := strings.TrimRight(o.cfg.BaseURL, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result, nil
	}

	resp, err := o.client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("failed to connect: %v", err)
		return result, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		result.Ok = true
	} else {
		result.Error = fmt.Sprintf("unexpected status: %d", resp.StatusCode)
	}

	return result, nil
}

// ollamaChatRequest is the request body for Ollama's /api/chat endpoint.
type ollamaChatRequest struct {
	Model    string             `json:"model"`
	Messages []ollamaChatMsg    `json:"messages"`
	Stream   bool               `json:"stream"`
	Options  *ollamaChatOptions `json:"options,omitempty"`
}

type ollamaChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
}

// ollamaChatResponse is the response from Ollama's /api/chat endpoint.
type ollamaChatResponse struct {
	Message ollamaChatMsg `json:"message"`
	Done    bool          `json:"done"`
}

func (o *OllamaChat) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	// Convert messages to Ollama format
	ollamaMsgs := make([]ollamaChatMsg, len(messages))
	for i, m := range messages {
		ollamaMsgs[i] = ollamaChatMsg{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	reqBody := ollamaChatRequest{
		Model:    o.cfg.Model,
		Messages: ollamaMsgs,
		Stream:   false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(o.cfg.BaseURL, "/") + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, snippet)
	}

	var chatResp ollamaChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	return chatResp.Message.Content, nil
}

// OpenAIChat implements ChatProvider using OpenAI-compatible API.
type OpenAIChat struct {
	cfg    ChatConfig
	client *http.Client
}

// NewOpenAIChat creates a new OpenAI-compatible chat provider.
func NewOpenAIChat(cfg ChatConfig) *OpenAIChat {
	return &OpenAIChat{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSecs) * time.Second,
		},
	}
}

func (o *OpenAIChat) Name() string { return "openai" }

func (o *OpenAIChat) Health(ctx context.Context) (*HealthResult, error) {
	result := &HealthResult{
		Provider: "openai",
		BaseURL:  o.cfg.BaseURL,
		Model:    o.cfg.Model,
	}

	// Try to list models to check connectivity
	url := strings.TrimRight(o.cfg.BaseURL, "/") + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result, nil
	}

	if o.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("failed to connect: %v", err)
		return result, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		result.Ok = true
	} else {
		body, _ := io.ReadAll(resp.Body)
		snippet := string(body)
		if len(snippet) > 100 {
			snippet = snippet[:100] + "..."
		}
		result.Error = fmt.Sprintf("status %d: %s", resp.StatusCode, snippet)
	}

	return result, nil
}

// openaiChatRequest is the request body for /v1/chat/completions.
type openaiChatRequest struct {
	Model       string          `json:"model,omitempty"`
	Messages    []openaiChatMsg `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type openaiChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiChatResponse is the response from /v1/chat/completions.
type openaiChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (o *OpenAIChat) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	// Convert messages to OpenAI format
	openaiMsgs := make([]openaiChatMsg, len(messages))
	for i, m := range messages {
		openaiMsgs[i] = openaiChatMsg{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	reqBody := openaiChatRequest{
		Model:       o.cfg.Model,
		Messages:    openaiMsgs,
		Temperature: 0.7,
		MaxTokens:   1024,
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
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, snippet)
	}

	var chatResp openaiChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("API returned 0 choices")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// EchoChat is a mock provider for testing when no LLM runtime is available.
// It echoes back the last user message with a prefix.
type EchoChat struct{}

// NewEchoChat creates a new echo chat provider.
func NewEchoChat() *EchoChat {
	return &EchoChat{}
}

func (e *EchoChat) Name() string { return "echo" }

func (e *EchoChat) Health(ctx context.Context) (*HealthResult, error) {
	return &HealthResult{
		Ok:       true,
		Provider: "echo",
		BaseURL:  "local",
		Model:    "echo-mock",
	}, nil
}

func (e *EchoChat) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	if len(messages) == 0 {
		return "Echo: (no messages)", nil
	}

	// Find the last user message
	var lastUserMsg string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUserMsg = messages[i].Content
			break
		}
	}

	if lastUserMsg == "" {
		return "Echo: (no user message found)", nil
	}

	return fmt.Sprintf("Echo: You said: %q\n\nThis is a mock response. Set CHAT_PROVIDER=ollama or CHAT_PROVIDER=openai for real LLM responses.", lastUserMsg), nil
}
