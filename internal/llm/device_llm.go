package llm

import (
	"context"
	"sync"
)

// DeviceLLMEndpointResolver returns a device's LLM endpoint when available.
type DeviceLLMEndpointResolver func() (baseURL, modelName string, ok bool)

// DeviceLLMProvider wraps an LLM Provider and prefers a device's endpoint when available.
// Uses Ollama/OpenAI-compat at the device's local_chat_endpoint for plan generation.
type DeviceLLMProvider struct {
	fallback Provider
	resolver DeviceLLMEndpointResolver
	timeout  int
	mu       sync.Mutex
}

// NewDeviceLLMProvider creates an LLM provider that prefers a device's endpoint.
func NewDeviceLLMProvider(fallback Provider, resolver DeviceLLMEndpointResolver, timeoutSecs int) *DeviceLLMProvider {
	if timeoutSecs <= 0 {
		timeoutSecs = 20
	}
	return &DeviceLLMProvider{
		fallback: fallback,
		resolver: resolver,
		timeout:  timeoutSecs,
	}
}

// Name returns the provider name.
func (d *DeviceLLMProvider) Name() string {
	return "device_llm_router"
}

// Plan generates a plan using device's LLM when available, else fallback.
func (d *DeviceLLMProvider) Plan(ctx context.Context, userText string, devicesJSON string) (string, error) {
	if baseURL, modelName, ok := d.resolver(); ok && baseURL != "" {
		cfg := Config{
			BaseURL:     baseURL,
			Model:       modelName,
			TimeoutSecs: d.timeout,
			Temperature: 0.2,
			MaxTokens:   900,
		}
		if modelName == "" {
			cfg.Model = "llama2"
		}
		deviceProvider := NewOpenAICompat(cfg)
		plan, err := deviceProvider.Plan(ctx, userText, devicesJSON)
		if err == nil {
			return plan, nil
		}
		// Fall through to fallback on device error
	}
	if d.fallback != nil {
		return d.fallback.Plan(ctx, userText, devicesJSON)
	}
	return "", nil
}
