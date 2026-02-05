// Package llm provides a pluggable LLM provider interface for plan generation.
package llm

import (
	"context"
	"fmt"
	"os"
	"strconv"
)

// Provider is the interface for LLM-based plan generation.
type Provider interface {
	// Name returns the provider name (e.g., "openai_compat").
	Name() string

	// Plan generates an execution plan JSON from user text and a device list.
	Plan(ctx context.Context, userText string, devicesJSON string) (planJSON string, err error)
}

// Config holds LLM provider configuration.
type Config struct {
	BaseURL     string
	APIKey      string
	Model       string
	TimeoutSecs int
	MaxTokens   int
	Temperature float64
}

// NewFromEnv creates a Provider from environment variables.
// Returns (nil, nil) if the provider is disabled.
func NewFromEnv() (Provider, error) {
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" || provider == "disabled" {
		return nil, nil
	}

	cfg := Config{
		BaseURL:     envOrDefault("LLM_BASE_URL", "http://127.0.0.1:1234"),
		APIKey:      os.Getenv("LLM_API_KEY"),
		Model:       os.Getenv("LLM_MODEL"),
		TimeoutSecs: envIntOrDefault("LLM_TIMEOUT_SECONDS", 20),
		MaxTokens:   envIntOrDefault("LLM_MAX_TOKENS", 900),
		Temperature: envFloatOrDefault("LLM_TEMPERATURE", 0.2),
	}

	switch provider {
	case "openai_compat":
		return NewOpenAICompat(cfg), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", provider)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envFloatOrDefault(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
