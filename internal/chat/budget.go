// Package chat provides utilities for chat session management
package chat

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	// DefaultMaxToolCallsPerTurn is the maximum number of tool calls allowed per user turn
	DefaultMaxToolCallsPerTurn = 6
	// DefaultMaxRepeatedSameTool is the maximum times the same tool+args can be called
	DefaultMaxRepeatedSameTool = 2
	// MaxOutputTailLength is the max length of output tail to store
	MaxOutputTailLength = 200
)

// ToolExecution records a single tool execution
type ToolExecution struct {
	ToolName   string    `json:"tool_name"`
	Args       string    `json:"args"`
	OutputTail string    `json:"output_tail"`
	Timestamp  time.Time `json:"timestamp"`
	Blocked    bool      `json:"blocked"`
	Reason     string    `json:"reason,omitempty"`
}

// ExecutionBudget tracks tool execution limits per user turn
type ExecutionBudget struct {
	MaxToolCallsPerTurn int `json:"max_tool_calls_per_turn"`
	MaxRepeatedSameTool int `json:"max_repeated_same_tool"`

	toolCallCount  int
	toolExecutions map[string]int // key: dedupe key, value: count
	executionLog   []ToolExecution
}

// NewExecutionBudget creates a new budget with default limits
func NewExecutionBudget() *ExecutionBudget {
	return &ExecutionBudget{
		MaxToolCallsPerTurn: DefaultMaxToolCallsPerTurn,
		MaxRepeatedSameTool: DefaultMaxRepeatedSameTool,
		toolExecutions:      make(map[string]int),
		executionLog:        make([]ToolExecution, 0),
	}
}

// Reset clears the budget for a new user turn
func (b *ExecutionBudget) Reset() {
	b.toolCallCount = 0
	b.toolExecutions = make(map[string]int)
	b.executionLog = make([]ToolExecution, 0)
}

// DedupeKey generates a unique key for a tool+args combination
func (b *ExecutionBudget) DedupeKey(toolName, args string) string {
	// Normalize args by parsing and re-encoding JSON if possible
	normalizedArgs := args
	var parsed interface{}
	if err := json.Unmarshal([]byte(args), &parsed); err == nil {
		if normalized, err := json.Marshal(parsed); err == nil {
			normalizedArgs = string(normalized)
		}
	}

	// Create a hash for long args
	combined := toolName + ":" + normalizedArgs
	if len(combined) > 100 {
		hash := sha256.Sum256([]byte(combined))
		return toolName + ":" + hex.EncodeToString(hash[:8])
	}
	return combined
}

// CanExecute checks if a tool execution is allowed
// Returns (allowed, reason)
func (b *ExecutionBudget) CanExecute(toolName, args string) (bool, string) {
	// Check total budget
	if b.toolCallCount >= b.MaxToolCallsPerTurn {
		return false, fmt.Sprintf("budget exhausted: %d/%d tool calls used",
			b.toolCallCount, b.MaxToolCallsPerTurn)
	}

	// Check dedupe
	key := b.DedupeKey(toolName, args)
	if count := b.toolExecutions[key]; count >= b.MaxRepeatedSameTool {
		return false, fmt.Sprintf("duplicate blocked: %s already called %d times with same args",
			toolName, count)
	}

	return true, ""
}

// RecordExecution records a tool execution (successful or blocked)
func (b *ExecutionBudget) RecordExecution(toolName, args, output string, blocked bool, reason string) {
	key := b.DedupeKey(toolName, args)

	if !blocked {
		b.toolCallCount++
		b.toolExecutions[key]++
	}

	// Truncate output tail
	outputTail := output
	if len(outputTail) > MaxOutputTailLength {
		outputTail = "..." + outputTail[len(outputTail)-MaxOutputTailLength:]
	}

	b.executionLog = append(b.executionLog, ToolExecution{
		ToolName:   toolName,
		Args:       args,
		OutputTail: outputTail,
		Timestamp:  time.Now(),
		Blocked:    blocked,
		Reason:     reason,
	})
}

// IsExhausted returns true if the budget is exhausted
func (b *ExecutionBudget) IsExhausted() bool {
	return b.toolCallCount >= b.MaxToolCallsPerTurn
}

// ToolCallCount returns the current number of tool calls
func (b *ExecutionBudget) ToolCallCount() int {
	return b.toolCallCount
}

// Summary returns a human-readable summary of the budget state
func (b *ExecutionBudget) Summary() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Execution budget exhausted (%d tool calls reached)\n\n",
		b.MaxToolCallsPerTurn))
	sb.WriteString("Tools executed this turn:\n")

	for i, exec := range b.executionLog {
		status := ""
		if exec.Blocked {
			status = fmt.Sprintf(" (blocked: %s)", exec.Reason)
		} else if exec.OutputTail != "" {
			// Show abbreviated output
			shortOutput := exec.OutputTail
			if len(shortOutput) > 50 {
				shortOutput = shortOutput[:47] + "..."
			}
			status = fmt.Sprintf(" -> %s", shortOutput)
		}
		sb.WriteString(fmt.Sprintf("  %d. %s%s\n", i+1, exec.ToolName, status))
	}

	sb.WriteString("\nSuggested next steps:\n")
	sb.WriteString("  - Register custom tools using the Tool interface\n")
	sb.WriteString("  - Run `edgecli --help` for available commands\n")

	return sb.String()
}

// ExecutionLog returns the list of executions for this turn
func (b *ExecutionBudget) ExecutionLog() []ToolExecution {
	return b.executionLog
}
