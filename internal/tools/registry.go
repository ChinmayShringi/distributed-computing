// Package tools provides the tool registry for CLI tool execution
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	cmdexec "github.com/edgecli/edgecli/internal/exec"
	"github.com/edgecli/edgecli/internal/mode"
)

// ToolResult is the result of executing a tool
type ToolResult struct {
	OK         bool   `json:"ok"`
	ExitCode   int    `json:"exit_code"`
	StdoutTail string `json:"stdout_tail"`
	StderrTail string `json:"stderr_tail"`
	Message    string `json:"message,omitempty"`
}

// StepResult represents the result of a single step in a multi-step tool
type StepResult struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Skipped bool   `json:"skipped,omitempty"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// RecipeResult represents the result of a multi-step tool recipe
type RecipeResult struct {
	Steps          []StepResult `json:"steps"`
	OverallSuccess bool         `json:"overall_success"`
	FailedAtStep   string       `json:"failed_at_step,omitempty"`
	FinalStatus    string       `json:"final_status"`
}

// Tool is the interface for all tools
type Tool interface {
	Name() string
	Description() string
	ArgsSchema() json.RawMessage
	IsDangerous() bool
	Run(ctx context.Context, args map[string]interface{}) (*ToolResult, error)
}

// Registry holds all registered tools
type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	r := &Registry{
		tools: make(map[string]Tool),
	}
	return r
}

// Register adds a tool to the registry
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// Get returns a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// Execute runs a tool by name with the given arguments
func (r *Registry) Execute(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error) {
	tool, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	return tool.Run(ctx, args)
}

// ExecuteWithMode runs a tool with mode-aware safety controls
// In DANGEROUS mode: executes directly without additional checks
// In SAFE mode: same as Execute (can add custom safety logic here)
func (r *Registry) ExecuteWithMode(ctx context.Context, name string, args map[string]interface{}, modeCtx *mode.ModeContext) (*ToolResult, error) {
	tool, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	// Add custom safety logic here if needed for SAFE mode
	// if !modeCtx.Mode.IsDangerous() { ... }

	return tool.Run(ctx, args)
}

// ListTools returns all registered tools
func (r *Registry) ListTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// DefaultRegistry returns the default tool registry
func DefaultRegistry() *Registry {
	return NewRegistry()
}

// Runner returns the command execution runner
func Runner() *cmdexec.Runner {
	return cmdexec.NewRunner()
}

// ---- Helper Functions ----

// BoolToExitCode converts a failed boolean to exit code
func BoolToExitCode(failed bool) int {
	if failed {
		return 1
	}
	return 0
}

// ParseBoolArg handles consent/boolean args that may be passed as bool or string
func ParseBoolArg(args map[string]interface{}, key string) bool {
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return v == "true" || v == "True" || v == "TRUE" || v == "1" || v == "yes" || v == "Yes" || v == "YES"
		case float64:
			return v != 0
		case int:
			return v != 0
		}
	}
	return false
}

// GetConsent returns true for dangerous tools (user already consented to dangerous mode),
// otherwise parses the consent arg from the request
func GetConsent(args map[string]interface{}, isDangerous bool) bool {
	if isDangerous {
		return true
	}
	return ParseBoolArg(args, "consent")
}
