package llm

import "encoding/json"

// ToolDefinition represents an OpenAI-style tool definition for function calling
type ToolDefinition struct {
	Type     string         `json:"type"`
	Function FunctionDef    `json:"function"`
}

// FunctionDef defines a function that the LLM can call
type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolDefinitions contains all available tools for LLM function calling
var ToolDefinitions = []ToolDefinition{
	{
		Type: "function",
		Function: FunctionDef{
			Name:        "get_capabilities",
			Description: "List all registered devices in the mesh with their capabilities, hardware info, and benchmarks. Call this first to discover available devices before running commands or reading files.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"include_benchmarks": {
						"type": "boolean",
						"description": "Include LLM benchmark data (prefill/decode tokens per second)",
						"default": true
					}
				},
				"required": []
			}`),
		},
	},
	{
		Type: "function",
		Function: FunctionDef{
			Name:        "execute_shell_cmd",
			Description: "Execute a shell command on a device. Most commands are allowed, but dangerous patterns (rm -rf, dd, mkfs, etc.) are blocked for safety. Use get_capabilities first to discover device IDs.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"device_id": {
						"type": "string",
						"description": "Target device ID from get_capabilities. Leave empty to use the local/coordinator device."
					},
					"command": {
						"type": "string",
						"description": "The full shell command to execute (e.g., 'df -h', 'ls -la /tmp', 'cat /etc/os-release')"
					},
					"timeout_ms": {
						"type": "integer",
						"description": "Command timeout in milliseconds",
						"default": 30000,
						"minimum": 1000,
						"maximum": 300000
					},
					"working_dir": {
						"type": "string",
						"description": "Working directory for command execution"
					}
				},
				"required": ["command"]
			}`),
		},
	},
	{
		Type: "function",
		Function: FunctionDef{
			Name:        "get_file",
			Description: "Read file contents from a device. Supports reading full files, head (first N bytes), tail (last N bytes), or a specific byte range. Use get_capabilities first to discover device IDs.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"device_id": {
						"type": "string",
						"description": "Target device ID from get_capabilities. Leave empty to read from local/coordinator device."
					},
					"path": {
						"type": "string",
						"description": "File path to read (absolute path or relative to shared directory)"
					},
					"read_mode": {
						"type": "string",
						"enum": ["full", "head", "tail", "range"],
						"description": "How to read the file: full (entire file), head (from beginning), tail (from end), range (specific byte range)",
						"default": "full"
					},
					"max_bytes": {
						"type": "integer",
						"description": "Maximum bytes to read (default 65536, max 10MB)",
						"default": 65536,
						"maximum": 10485760
					},
					"offset": {
						"type": "integer",
						"description": "Byte offset for range mode",
						"default": 0
					},
					"length": {
						"type": "integer",
						"description": "Bytes to read for range mode"
					}
				},
				"required": ["path"]
			}`),
		},
	},
}

// GetToolDefinitions returns a copy of all tool definitions
func GetToolDefinitions() []ToolDefinition {
	result := make([]ToolDefinition, len(ToolDefinitions))
	copy(result, ToolDefinitions)
	return result
}

// GetToolByName returns a tool definition by name, or nil if not found
func GetToolByName(name string) *ToolDefinition {
	for _, t := range ToolDefinitions {
		if t.Function.Name == name {
			return &t
		}
	}
	return nil
}

// ListToolNames returns the names of all available tools
func ListToolNames() []string {
	names := make([]string, len(ToolDefinitions))
	for i, t := range ToolDefinitions {
		names[i] = t.Function.Name
	}
	return names
}
