package llm

import (
	"context"
	"errors"
	"sync"
)

var ErrNoChatProvider = errors.New("no chat provider available")

// DeviceEndpointResolver returns a device's LLM endpoint when available.
// Returns (baseURL, modelName, ok). When ok is false, the fallback provider is used.
type DeviceEndpointResolver func() (baseURL, modelName string, ok bool)

// DeviceChatProvider routes chat to a device's local model when available,
// otherwise falls back to the configured chat provider.
type DeviceChatProvider struct {
	fallback ChatProvider
	resolver DeviceEndpointResolver
	timeout  int // seconds
	mu       sync.Mutex
}

// NewDeviceChatProvider creates a chat provider that prefers a device's LLM endpoint.
func NewDeviceChatProvider(fallback ChatProvider, resolver DeviceEndpointResolver, timeoutSecs int) *DeviceChatProvider {
	if timeoutSecs <= 0 {
		timeoutSecs = 60
	}
	return &DeviceChatProvider{
		fallback: fallback,
		resolver: resolver,
		timeout:  timeoutSecs,
	}
}

// Name returns the provider name.
func (d *DeviceChatProvider) Name() string {
	return "device_router"
}

// Health checks the device endpoint first, then fallback.
func (d *DeviceChatProvider) Health(ctx context.Context) (*HealthResult, error) {
	if baseURL, modelName, ok := d.resolver(); ok && baseURL != "" {
		cfg := ChatConfig{
			Provider:    "ollama",
			BaseURL:     baseURL,
			Model:       modelName,
			TimeoutSecs: d.timeout,
		}
		if modelName == "" {
			cfg.Model = "llama2"
		}
		deviceChat := NewOllamaChat(cfg)
		result, err := deviceChat.Health(ctx)
		if err == nil && result.Ok {
			result.Provider = "device_llm"
			return result, nil
		}
	}
	if d.fallback != nil {
		return d.fallback.Health(ctx)
	}
	return &HealthResult{
		Ok:       false,
		Provider: "device_router",
		Error:    "no device LLM available and no fallback configured",
	}, nil
}

// Chat routes to device's LLM when available, else fallback.
func (d *DeviceChatProvider) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	if baseURL, modelName, ok := d.resolver(); ok && baseURL != "" {
		cfg := ChatConfig{
			Provider:    "ollama",
			BaseURL:     baseURL,
			Model:       modelName,
			TimeoutSecs: d.timeout,
		}
		if modelName == "" {
			cfg.Model = "llama2"
		}
		deviceChat := NewOllamaChat(cfg)
		reply, err := deviceChat.Chat(ctx, messages)
		if err == nil {
			return reply, nil
		}
		// Fall through to fallback on device error
	}
	if d.fallback != nil {
		return d.fallback.Chat(ctx, messages)
	}
	return "", ErrNoChatProvider
}
