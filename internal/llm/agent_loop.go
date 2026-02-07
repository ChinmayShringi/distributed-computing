package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// DefaultMaxIterations is the default maximum number of tool calling iterations
const DefaultMaxIterations = 8

// SystemPrompt is the default system prompt for the agent
const SystemPrompt = `You are an AI assistant for a distributed orchestration system called EdgeMesh.
You have access to tools that can:
- List device capabilities (get_capabilities) - discovers all devices in the mesh
- Execute shell commands on devices (execute_shell_cmd) - runs commands remotely
- Read files from devices (get_file) - fetches file contents

Guidelines:
1. Call get_capabilities FIRST if you need device context or don't know available devices
2. Use device_id from get_capabilities to target specific devices - never invent device IDs
3. For file reads, prefer head/tail before full reads for large files
4. Dangerous shell commands (rm -rf, dd, mkfs, etc.) are blocked for safety
5. Summarize tool outputs concisely for the user
6. If a command fails, explain the error and suggest alternatives

Be helpful, direct, and efficient. Focus on solving the user's request.`

// AgentLoop manages the iterative tool calling loop
type AgentLoop struct {
	chat          ToolChatProvider
	executor      *ToolExecutor
	systemPrompt  string
	maxIterations int
}

// AgentLoopConfig contains configuration for the agent loop
type AgentLoopConfig struct {
	GRPCAddr      string
	SystemPrompt  string
	MaxIterations int
}

// NewAgentLoop creates a new agent loop
func NewAgentLoop(cfg AgentLoopConfig) (*AgentLoop, error) {
	// Create tool chat provider
	chat, err := NewToolChatFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create tool chat provider: %w", err)
	}

	// Create tool executor
	executor, err := NewToolExecutor(cfg.GRPCAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool executor: %w", err)
	}

	// Set defaults
	systemPrompt := cfg.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = SystemPrompt
	}

	maxIterations := cfg.MaxIterations
	if maxIterations <= 0 {
		maxIterations = getMaxIterationsFromEnv()
	}

	return &AgentLoop{
		chat:          chat,
		executor:      executor,
		systemPrompt:  systemPrompt,
		maxIterations: maxIterations,
	}, nil
}

// NewAgentLoopWithProvider creates an agent loop with a custom provider (useful for testing)
func NewAgentLoopWithProvider(chat ToolChatProvider, executor *ToolExecutor, systemPrompt string, maxIterations int) *AgentLoop {
	if systemPrompt == "" {
		systemPrompt = SystemPrompt
	}
	if maxIterations <= 0 {
		maxIterations = DefaultMaxIterations
	}
	return &AgentLoop{
		chat:          chat,
		executor:      executor,
		systemPrompt:  systemPrompt,
		maxIterations: maxIterations,
	}
}

// Close cleans up resources
func (a *AgentLoop) Close() error {
	if a.executor != nil {
		return a.executor.Close()
	}
	return nil
}

// AgentResponse is the final response from the agent
type AgentResponse struct {
	Reply        string         `json:"reply"`
	Iterations   int            `json:"iterations"`
	ToolCalls    []ToolCallInfo `json:"tool_calls,omitempty"`
	Error        string         `json:"error,omitempty"`
}

// ToolCallInfo records information about a tool call made during execution
type ToolCallInfo struct {
	Iteration int    `json:"iteration"`
	ToolName  string `json:"tool_name"`
	Arguments string `json:"arguments"`
	ResultLen int    `json:"result_len"`
}

// Run executes the agent loop for a user message, with history
func (a *AgentLoop) Run(ctx context.Context, userMessage string, history []ToolChatMessage) (*AgentResponse, error) {
	// Initialize conversation with system prompt
	messages := []ToolChatMessage{
		{Role: "system", Content: a.systemPrompt},
	}
	// Append history
	messages = append(messages, history...)
	
	// Append current user message
	messages = append(messages, ToolChatMessage{Role: "user", Content: userMessage})

	tools := GetToolDefinitions()
	var toolCallLog []ToolCallInfo

	for i := 0; i < a.maxIterations; i++ {
		// Send request to LLM
		resp, err := a.chat.ChatWithTools(ctx, messages, tools)
		if err != nil {
			return &AgentResponse{
				Iterations: i + 1,
				ToolCalls:  toolCallLog,
				Error:      fmt.Sprintf("LLM request failed: %v", err),
			}, nil
		}

		// If no tool calls, we're done
		if resp.FinishReason == "stop" || len(resp.ToolCalls) == 0 {
			return &AgentResponse{
				Reply:      resp.Content,
				Iterations: i + 1,
				ToolCalls:  toolCallLog,
			}, nil
		}

		// Add assistant message with tool calls
		messages = append(messages, ToolChatMessage{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// Execute each tool call
		for _, tc := range resp.ToolCalls {
			result, err := a.executor.Execute(ctx, tc)

			var resultJSON []byte
			if err != nil {
				resultJSON, _ = json.Marshal(map[string]string{
					"error": err.Error(),
				})
			} else {
				resultJSON, _ = json.Marshal(result)
			}

			// Log tool call
			toolCallLog = append(toolCallLog, ToolCallInfo{
				Iteration: i + 1,
				ToolName:  tc.Function.Name,
				Arguments: tc.Function.Arguments,
				ResultLen: len(resultJSON),
			})

			// Add tool result message
			messages = append(messages, ToolChatMessage{
				Role:       "tool",
				Content:    string(resultJSON),
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			})
		}
	}

	// Max iterations reached
	return &AgentResponse{
		Iterations: a.maxIterations,
		ToolCalls:  toolCallLog,
		Error:      fmt.Sprintf("max iterations (%d) reached without final response", a.maxIterations),
	}, nil
}

// RunSimple is a convenience method that returns just the reply string
func (a *AgentLoop) RunSimple(ctx context.Context, userMessage string) (string, error) {
	resp, err := a.Run(ctx, userMessage, nil)
	if err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", fmt.Errorf("%s", resp.Error)
	}
	return resp.Reply, nil
}

// getMaxIterationsFromEnv reads max iterations from environment
func getMaxIterationsFromEnv() int {
	if val := os.Getenv("AGENT_MAX_ITERATIONS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			return n
		}
	}
	return DefaultMaxIterations
}

// HealthCheck verifies both the LLM provider and gRPC server are reachable
func (a *AgentLoop) HealthCheck(ctx context.Context) (*HealthResult, error) {
	return a.chat.Health(ctx)
}
